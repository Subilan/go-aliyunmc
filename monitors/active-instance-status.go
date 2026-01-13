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
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/Subilan/go-aliyunmc/helpers/stream"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
)

var instanceStatusBroker = helpers.NewBroker[consts.InstanceStatus]()
var isInstancePresentBroker = helpers.NewBroker[bool]()

var instanceStatus consts.InstanceStatus
var instanceStatusMu sync.RWMutex
var isInstancePresent atomic.Bool

func SubscribeInstanceStatus() <-chan consts.InstanceStatus {
	return instanceStatusBroker.Subscribe()
}

func SnapshotInstanceStatus() consts.InstanceStatus {
	instanceStatusMu.RLock()
	defer instanceStatusMu.RUnlock()

	return instanceStatus
}

func syncInstanceStatusWithUser(logger *log.Logger) {
	instanceStatusUpdate := instanceStatusBroker.Subscribe()
	for newInstanceStatus := range instanceStatusUpdate {
		event := store.BuildInstanceEvent(store.InstanceEventActiveStatusUpdate, newInstanceStatus, true)
		err := stream.BroadcastAndSave(event)

		if err != nil {
			logger.Println("Error broadcast and save event", err)
		}
	}
}

func syncInstanceExternalDeletionWithUser(logger *log.Logger) {
	isInstancePresentUpdate := isInstancePresentBroker.Subscribe()
	for newIsInstancePresent := range isInstancePresentUpdate {
		if !newIsInstancePresent {
			event := store.BuildInstanceEvent(store.InstanceEventNotify, store.InstanceNotificationDeleted, true)
			err := stream.BroadcastAndSave(event)

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

	instanceStatusBroker.Publish(status)

	if status == consts.InstanceInvalid {
		if isInstancePresent.Load() == true {
			isInstancePresent.Store(false)
			isInstancePresentBroker.Publish(false)
		}
	} else {
		if isInstancePresent.Load() == false {
			isInstancePresent.Store(true)
			isInstancePresentBroker.Publish(true)
		}
	}
}

func ActiveInstance(quit chan bool) {
	logger := log.New(os.Stdout, "[ActiveInstance] ", log.LstdFlags)
	logger.Println("starting...")

	cfg := config.Cfg.Monitor.ActiveInstance

	ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)

	go instanceStatusBroker.Start()
	go isInstancePresentBroker.Start()
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
					if instanceStatus != consts.InstanceUnableToGet {
						setInstanceStatus(consts.InstanceUnableToGet)
					}
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
