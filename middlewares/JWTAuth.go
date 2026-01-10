package middlewares

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/handlers/auth"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, helpers.Details("请求头中缺少Authorization字段"))
			c.Abort()
			return
		}

		// Bearer <token> 格式
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, helpers.Details("Authorization头格式错误"))
			c.Abort()
			return
		}

		tokenString := parts[1]

		claims := &auth.TokenClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.Cfg.Base.JwtSecret), nil
		})

		if err != nil {
			if errors.Is(err, jwt.ErrSignatureInvalid) {
				c.JSON(http.StatusUnauthorized, helpers.Details("无效的JWT签名"))
				c.Abort()
				return
			}
			c.JSON(http.StatusUnauthorized, helpers.Details("无法解析JWT令牌: "+err.Error()))
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, helpers.Details("JWT令牌无效"))
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中，供后续处理器使用
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}
