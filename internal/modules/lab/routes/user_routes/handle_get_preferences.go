package user_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/common/context_util"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/dao"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleGetPreferences 获取当前用户的偏好设置
func HandleGetPreferences(c *gin.Context) (any, error) {
	userID, ok := context_util.GetUserID(c)
	if !ok {
		return nil, h.HttpError(http.StatusUnauthorized, "未登录")
	}

	prefs, err := dao.GetUserPreferences(userID)
	if err != nil {
		return nil, err
	}

	return prefs, nil
}
