package state_routes

import (
	"net/http"

	"go-aliyunmc/h"
	"go-aliyunmc/mid"
	"go-aliyunmc/monitors"
	"go-aliyunmc/sse"
	"go-aliyunmc/states"

	"github.com/gin-gonic/gin"
)

type stateProvider[T any] interface {
	Snapshot() states.State[T]
	Subscribe() (<-chan states.State[T], func())
}

func Bind(router *gin.Engine) {
	stateGroup := router.Group("/state")
	authorized := stateGroup.Group("")
	authorized.Use(mid.Auth())
	authorized.Use(mid.Perm())
	{
		authorized.GET("/snapshot/server-status", h.G(func(c *gin.Context) (any, error) {
			return monitors.GetServerStatusMonitor().Snapshot(), nil
		}))
		authorized.GET("/watch/server-status", newWatchHandler(
			func() stateProvider[states.ServerStatusState] { return monitors.GetServerStatusMonitor() },
			"server_status_snapshot", "server_status_update",
		))

		authorized.GET("/snapshot/instance-status", h.G(func(c *gin.Context) (any, error) {
			return monitors.GetInstanceStatusMonitor().Snapshot(), nil
		}))
		authorized.GET("/watch/instance-status", newWatchHandler(
			func() stateProvider[string] { return monitors.GetInstanceStatusMonitor() },
			"instance_status_snapshot", "instance_status_update",
		))
	}
}

func newWatchHandler[T any](getProvider func() stateProvider[T], snapshotEvent string, updateEvent string) gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := getProvider()

		client, err := sse.NewClient(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, h.Details(err))
			return
		}

		updates, unsubscribe := provider.Subscribe()
		defer unsubscribe()

		if err := client.SendEvent(snapshotEvent, provider.Snapshot()); err != nil {
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
