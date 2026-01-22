package users

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/gctx"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UpdateUserRequest struct {
	// Password 是新密码的明文
	Password string `json:"password" binding:"required"`
}

// HandleUserUpdate 对用户的信息进行修改
//
//	@Summary		更新用户信息
//	@Description	更新用户在数据库中存储的各个字段。目前仅支持更新密码。
//	@Tags			users
//	@Accept			json
//	@Param			userId				path	string				true	"目标用户ID"
//	@Param			updateuserrequest	body	UpdateUserRequest	true	"更新请求体"
//	@Success		200
//	@Failure		403	{object}	helpers.ErrorResp
//	@Failure		404	{object}	helpers.ErrorResp
//	@Failure		500	{object}	helpers.ErrorResp
//	@Router			/user/{userId} [patch]
func HandleUserUpdate() gin.HandlerFunc {
	return helpers.BodyHandler[UpdateUserRequest](func(body UpdateUserRequest, c *gin.Context) (any, error) {
		userId := c.Param("userId")

		if userId == "" {
			return nil, &helpers.HttpError{Code: http.StatusBadRequest, Details: "无效用户id"}
		}

		ownErr := gctx.MustOwnUserId(userId, c)

		if ownErr != nil {
			return nil, ownErr
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}

		result, err := db.Pool.ExecContext(c, "UPDATE users SET password_hash = ? WHERE id = ?", hash, userId)
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
