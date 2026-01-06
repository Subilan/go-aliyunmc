package tasks

import (
	"net/http"
	"strings"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
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

type retrievalType string

const (
	retrievalById     retrievalType = "id"
	retrievalByActive retrievalType = "active"
)

func getResponse(withPushedEvents bool, withJoinedPushedEvents bool, retrievalTyp retrievalType, retrievalArg string) (*GetTaskResponse, error) {
	var task store.Task

	stmt := "SELECT task_id, type, user_id, status, created_at FROM tasks "
	args := make([]any, 0, 2)

	if retrievalTyp == retrievalById {
		stmt += "WHERE task_id = ?"
		args = append(args, retrievalArg)
	} else {
		stmt += "WHERE type = ? AND status = ?"
		args = append(args, retrievalArg, store.TaskStatusRunning)
	}

	err := db.Pool.QueryRow(stmt, args...).Scan(&task.TaskId, &task.TaskType, &task.UserId, &task.Status, &task.CreatedAt)

	if err != nil {
		return nil, err
	}

	var pushedEvents []store.PushedEvent
	var joinedPushedEvents string

	if withPushedEvents || withJoinedPushedEvents {
		rows, err := db.Pool.Query("SELECT task_id, ord, type, content, created_at FROM pushed_events ev WHERE task_id = ? ORDER BY ord", task.TaskId)

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

		if withJoinedPushedEvents {
			var stringBuilder strings.Builder

			for _, event := range pushedEvents {
				stringBuilder.WriteString(event.Content)
			}

			joinedPushedEvents = stringBuilder.String()
			pushedEvents = nil
		}
	}

	return &GetTaskResponse{
		Task:               task,
		PushedEvents:       pushedEvents,
		JoinedPushedEvents: joinedPushedEvents,
	}, nil
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

		res, err := getResponse(query.WithPushedEvents, query.WithJoinedPushedEvents, retrievalById, taskId)

		if err != nil {
			return nil, err
		}

		return helpers.Data(res), nil
	})
}

func HandleGetActiveTaskByType() gin.HandlerFunc {
	return helpers.QueryHandler[GetTaskQuery](func(query GetTaskQuery, c *gin.Context) (any, error) {
		if query.WithJoinedPushedEvents && query.WithPushedEvents {
			return nil, &helpers.HttpError{Code: http.StatusBadRequest, Details: "互斥的选项"}
		}

		taskType := c.Query("type")

		if taskType == "" {
			return nil, &helpers.HttpError{Code: http.StatusBadRequest, Details: "must provide taskType"}
		}

		res, err := getResponse(query.WithPushedEvents, query.WithJoinedPushedEvents, retrievalByActive, taskType)

		if err != nil {
			return nil, err
		}

		return helpers.Data(res), nil
	})
}
