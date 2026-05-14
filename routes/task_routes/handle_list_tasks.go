package task_routes

import (
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ListTasksQuery struct {
	Status string `form:"status"`
	Sort   string `form:"sort"`
	Order  string `form:"order"`
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
}

type ListTasksResponse struct {
	Tasks []models.Task `json:"tasks"`
	Total int64         `json:"total"`
}

func HandleListTasks(query ListTasksQuery, c *gin.Context) (any, error) {
	if query.Limit <= 0 || query.Limit > 100 {
		query.Limit = 20
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	if query.Order != "asc" && query.Order != "desc" {
		query.Order = "desc"
	}

	var status *models.TaskStatus
	if query.Status != "" {
		s := models.TaskStatus(query.Status)
		status = &s
	}

	total, err := store.CountTasks(status)
	if err != nil {
		return nil, h.HttpError(http.StatusInternalServerError, "查询任务总数失败")
	}

	tasks, err := store.ListTasks(status, query.Sort, query.Order, query.Limit, query.Offset)
	if err != nil {
		return nil, err
	}

	return ListTasksResponse{Tasks: tasks, Total: total}, nil
}
