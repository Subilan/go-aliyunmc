package changelog_routes

import (
	"errors"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

type updateChangelogRequest struct {
	Title    string            `json:"title"`
	Body     string            `json:"body"`
	Category labmodels.LogType `json:"category"`
}

func HandleUpdate(req updateChangelogRequest, c *gin.Context) (any, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return nil, errors.New("无效的 id")
	}

	if req.Category != "" && req.Category != labmodels.LogTypePlatform && req.Category != labmodels.LogTypeServer {
		return nil, errors.New("category 必须是 platform 或 server")
	}

	updates := map[string]any{}
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Body != "" {
		updates["body"] = req.Body
	}
	if req.Category != "" {
		updates["category"] = req.Category
	}

	if len(updates) == 0 {
		return nil, errors.New("没有需要更新的字段")
	}

	if err := dao.UpdateChangelog(uint(id), updates); err != nil {
		return nil, err
	}

	return nil, nil
}
