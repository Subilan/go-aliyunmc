package users

import (
	"net/http"
	"strings"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/gctx"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

func HandleUpdateBindGameId() gin.HandlerFunc {
	return helpers.BodyHandler[BindGameIdRequest](func(body BindGameIdRequest, c *gin.Context) (any, error) {
		userId, err := gctx.ShouldGetUserId(c)

		if err != nil {
			return nil, err
		}

		_, exists := store.GetGameBound(userId)

		if !exists {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "找不到绑定记录"}
		}

		_, err = db.Pool.Exec("UPDATE game_bounds SET game_id = ? WHERE user_id = ?", body.GameId, userId)

		if err != nil {
			if strings.Contains(err.Error(), "1062") {
				return nil, &helpers.HttpError{Code: http.StatusConflict, Details: "已被绑定"}
			}
			return nil, err
		}

		return gin.H{}, nil
	})
}
