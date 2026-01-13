package monitors

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/Subilan/go-aliyunmc/helpers/stream"
	"github.com/mcstatus-io/mcutil/v4/query"
	"github.com/mcstatus-io/mcutil/v4/response"
	"github.com/mcstatus-io/mcutil/v4/status"
)

var isServerRunningBroker = helpers.NewBroker[bool]()
var onlinePlayersBroker = helpers.NewBroker[[]string]()
var playerCountBroker = helpers.NewBroker[int64]()

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
	isServerRunningUpdate := isServerRunningBroker.Subscribe()
	for newIsServerRunning := range isServerRunningUpdate {
		var data any

		if newIsServerRunning {
			data = store.ServerNotificationRunning
		} else {
			data = store.ServerNotificationClosed
		}

		event := store.BuildServerEvent(store.ServerEventNotify, data, true)
		err := stream.BroadcastAndSave(event)

		if err != nil {
			log.Println("cannot broadcast server event:", err)
		}
	}
}

func syncOnlineCountWithUser() {
	playerCountUpdate := playerCountBroker.Subscribe()
	for onlineCount := range playerCountUpdate {
		event := store.BuildServerEvent(store.ServerEventOnlineCountUpdate, onlineCount, true)
		err := stream.BroadcastAndSave(event)

		if err != nil {
			log.Println("cannot broadcast server event:", err)
		}
	}
}

func syncOnlinePlayersWithUser() {
	onlinePlayersUpdate := onlinePlayersBroker.Subscribe()

	for newOnlinePlayers := range onlinePlayersUpdate {
		marshalled, err := json.Marshal(newOnlinePlayers)

		if err != nil {
			log.Println("cannot marshal online players:", err)
			return
		}

		event := store.BuildServerEvent(store.ServerEventOnlinePlayersUpdate, string(marshalled), true)
		err = stream.BroadcastAndSave(event)

		if err != nil {
			log.Println("cannot broadcast server event:", err)
		}
	}
}

func setServerStatus(status bool) {
	isServerRunning.Store(status)
	isServerRunningBroker.Publish(status)

	if !status {
		playerCount.Store(0)
		playerCountBroker.Publish(0)

		onlinePlayersMu.Lock()
		onlinePlayers = []string{}
		onlinePlayersMu.Unlock()

		onlinePlayersBroker.Publish(onlinePlayers)
	}
}

func ServerStatus(quit chan bool) {
	var err error

	cfg := config.Cfg.Monitor.ServerStatus
	ticker := time.NewTicker(cfg.IntervalDuration())

	logger := log.New(os.Stdout, "[ServerStatus] ", log.LstdFlags)
	logger.Println("starting...")

	go isServerRunningBroker.Start()
	go onlinePlayersBroker.Start()
	go playerCountBroker.Start()
	go syncServerStatusWithUser()
	go syncOnlineCountWithUser()
	go syncOnlinePlayersWithUser()

	for {
		select {
		case <-ticker.C:
			func() {
				ctx, cancel := context.WithTimeout(context.Background(), cfg.TimeoutDuration())
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
				serverStatus, err = status.Modern(ctx, currentInstanceIp, config.Cfg.GetGamePort())
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
						playerCountBroker.Publish(playerCount.Load())
					}

					if playerCount.Load() > 0 {
						queryFull, err := query.Full(ctx, currentInstanceIp, config.Cfg.GetGamePort())

						if err != nil {
							log.Println("cannot query full:", err)
						} else {
							// rlock here?
							if !helpers.SameStringSlice(onlinePlayers, queryFull.Players) {
								onlinePlayersMu.Lock()
								onlinePlayers = queryFull.Players
								onlinePlayersMu.Unlock()
								onlinePlayersBroker.Publish(onlinePlayers)
							}
						}
					} else if len(onlinePlayers) > 0 {
						onlinePlayersMu.Lock()
						onlinePlayers = []string{}
						onlinePlayersMu.Unlock()
						onlinePlayersBroker.Publish(onlinePlayers)
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
