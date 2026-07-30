package mid

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Subilan/go-aliyunmc/env"
	"github.com/Subilan/go-aliyunmc/session"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// BotKey 是机器人的预共享密钥
var BotKey string

// BotTokenSecret 是机器人 JWT 的签名密钥。
var BotTokenSecret string

// InitBotAuth 由 main.go 在启动时调用，设置 bot 鉴权所需的参数。
func InitBotAuth(key, secret string) {
	BotKey = key
	BotTokenSecret = secret
}

// IsBotRequest 判断当前请求是否来自 bot（带有正确的 X-Bot-Key 头）。
func IsBotRequest(c *gin.Context) bool {
	return BotKey != "" && c.GetHeader("X-Bot-Key") == BotKey
}

func handleBotRequest(c *gin.Context, botKey string) {
	if BotKey == "" || botKey != BotKey {
		c.JSON(http.StatusForbidden, gin.H{"error": "无效的Bot密钥"})
		c.Abort()
		return
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少Bearer令牌"})
		c.Abort()
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	claims := &jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(BotTokenSecret), nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "令牌无效或已过期"})
		c.Abort()
		return
	}

	userIDFloat, ok := (*claims)["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "令牌格式无效"})
		c.Abort()
		return
	}

	var user models.User
	if err := store.DB.First(&user, uint(userIDFloat)).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		c.Abort()
		return
	}

	if user.Banned {
		c.JSON(http.StatusForbidden, gin.H{"error": "账号已被封禁"})
		c.Abort()
		return
	}

	c.Set("user", &user)
	c.Set("user_id", user.ID)
	c.Next()
}

// Auth 认证中间件，用于检查用户是否已登录。
// 支持两种鉴权通道：
//  1. Bot JWT 通道：请求带 X-Bot-Key + Authorization: Bearer <jwt>，由本中间件验证。
//  2. Session 通道：无 X-Bot-Key 头时走原有的 session cookie 鉴权。
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Bot JWT 鉴权通道（优先于 session / DEV 模式）
		if botKey := c.GetHeader("X-Bot-Key"); botKey != "" {
			handleBotRequest(c, botKey)
			return
		}

		// 原有 session 鉴权通道
		if env.DEV {
			user, err := store.EnsureDevUser()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "DEV用户初始化失败"})
				c.Abort()
				return
			}

			if user.Banned {
				c.JSON(http.StatusForbidden, gin.H{"error": "账号已被封禁"})
				c.Abort()
				return
			}

			c.Set("user", user)
			c.Set("user_id", user.ID)
			c.Next()
			return
		}

		// 从session中获取用户信息
		userID, exists := session.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			c.Abort()
			return
		}

		// 根据用户ID获取用户信息
		var user models.User
		if err := store.DB.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
			c.Abort()
			return
		}

		if user.Banned {
			c.JSON(http.StatusForbidden, gin.H{"error": "账号已被封禁"})
			c.Abort()
			return
		}

		// 将用户信息存储到context中
		c.Set("user", &user)
		c.Set("user_id", userID)

		c.Next()
	}
}
