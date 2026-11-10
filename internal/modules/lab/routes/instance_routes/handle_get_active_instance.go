package instance_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/dao"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleGetActiveInstance(c *gin.Context) (any, error) {
	activeInstance, err := dao.GetActiveInstanceDefaultNil()

	if err != nil {
		return nil, err
	}

	if activeInstance == nil {
		return nil, h.HttpError(http.StatusNotFound, "暂无活跃实例")
	}

	return activeInstance, nil
}
