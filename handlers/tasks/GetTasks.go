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

// HandleGetTasks 根据传入的查询参数，返回多条任务记录
//
//	@Summary		获取多条任务记录
//	@Description	根据传入的查询参数，返回多条任务记录。
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			gettasksquery	body		GetTasksQuery	true	"分页参数等"
//	@Success		200				{object}	helpers.DataResp[[]store.JoinedTask]
//	@Failure		404				{object}	helpers.ErrorResp
//	@Failure		500				{object}	helpers.ErrorResp
//	@Router			/task/s [get]
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

// TaskOverview 代表任务的总览信息
type TaskOverview struct {
	// SuccessCount 是成功任务的总数量
	SuccessCount int `json:"successCount"`
	// UnsuccessCount 是失败任务的总数量
	UnsuccessCount int `json:"unsuccessCount"`
	// Latest 是系统中最新一条任务的信息
	Latest store.JoinedTask `json:"latest,omitempty"`
}

// HandleGetTaskOverview 返回与任务相关的总览信息
//
//	@Summary		获取任务总览
//	@Description	返回系统任务模块的总览信息，如总任务数量、成功执行数量等。
//	@Tags			tasks
//	@Produce		json
//	@Success		200	{object}	helpers.DataResp[TaskOverview]
//	@Failure		404	{object}	helpers.ErrorResp
//	@Router			/task/overview [get]
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

		_ = db.Pool.QueryRow("SELECT t.task_id, t.type, t.status, t.created_at, t.updated_at, u.username FROM tasks t JOIN users u ON t.user_id = u.id ORDER BY t.created_at DESC LIMIT 1").
			Scan(&res.Latest.Id, &res.Latest.Type, &res.Latest.Status, &res.Latest.CreatedAt, &res.Latest.UpdatedAt, &res.Latest.Username)

		return helpers.Data(res), nil
	})
}
