package mid

import (
	"github.com/Subilan/go-aliyunmc/internal/common/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/session"

	"github.com/gin-gonic/gin"
)

// Auth 认证中间件，使用 lab 专属的 session cookie。
func Auth() gin.HandlerFunc {
	return mid.Auth(session.Manager)
}
