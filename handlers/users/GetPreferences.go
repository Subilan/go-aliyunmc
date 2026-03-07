package users

import (
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/gctx"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

// HandleGetPreferences 获取当前用户的所有偏好设置
//
//	@Summary		获取当前用户所有偏好
//	@Description	返回当前用户的所有偏好设置记录，格式为 map[key]value
//	@Tags			users
//	@Produce		json
//	@Success		200		{object}	helpers.DataResp[map[string]string]
//	@Failure		401		{object}	helpers.ErrorResp
//	@Failure		500		{object}	helpers.ErrorResp
//	@Router			/user/preferences [get]
func HandleGetPreferences() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		user, exists := gctx.ShouldGetUser(c)
		if !exists {
			return nil, &helpers.HttpError{Code: 401, Details: "未授权"}
		}

		prefs, err := store.GetAllUserPreferences(user.Username)
		if err != nil {
			return nil, err
		}

		// 转换为 map[string]string
		result := make(map[string]string)
		for _, pref := range prefs {
			result[string(pref.Key)] = pref.Value
		}

		return helpers.Data(result), nil
	})
}
