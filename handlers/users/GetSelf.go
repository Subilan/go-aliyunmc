package users

import (
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/gctx"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

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
