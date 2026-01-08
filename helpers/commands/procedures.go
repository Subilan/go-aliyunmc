package commands

import (
	"context"

	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/globals"
)

func StopAndArchiveServer(ctx context.Context, host string, by *int64) error {
	var marker = "stop_and_archive_server"

	stopServerCmd := MustGetCommand(consts.CmdTypeStopServer)
	archiveServerCmd := MustGetCommand(consts.CmdTypeArchiveServer)

	if globals.IsServerRunning {
		_, err := stopServerCmd.RunWithoutCooldown(ctx, host, by, &CommandRunOption{Output: true, Comment: marker})

		if err != nil {
			return err
		}
	}

	_, err := archiveServerCmd.RunWithoutCooldown(ctx, host, by, &CommandRunOption{Output: true, Comment: marker})

	if err != nil {
		return err
	}

	return nil
}
