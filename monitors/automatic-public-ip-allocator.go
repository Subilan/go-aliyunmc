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
)

func broadcastIpUpdate(logger *log.Logger, ip string) {
	event, err := store.BuildInstanceEvent(store.InstanceEventActiveIpUpdate, ip)

	if err != nil {
		logger.Println("cannot build event:", err)
	}

	err = stream.BroadcastAndSave(event)

	if err != nil {
		logger.Println("cannot broadcast and save event:", err)
	}
}

func PublicIP() {
	var err error
	var activeInstanceId string
	var cancel context.CancelFunc
	var ctx context.Context
	var allocatePublicIpAddressRequest *ecs20140526.AllocatePublicIpAddressRequest
	var allocatePublicIpAddressResponse *ecs20140526.AllocatePublicIpAddressResponse
	var ip string

	logger := log.New(os.Stdout, "[PublicIP] ", log.LstdFlags)
	logger.Println("starting...")

	for {
		ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)

		err = db.Pool.QueryRowContext(ctx, "SELECT instance_id FROM instances WHERE deleted_at IS NULL AND ip IS NULL").Scan(&activeInstanceId)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				if config.Cfg.Monitor.AutomaticPublicIpAllocator.Verbose {
					logger.Println("No active instance without ip addr is found, skipping")
				}
				goto end
			}

			logger.Printf("Cannot get active instance id: %v", err)
			goto end
		}

		allocatePublicIpAddressRequest = &ecs20140526.AllocatePublicIpAddressRequest{
			InstanceId: &activeInstanceId,
		}

		allocatePublicIpAddressResponse, err = globals.EcsClient.AllocatePublicIpAddress(allocatePublicIpAddressRequest)

		if err != nil {
			logger.Printf("Cannot allocate public ip: %v", err)
			goto end
		}

		ip = *allocatePublicIpAddressResponse.Body.IpAddress

		_, err = db.Pool.ExecContext(ctx, "UPDATE instances SET ip = ? WHERE instance_id = ?", ip, activeInstanceId)

		if err != nil {
			logger.Printf("Cannot update public ip: %v", err)
			goto end
		}

		logger.Printf("Successfully allocated public ip address: %v for instance %v", ip, activeInstanceId)
		broadcastIpUpdate(logger, ip)
	end:
		cancel()
		time.Sleep(time.Duration(config.Cfg.Monitor.AutomaticPublicIpAllocator.ExecutionInterval) * time.Second)
	}
}
