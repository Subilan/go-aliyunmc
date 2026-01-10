package monitors

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers/commands"
	"github.com/Subilan/go-aliyunmc/helpers/store"
)

func Backup(quit chan bool) {
	cfg := config.Cfg.Monitor.Backup
	var backupInterval = cfg.IntervalDuration()
	var retryInterval = cfg.RetryIntervalDuration()
	var backupTimeout = cfg.TimeoutDuration()
	logger := log.New(os.Stdout, "[Backup] ", log.LstdFlags)

	cmd := commands.MustGetCommand(consts.CmdTypeBackupWorlds)

	logger.Println("starting...")

	ticker := time.NewTicker(backupInterval)

	for {
		select {
		case <-ticker.C:
			func() {
				logger.Println("running new backup task")

				ctx, cancel := context.WithTimeout(context.Background(), backupTimeout)
				defer cancel()

				activeInstance, err := store.GetDeployedActiveInstance()

				if err != nil {
					logger.Println("no instance found, retry in", retryInterval)
					ticker.Reset(retryInterval)
					return
				}

				_, err = cmd.RunWithoutCooldown(ctx, *activeInstance.Ip, nil, nil)

				if err != nil {
					logger.Println("error:", err, "retry in", retryInterval)
					ticker.Reset(retryInterval)
					return
				}

				logger.Println("ok")
				logger.Println("next backup in", backupInterval.String())
				ticker.Reset(backupInterval)
			}()

		case <-quit:
			return
		}
	}
}
