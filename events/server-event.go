package events

import "github.com/gin-gonic/gin"

// ServerEventType 表示一个与服务器相关的事件类型
//   - ServerEventNotify 表示一个与服务器相关的通知事件
//   - ServerEventOnlineCountUpdate 表示服务器玩家数量的更新事件
//   - ServerEventOnlinePlayersUpdate 表示服务器在线玩家列表的更新事件
type ServerEventType string

const (
	ServerEventNotify              ServerEventType = "notify"
	ServerEventOnlineCountUpdate   ServerEventType = "online_count_update"
	ServerEventOnlinePlayersUpdate ServerEventType = "online_players_update"
)

const (
	// ServerNotificationClosed 表示服务器已关闭
	ServerNotificationClosed = "closed"
	// ServerNotificationRunning 表示服务器正在运行
	ServerNotificationRunning = "running"
)

// Server 创建一个指定类型、带有指定载荷的服务器事件
func Server(typ ServerEventType, data any, isPublic ...bool) *Event {
	public := false
	if len(isPublic) > 0 {
		public = isPublic[0]
	}
	return Stateless(gin.H{"type": typ, "data": data}, TypeServer, public)
}
