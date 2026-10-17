package task_routes

import (
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/sse"
	"github.com/Subilan/go-aliyunmc/tasks"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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

	executor, ok := tasks.GetExecutor(uint(taskIdUint))

	if !ok {
		c.JSON(http.StatusBadRequest, h.DetailsF("该任务不存在或已结束"))
		return
	}

	client, err := sse.NewClient(c)

	if err != nil {
		c.JSON(http.StatusInternalServerError, h.DetailsF("无法建立SSE连接：%s", err.Error()))
		return
	}

	unsubscribe, ok := executor.Subscribe(client)

	if !ok {
		c.JSON(http.StatusNotFound, h.DetailsF("该任务不存在或已结束"))
		return
	}

	defer unsubscribe()

	// Listen 会阻塞到客户端断开、任务结束或订阅 channel 被关闭为止。
	client.Listen()
}
