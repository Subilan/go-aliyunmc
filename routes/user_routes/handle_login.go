package user_routes

import (
	"go-aliyunmc/env"
	"go-aliyunmc/h"
	"go-aliyunmc/log_util"
	"go-aliyunmc/store"
	"go-aliyunmc/store/models"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// HandleLogin 用户登录处理函数
func HandleLogin(req LoginRequest, c *gin.Context) (any, error) {
	// 查找用户
	var user models.User
	if err := store.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return nil, err
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, h.HttpError(http.StatusUnauthorized, "用户名或密码错误")
	}

	// 设置session
	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Set("username", user.Username)

	// 如果选择"记住我"，设置session过期时间为7天
	if req.Remember {
		session.Options(sessions.Options{
			MaxAge: 7 * 24 * 60 * 60, // 7天
		})
	}

	if err := session.Save(); err != nil {
		return nil, err
	}

	// 在DEV模式下输出登录信息
	if env.DEV {
		log_util.Dev("用户登录: ID=%d, 用户名=%s, 角色=%s, 记住我=%v",
			user.ID, user.Username, user.Role, req.Remember)
	}
	return nil, nil
}
