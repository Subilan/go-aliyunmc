package monitors

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Subilan/go-aliyunmc/broker"
	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/events"
	"github.com/Subilan/go-aliyunmc/events/stream"
	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
)

var instanceIpRestored bool

var instanceIp string
var instanceIpMu sync.RWMutex

var instanceIpBroker = broker.New[string]()

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
	instanceIpUpdate := instanceIpBroker.Subscribe()
	for ip := range instanceIpUpdate {
		event := events.Instance(events.InstanceEventActiveIpUpdate, ip, true)
		err := stream.BroadcastAndSave(event)

		if err != nil {
			logger.Println("cannot broadcast and save event:", err)
		}
	}
}

func PublicIP(quit chan bool) {
	cfg := config.Cfg.Monitor.PublicIP
	logger := log.New(os.Stdout, "[PublicIP] ", log.LstdFlags)
	logger.Println("starting...")

	ticker := time.NewTicker(cfg.IntervalDuration())

	go instanceIpBroker.Start()
	go syncIpWithUser(logger)

	for {
		select {
		case <-ticker.C:
			func() {
				var activeInstanceId string

				ctx, cancel := context.WithTimeout(context.Background(), cfg.TimeoutDuration())
				defer cancel()

				err := db.Pool.QueryRowContext(ctx, "SELECT instance_id FROM instances WHERE deleted_at IS NULL AND ip IS NULL").Scan(&activeInstanceId)

				if err != nil {
					if errors.Is(err, sql.ErrNoRows) {
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

				instanceIpBroker.Publish(instanceIp)

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
