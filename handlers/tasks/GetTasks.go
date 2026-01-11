package tasks

import (
	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

type GetTasksQuery struct {
	helpers.Paginated
}

func HandleGetTasks() gin.HandlerFunc {
	return helpers.QueryHandler[GetTasksQuery](func(query GetTasksQuery, c *gin.Context) (any, error) {
		if query.PageSize == 0 {
			query.PageSize = 10
		}
		if query.Page == 0 {
			query.Page = 1
		}

		rows, err := db.Pool.Query(
			"SELECT t.task_id, t.`type`, t.`status`, t.created_at, t.updated_at, u.username FROM tasks t JOIN `users` u ON t.user_id = u.id ORDER BY t.created_at DESC LIMIT ? OFFSET ?",
			query.PageSize, (query.Page-1)*query.PageSize,
		)
		defer rows.Close()

		if err != nil {
			return nil, err
		}

		var tasks = make([]store.JoinedTask, 0, query.PageSize)

		for rows.Next() {
			var task store.JoinedTask
			err = rows.Scan(&task.Id, &task.Type, &task.Status, &task.CreatedAt, &task.UpdatedAt, &task.Username)

			if err != nil {
				return nil, err
			}

			tasks = append(tasks, task)
		}

		return helpers.Data(tasks), nil
	})
}

type TaskOverview struct {
	SuccessCount   int              `json:"successCount"`
	UnsuccessCount int              `json:"unsuccessCount"`
	Latest         store.JoinedTask `json:"latest"`
}

func HandleGetTaskOverview() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		var res TaskOverview

		err := db.Pool.QueryRow("SELECT COUNT(*) FROM tasks WHERE `status` = ?", consts.TaskStatusSuccess).Scan(&res.SuccessCount)

		if err != nil {
			return nil, err
		}

		err = db.Pool.QueryRow("SELECT COUNT(*) FROM tasks WHERE `status` IN (?, ?, ?)", consts.TaskStatusCancelled, consts.TaskStatusFailed, consts.TaskStatusTimedOut).Scan(&res.UnsuccessCount)

		if err != nil {
			return nil, err
		}

		err = db.Pool.QueryRow("SELECT t.task_id, t.type, t.status, t.created_at, t.updated_at, u.username FROM tasks t JOIN users u ON t.user_id = u.id ORDER BY t.created_at DESC LIMIT 1").
			Scan(&res.Latest.Id, &res.Latest.Type, &res.Latest.Status, &res.Latest.CreatedAt, &res.Latest.UpdatedAt, &res.Latest.Username)

		if err != nil {
			return nil, err
		}

		return helpers.Data(res), nil
	})
}
