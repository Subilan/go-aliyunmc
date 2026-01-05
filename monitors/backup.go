package monitors

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers/remote"
	"github.com/Subilan/go-aliyunmc/helpers/store"
)

var backupInterval = 10 * time.Minute
var retryInterval = 1 * time.Minute
var backupTimeout = 2 * time.Minute

func Backup() {
	var ctx context.Context
	var cancel context.CancelFunc
	var activeInstance *store.Instance
	var row sql.Result
	var recordId int64
	var err error
	var success bool

	logger := log.New(os.Stdout, "[BackupMonitor] ", log.LstdFlags)

	cmd, ok := globals.Commands[globals.CmdTypeBackupWorlds]

	if !ok {
		logger.Println("cannot start. no backup command found")
		return
	}

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

		if cmd.Prerequisite != nil && !cmd.Prerequisite() {
			logger.Println("backup command prerequisite not met")
			goto end
		}

		row, err = globals.Pool.ExecContext(ctx, "INSERT INTO `auto_command_exec` (`type`, `status`) VALUES (?, ?)", globals.CmdTypeBackupWorlds, "created")

		if err != nil {
			logger.Println("cannot insert into auto_command_exec table: " + err.Error())
			goto end
		}

		recordId, _ = row.LastInsertId()

		_, err = remote.RunCommandAsProdSync(ctx, *activeInstance.Ip, cmd.Content, false)

		if err != nil {
			logger.Println("backup command failed: ", err)
			_, _ = globals.Pool.ExecContext(ctx, "UPDATE `auto_command_exec` SET `status` = ? WHERE `id` = ?", "error", recordId)
			goto end
		}

		_, _ = globals.Pool.ExecContext(ctx, "UPDATE `auto_command_exec` SET `status` = ? WHERE `id` = ?", "success", recordId)

		logger.Println("successfully backed up")

		success = true

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
