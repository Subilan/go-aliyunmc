package events

import "github.com/gin-gonic/gin"

type ServerEventType string

const (
	ServerEventNotify              ServerEventType = "notify"
	ServerEventOnlineCountUpdate   ServerEventType = "online_count_update"
	ServerEventOnlinePlayersUpdate ServerEventType = "online_players_update"
)

const (
	ServerNotificationClosed  = "closed"
	ServerNotificationRunning = "running"
)

func Server(typ ServerEventType, data any, isPublic ...bool) *Event {
	public := false
	if len(isPublic) > 0 {
		public = isPublic[0]
	}
	return Stateless(gin.H{"type": typ, "data": data}, TypeServer, public)
}
