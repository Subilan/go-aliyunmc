package changelog_routes

import (
	"errors"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/models"

	"github.com/gin-gonic/gin"
)

type createChangelogRequest struct {
	Title    string            `json:"title" binding:"required"`
	Body     string            `json:"body" binding:"required"`
	Category labmodels.LogType `json:"category" binding:"required"`
}

func HandleCreate(req createChangelogRequest, c *gin.Context) (any, error) {
	if req.Category != labmodels.LogTypePlatform && req.Category != labmodels.LogTypeServer {
		return nil, errors.New("category 必须是 platform 或 server")
	}

	cl, err := dao.CreateChangelog(req.Title, req.Body, req.Category)
	if err != nil {
		return nil, err
	}

	return cl, nil
}
