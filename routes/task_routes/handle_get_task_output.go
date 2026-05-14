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

	go client.Listen()

	ok = executor.SubscribeOrFail(client)

	if !ok {
		c.JSON(http.StatusInternalServerError, h.DetailsF("无法注册输出监听器"))
		return
	}

	defer executor.Unsubscribe(client)

	<-client.Done()
}
