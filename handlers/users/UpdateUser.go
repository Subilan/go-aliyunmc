package users

import (
	"net/http"

	"github.com/Subilan/gomc-server/globals"
	"github.com/Subilan/gomc-server/helpers"
	"github.com/Subilan/gomc-server/helpers/ginContextCheckers"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UpdateUserRequest struct {
	Password string `json:"password" binding:"required"`
}

func Update() gin.HandlerFunc {
	return helpers.BodyHandler[UpdateUserRequest](func(body UpdateUserRequest, c *gin.Context) (any, error) {
		userId := c.Param("userId")

		if userId == "" {
			return nil, &helpers.HttpError{Code: http.StatusBadRequest, Details: "无效用户id"}
		}

		ownErr := ginContextCheckers.MustOwnUserId(userId, c)

		if ownErr != nil {
			return nil, ownErr
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}

		result, err := globals.Pool.ExecContext(c, "UPDATE users SET password_hash = ? WHERE id = ?", hash, userId)
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
