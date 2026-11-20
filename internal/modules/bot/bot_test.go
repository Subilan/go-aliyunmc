package bot

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/h"

	"github.com/gin-gonic/gin"
)

func newTestRouter(t *testing.T, mutate func(*config.Bot)) *gin.Engine {
	t.Helper()

	cfg := config.Bot{Enabled: true, RateLimitPerMinute: 0}
	if mutate != nil {
		mutate(&cfg)
	}

	rt := &botRuntime{limiter: newRateLimiter(cfg.RateLimitPerMinute)}
	t.Cleanup(rt.limiter.Stop)

	gin.SetMode(gin.TestMode)
	engine := gin.New()

	group := engine.Group(Prefix)
	group.Use(RequireLoopback())
	group.Use(RateLimit(rt))
	group.GET("/ping", h.G(handlePing))

	return engine
}

func doRequest(t *testing.T, engine *gin.Engine, remoteAddr string, headers http.Header) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest("GET", "/bot-v1/ping", nil)
	req.RemoteAddr = remoteAddr
	for name, values := range headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	return recorder
}

func TestRequireLoopbackAcceptsIPv4Loopback(t *testing.T) {
	engine := newTestRouter(t, nil)

	recorder := doRequest(t, engine, "127.0.0.1:54321", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("本机请求应当通过，状态码 %d，响应 %s", recorder.Code, recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"ok":true`)) {
		t.Fatalf("响应内容不正确: %s", recorder.Body.String())
	}
}

func TestRequireLoopbackAcceptsIPv6Loopback(t *testing.T) {
	engine := newTestRouter(t, nil)

	recorder := doRequest(t, engine, "[::1]:54321", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("IPv6 本机请求应当通过，状态码 %d，响应 %s", recorder.Code, recorder.Body.String())
	}
}

func TestRequireLoopbackRejectsExternalAddress(t *testing.T) {
	engine := newTestRouter(t, nil)

	for _, remoteAddr := range []string{"203.0.113.10:54321", "[2001:db8::1]:54321"} {
		recorder := doRequest(t, engine, remoteAddr, nil)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("%s 应当被拒绝，得到 %d", remoteAddr, recorder.Code)
		}
	}
}

func TestRequireLoopbackRejectsInvalidRemoteAddr(t *testing.T) {
	engine := newTestRouter(t, nil)

	recorder := doRequest(t, engine, "not-an-address", nil)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("无法解析的来源地址应当被拒绝，得到 %d", recorder.Code)
	}
}

func TestRequireLoopbackRejectsForwardedRequests(t *testing.T) {
	engine := newTestRouter(t, nil)

	for _, header := range []string{"Forwarded", "X-Forwarded-For", "X-Real-IP"} {
		headers := http.Header{}
		headers.Set(header, "203.0.113.10")
		recorder := doRequest(t, engine, "127.0.0.1:54321", headers)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("带 %s 的代理请求应当被拒绝，得到 %d", header, recorder.Code)
		}
	}
}

func TestRateLimit(t *testing.T) {
	engine := newTestRouter(t, func(cfg *config.Bot) {
		cfg.RateLimitPerMinute = 1
	})

	first := doRequest(t, engine, "127.0.0.1:54321", nil)
	if first.Code != http.StatusOK {
		t.Fatalf("第一次请求应当通过，得到 %d", first.Code)
	}

	second := doRequest(t, engine, "127.0.0.1:54321", nil)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("第二次请求应当被限流，得到 %d", second.Code)
	}
}
