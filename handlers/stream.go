package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/Subilan/gomc-server/globals"
	"github.com/Subilan/gomc-server/helpers"
	"github.com/Subilan/gomc-server/helpers/store"
	"github.com/Subilan/gomc-server/helpers/stream"
	"github.com/gin-gonic/gin"
	"go.jetify.com/sse"
)

func HandleBeginStream() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userId, exists := c.Get("user_id")

		if !exists {
			c.JSON(http.StatusUnauthorized, helpers.Details("找不到用户凭据"))
			return
		}

		userIdInt := int(userId.(int64))

		lastEventId := c.GetHeader("Last-Event-Id")
		lastState, _ := store.PushedEventStateFromString(lastEventId)

		conn, err := sse.Upgrade(ctx, c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, helpers.Details(err.Error()))
			return
		}
		defer conn.Close()

		userStream := stream.RegisterUser(userIdInt, conn, ctx)
		defer stream.UnregisterUser(userIdInt)

		lastOrd := 0
		// 前端携带了Last-Event-Id，需要进行同步
		if lastState != nil {
			var taskStatus string
			err = globals.Pool.QueryRow("SELECT `status` FROM tasks WHERE task_id = ?", lastState.TaskId).Scan(&taskStatus)

			if err != nil {
				log.Println("cannot get task status:", err)
				_ = conn.SendEvent(ctx, helpers.ErrorEvent("invalid last-event-id, cannot retrieve status from db", helpers.EventErrorInvalidLastEventId))
				return
			}

			if taskStatus != "running" {
				_ = conn.SendEvent(ctx, helpers.ErrorEvent("invalid last-event-id, corresponding task is not running, please use dedicated GET handlers instead", helpers.EventErrorInvalidLastEventId))
				return
			}

			// 获取当前数据库所有相关信息
			pushedEvents, err := store.GetPushedEventsAfterOrd(*lastState.TaskId, *lastState.Ord)

			if err != nil {
				log.Println("cannot get addendum from database", err.Error())
			} else {
				for _, event := range pushedEvents {
					log.Printf("sending addendum %v\n", event)
					err = conn.SendEvent(ctx, event.SSE())
					if err != nil {
						log.Println("cannot send event: ", err.Error())
						continue
					}
					lastOrd = *event.Ord
					log.Printf("debug: lastOrd -> %v\n", lastOrd)
				}
			}
		}

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case e := <-userStream.Chan:
				state, _ := store.PushedEventStateFromString(e.ID)

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
