package users

import (
	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/gctx"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

type UpsertPreferenceRequest struct {
	Key   consts.UserPreferenceKey `json:"key" binding:"required"`
	Value string                   `json:"value" binding:"required"`
}

// HandleUpsertPreference 设置或更新用户的偏好设置
//
//	@Summary		设置或更新用户偏好
//	@Description	设置或更新用户的偏好设置，如果记录已存在则更新，不存在则插入
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			body	body	UpsertPreferenceRequest	true	"偏好设置"
//	@Success		200		{object}	gin.H
//	@Failure		400		{object}	helpers.ErrorResp
//	@Failure		500		{object}	helpers.ErrorResp
//	@Router			/user/preferences [put]
func HandleUpsertPreference() gin.HandlerFunc {
	return helpers.BodyHandler[UpsertPreferenceRequest](func(body UpsertPreferenceRequest, c *gin.Context) (any, error) {
		user, exists := gctx.ShouldGetUser(c)
		if !exists {
			return nil, &helpers.HttpError{Code: 401, Details: "未授权"}
		}

		// 检查 key 是否为有效的枚举值
		if !body.Key.Valid() {
			return nil, &helpers.HttpError{Code: 400, Details: "无效的偏好设置键"}
		}

		err := store.SetUserPreference(user.Username, body.Key, body.Value)
		if err != nil {
			return nil, err
		}

		return gin.H{}, nil
	})
}
