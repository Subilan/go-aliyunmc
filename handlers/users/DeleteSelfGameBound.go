package users

import (
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/gctx"
	"github.com/gin-gonic/gin"
)

func HandleDeleteSelfGameBound() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		userId, err := gctx.ShouldGetUserId(c)

		if err != nil {
			return nil, err
		}

		_, err = db.Pool.Exec("DELETE FROM game_bounds WHERE user_id = ?", userId)

		if err != nil {
			return nil, err
		}

		return gin.H{}, nil
	})
}
