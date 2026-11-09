package user_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/context_util"
	"github.com/Subilan/go-aliyunmc/internal/h"
	"github.com/Subilan/go-aliyunmc/internal/perms"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateUserRequest 管理员创建用户请求。
type CreateUserRequest struct {
	Username string     `json:"username"`
	Password string     `json:"password"`
	Role     perms.Role `json:"role"`
}

// HandleCreateUser 由管理员创建用户并指定初始权限。
func HandleCreateUser(req CreateUserRequest, c *gin.Context) (any, error) {
	// 登录与固定路由权限已由 Auth、Perm 中间件校验。
	currentRole, _ := context_util.GetUserRole(c)
	if !req.Role.IsValid() {
		return nil, h.HttpError(http.StatusBadRequest, "无效的权限等级")
	}
	if !currentRole.Gte(req.Role) {
		return nil, h.HttpError(http.StatusForbidden, "不能创建权限高于自己的用户")
	}

	return createUser(req.Username, req.Password, req.Role)
}
