package bot

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/botkey"
	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/paths"

	"github.com/gin-gonic/gin"
)

type echoBody struct {
	Message string `json:"message"`
}

var nonceCounter atomic.Int64

func nextNonce() string {
	return fmt.Sprintf("nonce-%016d", nonceCounter.Add(1))
}

func newTestRuntime(t *testing.T, mutate func(*config.Bot)) (*botRuntime, ed25519.PrivateKey) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("生成测试密钥失败: %v", err)
	}

	cfg := config.Bot{
		Enabled:               true,
		KeyID:                 "default",
		TimestampToleranceSec: 300,
		NonceCapacity:         1000,
		MaxBodyBytes:          1 << 20,
		RateLimitPerMinute:    0,
	}
	if mutate != nil {
		mutate(&cfg)
	}

	rt := &botRuntime{
		cfg:       cfg,
		keyID:     cfg.KeyID,
		publicKey: pub,
		nonces:    newNonceCache(2*time.Duration(cfg.TimestampToleranceSec)*time.Second, cfg.NonceCapacity),
		limiter:   newRateLimiter(cfg.RateLimitPerMinute),
		now:       time.Now,
	}
	t.Cleanup(func() {
		rt.nonces.Stop()
		rt.limiter.Stop()
	})

	return rt, priv
}

func newTestRouter(rt *botRuntime) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	group := engine.Group(Prefix)
	group.Use(Verify(rt))
	group.Use(RateLimit(rt))

	group.GET("/ping", h.G(handlePing))
	group.POST("/echo", h.B(func(body echoBody, _ *gin.Context) (any, error) {
		return body, nil
	}))

	return engine
}

type requestSpec struct {
	method string
	uri    string
	body   []byte
	ts     int64
	nonce  string
}

func signRequest(priv ed25519.PrivateKey, keyID string, spec requestSpec) http.Header {
	canonical := Canonical(spec.method, spec.uri, keyID, strconv.FormatInt(spec.ts, 10), spec.nonce, spec.body)
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte(canonical)))

	headers := http.Header{}
	headers.Set(headerKeyID, keyID)
	headers.Set(headerTimestamp, strconv.FormatInt(spec.ts, 10))
	headers.Set(headerNonce, spec.nonce)
	headers.Set(headerSignature, signature)
	return headers
}

func doRequest(t *testing.T, engine *gin.Engine, method, uri string, body []byte, headers http.Header) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, uri, bytes.NewReader(body))
	for name, values := range headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	return recorder
}

