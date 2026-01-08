package monitors

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/Subilan/go-aliyunmc/helpers/stream"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
)

var instanceIpRestored bool

var instanceIp string
var instanceIpMu sync.RWMutex

var instanceIpUpdate = make(chan string)

func RestoreInstanceIp(ip string) {
	if instanceIpRestored {
		log.Fatalln("double restoring instance ip is not permitted")
	}

	instanceIpMu.Lock()
	instanceIp = ip
	instanceIpMu.Unlock()
	instanceIpRestored = true
}

func SnapshotInstanceIp() string {
	instanceIpMu.RLock()
	defer instanceIpMu.RUnlock()

	return instanceIp
}

func syncIpWithUser(logger *log.Logger) {
	for ip := range instanceIpUpdate {
		event, err := store.BuildInstanceEvent(store.InstanceEventActiveIpUpdate, ip)

		if err != nil {
			logger.Println("cannot build event:", err)
		}

		err = stream.BroadcastAndSave(event)

		if err != nil {
			logger.Println("cannot broadcast and save event:", err)
		}
	}
}

func PublicIP(quit chan bool) {
	logger := log.New(os.Stdout, "[PublicIP] ", log.LstdFlags)
	logger.Println("starting...")

	ticker := time.NewTicker(time.Duration(config.Cfg.Monitor.AutomaticPublicIpAllocator.ExecutionInterval) * time.Second)

	go syncIpWithUser(logger)

	for {
		select {
		case <-ticker.C:
			func() {
				var activeInstanceId string

				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()

				err := db.Pool.QueryRowContext(ctx, "SELECT instance_id FROM instances WHERE deleted_at IS NULL AND ip IS NULL").Scan(&activeInstanceId)

				if err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						if config.Cfg.Monitor.AutomaticPublicIpAllocator.Verbose {
							logger.Println("No active instance without ip addr is found, skipping")
						}
						return
					}

					logger.Printf("Cannot get active instance id: %v", err)
					return
				}

				allocatePublicIpAddressRequest := &ecs20140526.AllocatePublicIpAddressRequest{
					InstanceId: &activeInstanceId,
				}

				allocatePublicIpAddressResponse, err := globals.EcsClient.AllocatePublicIpAddress(allocatePublicIpAddressRequest)

				if err != nil {
					logger.Printf("Cannot allocate public ip: %v", err)
					return
				}

				ip := *allocatePublicIpAddressResponse.Body.IpAddress

				instanceIpMu.Lock()
				instanceIp = ip
				instanceIpMu.Unlock()

				instanceIpUpdate <- ip

				_, err = db.Pool.ExecContext(ctx, "UPDATE instances SET ip = ? WHERE instance_id = ?", ip, activeInstanceId)

				if err != nil {
					logger.Printf("Cannot update public ip: %v", err)
					return
				}

				logger.Printf("Successfully allocated public ip address: %v for instance %v", ip, activeInstanceId)
			}()

		case <-quit:
			return
		}
	}
}
