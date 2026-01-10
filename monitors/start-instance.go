package monitors

import (
	"log"
	"os"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
)

func StartActiveInstanceWhenReady() {
	var err error

	logger := log.New(os.Stdout, "[StartActiveInstanceWhenReady] ", log.LstdFlags)
	cfg := config.Cfg.Monitor.StartInstance

	var instanceId string

	err = db.Pool.QueryRow("SELECT instance_id FROM instances WHERE deleted_at IS NULL").Scan(&instanceId)

	if err != nil {
		logger.Println("Error getting instance id:", err)
		return
	}

	timer := time.NewTimer(cfg.TimeoutDuration()) // timeout
	ticker := time.NewTicker(cfg.IntervalDuration())

	for {
		select {
		case <-ticker.C:
			if SnapshotInstanceStatus() != consts.InstanceStopped || SnapshotInstanceIp() == "" {
				logger.Println("instance not ready, retry in", cfg.IntervalDuration())
				continue
			}

			_, err = globals.EcsClient.StartInstance(&client.StartInstanceRequest{InstanceId: tea.String(instanceId)})

			if err != nil {
				logger.Println("cannot start instance in StartActiveInstanceWhenReady monitor")
			}

			logger.Println("successfully triggered instance start")
			return

		case <-timer.C:
			logger.Println("Timed out waiting for instance to be ready to be started")
			return
		}
	}
}
