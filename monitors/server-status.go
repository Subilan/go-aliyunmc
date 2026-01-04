package monitors

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/Subilan/go-aliyunmc/helpers/stream"
	"github.com/mcstatus-io/mcutil/v4/query"
	"github.com/mcstatus-io/mcutil/v4/response"
	"github.com/mcstatus-io/mcutil/v4/status"
)

func syncServerStatusWithUser() {
	var data any

	if globals.IsServerRunning {
		data = store.ServerNotificationRunning
	} else {
		data = store.ServerNotificationClosed
	}

	log.Println("syncing server status with user: ", data)

	event, err := store.BuildServerEvent(store.ServerEventNotify, data)

	if err != nil {
		log.Println("cannot build server event:", err)
	} else {
		err = stream.BroadcastAndSave(event)

		if err != nil {
			log.Println("cannot broadcast server event:", err)
		}
	}
}

func syncOnlineCountWithUser() {
	event, err := store.BuildServerEvent(store.ServerEventOnlineCountUpdate, globals.PlayerCount)

	if err != nil {
		log.Println("cannot build server event:", err)
		return
	}

	err = stream.BroadcastAndSave(event)

	if err != nil {
		log.Println("cannot broadcast server event:", err)
	}
}

func syncOnlinePlayersWithUser() {
	marshalled, err := json.Marshal(globals.OnlinePlayers)

	if err != nil {
		log.Println("cannot marshal online players:", err)
		return
	}

	event, err := store.BuildServerEvent(store.ServerEventOnlinePlayersUpdate, string(marshalled))

	if err != nil {
		log.Println("cannot build server event:", err)
		return
	}

	err = stream.BroadcastAndSave(event)

	if err != nil {
		log.Println("cannot broadcast server event:", err)
	}
}

func sameStringSlice(x, y []string) bool {
	if len(x) != len(y) {
		return false
	}
	diff := make(map[string]int, len(x))
	for _, _x := range x {
		diff[_x]++
	}
	for _, _y := range y {
		if _, ok := diff[_y]; !ok {
			return false
		}
		diff[_y]--
		if diff[_y] == 0 {
			delete(diff, _y)
		}
	}
	return len(diff) == 0
}

func ServerStatusMonitor() {
	var activeInstance *store.Instance
	var ctx context.Context
	var cancel context.CancelFunc
	var err error
	var queryFull *response.QueryFull

	logger := log.New(os.Stdout, "[ServerStatusMonitor] ", log.LstdFlags)
	logger.Println("Starting server status monitor")

	for {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)

		activeInstance = store.GetActiveInstance()

		if activeInstance == nil {
			if globals.IsServerRunning != false {
				globals.IsServerRunning = false
				syncServerStatusWithUser()
			}
			goto end
		}

		globals.ServerStatus, err = status.Modern(ctx, *activeInstance.Ip, 25565)

		if err != nil {
			if globals.IsServerRunning != false {
				globals.IsServerRunning = false
				syncServerStatusWithUser()
			}
			goto end
		}

		if globals.ServerStatus.Players.Online == nil {
			log.Println("warn: online player being nil")
		} else {
			if globals.PlayerCount != *globals.ServerStatus.Players.Online {
				globals.PlayerCount = *globals.ServerStatus.Players.Online
				syncOnlineCountWithUser()
			}

			if globals.PlayerCount > 0 {
				queryFull, err = query.Full(ctx, *activeInstance.Ip, 25565)

				if err != nil {
					log.Println("cannot query full:", err)
				} else {
					if !sameStringSlice(globals.OnlinePlayers, queryFull.Players) {
						globals.OnlinePlayers = queryFull.Players
						syncOnlinePlayersWithUser()
					}
				}
			}
		}

		if globals.IsServerRunning != true {
			globals.IsServerRunning = true
			syncServerStatusWithUser()
		}

	end:
		time.Sleep(5 * time.Second)
		cancel()
	}
}
