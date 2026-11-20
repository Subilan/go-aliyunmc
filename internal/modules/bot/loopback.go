package bot

import (
	"net"
	"net/http"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/log_util"

	"github.com/gin-gonic/gin"
)

// RequireLoopback 只放行直接来自本机回环地址的请求。
// 携带转发头的请求视为经过反向代理，即使代理与服务同机也拒绝，
// 避免 /bot-v1 通过反向代理暴露到外部。
func RequireLoopback() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isLoopbackRemote(c.Request.RemoteAddr) || hasForwardedHeaders(c.Request) {
			log_util.Warn("机器人接口拒绝非本机请求: method=%s path=%s remote_addr=%s client_ip=%s",
				c.Request.Method, c.Request.URL.Path, c.Request.RemoteAddr, c.ClientIP())
			c.AbortWithStatusJSON(http.StatusForbidden, h.DetailsF("仅允许本机访问"))
			return
		}
		c.Next()
	}
}

// isLoopbackRemote 判断 RemoteAddr 是否为本机回环地址。
func isLoopbackRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// hasForwardedHeaders 判断请求是否经过反向代理转发。
func hasForwardedHeaders(r *http.Request) bool {
	return r.Header.Get("Forwarded") != "" ||
		r.Header.Get("X-Forwarded-For") != "" ||
		r.Header.Get("X-Real-IP") != ""
}
