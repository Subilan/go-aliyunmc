package globals

import "github.com/mcstatus-io/mcutil/v4/response"

var IsServerRunning = false
var ServerStatus *response.StatusModern
var PlayerCount int64 = 0
var OnlinePlayers = make([]string, 0, 20)
