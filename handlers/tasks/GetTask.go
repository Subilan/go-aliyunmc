package tasks

import (
	"net/http"
	"strings"

	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

// GetTaskQuery 定义 HandleGetTask 接口的查询格式
// 在这里 WithPushedEvents 和 WithJoinedPushedEvents 两个选项是互斥的
type GetTaskQuery struct {
	// WithPushedEvents 指定是否在结果中包含与该任务相关的所有推送事件
	WithPushedEvents bool `form:"withPushedEvents"`

	// WithJoinedPushedEvents 指定是否在结果中包含与该任务相关的所有推送事件内容的合并值
	WithJoinedPushedEvents bool `form:"withJoinedPushedEvents"`
}

// GetTaskResponse 是 HandleGetTask 接口的返回数据结构
type GetTaskResponse struct {
	// store.Task 是返回数据结构的主体部分
	store.Task

	// PushedEvents 由 GetTaskQuery.WithPushedEvents 指定是否包含
	PushedEvents []store.PushedEvent `json:"pushedEvents,omitempty"`

	// PushedEvents 由 GetTaskQuery.WithJoinedPushedEvents 指定是否包含
	JoinedPushedEvents string `json:"joinedPushedEvents,omitempty"`
}

// HandleGetTask 接口用于获取数据库中的一条指定的任务记录，可以选择性地包含与该任务记录关联的推送事件内容
func HandleGetTask() gin.HandlerFunc {
	return helpers.QueryHandler[GetTaskQuery](func(query GetTaskQuery, c *gin.Context) (any, error) {
		// 为了避免冗余，二者互斥
		if query.WithJoinedPushedEvents && query.WithPushedEvents {
			return nil, &helpers.HttpError{Code: http.StatusBadRequest, Details: "互斥的选项"}
		}

		taskId := c.Param("taskId")

		if taskId == "" {
			return nil, &helpers.HttpError{Code: http.StatusBadRequest, Details: "must provide taskId"}
		}

		var task store.Task

		err := globals.Pool.QueryRow("SELECT task_id, type, user_id, status, created_at FROM tasks WHERE task_id = ?", taskId).Scan(&task.TaskId, &task.TaskType, &task.UserId, &task.Status, &task.CreatedAt)

		if err != nil {
			return nil, err
		}

		var pushedEvents []store.PushedEvent
		var joinedPushedEvents string

		if query.WithPushedEvents || query.WithJoinedPushedEvents {
			rows, err := globals.Pool.Query("SELECT task_id, ord, type, content, created_at FROM pushed_events ev WHERE task_id = ? ORDER BY ord", taskId)

			if err != nil {
				return nil, err
			}

			for rows.Next() {
				var event store.PushedEvent
				err = rows.Scan(&event.TaskId, &event.Ord, &event.Type, &event.Content, &event.CreatedAt)
				if err != nil {
					return nil, err
				}
				pushedEvents = append(pushedEvents, event)
			}

			if query.WithJoinedPushedEvents {
				var stringBuilder strings.Builder

				for _, event := range pushedEvents {
					stringBuilder.WriteString(event.Content)
				}

				joinedPushedEvents = stringBuilder.String()
				pushedEvents = nil
			}
		}

		return helpers.Data(GetTaskResponse{Task: task, PushedEvents: pushedEvents, JoinedPushedEvents: joinedPushedEvents}), nil
	})
}