func TestVerifyAcceptsValidRequest(t *testing.T) {
	rt, priv := newTestRuntime(t, nil)
	engine := newTestRouter(rt)

	headers := signRequest(priv, rt.keyID, requestSpec{
		method: "GET",
		uri:    "/bot-v1/ping",
		ts:     time.Now().Unix(),
		nonce:  nextNonce(),
	})

	recorder := doRequest(t, engine, "GET", "/bot-v1/ping", nil, headers)
	if recorder.Code != http.StatusOK {
		t.Fatalf("合法请求应当通过，状态码 %d，响应 %s", recorder.Code, recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"ok":true`)) {
		t.Fatalf("响应内容不正确: %s", recorder.Body.String())
	}
}

func TestVerifyRejectsInvalidRequests(t *testing.T) {
	rt, priv := newTestRuntime(t, nil)
	engine := newTestRouter(rt)
	now := time.Now().Unix()

	valid := func() http.Header {
		return signRequest(priv, rt.keyID, requestSpec{
			method: "GET",
			uri:    "/bot-v1/ping",
			ts:     now,
			nonce:  nextNonce(),
		})
	}

	t.Run("缺少认证头", func(t *testing.T) {
		recorder := doRequest(t, engine, "GET", "/bot-v1/ping", nil, nil)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，得到 %d", recorder.Code)
		}
	})

	t.Run("key_id 不匹配", func(t *testing.T) {
		headers := valid()
		headers.Set(headerKeyID, "other")
		recorder := doRequest(t, engine, "GET", "/bot-v1/ping", nil, headers)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，得到 %d", recorder.Code)
		}
	})

	t.Run("时间戳过旧", func(t *testing.T) {
		headers := signRequest(priv, rt.keyID, requestSpec{
			method: "GET",
			uri:    "/bot-v1/ping",
			ts:     now - int64(rt.cfg.TimestampToleranceSec) - 1,
			nonce:  nextNonce(),
		})
		recorder := doRequest(t, engine, "GET", "/bot-v1/ping", nil, headers)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，得到 %d", recorder.Code)
		}
	})

	t.Run("时间戳过新", func(t *testing.T) {
		headers := signRequest(priv, rt.keyID, requestSpec{
			method: "GET",
			uri:    "/bot-v1/ping",
			ts:     now + int64(rt.cfg.TimestampToleranceSec) + 1,
			nonce:  nextNonce(),
		})
		recorder := doRequest(t, engine, "GET", "/bot-v1/ping", nil, headers)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，得到 %d", recorder.Code)
		}
	})

	t.Run("nonce 格式错误", func(t *testing.T) {
		headers := valid()
		headers.Set(headerNonce, "short")
		recorder := doRequest(t, engine, "GET", "/bot-v1/ping", nil, headers)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，得到 %d", recorder.Code)
		}
	})

	t.Run("认证头重复", func(t *testing.T) {
		headers := valid()
		headers.Add(headerKeyID, rt.keyID)
		recorder := doRequest(t, engine, "GET", "/bot-v1/ping", nil, headers)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，得到 %d", recorder.Code)
		}
	})

	t.Run("签名无效", func(t *testing.T) {
		headers := valid()
		raw, err := base64.StdEncoding.DecodeString(headers.Get(headerSignature))
		if err != nil {
			t.Fatalf("测试签名解码失败: %v", err)
		}
		raw[0] ^= 0xff
		headers.Set(headerSignature, base64.StdEncoding.EncodeToString(raw))
		recorder := doRequest(t, engine, "GET", "/bot-v1/ping", nil, headers)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，得到 %d", recorder.Code)
		}
	})

	t.Run("URI 被篡改", func(t *testing.T) {
		headers := valid()
		recorder := doRequest(t, engine, "GET", "/bot-v1/ping?x=1", nil, headers)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，得到 %d", recorder.Code)
		}
	})
}

func TestVerifyRejectsTamperedBody(t *testing.T) {
	rt, priv := newTestRuntime(t, nil)
	engine := newTestRouter(rt)

	signedBody := []byte(`{"message":"hello"}`)
	headers := signRequest(priv, rt.keyID, requestSpec{
		method: "POST",
		uri:    "/bot-v1/echo",
		body:   signedBody,
		ts:     time.Now().Unix(),
		nonce:  nextNonce(),
	})
	headers.Set("Content-Type", "application/json")

	recorder := doRequest(t, engine, "POST", "/bot-v1/echo", []byte(`{"message":"evil"}`), headers)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("body 被篡改时应当拒绝，得到 %d", recorder.Code)
	}
}

func TestVerifyRejectsReplay(t *testing.T) {
	rt, priv := newTestRuntime(t, nil)
	engine := newTestRouter(rt)

	headers := signRequest(priv, rt.keyID, requestSpec{
		method: "GET",
		uri:    "/bot-v1/ping",
		ts:     time.Now().Unix(),
		nonce:  nextNonce(),
	})

	first := doRequest(t, engine, "GET", "/bot-v1/ping", nil, headers)
	if first.Code != http.StatusOK {
		t.Fatalf("首次请求应当通过，得到 %d", first.Code)
	}

	second := doRequest(t, engine, "GET", "/bot-v1/ping", nil, headers)
	if second.Code != http.StatusUnauthorized {
		t.Fatalf("重放应当被拒绝，得到 %d", second.Code)
	}
}

func TestVerifyRestoresBodyForBinding(t *testing.T) {
	rt, priv := newTestRuntime(t, nil)
	engine := newTestRouter(rt)

	body := []byte(`{"message":"hello"}`)
	headers := signRequest(priv, rt.keyID, requestSpec{
		method: "POST",
		uri:    "/bot-v1/echo",
		body:   body,
		ts:     time.Now().Unix(),
		nonce:  nextNonce(),
	})
	headers.Set("Content-Type", "application/json")

	recorder := doRequest(t, engine, "POST", "/bot-v1/echo", body, headers)
	if recorder.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d，响应 %s", recorder.Code, recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"message":"hello"`)) {
		t.Fatalf("请求体未被正确绑定: %s", recorder.Body.String())
	}
}

