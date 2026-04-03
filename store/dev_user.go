package store

import (
	"errors"
	"go-aliyunmc/store/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const DevUsername = "dev"
const DevPassword = "dev"
const DevRole = "superuser"

// EnsureDevUser 确保固定的 DEV 用户存在，不存在时自动创建。
func EnsureDevUser() (*models.User, error) {
	var user models.User
	if err := DB.Where("username = ?", DevUsername).First(&user).Error; err == nil {
		return &user, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(DevPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user = models.User{
		Username:     DevUsername,
		PasswordHash: string(hashedPassword),
		Role:         DevRole,
	}

	if err := DB.Create(&user).Error; err != nil {
		// 处理并发创建场景：若已被其他请求创建，则直接读取并返回。
		var existing models.User
		if readErr := DB.Where("username = ?", DevUsername).First(&existing).Error; readErr == nil {
			return &existing, nil
		}
		return nil, err
	}

	return &user, nil
}
