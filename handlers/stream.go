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
		lastStateSynchronized := false

		conn, err := sse.Upgrade(ctx, c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, helpers.Details(err.Error()))
			return
		}
		defer conn.Close()

		var userStream = stream.RegisterStream(userIdInt, conn, ctx)
		defer stream.UnregisterStream(userIdInt)

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case e := <-userStream.Chan:
				// 判断是否存在有效的同步任务（出现在重连且存在task_id、ord信息时）
				if lastState != nil && !lastStateSynchronized {
					// 前端携带了Last-Event-Id，需要进行同步

					// 获取当前传来信息的状态
					currentState, err := stream.StateFromString(e.ID)

					if err != nil {
						log.Println("cannot get current state from string: ", currentState, "; error: ", err.Error())
					} else {
						var rows *sql.Rows
						rows, err = globals.Pool.Query("SELECT task_id, ord, type, is_error, content FROM pushed_events WHERE task_id = ? AND ord > ? AND ord < ?", lastState.TaskId, lastState.Ord, currentState.Ord)

						if err != nil {
							log.Println("cannot get addendum from database: ", err.Error())
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

								err = conn.SendEvent(ctx, stream.BuildEvent(stream.Event{
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
							}
							lastStateSynchronized = true
						}
					}
				}
				err = conn.SendEvent(ctx, e)
				if err != nil {
					log.Println("cannot send event:", err)
					return
				}

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
