package user_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/context_util"
	"github.com/Subilan/go-aliyunmc/internal/h"
	"github.com/Subilan/go-aliyunmc/internal/store"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleGetPreferences 获取当前用户的偏好设置
func HandleGetPreferences(c *gin.Context) (any, error) {
	userID, ok := context_util.GetUserID(c)
	if !ok {
		return nil, h.HttpError(http.StatusUnauthorized, "未登录")
	}

	prefs, err := store.GetUserPreferences(userID)
	if err != nil {
		return nil, err
	}

	return prefs, nil
}
