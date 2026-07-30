package user_routes

import (
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"
	"net/http"
	"time"

	"github.com/Subilan/go-aliyunmc/session"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// HandleLogin 用户登录处理函数
func HandleLogin(req LoginRequest, c *gin.Context) (any, error) {
	// 检查是否已登录，防止重复登录
	if _, exists := session.GetUserID(c); exists {
		return nil, h.HttpError(http.StatusBadRequest, "你已经登录了")
	}

	// 查找用户
	var user models.User
	if err := store.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return nil, err
	}

	// 检查是否被封禁
	if user.Banned {
		return nil, h.HttpError(http.StatusForbidden, "账号已被封禁")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, h.HttpError(http.StatusUnauthorized, "用户名或密码错误")
	}

	session.Set(c, session.KeyUserId, user.ID)
	session.Set(c, session.KeyUsername, user.Username)

	if req.Remember {
		session.SetDeadline(c, 7*24*time.Hour)
	}

	return nil, nil
}
