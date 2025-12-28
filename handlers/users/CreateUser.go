package users

import (
	"net/http"

	"github.com/Subilan/gomc-server/globals"
	"github.com/Subilan/gomc-server/helpers"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func HandleCreateUser() gin.HandlerFunc {
	return helpers.BodyHandler[CreateUserRequest](func(body CreateUserRequest, c *gin.Context) (any, error) {
		hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)

		if err != nil {
			return nil, err
		}

		_, err = globals.Pool.ExecContext(c, "INSERT INTO users (username, password_hash) VALUES (?, ?)", body.Username, hash)

		if err != nil {
			if helpers.IsDuplicateEntryError(err) {
				return nil, &helpers.HttpError{
					Code:    http.StatusBadRequest,
					Details: "用户名重复",
				}
			}
			return nil, err
		}

		return gin.H{}, nil
	})
}
