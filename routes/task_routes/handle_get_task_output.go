package task_routes

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/sse"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"
	"github.com/Subilan/go-aliyunmc/tasks"

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

	afterSeq, hasLastEventID, err := parseLastEventID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, h.DetailsF("Last-Event-ID格式错误"))
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

	streamFinishedTaskOutput(c, uint(taskIdUint), afterSeq, hasLastEventID)
}

// parseLastEventID 读取 SSE 重连时浏览器自动携带的 Last-Event-ID 请求头。
func parseLastEventID(c *gin.Context) (uint64, bool, error) {
	raw := strings.TrimSpace(c.GetHeader("Last-Event-ID"))
	if raw == "" {
		return 0, false, nil
	}

	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, true, err
	}

	return id, true, nil
}

// streamFinishedTaskOutput 在任务没有正在运行的执行器时，从数据库回放其输出。
// 不带 Last-Event-ID 时从头回放；带 Last-Event-ID 时只回放其后的行。
func streamFinishedTaskOutput(c *gin.Context, taskID uint, afterSeq uint64, hasLastEventID bool) {
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

	lastSeq, err := store.GetLastTaskOutputSeq(taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, h.DetailsF("读取任务输出失败：%s", err.Error()))
		return
	}

	// task_done 的事件 ID 是 lastSeq+1。客户端已经收到它（或更靠后的 ID）时，
	// 返回 204 让 EventSource 停止自动重连。
	if hasLastEventID && afterSeq > lastSeq {
		c.Status(http.StatusNoContent)
		return
	}

	client, err := sse.NewClient(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, h.DetailsF("无法建立SSE连接：%s", err.Error()))
		return
	}

	tasks.ReplayTaskOutput(client, task, lastSeq, afterSeq)
	client.Listen()
}
