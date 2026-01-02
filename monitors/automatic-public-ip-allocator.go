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
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
)

type AutomaticPublicIPAllocator struct {
	stopSign    bool
	runningSign bool
	ctx         context.Context
	errChan     chan error
	logger      *log.Logger
}

var GlobalAutomaticPublicIPAllocator *AutomaticPublicIPAllocator

func NewAutomaticPublicIPAllocator(ctx context.Context, errChan chan error) *AutomaticPublicIPAllocator {
	logger := log.New(os.Stdout, "[AutomaticPublicIPAllocator] ", log.LstdFlags)
	return &AutomaticPublicIPAllocator{
		stopSign:    false,
		runningSign: false,
		ctx:         ctx,
		errChan:     errChan,
		logger:      logger,
	}
}

func (m *AutomaticPublicIPAllocator) Run() {
	m.runningSign = true
	m.stopSign = false

	go m.Main()
}

const AutomaticPublicIPAllocatorCycleTimeout = 20 * time.Second

func (m *AutomaticPublicIPAllocator) Main() {
	var err error
	var activeInstanceId string
	var cancel context.CancelFunc
	var ctx context.Context
	var allocatePublicIpAddressRequest *ecs20140526.AllocatePublicIpAddressRequest
	var allocatePublicIpAddressResponse *ecs20140526.AllocatePublicIpAddressResponse

	for {
		if m.stopSign {
			break
		}

		ctx, cancel = context.WithTimeout(m.ctx, AutomaticPublicIPAllocatorCycleTimeout)

		err = globals.Pool.QueryRowContext(ctx, "SELECT instance_id FROM instances WHERE deleted_at IS NULL AND ip IS NULL").Scan(&activeInstanceId)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				if config.Cfg.Monitor.AutomaticPublicIpAllocator.Verbose {
					m.logger.Println("No active instance without ip addr is found, skipping")
				}
				goto end
			}

			m.logger.Printf("Cannot get active instance id: %v", err)
			m.errChan <- err
			goto end
		}

		allocatePublicIpAddressRequest = &ecs20140526.AllocatePublicIpAddressRequest{
			InstanceId: &activeInstanceId,
		}

		allocatePublicIpAddressResponse, err = globals.EcsClient.AllocatePublicIpAddress(allocatePublicIpAddressRequest)

		if err != nil {
			m.logger.Printf("Cannot allocate public ip: %v", err)
			m.errChan <- err
			goto end
		}

		_, err = globals.Pool.ExecContext(ctx, "UPDATE instances SET ip = ? WHERE instance_id = ?", *allocatePublicIpAddressResponse.Body.IpAddress, activeInstanceId)

		if err != nil {
			m.logger.Printf("Cannot update public ip: %v", err)
			m.errChan <- err
			goto end
		}

		m.logger.Printf("Successfully allocated public ip address: %v for instance %v", *allocatePublicIpAddressResponse.Body.IpAddress, activeInstanceId)

	end:
		cancel()
		time.Sleep(time.Duration(config.Cfg.Monitor.AutomaticPublicIpAllocator.ExecutionInterval) * time.Second)
	}
}
