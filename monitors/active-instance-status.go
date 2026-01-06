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

type ActiveInstanceStatusMonitor struct {
	skipSign    bool
	stopSign    bool
	runningSign bool
	ctx         context.Context
	errChan     chan error
	logger      *log.Logger
}

const ActiveInstanceStatusMonitorCycleTimeout = 30 * time.Second

var GlobalActiveInstanceStatusMonitor *ActiveInstanceStatusMonitor

func NewActiveInstanceStatusMonitor(ctx context.Context, errChan chan error) *ActiveInstanceStatusMonitor {
	logger := log.New(os.Stdout, "[ActiveInstanceStatusMonitor] ", log.LstdFlags)
	return &ActiveInstanceStatusMonitor{
		ctx:     ctx,
		errChan: errChan,
		logger:  logger,
	}
}

func (m *ActiveInstanceStatusMonitor) Run() {
	m.logger.Println("Starting")
	m.stopSign = false
	m.skipSign = false
	m.runningSign = true
	go m.Main()
}

func (m *ActiveInstanceStatusMonitor) Pause() {
	m.logger.Println("Pausing subsequent execution")
	m.skipSign = true
}

func (m *ActiveInstanceStatusMonitor) Stop() {
	m.logger.Println("Stopping")
	m.stopSign = true
	m.runningSign = false
}

func (m *ActiveInstanceStatusMonitor) syncWithUser(status string, save bool) {
	event, err := store.BuildInstanceEvent(store.InstanceEventActiveStatusUpdate, status)

	if err != nil {
		m.logger.Printf("Error building event: %v", err)
		return
	}

	if save {
		err = stream.BroadcastAndSave(event)

		if err != nil {
			m.logger.Printf("Error sending event: %v", err)
		}
	} else {
		stream.Broadcast(event)
	}
}

func (m *ActiveInstanceStatusMonitor) Main() {
	var err error
	var updateRes sql.Result
	var rowsAffected int64
	var ctx context.Context
	var cancel context.CancelFunc
	var activeInstanceId string
	var describeInstanceStatusRequest *ecs20140526.DescribeInstanceStatusRequest
	var describeInstanceStatusResponse *ecs20140526.DescribeInstanceStatusResponse

	cfg := config.Cfg.Monitor.ActiveInstanceStatusMonitor

	for {
		if m.stopSign {
			break
		}

		if m.skipSign {
			goto end
		}

		ctx, cancel = context.WithTimeout(m.ctx, ActiveInstanceStatusMonitorCycleTimeout)

		err = db.Pool.QueryRowContext(ctx, "SELECT instance_id FROM instances WHERE deleted_at IS NULL").Scan(&activeInstanceId)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				if cfg.Verbose {
					m.logger.Println("No active instance found, skipping")
				}
				goto end
			}

			m.logger.Printf("Error querying active instance status: %v\n", err)
			m.errChan <- err
			goto end
		}

		describeInstanceStatusRequest = &ecs20140526.DescribeInstanceStatusRequest{
			RegionId:   tea.String(config.Cfg.Aliyun.RegionId),
			InstanceId: tea.StringSlice([]string{activeInstanceId}),
		}

		describeInstanceStatusResponse, err = globals.EcsClient.DescribeInstanceStatus(describeInstanceStatusRequest)

		if err != nil {
			m.syncWithUser("unable_to_get", false)
			m.logger.Printf("Error describing active instance status: %v\n", err)
			m.errChan <- err
			goto end
		}

		if len(describeInstanceStatusResponse.Body.InstanceStatuses.InstanceStatus) > 0 {
			status := tea.StringValue(describeInstanceStatusResponse.Body.InstanceStatuses.InstanceStatus[0].Status)

			updateRes, err = db.Pool.ExecContext(ctx, "UPDATE instance_statuses SET status = ? WHERE instance_id = ?", status, activeInstanceId)

			if err != nil {
				m.logger.Printf("Error updating active instance status: %v\n", err)
				m.errChan <- err
				goto end
			}

			rowsAffected, err = updateRes.RowsAffected()

			if err != nil {
				m.logger.Printf("Why is this even possible? Updated active instance status to %s\n", status)
				m.errChan <- err
				goto end
			}

			if rowsAffected > 0 {
				// 推送并记录实例状态的更新
				m.syncWithUser(status, true)
				m.logger.Printf("Updated active instance status to %s\n", status)
			}
		} else {
			if cfg.Verbose {
				m.logger.Println("No instance returned from remote")
			}
		}

	end:
		cancel()
		time.Sleep(time.Duration(config.Cfg.Monitor.ActiveInstanceStatusMonitor.ExecutionInterval) * time.Second)
	}
}
