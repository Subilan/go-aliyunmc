package user_routes

import (
	"go-aliyunmc/h"
	"go-aliyunmc/perms"
	"go-aliyunmc/store"
	"go-aliyunmc/store/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// HandleRegister 用户注册处理函数
func HandleRegister(req RegisterRequest, c *gin.Context) (any, error) {
	// 检查用户名是否已存在
	var existingUser models.User
	if err := store.DB.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		return nil, h.HttpError(http.StatusConflict, "用户名已存在")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 创建用户
	user := models.User{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Role:         perms.RoleBasic,
	}

	if err := store.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	return nil, nil
}
