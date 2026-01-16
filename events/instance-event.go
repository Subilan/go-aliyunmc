package events

import "github.com/gin-gonic/gin"

// InstanceEventType 表示一个与实例有关的事件类型
//   - InstanceEventNotify 表示一个关于实例的通知事件。此事件没有具体的参数（载荷），主要用于通知前端进行特定的操作或者更新。
//   - InstanceEventActiveStatusUpdate 表示实例状态更新事件，主要由 monitors.ActiveInstance 触发
//   - InstanceEventActiveIpUpdate 表示实例IP地址更新事件，主要由 monitors.PublicIP 触发
//   - InstanceEventCreated 表示实例被创建事件，此事件上会携带新实例信息（store.Instance）的载荷
//   - InstanceEventDeploymentTaskStatusUpdate 表示部署任务的状态的更新，主要由执行部署任务的 goroutine 触发
//   - InstanceEventCreateAndDeployFailed 表示“一键开启服务器”功能流程的失败，参见 instances.HandleCreateAndDeployInstance
//   - InstanceEventCreateAndDeployStep 表示“一键开启服务器”功能过程的状态更新，用于前端更新页面或告知用户，参见 instances.HandleCreateAndDeployInstance
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
	// InstanceNotificationDeleted 表示实例被删除
	InstanceNotificationDeleted = "instance_deleted"
)

// Instance 创建一个指定类型、带有指定载荷的实例事件
func Instance(typ InstanceEventType, data any, isPublic ...bool) *Event {
	public := false
	if len(isPublic) > 0 {
		public = isPublic[0]
	}

	return Stateless(gin.H{"type": typ, "data": data}, TypeInstance, public)
}
