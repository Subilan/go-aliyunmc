package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/Subilan/go-aliyunmc/events"
	"github.com/Subilan/go-aliyunmc/events/stream"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
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
		lastState, _ := events.StateFromString(lastEventId)

		conn, err := sse.Upgrade(ctx, c.Writer, sse.WithWriteTimeout(10*time.Second), sse.WithHeartbeatInterval(5*time.Second))
		if err != nil {
			c.JSON(http.StatusInternalServerError, helpers.Details(err.Error()))
			return
		}
		defer conn.Close()

		streamId, userStream := stream.Register(userIdInt, conn, ctx)
		defer stream.Unregister(streamId)

		_ = conn.SendEvent(ctx, events.Sync(events.SyncEventClearLastEventId).SSE())

		lastOrd := 0
		// 前端携带了Last-Event-Id，需要进行同步
		if lastState != nil {
			var taskStatus string
			err = db.Pool.QueryRow("SELECT `status` FROM tasks WHERE task_id = ?", lastState.TaskId).Scan(&taskStatus)

			if err != nil {
				log.Println("cannot get task status of task id", lastState.TaskId)
				_ = conn.SendEvent(ctx, events.Error("invalid last-event-id, cannot retrieve status from db").SSE())
				return
			}

			// 获取当前数据库所有相关信息
			pushedEvents, err := store.GetEventsOfTask(*lastState.TaskId)

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

		for {
			select {
			case e := <-userStream.Chan:
				state, _ := events.StateFromString(e.ID)

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

			case <-ctx.Done():
				return
			}
		}
	}
}
