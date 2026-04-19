package state_routes

import (
	"net/http"

	"go-aliyunmc/h"
	"go-aliyunmc/mid"
	"go-aliyunmc/sse"
	"go-aliyunmc/states"

	"github.com/gin-gonic/gin"
)

func Bind(router *gin.Engine) {
	stateGroup := router.Group("/state")
	authorized := stateGroup.Group("")
	authorized.Use(mid.Auth())
	authorized.Use(mid.Perm())
	{
		bindStateRoutes[states.ServerStatusState](authorized, "server-status", states.HSKeyServerStatus, "server_status_snapshot", "server_status_update")
		bindStateRoutes[string](authorized, "instance-status", states.HSKeyInstanceStatus, "instance_status_snapshot", "instance_status_update")
	}
}

func bindStateRoutes[T comparable](group *gin.RouterGroup, name string, key string, snapshotEvent string, updateEvent string) {
	group.GET("/snapshot/"+name, h.G(func(c *gin.Context) (any, error) {
		store, ok := states.GetRecordedHubbedStore[T](key)
		if !ok {
			return nil, h.HttpError(http.StatusServiceUnavailable, "暂无监控数据")
		}
		return store.Snapshot(), nil
	}))
	group.GET("/watch/"+name, handleStateWatch[T](key, snapshotEvent, updateEvent))
}

func handleStateWatch[T comparable](key string, snapshotEvent string, updateEvent string) gin.HandlerFunc {
	return func(c *gin.Context) {
		store, ok := states.GetRecordedHubbedStore[T](key)
		if !ok {
			c.JSON(http.StatusServiceUnavailable, h.Details(h.HttpError(http.StatusServiceUnavailable, "monitor尚未初始化")))
			return
		}

		client, err := sse.NewClient(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, h.Details(err))
			return
		}

		updates, unsubscribe := store.Subscribe()
		defer unsubscribe()

		if err := client.SendEvent(snapshotEvent, store.Snapshot()); err != nil {
			c.JSON(http.StatusInternalServerError, h.Details(err))
			return
		}

		go func() {
			for {
				select {
				case <-client.Done():
					return
				case snapshot, ok := <-updates:
					if !ok {
						return
					}
					if err := client.SendEvent(updateEvent, snapshot); err != nil {
						return
					}
				}
			}
		}()

		client.Listen()
	}
}
