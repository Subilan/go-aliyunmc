package states

var (
	HSKeyInstanceStatus = "instance_status_monitor"
	HSKeyServerStatus = "server_status_monitor"
)

type ServerStatusState struct {
	Online      bool  `json:"online"`
	PlayerCount int64 `json:"playerCount"`
}