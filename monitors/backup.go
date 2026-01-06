package monitors

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers/store"
)

var backupInterval = 10 * time.Minute
var retryInterval = 1 * time.Minute
var backupTimeout = 2 * time.Minute

func Backup() {
	var ctx context.Context
	var cancel context.CancelFunc
	var activeInstance *store.Instance
	var err error
	var success bool

	logger := log.New(os.Stdout, "[BackupMonitor] ", log.LstdFlags)

	cmd := globals.MustGetCommand(globals.CmdTypeBackupWorlds)

	logger.Println("starting...")

	for {
		success = false

		logger.Println("running new backup task")

		ctx, cancel = context.WithTimeout(context.Background(), backupTimeout)

		activeInstance = store.GetActiveInstance()

		if activeInstance == nil {
			goto end
		}

		if activeInstance.Ip == nil {
			logger.Println("active instance does not have an ip allocated, skipping")
			goto end
		}

		_, err = cmd.Run(ctx, *activeInstance.Ip, nil, &globals.CommandRunOption{IgnoreCooldown: true})

		success = err != nil

	end:
		cancel()

		if success {
			logger.Println("next backup in", backupInterval.String())
			time.Sleep(backupInterval)
		} else {
			logger.Println("retry in", retryInterval.String())
			time.Sleep(retryInterval)
		}
	}
}
