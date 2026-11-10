package user_routes

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/internal/common/context_util"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/session"

	"github.com/gin-gonic/gin"
)

// HandleDeleteSelf 注销用户处理函数
func HandleDeleteSelf(c *gin.Context) (any, error) {
	// 从context中获取当前登录用户
	currentUser, exists := context_util.GetUser(c)

	if !exists {
		return nil, h.HttpError(http.StatusUnauthorized, "未登录")
	}

	// 删除用户
	if err := store.DB.Delete(&currentUser).Error; err != nil {
		return nil, err
	}

	session.Logout(c)

	return nil, nil
}
