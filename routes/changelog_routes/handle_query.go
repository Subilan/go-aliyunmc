package changelog_routes

import (
	"github.com/Subilan/go-aliyunmc/context_util"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"

	"github.com/gin-gonic/gin"
)

type changelogQuery struct {
	SortBy   string         `form:"sortBy" binding:"omitempty,oneof=asc desc"`
	Page     int            `form:"page"`
	PageSize int            `form:"pageSize"`
	Category models.LogType `form:"category" binding:"omitempty,oneof=platform server"`
}

type changelogQueryResult struct {
	Items    []store.ChangelogItem `json:"items"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}

func HandleQuery(q changelogQuery, c *gin.Context) (any, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 5 || q.PageSize > 50 {
		q.PageSize = 10
	}

	userID, _ := context_util.GetUserID(c)

	items, total, err := store.QueryChangelogs(q.SortBy, q.Page, q.PageSize, q.Category, userID)
	if err != nil {
		return nil, err
	}

	return changelogQueryResult{
		Items:    items,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}
