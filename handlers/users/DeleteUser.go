package users

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/ginContextCheckers"
	"github.com/gin-gonic/gin"
)

func HandleUserDelete() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		userId := c.Param("userId")

		if userId == "" {
			return nil, &helpers.HttpError{
				Code:    http.StatusBadRequest,
				Details: "无效用户id",
			}
		}

		ownErr := ginContextCheckers.MustOwnUserId(userId, c)

		if ownErr != nil {
			return nil, ownErr
		}

		result, err := db.Pool.ExecContext(c, "DELETE FROM users WHERE id = ?", userId)
		if err != nil {
			return nil, err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}

		if rowsAffected == 0 {
			return nil, &helpers.HttpError{
				Code:    http.StatusNotFound,
				Details: "用户不存在",
			}
		}

		return gin.H{}, nil
	})
}
