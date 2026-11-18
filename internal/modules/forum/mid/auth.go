// Package mid 提供论坛的认证、权限与限流中间件。
package mid

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/context_util"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/session_util"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/perms"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/session"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/settings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Auth 要求论坛前台登录。
func Auth() gin.HandlerFunc {
	return authenticate(session.Manager, true)
}

// OptionalAuth 尝试解析论坛前台登录态，游客继续访问。
func OptionalAuth() gin.HandlerFunc {
	return authenticate(session.Manager, false)
}

// sessionUser 读取 session 中的用户。用户不存在时清除 session；
// 数据库错误原样返回，由调用方决定如何处理。
func sessionUser(c *gin.Context, manager *session_util.Manager) (*models.User, error) {
	userID, ok := manager.GetUserID(c)
	if !ok {
		return nil, nil
	}

	var user models.User
	err := store.DB.First(&user, userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		_ = manager.Destroy(c)
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// AdminAuth 要求管理后台登录，并且至少拥有一个 admin.access。
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := session.AdminManager
		user, err := sessionUser(c, manager)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, h.DetailsF("读取用户失败"))
			return
		}
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, h.DetailsF("未登录"))
			return
		}

		if user.Banned {
			_ = manager.Destroy(c)
			c.AbortWithStatusJSON(http.StatusForbidden, h.DetailsF("账号已被封禁"))
			return
		}

		if !perms.CanAccessAdmin(user) {
			c.AbortWithStatusJSON(http.StatusForbidden, h.DetailsF("无管理权限"))
			return
		}

		c.Set(context_util.UserKey, user)
		c.Set(context_util.UserIdKey, user.ID)
		c.Next()
	}
}

func authenticate(manager *session_util.Manager, required bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := sessionUser(c, manager)
		if err != nil || user == nil {
			if required {
				c.AbortWithStatusJSON(http.StatusUnauthorized, h.DetailsF("未登录"))
				return
			}
			c.Next()
			return
		}

		if user.Banned {
			if required {
				c.AbortWithStatusJSON(http.StatusForbidden, h.DetailsF("账号已被封禁"))
				return
			}
			c.Next()
			return
		}

		c.Set(context_util.UserKey, user)
		c.Set(context_util.UserIdKey, user.ID)
		c.Next()
	}
}

// RequirePermission 要求当前用户拥有指定全局权限点。
func RequirePermission(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := context_util.GetUser(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, h.DetailsF("未登录"))
			return
		}

		if !perms.Allow(user, "*", code) {
			c.AbortWithStatusJSON(http.StatusForbidden, h.DetailsF("权限不足"))
			return
		}

		c.Next()
	}
}

// OptionalRead 在 allow_guest_read=false 时要求游客登录。
// 必须放在 OptionalAuth 之后。
func OptionalRead() gin.HandlerFunc {
	return func(c *gin.Context) {
		if settings.AllowGuestRead() {
			c.Next()
			return
		}

		if _, ok := context_util.GetUser(c); ok {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, h.DetailsF("请先登录"))
	}
}

// CurrentUser 返回当前请求用户。
func CurrentUser(c *gin.Context) (*models.User, bool) {
	return context_util.GetUser(c)
}

// RequireAnyPermission 要求当前用户拥有给定权限点中的任意一个（全局范围）。
func RequireAnyPermission(codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := context_util.GetUser(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, h.DetailsF("未登录"))
			return
		}

		for _, code := range codes {
			if perms.Allow(user, "*", code) {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, h.DetailsF("权限不足"))
	}
}

// OriginCheck 对带 Origin 的写请求校验来源：同源请求始终放行，
// 跨源请求必须出现在 CORS 允许列表中。未带 Origin 的请求（curl、服务端调用）不强制校验。
// 配置中的 "*" 不会被视为通配，避免 CSRF 防护被错误配置绕过。
func OriginCheck() gin.HandlerFunc {
	allowed := config.Get().Base.Cors.AllowOrigins

	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		origin := strings.TrimRight(c.GetHeader("Origin"), "/")
		if origin == "" {
			c.Next()
			return
		}

		if sameOrigin(c, origin) {
			c.Next()
			return
		}

		for _, candidate := range allowed {
			candidate = strings.TrimRight(candidate, "/")
			if candidate == "" || candidate == "*" {
				continue
			}
			if strings.EqualFold(candidate, origin) {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, h.DetailsF("来源不被允许"))
	}
}

// sameOrigin 判断 Origin 是否与当前请求同源（忽略 scheme，允许反向代理改写协议）。
func sameOrigin(c *gin.Context, origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	return strings.EqualFold(parsed.Host, c.Request.Host)
}
