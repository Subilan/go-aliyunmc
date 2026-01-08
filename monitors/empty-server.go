package monitors

import (
	"context"
	"log"
	"os"
	"time"

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
	activeInstance, err := store.GetIpAllocatedActiveInstance()

	if err != nil {
		logger.Println("cannot get ip allocated active instance:", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := commands.StopAndArchiveServer(ctx, *activeInstance.Ip, nil); err != nil {
		logger.Println("cannot stop and archive server:", err)
		return
	}

	if err := helpers.DeleteInstance(ctx, activeInstance.InstanceId, true); err != nil {
		logger.Println("cannot delete instance:", err)
		return
	}
}

func EmptyServer() {
	logger := log.New(os.Stdout, "[EmptyServer] ", log.LstdFlags)

	const emptyTimeout = 1 * time.Hour

	var (
		state = emptyServerStateIdle
		timer *time.Timer
	)

	for {
		select {
		case cnt := <-playerCountUpdate:
			switch {
			case cnt > 0:
				// 有玩家进入，取消空服计时
				if timer != nil {
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timer = nil
				}
				state = emptyServerStateIdle
				logger.Println("player joined, cancel empty-server timer")

			case cnt == 0 && state == emptyServerStateIdle:
				// 从非空转为空，启动计时
				timer = time.NewTimer(emptyTimeout)
				state = emptyServerStateWatching
				logger.Println("server empty, start empty-server timer")
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
		}
	}
}
