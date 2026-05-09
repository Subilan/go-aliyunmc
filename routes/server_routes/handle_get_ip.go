package server_routes

import (
	"go-aliyunmc/h"
	"go-aliyunmc/store"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleGetIP(c *gin.Context) (any, error) {
	defaultValue := c.Query("defaultValue")

	ip, err := store.GetActiveInstanceIpNonEmpty()

	if err != nil {
		if defaultValue != "" {
			return defaultValue, nil
		}
		return nil, h.HttpError(http.StatusNotFound, "无法获取服务器IP")
	}

	return ip, nil
}
