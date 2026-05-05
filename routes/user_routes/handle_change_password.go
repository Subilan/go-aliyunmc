package user_routes

import (
	"go-aliyunmc/context_util"
	"go-aliyunmc/h"
	"go-aliyunmc/store"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type changePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

func HandleChangePassword(req changePasswordRequest, c *gin.Context) (any, error) {
	// 从context中获取当前登录用户
	currentUser, exists := context_util.GetUser(c)

	if !exists {
		return nil, h.HttpError(http.StatusUnauthorized, "未登录")
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(currentUser.PasswordHash), []byte(req.OldPassword)); err != nil {
		return nil, h.HttpError(http.StatusUnauthorized, "原密码不匹配")
	}

	// 更新为新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	currentUser.PasswordHash = string(hashedPassword)
	if err := store.DB.Save(&currentUser).Error; err != nil {
		return nil, err
	}

	return nil, nil
}