func TestVerifyRejectsOversizedBody(t *testing.T) {
	rt, priv := newTestRuntime(t, func(cfg *config.Bot) {
		cfg.MaxBodyBytes = 8
	})
	engine := newTestRouter(rt)

	body := bytes.Repeat([]byte("x"), 32)
	headers := signRequest(priv, rt.keyID, requestSpec{
		method: "POST",
		uri:    "/bot-v1/echo",
		body:   body,
		ts:     time.Now().Unix(),
		nonce:  nextNonce(),
	})
	headers.Set("Content-Type", "application/json")

	recorder := doRequest(t, engine, "POST", "/bot-v1/echo", body, headers)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("期望 413，得到 %d", recorder.Code)
	}
}

func TestRateLimit(t *testing.T) {
	rt, priv := newTestRuntime(t, func(cfg *config.Bot) {
		cfg.RateLimitPerMinute = 1
	})
	engine := newTestRouter(rt)

	sign := func() http.Header {
		return signRequest(priv, rt.keyID, requestSpec{
			method: "GET",
			uri:    "/bot-v1/ping",
			ts:     time.Now().Unix(),
			nonce:  nextNonce(),
		})
	}

	first := doRequest(t, engine, "GET", "/bot-v1/ping", nil, sign())
	if first.Code != http.StatusOK {
		t.Fatalf("第一次请求应当通过，得到 %d", first.Code)
	}

	second := doRequest(t, engine, "GET", "/bot-v1/ping", nil, sign())
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("第二次请求应当被限流，得到 %d", second.Code)
	}
}

func TestValidateConfig(t *testing.T) {
	original := config.TryGet()
	t.Cleanup(func() { config.Set(original) })

	stateDir := t.TempDir()
	t.Setenv(paths.StateDirEnv, stateDir)
	configDir := filepath.Join(stateDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("创建配置目录失败: %v", err)
	}

	privatePEM, publicPEM, err := botkey.Generate()
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}
	publicPath := filepath.Join(configDir, "bot_public.pem")
	privatePath := filepath.Join(configDir, "bot_private.pem")
	if err := os.WriteFile(publicPath, publicPEM, 0644); err != nil {
		t.Fatalf("写入公钥失败: %v", err)
	}
	if err := os.WriteFile(privatePath, privatePEM, 0600); err != nil {
		t.Fatalf("写入私钥失败: %v", err)
	}

	valid := config.Bot{
		Enabled:       true,
		KeyID:         "default",
		PublicKeyPath: "bot_public.pem",
	}

	t.Run("未启用时直接通过", func(t *testing.T) {
		config.Set(&config.Config{Bot: config.Bot{Enabled: false}})
		if err := ValidateConfig(); err != nil {
			t.Fatalf("未启用时不应当报错: %v", err)
		}
	})

	t.Run("合法配置", func(t *testing.T) {
		config.Set(&config.Config{Bot: valid})
		if err := ValidateConfig(); err != nil {
			t.Fatalf("合法配置不应当报错: %v", err)
		}
	})

	t.Run("绝对路径被拒绝", func(t *testing.T) {
		cfg := valid
		cfg.PublicKeyPath = publicPath
		config.Set(&config.Config{Bot: cfg})
		err := ValidateConfig()
		if err == nil {
			t.Fatal("绝对路径应当被拒绝")
		}
		if !strings.Contains(err.Error(), "绝对路径") {
			t.Errorf("报错未说明不支持绝对路径: %v", err)
		}
	})

	t.Run("公钥文件不存在", func(t *testing.T) {
		cfg := valid
		cfg.PublicKeyPath = "missing.pem"
		config.Set(&config.Config{Bot: cfg})
		if err := ValidateConfig(); err == nil {
			t.Fatal("公钥文件不存在时应当报错")
		}
	})

	t.Run("把私钥当公钥", func(t *testing.T) {
		cfg := valid
		cfg.PublicKeyPath = "bot_private.pem"
		config.Set(&config.Config{Bot: cfg})
		if err := ValidateConfig(); err == nil {
			t.Fatal("私钥文件不应当被接受为公钥")
		}
	})

	t.Run("key_id 不合法", func(t *testing.T) {
		cfg := valid
		cfg.KeyID = "bad id!"
		config.Set(&config.Config{Bot: cfg})
		if err := ValidateConfig(); err == nil {
			t.Fatal("非法 key_id 应当报错")
		}
	})
}
