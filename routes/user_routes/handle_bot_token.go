package user_routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/mid"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type botTokenRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// HandleBotToken 供 Bot 用 X-Bot-Key + 用户名密码换取长期 JWT。
func HandleBotToken(req botTokenRequest, c *gin.Context) (any, error) {
	if !mid.IsBotRequest(c) {
		return nil, h.HttpError(http.StatusForbidden, "无效的Bot密钥")
	}

	var user models.User
	if err := store.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return nil, h.HttpError(http.StatusUnauthorized, "用户名或密码错误")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, h.HttpError(http.StatusUnauthorized, "用户名或密码错误")
	}

	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     string(user.Role),
		"exp":      time.Now().Add(time.Duration(C.BotToken.ExpireSeconds) * time.Second).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(C.BotToken.Secret))
	if err != nil {
		return nil, fmt.Errorf("生成令牌失败：%v", err)
	}

	return gin.H{
		"token":          tokenString,
		"username":       user.Username,
		"role":           user.Role,
		"whitelist_uuid": user.WhitelistUUID,
	}, nil
}
