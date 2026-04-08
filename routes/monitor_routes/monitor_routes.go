package monitor_routes

import (
	"net/http"

	"go-aliyunmc/h"
	"go-aliyunmc/mid"
	"go-aliyunmc/monitors"
	"go-aliyunmc/sse"

	"github.com/gin-gonic/gin"
)

func Bind(router *gin.Engine) {
	monitorGroup := router.Group("/monitor")
	authorized := monitorGroup.Group("")
	authorized.Use(mid.Auth())
	{
		bindMonitorRoutes(authorized, "server-status", monitors.ServerStatus, "server_status_snapshot", "server_status_update")
		bindMonitorRoutes(authorized, "instance-status", monitors.InstanceStatus, "instance_status_snapshot", "instance_status_update")
	}
}

func bindMonitorRoutes[T any](group *gin.RouterGroup, name string, getMonitor func() monitors.StateMonitor[T], snapshotEvent string, updateEvent string) {
	group.GET("/snapshot/"+name, h.G(func(c *gin.Context) (any, error) {
		monitor := getMonitor()
		if monitor == nil {
			return nil, h.HttpError(http.StatusServiceUnavailable, "monitor尚未初始化")
		}
		return monitor.Snapshot(), nil
	}))
	group.GET("/watch/"+name, handleMonitorWatch(getMonitor, snapshotEvent, updateEvent))
}

func handleMonitorWatch[T any](getMonitor func() monitors.StateMonitor[T], snapshotEvent string, updateEvent string) gin.HandlerFunc {
	return func(c *gin.Context) {
		monitor := getMonitor()
		if monitor == nil {
			c.JSON(http.StatusServiceUnavailable, h.Details(h.HttpError(http.StatusServiceUnavailable, "monitor尚未初始化")))
			return
		}

		client, err := sse.NewClient(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, h.Details(err))
			return
		}

		updates, unsubscribe := monitor.Subscribe()
		defer unsubscribe()

		if err := client.SendEvent(snapshotEvent, monitor.Snapshot()); err != nil {
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
