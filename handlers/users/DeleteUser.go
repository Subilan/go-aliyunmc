package users

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/gctx"
	"github.com/gin-gonic/gin"
)

// HandleUserDelete godoc
//
//	@Summary		注销用户
//	@Description	删除某个用户，使得该用户名以后无法再次登录。此接口仅适用于用户主动注销，如果请求参数携带的用户ID与凭证中的用户ID不匹配，返回403
//	@Tags			users
//	@Param			userId	path	string	true	"删除目标用户ID"
//	@Produce		json
//	@Success		200
//	@Failure		403	{object}	helpers.ErrorResp
//	@Failure		409	{object}	helpers.ErrorResp
//	@Failure		500	{object}	helpers.ErrorResp
//	@Router			/user/{userId} [delete]
func HandleUserDelete() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		userId := c.Param("userId")

		if userId == "" {
			return nil, &helpers.HttpError{
				Code:    http.StatusBadRequest,
				Details: "无效用户id",
			}
		}

		ownErr := gctx.MustOwnUserId(userId, c)

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
