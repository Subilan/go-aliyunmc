package user_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/common/context_util"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleSetPreferences 设置当前用户的偏好
func HandleSetPreferences(prefs labmodels.Preferences, c *gin.Context) (any, error) {
	userID, ok := context_util.GetUserID(c)

	if !ok {
		return nil, h.HttpError(http.StatusUnauthorized, "未登录")
	}

	if err := dao.SetUserPreferences(userID, prefs); err != nil {
		return nil, err
	}

	return nil, nil
}
