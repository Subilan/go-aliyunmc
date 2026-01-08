package monitors

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/Subilan/go-aliyunmc/helpers/stream"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
)

func syncWithUser(logger *log.Logger, status string, save bool) {
	event, err := store.BuildInstanceEvent(store.InstanceEventActiveStatusUpdate, status)

	if err != nil {
		logger.Printf("Error building event: %v", err)
		return
	}

	if save {
		err = stream.BroadcastAndSave(event)

		if err != nil {
			logger.Printf("Error sending event: %v", err)
		}
	} else {
		stream.Broadcast(event)
	}
}

func ActiveInstanceStatus() {
	var err error
	var updateRes sql.Result
	var rowsAffected int64
	var ctx context.Context
	var cancel context.CancelFunc
	var activeInstanceId string
	var describeInstanceStatusRequest *ecs20140526.DescribeInstanceStatusRequest
	var describeInstanceStatusResponse *ecs20140526.DescribeInstanceStatusResponse

	logger := log.New(os.Stdout, "[ActiveInstanceStatus] ", log.LstdFlags)
	logger.Println("starting...")

	cfg := config.Cfg.Monitor.ActiveInstanceStatusMonitor

	for {
		ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)

		err = db.Pool.QueryRowContext(ctx, "SELECT instance_id FROM instances WHERE deleted_at IS NULL").Scan(&activeInstanceId)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				if cfg.Verbose {
					logger.Println("No active instance found, skipping")
				}
				goto end
			}

			logger.Printf("Error querying active instance status: %v\n", err)
			goto end
		}

		describeInstanceStatusRequest = &ecs20140526.DescribeInstanceStatusRequest{
			RegionId:   tea.String(config.Cfg.Aliyun.RegionId),
			InstanceId: tea.StringSlice([]string{activeInstanceId}),
		}

		describeInstanceStatusResponse, err = globals.EcsClient.DescribeInstanceStatus(describeInstanceStatusRequest)

		if err != nil {
			syncWithUser(logger, "unable_to_get", false)
			logger.Printf("Error describing active instance status: %v\n", err)
			goto end
		}

		if len(describeInstanceStatusResponse.Body.InstanceStatuses.InstanceStatus) > 0 {
			status := tea.StringValue(describeInstanceStatusResponse.Body.InstanceStatuses.InstanceStatus[0].Status)

			updateRes, err = db.Pool.ExecContext(ctx, "UPDATE instance_statuses SET status = ? WHERE instance_id = ?", status, activeInstanceId)

			if err != nil {
				logger.Printf("Error updating active instance status: %v\n", err)
				goto end
			}

			rowsAffected, err = updateRes.RowsAffected()

			if err != nil {
				logger.Printf("Why is this even possible? Updated active instance status to %s\n", status)
				goto end
			}

			if rowsAffected > 0 {
				// 推送并记录实例状态的更新
				syncWithUser(logger, status, true)
				logger.Printf("Updated active instance status to %s\n", status)
			}
		} else {
			if cfg.Verbose {
				logger.Println("No instance returned from remote")
			}
		}

	end:
		cancel()
		time.Sleep(time.Duration(config.Cfg.Monitor.ActiveInstanceStatusMonitor.ExecutionInterval) * time.Second)
	}
}
