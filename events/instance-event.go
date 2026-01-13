package events

import "github.com/gin-gonic/gin"

type InstanceEventType string

const (
	InstanceEventNotify                     InstanceEventType = "notify"
	InstanceEventActiveStatusUpdate         InstanceEventType = "active_status_update"
	InstanceEventActiveIpUpdate             InstanceEventType = "active_ip_update"
	InstanceEventCreated                    InstanceEventType = "created"
	InstanceEventDeploymentTaskStatusUpdate InstanceEventType = "deployment_task_status_update"
	InstanceEventCreateAndDeployFailed      InstanceEventType = "create_and_deploy_failed"
	InstanceEventCreateAndDeployStep        InstanceEventType = "create_and_deploy_step"
)

const (
	InstanceNotificationDeleted = "instance_deleted"
)

func Instance(typ InstanceEventType, data any, isPublic ...bool) *Event {
	public := false
	if len(isPublic) > 0 {
		public = isPublic[0]
	}

	return Stateless(gin.H{"type": typ, "data": data}, TypeInstance, public)
}
