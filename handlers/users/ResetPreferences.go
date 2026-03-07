package users

import (
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/gctx"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

// HandleResetPreferences 重置用户的所有偏好设置（删除所有记录）
//
//	@Summary		重置所有用户偏好
//	@Description	删除用户的所有偏好设置记录，使其恢复为默认值
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Success		200		{object}	gin.H
//	@Failure		401		{object}	helpers.ErrorResp
//	@Failure		500		{object}	helpers.ErrorResp
//	@Router			/user/preferences/reset [post]
func HandleResetPreferences() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		user, exists := gctx.ShouldGetUser(c)
		if !exists {
			return nil, &helpers.HttpError{Code: 401, Details: "未授权"}
		}

		err := store.DeleteAllUserPreferences(user.Username)
		if err != nil {
			return nil, err
		}

		return gin.H{}, nil
	})
}
