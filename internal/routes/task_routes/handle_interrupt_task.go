package task_routes

import (
	"errors"
	"net/http"

	"github.com/Subilan/go-aliyunmc/internal/context_util"
	"github.com/Subilan/go-aliyunmc/internal/h"
	"github.com/Subilan/go-aliyunmc/internal/store"
	"github.com/Subilan/go-aliyunmc/internal/tasks"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HandleInterruptTask 请求中止正在执行的任务。
//
// 返回 202 Accepted 表示中断请求已经发出，不表示任务已经停止：任务函数的退出与
// 终态收尾由 monitor 异步完成。调用方应通过该任务的 SSE 流（task_done）或
// GET /task/:id 确认最终状态。
func HandleInterruptTask(c *gin.Context) {
	taskID, ok := context_util.ParamUint(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, h.DetailsF("无效的任务 id"))
		return
	}

	executor, ok := tasks.GetExecutor(taskID)
	if !ok {
		// 区分「任务不存在」和「任务已结束」，便于调用方判断是否需要处理。
		if _, err := store.GetTaskRecord(taskID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, h.DetailsF("任务不存在"))
				return
			}
			c.JSON(http.StatusInternalServerError, h.Details(err))
			return
		}
		c.JSON(http.StatusConflict, h.DetailsF("任务不在运行中"))
		return
	}

	executor.Interrupt()
	c.Status(http.StatusAccepted)
}
