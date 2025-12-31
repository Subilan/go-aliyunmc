package auth

import (
	"net/http"
	"time"

	"github.com/Subilan/gomc-server/config"
	"github.com/Subilan/gomc-server/globals"
	"github.com/Subilan/gomc-server/helpers"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type GetTokenRequest struct {
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
	KeepAlive bool   `json:"keepAlive"`
}

type TokenClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func GetToken() gin.HandlerFunc {
	return helpers.BodyHandler[GetTokenRequest](func(body GetTokenRequest, c *gin.Context) (any, error) {
		// 查询用户信息
		row := globals.Pool.QueryRowContext(c, "SELECT id, username, password_hash FROM users WHERE username = ?", body.Username)

		var id int64
		var username string
		var passwordHash string

		err := row.Scan(&id, &username, &passwordHash)
		if err != nil {
			return nil, &helpers.HttpError{
				Code:    http.StatusUnauthorized,
				Details: "用户名或密码错误",
			}
		}

		// 验证密码
		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(body.Password)); err != nil {
			return nil, &helpers.HttpError{
				Code:    http.StatusUnauthorized,
				Details: "用户名或密码错误",
			}
		}

		// 设置token过期时间
		expirationTime := time.Hour * 24 * 7 // 默认7天
		if body.KeepAlive {
			expirationTime = time.Hour * 24 * 30 // 如果KeepAlive为true，则为30天
		}

		// 创建JWT token
		claims := &TokenClaims{
			UserID:   id,
			Username: username,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(expirationTime)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Issuer:    "gomc-server",
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(config.Cfg.Server.JwtSecret))
		if err != nil {
			return nil, err
		}

		return helpers.Data(tokenString), nil
	})
}
