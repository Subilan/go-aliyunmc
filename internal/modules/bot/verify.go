package bot

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/log_util"

	"github.com/gin-gonic/gin"
)

const (
	headerKeyID     = "X-Bot-Key-Id"
	headerTimestamp = "X-Bot-Timestamp"
	headerNonce     = "X-Bot-Nonce"
	headerSignature = "X-Bot-Signature"
)

var (
	timestampPattern = regexp.MustCompile(`^[0-9]{1,20}$`)
	noncePattern     = regexp.MustCompile(`^[A-Za-z0-9_-]{16,128}$`)
)

// Verify 验证机器人请求签名、时间窗口与 nonce 重放，并把 principal 写入 context。
func Verify(rt *botRuntime) gin.HandlerFunc {
	return func(c *gin.Context) {
		keyID, ok := singleHeader(c, headerKeyID)
		if !ok {
			reject(c, "缺少或重复的 X-Bot-Key-Id")
			return
		}
		if keyID != rt.keyID {
			reject(c, "key_id 不匹配")
			return
		}

		timestamp, ok := singleHeader(c, headerTimestamp)
		if !ok || !timestampPattern.MatchString(timestamp) {
			reject(c, "X-Bot-Timestamp 缺失或格式错误")
			return
		}
		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil || abs64(rt.now().Unix()-ts) > int64(rt.cfg.TimestampToleranceSec) {
			reject(c, "时间戳超出容差")
			return
		}

		nonce, ok := singleHeader(c, headerNonce)
		if !ok || !noncePattern.MatchString(nonce) {
			reject(c, "X-Bot-Nonce 缺失或格式错误")
			return
		}

		signature, ok := singleHeader(c, headerSignature)
		if !ok {
			reject(c, "X-Bot-Signature 缺失或重复")
			return
		}
		sig, err := base64.StdEncoding.DecodeString(signature)
		if err != nil || len(sig) != ed25519.SignatureSize {
			reject(c, "签名格式错误")
			return
		}

		body, ok := readBody(c, rt.cfg.MaxBodyBytes)
		if !ok {
			return
		}

		canonical := Canonical(c.Request.Method, requestURI(c.Request), keyID, timestamp, nonce, body)
		if !ed25519.Verify(rt.publicKey, []byte(canonical), sig) {
			reject(c, "签名无效")
			return
		}

		if !rt.nonces.CheckAndStore(keyID + ":" + nonce) {
			reject(c, "nonce 重放")
			return
		}

		SetPrincipal(c, &Principal{KeyID: keyID, Scopes: rt.scopes})
		c.Next()
	}
}

// readBody 读取请求体并按原样恢复，供后续 h.B[T] 绑定。
func readBody(c *gin.Context, maxBytes int64) ([]byte, bool) {
	if c.Request.Body == nil {
		return nil, true
	}

	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, h.DetailsF("请求体过大"))
			return nil, false
		}
		reject(c, "读取请求体失败")
		return nil, false
	}

	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	return body, true
}

func singleHeader(c *gin.Context, name string) (string, bool) {
	values := c.Request.Header.Values(name)
	if len(values) != 1 {
		return "", false
	}
	return values[0], true
}

func requestURI(r *http.Request) string {
	if r.RequestURI != "" {
		return r.RequestURI
	}
	return r.URL.RequestURI()
}

func reject(c *gin.Context, reason string) {
	log_util.Warn("机器人认证失败: reason=%s method=%s path=%s client_ip=%s",
		reason, c.Request.Method, c.Request.URL.Path, c.ClientIP())
	c.AbortWithStatusJSON(http.StatusUnauthorized, h.DetailsF("机器人认证失败"))
}

func abs64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}
