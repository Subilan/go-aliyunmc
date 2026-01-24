package users

import (
	"database/sql"
	"net/http"

	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type CreateUserRequest struct {
	// Username 是要创建用户的用户名，不可与其他用户重复
	Username string `json:"username" binding:"required" validate:"required"`
	// Password 是要创建用户的密码明文
	Password string `json:"password" binding:"required" validate:"required"`
}

// HandleCreateUser godoc
//
//	@Summary		创建用户
//	@Description	根据提供的用户名和密码创建一个新的用户。用户名是唯一的，如果与现有用户重复则返回409。
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			createuserrequest	body	CreateUserRequest	true	"创建用户请求体"
//	@Success		200
//	@Failure		409	{object}	helpers.ErrorResp
//	@Failure		500	{object}	helpers.ErrorResp
//	@Router			/user [post]
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
					Code:    http.StatusConflict,
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

		_, err = tx.ExecContext(c, "INSERT INTO user_roles (user_id, `role`) VALUES (?, ?)", userId, consts.UserRoleUser)

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
