package ginContextCheckers

import (
	"net/http"
	"strconv"

	"github.com/Subilan/gomc-server/helpers"
	"github.com/gin-gonic/gin"
)

func MustOwnUserId(id string, c *gin.Context) error {
	currentUserId, exists := c.Get("user_id")

	if !exists {
		return &helpers.HttpError{Code: http.StatusForbidden, Details: "找不到凭据"}
	}

	if strconv.FormatInt(currentUserId.(int64), 10) != id {
		return &helpers.HttpError{Code: http.StatusForbidden, Details: "用户不匹配"}
	}

	return nil
}
