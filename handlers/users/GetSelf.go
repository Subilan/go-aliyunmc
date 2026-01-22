package users

import (
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/gctx"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

// HandleGetSelf 根据传入请求头中的凭证信息，返回该用户的非敏感字段数据。
//
//	@Summary		获取当前已登入用户信息
//	@Description	根据传入的凭证返回该用户的非敏感信息，例如用户名、ID、创建时间、权限组等信息。
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	helpers.DataResp[store.User]
//	@Failure		404	{object}	helpers.ErrorResp
//	@Failure		403	{object}	helpers.ErrorResp
//	@Router			/user [get]
func HandleGetSelf() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		userId, _ := gctx.ShouldGetUserId(c)

		var user store.User

		err := db.Pool.QueryRow("SELECT u.id, u.username, u.created_at, r.role FROM users u JOIN user_roles r ON u.id = r.user_id WHERE u.id=?", userId).Scan(&user.Id, &user.Username, &user.CreatedAt, &user.Role)

		if err != nil {
			return nil, err
		}

		return helpers.Data(user), nil
	})
}
