package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/Subilan/gomc-server/globals"
	"github.com/Subilan/gomc-server/helpers"
	"github.com/Subilan/gomc-server/stream"
	"github.com/gin-gonic/gin"
	"go.jetify.com/sse"
)

func BeginStream() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userId, exists := c.Get("user_id")

		if !exists {
			c.JSON(http.StatusUnauthorized, helpers.Details("找不到用户凭据"))
			return
		}

		userIdInt := int(userId.(int64))

		lastEventId := c.GetHeader("Last-Event-Id")
		lastState, _ := stream.StateFromString(lastEventId)

		conn, err := sse.Upgrade(ctx, c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, helpers.Details(err.Error()))
			return
		}
		defer conn.Close()

		userStream := stream.RegisterStream(userIdInt, conn, ctx)
		defer stream.UnregisterStream(userIdInt)

		lastOrd := 0
		// 前端携带了Last-Event-Id，需要进行同步
		if lastState != nil {
			var taskStatus string
			err = globals.Pool.QueryRow("SELECT `status` FROM tasks WHERE task_id = ?", lastState.TaskId).Scan(&taskStatus)

			if err != nil {
				log.Println("cannot get task status:", err)
				_ = conn.SendEvent(ctx, stream.ErrorEvent("invalid last-event-id, cannot retrieve status from db", stream.EventErrorInvalidLastEventId))
				return
			}

			if taskStatus != "running" {
				_ = conn.SendEvent(ctx, stream.ErrorEvent("invalid last-event-id, corresponding task is not running, please use dedicated GET handlers instead", stream.EventErrorInvalidLastEventId))
				return
			}

			// 获取当前数据库所有相关信息
			var rows *sql.Rows
			rows, err = globals.Pool.Query("SELECT task_id, ord, type, is_error, content FROM pushed_events WHERE task_id = ? AND ord > ? ORDER BY ord", lastState.TaskId, lastState.Ord)

			if err != nil {
				log.Println("cannot get addendum from database", err.Error())
			} else {
				for rows.Next() {
					var taskId string
					var ord int
					var typ stream.Type
					var isError bool
					var content string
					err = rows.Scan(&taskId, &ord, &typ, &isError, &content)

					if err != nil {
						log.Println("cannot scan row: ", err.Error())
						continue
					}

					log.Printf("sending addendum taskId=%v, ord=%v, type=%v, content=%v\n", taskId, ord, typ, content)

					err = conn.SendEvent(ctx, stream.DataEvent(stream.Event{
						State: &stream.State{
							Ord:    &ord,
							TaskId: &taskId,
							Type:   typ,
						},
						IsError: isError,
						Content: content,
					}))

					if err != nil {
						log.Println("cannot send event: ", err.Error())
						continue
					}

					lastOrd = ord
					log.Printf("debug: lastOrd -> %v\n", lastOrd)
				}
			}
		}

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case e := <-userStream.Chan:
				state, _ := stream.StateFromString(e.ID)

				if lastState != nil {
					if state != nil && state.TaskId != nil && state.Ord != nil {
						if *state.TaskId == *lastState.TaskId && *state.Ord <= lastOrd {
							continue
						}
					}
				}

				err = conn.SendEvent(ctx, e)

				if err != nil {
					log.Println("cannot send event:", err)
					return
				}

				lastOrd = *state.Ord
				log.Printf("debug: lastOrd -> %v\n", lastOrd)

			case <-ticker.C:
				if err := conn.SendComment(ctx, "ping"); err != nil {
					// 心跳的错误处理应该保持简单
					log.Println("cannot ping client:", err)
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}
}
