package monitors

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/commands"
	"github.com/Subilan/go-aliyunmc/helpers/store"
)

type emptyServerState int

const (
	emptyServerStateIdle emptyServerState = iota
	emptyServerStateWatching
	emptyServerStateDeleting
)

func safeDeleteServer(logger *log.Logger) {
	activeInstance, err := store.GetDeployedActiveInstance()

	if err != nil {
		logger.Println("instance not found, skipping")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := commands.StopAndArchiveServer(ctx, *activeInstance.Ip, nil, "The server has been empty for too long."); err != nil {
		logger.Println("cannot stop and archive server:", err)
		return
	}

	logger.Println("stop and archive server successfully")

	if err := helpers.DeleteInstance(ctx, activeInstance.InstanceId, true); err != nil {
		logger.Println("cannot delete instance:", err)
		return
	}

	logger.Println("delete instance successfully")
}

func EmptyServer(quit chan bool) {
	cfg := config.Cfg.Monitor.EmptyServer
	logger := log.New(os.Stdout, "[EmptyServer] ", log.LstdFlags)
	logger.Println("starting...")

	var emptyTimeout = cfg.EmptyTimeoutDuration()

	var (
		state = emptyServerStateIdle
		timer *time.Timer
	)

	playerCountUpdate := playerCountBroker.Subscribe()

	for {
		select {
		case cnt := <-playerCountUpdate:
			switch {
			case cnt > 0 || cnt == -1:
				if timer != nil {
					timer.Stop()
					timer = nil
				}
				state = emptyServerStateIdle

				if cnt > 0 {
					logger.Println("player joined, cancel empty-server timer")
				} else {
					logger.Println("player count being -1, cancel empty-server timer")
				}

			case cnt == 0 && state == emptyServerStateIdle:
				// 从非空转为空，启动计时
				// 注意，如果服务器因为某种原因被关闭没有运行，也认为玩家数量为0
				timer = time.NewTimer(emptyTimeout)
				state = emptyServerStateWatching
				logger.Println("server empty, starting empty-server timer. the server will be safely deleted in", emptyTimeout)
			}

		case <-func() <-chan time.Time {
			if timer != nil {
				return timer.C
			}
			return nil
		}():
			// edge case
			if playerCount.Load() > 0 {
				state = emptyServerStateIdle
				timer = nil
				logger.Println("timer fired but players exist, abort")
				continue
			}

			state = emptyServerStateDeleting
			timer = nil

			logger.Println("empty timeout reached, deleting server")
			safeDeleteServer(logger)

			state = emptyServerStateIdle

		case <-quit:
			return
		}
	}
}
