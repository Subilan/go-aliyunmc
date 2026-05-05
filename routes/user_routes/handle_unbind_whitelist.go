package user_routes

import (
	"net/http"

	"go-aliyunmc/context_util"
	"go-aliyunmc/h"
	"go-aliyunmc/store"

	"github.com/gin-gonic/gin"
)

// HandleUnbindWhitelist 解除当前用户与白名单项目的绑定
func HandleUnbindWhitelist(c *gin.Context) (any, error) {
	user, exists := context_util.GetUser(c)
	if !exists {
		return nil, h.HttpError(http.StatusUnauthorized, "未登录")
	}

	if user.WhitelistUUID == nil {
		return nil, h.HttpError(http.StatusBadRequest, "未绑定白名单")
	}

	user.WhitelistUUID = nil
	if err := store.DB.Save(user).Error; err != nil {
		return nil, err
	}

	return nil, nil
}
