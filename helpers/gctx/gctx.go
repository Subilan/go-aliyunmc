package gctx

import (
	"net/http"
	"strconv"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/gin-gonic/gin"
)

func MustOwnUserId(id string, c *gin.Context) error {
	inputId, err := strconv.ParseInt(id, 10, 64)

	if err != nil {
		return &helpers.HttpError{Code: http.StatusBadRequest, Details: "输入ID无效"}
	}

	userId, err := ShouldGetUserId(c)

	if err != nil {
		return err
	}

	if userId != inputId {
		return &helpers.HttpError{Code: http.StatusForbidden, Details: "用户不匹配"}
	}

	return nil
}

func ShouldGetUserId(c *gin.Context) (int64, error) {
	currentUserId, exists := c.Get("user_id")

	if !exists {
		return 0, &helpers.HttpError{Code: http.StatusForbidden, Details: "找不到用户凭据"}
	}

	result, ok := currentUserId.(int64)

	if !ok {
		return 0, &helpers.HttpError{Code: http.StatusForbidden, Details: "用户ID无效"}
	}

	return result, nil
}
