package instances

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Subilan/gomc-server/globals"
	"github.com/Subilan/gomc-server/helpers"
	"github.com/Subilan/gomc-server/helpers/remote"
	"github.com/Subilan/gomc-server/helpers/store"
	"github.com/Subilan/gomc-server/helpers/stream"
	"github.com/Subilan/gomc-server/helpers/tasks"
	"github.com/Subilan/gomc-server/helpers/templateData"
	"github.com/gin-gonic/gin"
)

func HandleDeployInstance() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		var userId int

		userIdStr, exists := c.Get("user_id")

		if !exists || userIdStr == "" {
			return nil, &helpers.HttpError{Code: http.StatusUnauthorized, Details: "无效用户ID"}
		}

		userId = int(userIdStr.(int64))

		// 检查是否存在有ip且正在运行的实例可供部署
		_, ip, err := store.GetRunningInstanceBrief()

		if err != nil {
			return nil, err
		}

		// 检查是否存在部署任务正在运行
		runningTaskCnt, err := store.GetRunningTaskCount(store.TaskTypeInstanceDeployment)

		if err != nil {
			return nil, err
		}

		if runningTaskCnt != 0 {
			return nil, &helpers.HttpError{Code: http.StatusConflict, Details: "已经存在部署任务正在运行"}
		}

		// 为新的部署任务分配UUID并插入记录
		taskId, err := store.InsertTask(store.TaskTypeInstanceDeployment, userId)

		if err != nil {
			return nil, err
		}

		// 创建当前任务的全局流
		stream.Create(taskId)

		// 创建超时上下文
		runCtx, cancelRunCtx := context.WithTimeout(context.Background(), 5*time.Minute)
		tasks.Register(cancelRunCtx, taskId)

		// 运行并借助全局流输出内容
		go remote.RunScriptAsRootAsync(runCtx, ip, "deploy.tmpl.sh", templateData.Deploy(),
			func(bytes []byte) {
				log.Println("debug: deploy.sh stdout: ", string(bytes))

				state := stream.GetState(taskId)

				err = stream.BroadcastAndSave(store.PushedEvent{
					PushedEventState: *state,
					Content:          string(bytes),
				})

				stream.IncrOrd(taskId)

				if err != nil {
					log.Println(err.Error())
					log.Printf("cannot send and save script step: userid=%v, deployment, is_error=false, content=%s\n", userId, string(bytes))
				}
			},
			func(err error) {
				log.Println("dedug: deploy.sh stderr: ", err.Error())

				state := stream.GetState(taskId)
				sendAndSaveError := stream.BroadcastAndSave(store.PushedEvent{
					PushedEventState: *state,
					IsError:          true,
					Content:          err.Error(),
				})

				if sendAndSaveError != nil {
					log.Println(sendAndSaveError.Error())
					log.Printf("cannot send and save script step: userid=%v, deployment, is_error=true, content=%s\n", userId, err.Error())
				}

				var status = store.TaskStatusFailed

				if errors.Is(err, context.Canceled) {
					status = store.TaskStatusCancelled
				}

				if errors.Is(err, context.DeadlineExceeded) {
					status = store.TaskStatusTimedOut
				}

				_, err = globals.Pool.Exec("UPDATE tasks SET status = ? WHERE task_id = ?", status, taskId)

				if err != nil {
					log.Println("cannot update task status: " + err.Error())
				}
			},
			func() {
				_, err = globals.Pool.Exec("UPDATE tasks SET status = ? WHERE task_id = ?", store.TaskStatusSuccess, taskId)
				if err != nil {
					log.Println("cannot update task status to success: " + err.Error())
				}
			},
			func() {
				tasks.Unregister(taskId)
				stream.Delete(taskId)
			},
		)

		return helpers.Data(taskId), nil
	})
}
