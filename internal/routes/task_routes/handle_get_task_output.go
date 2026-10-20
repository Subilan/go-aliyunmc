package task_routes

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Subilan/go-aliyunmc/internal/h"
	"github.com/Subilan/go-aliyunmc/internal/sse"
	"github.com/Subilan/go-aliyunmc/internal/store"
	"github.com/Subilan/go-aliyunmc/internal/store/models"
	"github.com/Subilan/go-aliyunmc/internal/tasks"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func HandleGetTaskOutput(c *gin.Context) {
	taskId := c.Param("id")

	if taskId == "" {
		c.JSON(http.StatusBadRequest, h.DetailsF("task_id不可为空"))
		return
	}

	taskIdUint, err := strconv.ParseUint(taskId, 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, h.DetailsF("task_id格式错误"))
		return
	}

	afterSeq, hasLastEventID, done, err := parseLastEventID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, h.DetailsF("Last-Event-ID格式错误"))
		return
	}

	if done {
		// 客户端已经收到 task_done，输出流没有更多内容，返回 204 让 EventSource 停止重连。
		c.Status(http.StatusNoContent)
		return
	}

	if executor, ok := tasks.GetExecutor(uint(taskIdUint)); ok {
		client, err := sse.NewClient(c)

		if err != nil {
			c.JSON(http.StatusInternalServerError, h.DetailsF("无法建立SSE连接：%s", err.Error()))
			return
		}

		var (
			unsubscribe func()
			subscribed  bool
		)

		if hasLastEventID {
			// 浏览器重连：先回放错过的持久化输出，再继续接收实时事件。
			unsubscribe, subscribed = executor.SubscribeFrom(client, afterSeq)
		} else {
			unsubscribe, subscribed = executor.Subscribe(client)
		}

		if subscribed {
			defer unsubscribe()
			client.Listen()
			return
		}

		// 执行器恰好在此刻结束，退回从数据库回放。
	}

	streamFinishedTaskOutput(c, uint(taskIdUint), afterSeq)
}

// parseLastEventID 读取 SSE 重连时浏览器自动携带的 Last-Event-ID 请求头。
// done 表示客户端已经消费到 task_done；其余情况返回输出序号。
func parseLastEventID(c *gin.Context) (afterSeq uint64, hasLastEventID, done bool, err error) {
	raw := strings.TrimSpace(c.GetHeader("Last-Event-ID"))
	if raw == "" {
		return 0, false, false, nil
	}
	if raw == tasks.TaskDoneEventID {
		return 0, true, true, nil
	}

	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, true, false, err
	}

	return id, true, false, nil
}

// streamFinishedTaskOutput 在任务没有正在运行的执行器时，从数据库回放其输出。
// afterSeq 之前的行不再回放；为 0 时从头回放。
func streamFinishedTaskOutput(c *gin.Context, taskID uint, afterSeq uint64) {
	task, err := store.GetTaskRecord(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, h.DetailsF("该任务不存在或已结束"))
			return
		}
		c.JSON(http.StatusInternalServerError, h.DetailsF("读取任务失败：%s", err.Error()))
		return
	}

	// 只有进入终态的任务才有确定的 task_done；没有执行器的非终态任务无法回放。
	if task.Status != models.TaskStatusSuccess && task.Status != models.TaskStatusFailed {
		c.JSON(http.StatusBadRequest, h.DetailsF("该任务不存在或已结束"))
		return
	}

	client, err := sse.NewClient(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, h.DetailsF("无法建立SSE连接：%s", err.Error()))
		return
	}

	tasks.ReplayTaskOutput(client, task, afterSeq)
	client.Listen()
}
