package instances

import (
	"context"
	"database/sql"
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
	"github.com/google/uuid"
)

func HandleDeployInstance() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		var userId int

		userIdStr, exists := c.Get("user_id")

		if !exists || userIdStr == "" {
			return nil, &helpers.HttpError{Code: http.StatusUnauthorized, Details: "无效用户ID"}
		}

		userId = int(userIdStr.(int64))

		var instanceId string
		var ip string

		checkCtx, cancelCheckCtx := context.WithTimeout(c, 10*time.Second)
		defer cancelCheckCtx()

		// 检查是否存在有ip且正在运行的实例可供部署
		err := globals.Pool.QueryRowContext(checkCtx, "SELECT i.instance_id, i.ip FROM instances i JOIN instance_statuses s ON i.instance_id = s.instance_id WHERE i.ip IS NOT NULL AND i.deleted_at IS NULL AND s.status = 'Running'").Scan(&instanceId, &ip)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, &helpers.HttpError{Code: http.StatusBadRequest, Details: "没有符合要求的实例"}
			}

			return nil, err
		}

		// 检查是否存在部署任务正在运行
		var cnt int

		err = globals.Pool.QueryRowContext(checkCtx, "SELECT COUNT(*) FROM tasks WHERE `type` = ? AND status = ?", store.TaskTypeInstanceDeployment, store.TaskStatusRunning).Scan(&cnt)

		if err != nil {
			return nil, err
		}

		if cnt != 0 {
			return nil, &helpers.HttpError{Code: http.StatusConflict, Details: "已经存在部署任务正在运行"}
		}

		// 为新的部署任务分配UUID并插入记录
		taskIdU, err := uuid.NewRandom()

		_, err = globals.Pool.ExecContext(checkCtx, "INSERT INTO tasks (task_id, `type`, user_id) VALUES (?, ?, ?)", taskIdU.String(), store.TaskTypeInstanceDeployment, userId)

		if err != nil {
			return nil, err
		}

		taskId := taskIdU.String()

		cancelCheckCtx()

		// 创建当前任务的全局流
		stream.Create(taskId)

		// 创建超时上下文
		runCtx, cancelRunCtx := context.WithTimeout(context.Background(), 5*time.Minute)
		tasks.Register(cancelRunCtx, taskId)

		// 运行并借助全局流输出内容
		go remote.RunScriptAsRootAsync(runCtx, ip, "deploy.tmpl.sh", templateData.Deploy(), func(bytes []byte) {
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
		}, func(err error) {
			tasks.Unregister(taskId)

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
		}, func() {
			tasks.Unregister(taskId)
			_, err = globals.Pool.Exec("UPDATE tasks SET status = ? WHERE task_id = ?", store.TaskStatusSuccess, taskId)

			if err != nil {
				log.Println("cannot update task status to success: " + err.Error())
			}
		})

		return helpers.Data(taskId), nil
	})
}
