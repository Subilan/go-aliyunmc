package users

import (
	"database/sql"
	"net/http"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
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

		tx, err := db.Pool.BeginTx(c, &sql.TxOptions{})

		if err != nil {
			return nil, err
		}

		result, err := tx.ExecContext(c, "INSERT INTO users (username, password_hash) VALUES (?, ?)", body.Username, hash)

		if err != nil {
			_ = tx.Rollback()
			if store.IsDuplicateEntryError(err) {
				return nil, &helpers.HttpError{
					Code:    http.StatusBadRequest,
					Details: "用户名重复",
				}
			}
			return nil, err
		}

		userId, err := result.LastInsertId()

		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}

		_, err = tx.ExecContext(c, "INSERT INTO user_roles (user_id, `role`) VALUES (?, ?)", userId, "user")

		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}

		err = tx.Commit()

		if err != nil {
			return nil, err
		}

		return gin.H{}, nil
	})
}
