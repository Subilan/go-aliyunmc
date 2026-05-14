package user_routes

import (
	"github.com/Subilan/go-aliyunmc/context_util"
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleSetPreferences 设置当前用户的偏好
func HandleSetPreferences(prefs models.Preferences, c *gin.Context) (any, error) {
	userID, ok := context_util.GetUserID(c)
	
	if !ok {
		return nil, h.HttpError(http.StatusUnauthorized, "未登录")
	}

	if err := store.SetUserPreferences(userID, prefs); err != nil {
		return nil, err
	}

	return nil, nil
}
