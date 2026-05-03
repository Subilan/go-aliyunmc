package instance_routes

import (
	"go-aliyunmc/h"
	"go-aliyunmc/store"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleGetActiveInstance(c *gin.Context) (any, error) {
	activeInstance, err := store.GetActiveInstanceDefaultNil()

	if err != nil {
		return nil, err
	}

	if activeInstance == nil {
		return nil, h.HttpError(http.StatusNotFound, "暂无活跃实例")
	}

	return activeInstance, nil
}