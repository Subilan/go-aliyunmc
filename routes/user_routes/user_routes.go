package user_routes

import (
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/mid"

	"github.com/gin-gonic/gin"
)

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginRequest 用户登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Remember bool   `json:"remember"`
}

// Bind 注册用户路由
func Bind(router *gin.Engine) {
	userGroup := router.Group("/user")
	{
		// 公开访问的路由（无需登录）
		userGroup.POST("/register", h.B(HandleRegister))
		userGroup.POST("/login", h.B(HandleLogin))
		// 需要登录的路由
		authorized := userGroup.Group("")
		authorized.Use(mid.Auth())
		authorized.Use(mid.Perm())
		{
			authorized.GET("/profile", h.G(HandleGetProfile))
			authorized.GET("/logout", h.G(HandleLogout))
			authorized.POST("/change-password", h.B(HandleChangePassword))
			authorized.DELETE("", h.G(HandleDeleteSelf))
			authorized.POST("/whitelist/bind", h.B(HandleBindWhitelist))
			authorized.POST("/whitelist/unbind", h.G(HandleUnbindWhitelist))
			authorized.GET("/preferences", h.G(HandleGetPreferences))
			authorized.PUT("/preferences", h.B(HandleSetPreferences))
			authorized.GET("/permissions", h.G(HandleGetPermissions))
			authorized.GET("/chat-token", mid.Whitelisted(), h.G(HandleGetChatToken))
			authorized.GET("/s", h.Q(HandleListUsers))
			authorized.POST("/ban", h.B(HandleBanUser))
			authorized.POST("/unban", h.B(HandleUnbanUser))
		}
	}
}
