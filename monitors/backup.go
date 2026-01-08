package monitors

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers/commands"
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

	cmd := commands.MustGetCommand(consts.CmdTypeBackupWorlds)

	logger.Println("starting...")

	for {
		success = false

		logger.Println("running new backup task")

		ctx, cancel = context.WithTimeout(context.Background(), backupTimeout)

		activeInstance, err = store.GetIpAllocatedActiveInstance()

		if err != nil {
			goto end
		}

		_, err = cmd.RunWithoutCooldown(ctx, *activeInstance.Ip, nil, nil)

		if err != nil {
			logger.Println("error:", err)
		}

		success = err == nil

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
