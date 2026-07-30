package user_routes

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"

	"github.com/gin-gonic/gin"
)

// ListUsersQuery 用户列表查询参数
type ListUsersQuery struct {
	Banned   string `form:"banned"`   // "true" / "false" / ""（全部）
	Username string `form:"username"` // 模糊搜索
	Sort     string `form:"sort"`
	Order    string `form:"order"`
	Limit    int    `form:"limit"`
	Offset   int    `form:"offset"`
}

// ListUsersResponse 用户列表响应
type ListUsersResponse struct {
	Users []models.User `json:"users"`
	Total int64         `json:"total"`
}

// HandleListUsers 查询用户列表（仅 operator 及以上可访问）
func HandleListUsers(query ListUsersQuery, c *gin.Context) (any, error) {
	if query.Limit <= 0 || query.Limit > 100 {
		query.Limit = 20
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	if query.Order != "asc" && query.Order != "desc" {
		query.Order = "desc"
	}

	var banned *bool
	switch query.Banned {
	case "true":
		t := true
		banned = &t
	case "false":
		f := false
		banned = &f
	}

	total, err := store.CountUsers(banned, query.Username)
	if err != nil {
		return nil, h.HttpError(http.StatusInternalServerError, "查询用户总数失败")
	}

	users, err := store.ListUsers(banned, query.Username, query.Sort, query.Order, query.Limit, query.Offset)
	if err != nil {
		return nil, err
	}

	return ListUsersResponse{Users: users, Total: total}, nil
}
