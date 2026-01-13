package instances

import (
	"time"

	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/events"
	"github.com/Subilan/go-aliyunmc/events/stream"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/commands"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/Subilan/go-aliyunmc/monitors"
	"github.com/gin-gonic/gin"
)

func HandleCreateAndDeployInstance() gin.HandlerFunc {
	return helpers.QueryHandler[CreateInstanceQuery](func(query CreateInstanceQuery, c *gin.Context) (any, error) {
		createInstanceFunc := createPreferredInstance()
		deployInstanceFunc := deployInstance()

		go func() {
			_, err := createInstanceFunc(query, c)

			if err != nil {
				event := events.Instance(events.InstanceEventCreateAndDeployFailed, err.Error())
				stream.Broadcast(event)
				return
			}

			timeout := time.NewTimer(25 * time.Second)
			instanceStatusUpdate := monitors.SubscribeInstanceStatus()

		loop1:
			for {
				select {
				case status := <-instanceStatusUpdate:
					if status == consts.InstanceRunning {
						event := events.Instance(events.InstanceEventCreateAndDeployStep, "waiting for instance to be initialized")
						stream.Broadcast(event)
						time.Sleep(20 * time.Second)
						_, err = deployInstanceFunc(c)

						if err != nil {
							event := events.Instance(events.InstanceEventCreateAndDeployFailed, err.Error())
							stream.Broadcast(event)
							return
						}

						event = events.Instance(events.InstanceEventCreateAndDeployStep, "requested instance deployment")
						stream.Broadcast(event)
						break loop1
					}
				case <-timeout.C:
					event := events.Instance(events.InstanceEventCreateAndDeployFailed, "timeout waiting for instance to be running")
					stream.Broadcast(event)
					return
				}
			}

			timeout = time.NewTimer(5 * time.Minute)
			deployInstanceStatusUpdate := SubscribeDeployInstanceTaskStatus()

		loop2:
			for {
				select {
				case taskStatus := <-deployInstanceStatusUpdate:
					if taskStatus == consts.TaskStatusSuccess {
						cmd, ok := commands.ShouldGetCommand(consts.CmdTypeStartServer)

						if !ok {
							event := events.Instance(events.InstanceEventCreateAndDeployFailed, "cannot find start_server command")
							stream.Broadcast(event)
							return
						}

						instance, err := store.GetDeployedActiveInstance()

						if err != nil {
							event := events.Instance(events.InstanceEventCreateAndDeployFailed, "cannot get instance: "+err.Error())
							stream.Broadcast(event)
							return
						}

						func() {
							ctx, cancel := cmd.DefaultContext()
							defer cancel()

							_, err = cmd.RunWithoutCooldown(ctx, *instance.Ip, nil, nil)

							if err != nil {
								event := events.Instance(events.InstanceEventCreateAndDeployFailed, "cannot start server: "+err.Error())
								stream.Broadcast(event)
								return
							}

							event := events.Instance(events.InstanceEventCreateAndDeployStep, "requested server start")
							stream.Broadcast(event)
						}()
						break loop2
					} else if taskStatus == consts.TaskStatusFailed {
						event := events.Instance(events.InstanceEventCreateAndDeployFailed, "deploy task failed")
						stream.Broadcast(event)
						return
					}
				case <-timeout.C:
					event := events.Instance(events.InstanceEventCreateAndDeployFailed, "timeout waiting for instance to be deployed")
					stream.Broadcast(event)
					return
				}
			}
		}()

		return gin.H{}, nil
	})
}
