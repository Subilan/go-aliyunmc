package user_routes

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/Subilan/go-aliyunmc/internal/h"
	"github.com/Subilan/go-aliyunmc/internal/perms"
	"github.com/Subilan/go-aliyunmc/internal/store"
	"github.com/Subilan/go-aliyunmc/internal/store/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	validUsernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{4,12}$`)
	hasLetterPattern     = regexp.MustCompile(`[A-Za-z]`)
	hasDigitPattern      = regexp.MustCompile(`[0-9]`)
)

// HandleRegister 用户注册处理函数
func HandleRegister(req RegisterRequest, c *gin.Context) (any, error) {
	return createUser(req.Username, req.Password, perms.RoleBasic)
}

func createUser(username, password string, role perms.Role) (*models.User, error) {
	if !validUsernamePattern.MatchString(username) {
		return nil, h.HttpError(http.StatusBadRequest, "用户名需为 4~12 个字符，且只能包含英文字母、数字和下划线")
	}
	if len(password) < 8 || len(password) > 20 ||
		!hasLetterPattern.MatchString(password) ||
		!hasDigitPattern.MatchString(password) {
		return nil, h.HttpError(http.StatusBadRequest, "密码长度需为 8~20 个字符，且必须包含英文字母和数字")
	}

	// 检查用户名是否已存在
	var existingUser models.User
	err := store.DB.Where("username = ?", username).First(&existingUser).Error
	if err == nil {
		return nil, h.HttpError(http.StatusConflict, "用户名已存在")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 创建用户
	user := models.User{
		Username:     username,
		PasswordHash: string(hashedPassword),
		Role:         role,
	}

	if err := store.DB.Create(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, h.HttpError(http.StatusConflict, "用户名已存在")
		}
		return nil, err
	}

	return &user, nil
}
