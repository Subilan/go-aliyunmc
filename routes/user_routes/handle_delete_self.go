package user_routes

import (
	"github.com/Subilan/go-aliyunmc/context_util"
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/store"
	"net/http"

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
	
	context_util.Logout(c)

	return nil, nil
}
