package changelog_routes

import (
	"errors"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"

	"github.com/gin-gonic/gin"
)

type createChangelogRequest struct {
	Title    string         `json:"title" binding:"required"`
	Body     string         `json:"body" binding:"required"`
	Category models.LogType `json:"category" binding:"required"`
}

func HandleCreate(req createChangelogRequest, c *gin.Context) (any, error) {
	if req.Category != models.LogTypePlatform && req.Category != models.LogTypeServer {
		return nil, errors.New("category 必须是 platform 或 server")
	}

	cl, err := store.CreateChangelog(req.Title, req.Body, req.Category)
	if err != nil {
		return nil, err
	}

	return cl, nil
}
