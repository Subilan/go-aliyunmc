package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/remote"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
	"github.com/mcstatus-io/mcutil/v4/rcon"
)

type ExecuteQuery struct {
	CommandType   CommandType `form:"commandType" binding:"required,oneof=start_server stop_server"`
	WaitForOutput bool        `form:"waitForOutput"`
}

type CommandExecuteLocation string

const (
	ExecuteLocationServer CommandExecuteLocation = "server"
	ExecuteLocationShell                         = "shell"
)

type CommandType string

var commandTypeStartServerCooldownLeft = 0

func (c CommandType) Prerequisite() error {
	switch c {
	case CmdTypeStartServer:
		if globals.IsServerRunning {
			return &helpers.HttpError{Code: http.StatusConflict, Details: "server is already running"}
		}
		if commandTypeStartServerCooldownLeft > 0 {
			return &helpers.HttpError{Code: http.StatusForbidden, Details: "this command is still in cooldown"}
		}
	case CmdTypeStopServer:
		if !globals.IsServerRunning {
			return &helpers.HttpError{Code: http.StatusServiceUnavailable, Details: "server is not running"}
		}
	}

	return nil
}

func (c CommandType) Command() (CommandExecuteLocation, []string) {
	switch c {
	case CmdTypeStartServer:
		// TODO: 必须先cd到目录下，再执行start.sh，因为start.sh里面写的很可能是相对路径
		return ExecuteLocationShell, []string{"cd /home/mc/server/archive && ./start.sh && sleep 0.5 && screen -S server -Q select . >/dev/null || echo 'server cannot be started'"}
	case CmdTypeStopServer:
		return ExecuteLocationServer, []string{"stop"}
	}

	return "", nil
}

func (c CommandType) SetupCooldown() {
	switch c {
	case CmdTypeStartServer:
		commandTypeStartServerCooldownLeft = 60
		go func() {
			for commandTypeStartServerCooldownLeft > 0 {
				commandTypeStartServerCooldownLeft -= 1
				time.Sleep(time.Second)
			}
		}()
	}
}

const (
	CmdTypeStartServer CommandType = "start_server"
	CmdTypeStopServer  CommandType = "stop_server"
)

// HandleServerExecute 尝试在活动实例上运行一个操作，该操作必须在预先固定的有限操作中选取一个。无论运行成功还是失败，该接口总是返回输出的内容。
func HandleServerExecute() gin.HandlerFunc {
	return helpers.QueryHandler[ExecuteQuery](func(body ExecuteQuery, c *gin.Context) (any, error) {
		activeInstance := store.GetActiveInstance()

		if activeInstance == nil {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "no active instance present"}
		}

		if err := body.CommandType.Prerequisite(); err != nil {
			return nil, err
		}

		ctx, cancel := context.WithTimeout(c, 10*time.Second)
		defer cancel()

		loc, cmd := body.CommandType.Command()

		var output []byte
		var err error

		if loc == ExecuteLocationShell {
			output, err = remote.RunCommandAsProdSync(ctx, activeInstance.Ip, cmd)
		}

		if loc == ExecuteLocationServer {
			if !globals.IsServerRunning {
				return nil, &helpers.HttpError{Code: http.StatusServiceUnavailable, Details: "server not running"}
			}

			rconClient, rconClientErr := rcon.Dial(activeInstance.Ip, 25575)

			if rconClientErr != nil {
				return nil, &helpers.HttpError{Code: http.StatusServiceUnavailable, Details: "cannot dial server"}
			}

			if err := rconClient.Login("subilan1999"); err != nil {
				return nil, &helpers.HttpError{Code: http.StatusInternalServerError, Details: "cannot login"}
			}

			var messages strings.Builder

			for _, serverCmd := range cmd {
				if err := rconClient.Run(serverCmd); err != nil {
					return nil, &helpers.HttpError{Code: http.StatusInternalServerError, Details: "cannot execute command"}
				}

				if body.WaitForOutput {
					messages.WriteString(<-rconClient.Messages)
				}
			}

			if body.WaitForOutput {
				output = []byte(messages.String())
			} else {
				output = []byte{}
			}
		}

		body.CommandType.SetupCooldown()

		if err != nil {
			return helpers.Data(gin.H{"error": err.Error(), "output": string(output)}), nil
		}

		return helpers.Data(gin.H{"error": nil, "output": string(output)}), nil
	})
}
