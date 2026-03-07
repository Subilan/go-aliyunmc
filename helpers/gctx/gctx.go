package gctx

import (
	"net/http"
	"strconv"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

func MustOwnUserId(id string, c *gin.Context) error {
	inputId, err := strconv.ParseInt(id, 10, 64)

	if err != nil {
		return &helpers.HttpError{Code: http.StatusBadRequest, Details: "输入 ID 无效"}
	}

	userId, err := ShouldGetUserId(c)

	if err != nil {
		return err
	}

	if userId != inputId {
		return &helpers.HttpError{Code: http.StatusForbidden, Details: "用户不匹配"}
	}

	return nil
}

func ShouldGetUserId(c *gin.Context) (int64, error) {
	currentUserId, exists := c.Get("user_id")

	if !exists {
		return 0, &helpers.HttpError{Code: http.StatusForbidden, Details: "找不到用户凭据"}
	}

	result, ok := currentUserId.(int64)

	if !ok {
		return 0, &helpers.HttpError{Code: http.StatusForbidden, Details: "用户 ID 无效"}
	}

	return result, nil
}

// ShouldGetUser 从上下文中获取当前用户信息
func ShouldGetUser(c *gin.Context) (*store.User, bool) {
	userId, err := ShouldGetUserId(c)
	if err != nil {
		return nil, false
	}

	var user store.User
	err = db.Pool.QueryRow("SELECT u.id, u.username, u.created_at, r.role FROM users u JOIN user_roles r ON u.id = r.user_id WHERE u.id = ?", userId).
		Scan(&user.Id, &user.Username, &user.CreatedAt, &user.Role)

	if err != nil {
		return nil, false
	}

	return &user, true
}
