package instances

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/gctx"
	"github.com/Subilan/go-aliyunmc/helpers/remote"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/Subilan/go-aliyunmc/helpers/stream"
	"github.com/Subilan/go-aliyunmc/helpers/tasks"
	"github.com/Subilan/go-aliyunmc/helpers/templateData"
	"github.com/Subilan/go-aliyunmc/monitors"
	"github.com/gin-gonic/gin"
)

var deployInstanceMutex sync.Mutex
var deployInstanceTaskStatusBroker = helpers.NewBroker[store.TaskStatus]()

func StartDeployInstanceTaskStatusBroker() {
	go deployInstanceTaskStatusBroker.Start()
}

func SubscribeDeployInstanceTaskStatus() <-chan store.TaskStatus {
	return deployInstanceTaskStatusBroker.Subscribe()
}

func syncDeployInstanceStatusWithUser(taskId string) {
	taskStatusUpdate := deployInstanceTaskStatusBroker.Subscribe()

	for taskStatus := range taskStatusUpdate {
		updateAndSend(taskId, taskStatus)
	}
}

func updateAndSend(taskId string, taskStatus store.TaskStatus) {
	_, err := db.Pool.Exec("UPDATE tasks SET status = ? WHERE task_id = ?", taskStatus, taskId)

	if err != nil {
		log.Println("cannot update task status: " + err.Error())
	}

	event := store.BuildInstanceEvent(store.InstanceEventDeploymentTaskStatusUpdate, taskStatus)
	err = stream.BroadcastAndSave(event)

	if err != nil {
		log.Println("cannot send and save instance event", err)
	}
}

func deployInstance() helpers.BasicHandlerFunc {
	return func(c *gin.Context) (any, error) {
		ok := deployInstanceMutex.TryLock()
		if !ok {
			return nil, &helpers.HttpError{Code: http.StatusForbidden, Details: "instance is being deployed"}
		}
		defer deployInstanceMutex.Unlock()

		userId, err := gctx.ShouldGetUserId(c)

		if err != nil {
			return nil, err
		}

		// 检查实例是否处于运行状态
		if monitors.SnapshotInstanceStatus() != consts.InstanceRunning {
			return nil, &helpers.HttpError{Code: http.StatusBadRequest, Details: "instance is not running"}
		}

		var ip string

		err = db.Pool.QueryRow("SELECT ip FROM instances WHERE ip IS NOT NULL AND deleted_at IS NULL AND deployed = 0").Scan(&ip)

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

		// 与用户和数据库同步
		go syncDeployInstanceStatusWithUser(taskId)

		// 创建当前任务的全局流
		stream.CreateState(taskId)

		// 创建超时上下文
		runCtx, cancelRunCtx := context.WithTimeout(context.Background(), 5*time.Minute)

		// 记录取消函数
		tasks.Register(cancelRunCtx, taskId)

		// 更新为运行状态（此时不进行Publish）
		updateAndSend(taskId, store.TaskStatusRunning)

		// 运行并借助全局流输出内容
		go remote.RunScriptAsRootAsync(runCtx, ip, "deploy.tmpl.sh", templateData.Deploy(),
			func(bytes []byte) {
				//log.Println("debug: deploy.sh stdout: ", string(bytes))

				state := stream.GetState(taskId)

				err = stream.BroadcastAndSave(&store.PushedEvent{
					PushedEventState: *state,
					Content:          string(bytes),
				})

				stream.IncrStateOrd(taskId)

				if err != nil {
					log.Println(err.Error())
					log.Printf("cannot send and save script step: userid=%v, deployment, is_error=false, content=%s\n", userId, string(bytes))
				}
			},
			func(err error) {
				log.Println("dedug: deploy.sh stderr: ", err.Error())

				state := stream.GetState(taskId)
				sendAndSaveError := stream.BroadcastAndSave(&store.PushedEvent{
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

				deployInstanceTaskStatusBroker.Publish(status)
			},
			func() {
				_, err = db.Pool.Exec("UPDATE instances SET deployed = 1 WHERE deleted_at IS NULL")
				if err != nil {
					log.Println("cannot update instance deployed status: " + err.Error())
				}

				deployInstanceTaskStatusBroker.Publish(store.TaskStatusSuccess)
			},
			func() {
				tasks.Unregister(taskId)
				stream.DeleteState(taskId)
			},
		)

		return helpers.Data(taskId), nil
	}
}

func HandleDeployInstance() gin.HandlerFunc {
	return helpers.BasicHandler(deployInstance())
}
