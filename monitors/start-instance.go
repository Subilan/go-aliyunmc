package monitors

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"time"

	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
)

func StartInstanceWhenReady() {
	var err error

	logger := log.New(os.Stdout, "[StartInstanceWhenReadyMonitor] ", log.LstdFlags)

	logger.Println("Starting")

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		var instanceId string
		err = db.Pool.QueryRowContext(ctx, "SELECT i.instance_id FROM instances i JOIN instance_statuses s ON i.instance_id = s.instance_id WHERE i.deleted_at IS NULL AND s.status = 'Stopped'").Scan(&instanceId)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.Println("instance not ready, skipping")
				goto end
			}

			logger.Println("cannot get active instance in StartInstanceWhenReady monitor")
			goto end
		}

		_, err = globals.EcsClient.StartInstance(&client.StartInstanceRequest{InstanceId: tea.String(instanceId)})

		cancel()

		if err != nil {
			logger.Println("cannot start instance in StartInstanceWhenReady monitor")
			break
		}

		logger.Println("successfully started instance")

		break

	end:
		cancel()
		time.Sleep(5 * time.Second)
	}
}
