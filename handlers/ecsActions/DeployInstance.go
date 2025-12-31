package ecsActions

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Subilan/gomc-server/globals"
	"github.com/Subilan/gomc-server/helpers"
	"github.com/Subilan/gomc-server/stream"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const TaskTypeInstanceDeployment = "instance_deployment"

func DeployInstance() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		var userId int

		userIdStr, exists := c.Get("user_id")

		if !exists || userIdStr == "" {
			return nil, helpers.HttpError{Code: http.StatusUnauthorized, Details: "无效用户ID"}
		}

		userId = int(userIdStr.(int64))

		var instanceId string
		var ip string

		ctx, cancel := context.WithTimeout(c, 10*time.Second)
		defer cancel()

		// 检查是否存在有ip且正在运行的实例可供部署
		err := globals.Pool.QueryRowContext(ctx, "SELECT i.instance_id, i.ip FROM instances i JOIN instance_statuses s ON i.instance_id = s.instance_id WHERE i.ip IS NOT NULL AND i.deleted_at IS NULL AND s.status = 'Running'").Scan(&instanceId, &ip)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, helpers.HttpError{Code: http.StatusBadRequest, Details: "没有符合要求的实例"}
			}

			return nil, err
		}

		// 检查是否存在部署任务正在运行
		var cnt int

		err = globals.Pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE `type` = ? AND status = 'running'", TaskTypeInstanceDeployment).Scan(&cnt)

		if err != nil {
			return nil, err
		}

		if cnt != 0 {
			return nil, helpers.HttpError{Code: http.StatusConflict, Details: "已经存在部署任务正在运行"}
		}

		// 为新的部署任务分配UUID并插入记录
		taskIdU, err := uuid.NewRandom()

		_, err = globals.Pool.ExecContext(ctx, "INSERT INTO tasks (task_id, `type`, user_id) VALUES (?, ?, ?)", taskIdU.String(), TaskTypeInstanceDeployment, userId)

		if err != nil {
			return nil, err
		}

		taskId := taskIdU.String()

		// 创建当前任务的全局流
		stream.BeginGlobalStream(taskId, stream.Deployment)

		// 运行并借助全局流输出内容
		go helpers.RunRemoteScripts(ip, "deploy.tmpl.sh", func(bytes []byte) {
			log.Println("debug: deploy.sh stdout: ", string(bytes))
			err = stream.BroadcastAndSave(stream.Event{
				State:   stream.GetGlobalStreamState(taskId),
				Content: string(bytes),
			})

			stream.IncrGlobalStreamOrd(taskId)

			if err != nil {
				log.Println(err.Error())
				log.Printf("cannot send and save script step: userid=%v, deployment, is_error=false, content=%s\n", userId, string(bytes))
			}
		}, func(err error) {
			log.Println("dedug: deploy.sh stderr: ", err.Error())
			sendAndSaveError := stream.BroadcastAndSave(stream.Event{
				State:   stream.GetGlobalStreamState(taskId),
				IsError: true,
				Content: err.Error(),
			})

			if sendAndSaveError != nil {
				log.Println(sendAndSaveError.Error())
				log.Printf("cannot send and save script step: userid=%v, deployment, is_error=true, content=%s\n", userId, err.Error())
			}

			_, err = globals.Pool.Exec("UPDATE tasks SET status = 'failed' WHERE task_id = ?", taskId)

			if err != nil {
				log.Println("cannot update task status to failed: " + err.Error())
			}
		}, func() {
			_, err = globals.Pool.Exec("UPDATE tasks SET status = 'success' WHERE task_id = ?", taskId)

			if err != nil {
				log.Println("cannot update task status to success: " + err.Error())
			}
		})

		return gin.H{}, nil
	})
}
