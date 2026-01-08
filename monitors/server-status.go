package monitors

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/Subilan/go-aliyunmc/helpers/stream"
	"github.com/mcstatus-io/mcutil/v4/query"
	"github.com/mcstatus-io/mcutil/v4/response"
	"github.com/mcstatus-io/mcutil/v4/status"
)

var serverRunningUpdate = make(chan bool, 1)
var onlinePlayersUpdate = make(chan []string)
var playerCountUpdate = make(chan int64)

var isServerRunning atomic.Bool
var playerCount atomic.Int64

var serverStatus *response.StatusModern
var serverStatusMu sync.RWMutex
var onlinePlayers = make([]string, 0, 20)
var onlinePlayersMu sync.RWMutex

// SnapshotServerStatus 返回截止目前最新的服务器状态
func SnapshotServerStatus() *response.StatusModern {
	serverStatusMu.RLock()
	defer serverStatusMu.RUnlock()

	return serverStatus
}

// SnapshotOnlinePlayers 返回截止目前最新的玩家列表
func SnapshotOnlinePlayers() []string {
	onlinePlayersMu.RLock()
	defer onlinePlayersMu.RUnlock()

	return onlinePlayers
}

// SnapshotIsServerRunning 返回截止目前最新的服务器运行状态
func SnapshotIsServerRunning() bool {
	return isServerRunning.Load()
}

func syncServerStatusWithUser() {
	for serverRunning := range serverRunningUpdate {
		var data any

		if serverRunning {
			data = store.ServerNotificationRunning
		} else {
			data = store.ServerNotificationClosed
		}

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
}

func syncOnlineCountWithUser() {
	for onlineCount := range playerCountUpdate {
		event, err := store.BuildServerEvent(store.ServerEventOnlineCountUpdate, onlineCount)

		if err != nil {
			log.Println("cannot build server event:", err)
			return
		}

		err = stream.BroadcastAndSave(event)

		if err != nil {
			log.Println("cannot broadcast server event:", err)
		}
	}
}

func syncOnlinePlayersWithUser() {
	for onlinePlayers := range onlinePlayersUpdate {
		marshalled, err := json.Marshal(onlinePlayers)

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
}

func setServerStatus(status bool) {
	isServerRunning.Store(status)
	serverRunningUpdate <- status

	if !status {
		playerCount.Store(0)
		playerCountUpdate <- 0

		onlinePlayersMu.Lock()
		onlinePlayers = []string{}
		onlinePlayersMu.Unlock()

		onlinePlayersUpdate <- onlinePlayers
	}
}

func ServerStatus(quit chan bool) {
	var err error

	ticker := time.NewTicker(5 * time.Second)

	logger := log.New(os.Stdout, "[ServerStatus] ", log.LstdFlags)
	logger.Println("starting...")

	go syncServerStatusWithUser()
	go syncOnlineCountWithUser()
	go syncOnlinePlayersWithUser()

	for {
		select {
		case <-ticker.C:
			func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				currentInstanceStatus := SnapshotInstanceStatus()
				currentInstanceIp := SnapshotInstanceIp()

				if currentInstanceStatus != consts.InstanceRunning || currentInstanceIp == "" {
					if isServerRunning.Load() == true {
						setServerStatus(false)
					}
					return
				}

				serverStatusMu.Lock()
				serverStatus, err = status.Modern(ctx, currentInstanceIp, 25565)
				serverStatusMu.Unlock()

				if err != nil {
					if isServerRunning.Load() == true {
						setServerStatus(false)
					}
					return
				}

				if serverStatus.Players.Online == nil {
					log.Println("warn: unexpected online player count being nil")
				} else {
					if playerCount.Load() != *serverStatus.Players.Online {
						playerCount.Store(*serverStatus.Players.Online)
						playerCountUpdate <- playerCount.Load()
					}

					if playerCount.Load() > 0 {
						queryFull, err := query.Full(ctx, currentInstanceIp, 25565)

						if err != nil {
							log.Println("cannot query full:", err)
						} else {
							// rlock here?
							if !helpers.SameStringSlice(onlinePlayers, queryFull.Players) {
								onlinePlayersMu.Lock()
								onlinePlayers = queryFull.Players
								onlinePlayersMu.Unlock()
								onlinePlayersUpdate <- queryFull.Players
							}
						}
					} else if len(onlinePlayers) > 0 {
						onlinePlayersMu.Lock()
						onlinePlayers = []string{}
						onlinePlayersMu.Unlock()
						onlinePlayersUpdate <- onlinePlayers
					}
				}

				if isServerRunning.Load() == false {
					setServerStatus(true)
				}
			}()

		case <-quit:
			return
		}
	}
}
