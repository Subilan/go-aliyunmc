package monitors

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"time"

	"github.com/Subilan/gomc-server/config"
	"github.com/Subilan/gomc-server/globals"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
)

type ActiveInstanceStatusMonitor struct {
	SkipSign    bool
	StopSign    bool
	RunningSign bool
	Ctx         context.Context
	ErrChan     chan error
	Logger      *log.Logger
}

const ActiveInstanceStatusMonitorCycleTimeout = 30 * time.Second

var GlobalActiveInstanceStatusMonitor *ActiveInstanceStatusMonitor

func NewActiveInstanceStatusMonitor(ctx context.Context, errChan chan error) *ActiveInstanceStatusMonitor {
	logger := log.New(os.Stdout, "[ActiveInstanceStatusMonitor] ", 0)
	return &ActiveInstanceStatusMonitor{
		Ctx:     ctx,
		ErrChan: errChan,
		Logger:  logger,
	}
}

func (m *ActiveInstanceStatusMonitor) Run() {
	m.Logger.Println("Starting")
	m.StopSign = false
	m.SkipSign = false
	m.RunningSign = true
	go m.Main()
}

func (m *ActiveInstanceStatusMonitor) Pause() {
	m.Logger.Println("Pausing subsequent execution")
	m.SkipSign = true
}

func (m *ActiveInstanceStatusMonitor) Stop() {
	m.Logger.Println("Stopping")
	m.StopSign = true
	m.RunningSign = false
}

func (m *ActiveInstanceStatusMonitor) Main() {
	var err error
	cfg := config.Cfg.Monitor.ActiveInstanceStatusMonitor

	for {
		var updateRes sql.Result
		var rowsAffected int64
		var ctx context.Context
		var cancel context.CancelFunc
		var activeInstanceId string
		var describeInstanceStatusRequest *ecs20140526.DescribeInstanceStatusRequest
		var describeInstanceStatusResponse *ecs20140526.DescribeInstanceStatusResponse

		if m.StopSign {
			break
		}

		if m.SkipSign {
			goto end
		}

		ctx, cancel = context.WithTimeout(m.Ctx, ActiveInstanceStatusMonitorCycleTimeout)

		err = globals.Pool.QueryRowContext(ctx, "SELECT instance_id FROM instances WHERE deleted_at IS NULL").Scan(&activeInstanceId)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				if cfg.Verbose {
					m.Logger.Println("No active instance found, skipping")
				}
				goto end
			}

			m.Logger.Printf("Error querying active instance status: %v\n", err)
			m.ErrChan <- err
			goto end
		}

		describeInstanceStatusRequest = &ecs20140526.DescribeInstanceStatusRequest{
			RegionId:   tea.String(config.Cfg.Aliyun.RegionId),
			InstanceId: tea.StringSlice([]string{activeInstanceId}),
		}

		describeInstanceStatusResponse, err = globals.EcsClient.DescribeInstanceStatus(describeInstanceStatusRequest)

		if err != nil {
			m.Logger.Printf("Error describing active instance status: %v\n", err)
			m.ErrChan <- err
			goto end
		}

		if len(describeInstanceStatusResponse.Body.InstanceStatuses.InstanceStatus) > 0 {
			status := tea.StringValue(describeInstanceStatusResponse.Body.InstanceStatuses.InstanceStatus[0].Status)

			updateRes, err = globals.Pool.ExecContext(ctx, "UPDATE instance_statuses SET status = ? WHERE instance_id = ?", status, activeInstanceId)

			if err != nil {
				m.Logger.Printf("Error updating active instance status: %v\n", err)
				m.ErrChan <- err
				goto end
			}

			rowsAffected, err = updateRes.RowsAffected()

			if err != nil {
				m.Logger.Printf("You don't seem to have RowsAffected! Updated active instance status to %s\n", status)
				m.ErrChan <- err
				goto end
			}

			if rowsAffected > 0 {
				m.Logger.Printf("Updated active instance status to %s\n", status)
			}
		} else {
			if cfg.Verbose {
				m.Logger.Println("No instance returned from remote")
			}
		}

	end:
		cancel()
		time.Sleep(5 * time.Second)
	}
}
