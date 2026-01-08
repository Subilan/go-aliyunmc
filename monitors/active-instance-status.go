package monitors

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/Subilan/go-aliyunmc/helpers/stream"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
)

var instanceStatus consts.InstanceStatus
var instanceStatusUpdate = make(chan consts.InstanceStatus)
var instanceStatusMu sync.RWMutex

var instanceExist atomic.Bool
var instanceExistUpdate = make(chan bool)

func SnapshotInstanceStatus() consts.InstanceStatus {
	instanceStatusMu.RLock()
	defer instanceStatusMu.RUnlock()

	return instanceStatus
}

func syncInstanceStatusWithUser(logger *log.Logger) {
	for currentInstanceStatus := range instanceStatusUpdate {
		event, err := store.BuildInstanceEvent(store.InstanceEventActiveStatusUpdate, currentInstanceStatus)

		if err != nil {
			logger.Println("Error building event", err)
			continue
		}

		err = stream.BroadcastAndSave(event)

		if err != nil {
			logger.Println("Error broadcast and save event", err)
		}
	}
}

func syncInstanceExternalDeletionWithUser(logger *log.Logger) {
	for currentInstanceExist := range instanceExistUpdate {
		if !currentInstanceExist {
			event, err := store.BuildInstanceEvent(store.InstanceEventNotify, store.InstanceNotificationDeleted)

			if err != nil {
				logger.Println("Error building event", err)
				continue
			}

			err = stream.BroadcastAndSave(event)

			if err != nil {
				logger.Println("Error broadcast and save event", err)
			}
		}
	}
}

func setInstanceStatus(status consts.InstanceStatus) {
	instanceStatusMu.Lock()
	instanceStatus = status
	instanceStatusMu.Unlock()

	instanceStatusUpdate <- status

	if status == consts.InstanceInvalid {
		if instanceExist.Load() == true {
			instanceExist.Store(false)
			instanceExistUpdate <- false
		}
	} else {
		if instanceExist.Load() == false {
			instanceExist.Store(true)
			instanceExistUpdate <- true
		}
	}
}

func ActiveInstance(quit chan bool) {
	logger := log.New(os.Stdout, "[ActiveInstance] ", log.LstdFlags)
	logger.Println("starting...")

	cfg := config.Cfg.Monitor.ActiveInstanceStatusMonitor

	ticker := time.NewTicker(time.Duration(cfg.ExecutionInterval) * time.Second)

	go syncInstanceExternalDeletionWithUser(logger)
	go syncInstanceStatusWithUser(logger)

	for {
		select {
		case <-ticker.C:
			func() {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()

				var activeInstanceId string

				err := db.Pool.QueryRowContext(ctx, "SELECT instance_id FROM instances WHERE deleted_at IS NULL").Scan(&activeInstanceId)

				if err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						return
					}

					logger.Printf("Error querying active instance status: %v\n", err)
					return
				}

				describeInstanceStatusRequest := &ecs20140526.DescribeInstanceStatusRequest{
					RegionId:   tea.String(config.Cfg.Aliyun.RegionId),
					InstanceId: tea.StringSlice([]string{activeInstanceId}),
				}

				describeInstanceStatusResponse, err := globals.EcsClient.DescribeInstanceStatus(describeInstanceStatusRequest)

				if err != nil {
					event, _ := store.BuildInstanceEvent(store.InstanceEventActiveStatusUpdate, "unable_to_get")
					stream.Broadcast(event)
					logger.Printf("Error describing active instance status: %v\n", err)
					return
				}

				if len(describeInstanceStatusResponse.Body.InstanceStatuses.InstanceStatus) > 0 {
					newInstanceStatus := consts.InstanceStatus(tea.StringValue(describeInstanceStatusResponse.Body.InstanceStatuses.InstanceStatus[0].Status))

					if instanceStatus != newInstanceStatus {
						setInstanceStatus(newInstanceStatus)
						logger.Printf("Updated active instance status to %s\n", instanceStatus)
					}
				} else {
					// 请求成功了但没有找到符合要求的实例，说明实例被外部删除
					setInstanceStatus(consts.InstanceInvalid)

					logger.Println("Active instance is externally deleted, updating database.")

					_, err = db.Pool.ExecContext(ctx, "UPDATE instances SET deleted_at = CURRENT_TIMESTAMP WHERE instance_id = ?", activeInstanceId)

					if err != nil {
						logger.Printf("Error updating active instance status: %v\n", err)
						return
					}
				}
			}()

		case <-quit:
			return
		}
	}
}
