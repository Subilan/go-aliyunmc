package users

import (
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/gctx"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

type SelfGameBoundResponse struct {
	store.GameBound
	Exists bool `json:"exists"`
}

func HandleGetSelfGameBound() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		userId, err := gctx.ShouldGetUserId(c)

		if err != nil {
			return nil, err
		}

		bound, exists := store.GetGameBound(userId)

		if !exists {
			return helpers.Data(SelfGameBoundResponse{Exists: false}), nil
		}

		return helpers.Data(SelfGameBoundResponse{Exists: true, GameBound: *bound}), nil
	})
}
