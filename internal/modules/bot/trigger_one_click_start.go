package bot

import (
	"errors"
	"net/http"
	"sync"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	labmodels "github.com/Subilan/go-aliyunmc/internal/modules/lab/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/tasks"

	"github.com/gin-gonic/gin"
)

// oneClickStartMu 串行化机器人侧的一键启动触发。
// 任务框架的互斥分组负责跨模块互斥，这里只消除同一接口并发触发时的抢占窗口，
// 让后到的请求能稳定地识别出正在执行的任务。
var oneClickStartMu sync.Mutex

type botOneClickStartResult struct {
	TaskID         uint   `json:"taskId"`
	Status         string `json:"status"`
	AlreadyRunning bool   `json:"alreadyRunning"`
}

// handleTriggerOneClickStart 以机器人身份触发一键启动游戏任务。
// 执行期间重复触发返回同一个任务，不会创建新的任务记录。
func handleTriggerOneClickStart(_ *gin.Context) (any, error) {
	oneClickStartMu.Lock()
	defer oneClickStartMu.Unlock()

	if taskID, ok := tasks.GetRunningTaskIDByType(labmodels.TaskTypeOneClickStartGame); ok {
		return botOneClickStartResult{
			TaskID:         taskID,
			Status:         string(labmodels.TaskStatusRunning),
			AlreadyRunning: true,
		}, nil
	}

	_, task, err := tasks.TriggerTaskBot(labmodels.TaskTypeOneClickStartGame, nil)
	if err == nil {
		return botOneClickStartResult{
			TaskID: task.ID,
			Status: string(task.Status),
		}, nil
	}

	if errors.Is(err, tasks.ErrTaskTypeExecuting) {
		// 在机器人触发之外，lab 也可能同时发起部署链任务；重新确认一次是否就是一键启动。
		if taskID, ok := tasks.GetRunningTaskIDByType(labmodels.TaskTypeOneClickStartGame); ok {
			return botOneClickStartResult{
				TaskID:         taskID,
				Status:         string(labmodels.TaskStatusRunning),
				AlreadyRunning: true,
			}, nil
		}
		return nil, h.HttpError(http.StatusConflict, "已有其他任务正在执行，请稍后再试")
	}

	return nil, err
}
