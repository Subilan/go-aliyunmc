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
	CommandType   globals.CommandType `form:"commandType" binding:"required"`
	WaitForOutput bool                `form:"waitForOutput"`
}

// HandleServerExecute 尝试在活动实例上运行一个操作，该操作必须在预先固定的有限操作中选取一个。
func HandleServerExecute() gin.HandlerFunc {
	return helpers.QueryHandler[ExecuteQuery](func(body ExecuteQuery, c *gin.Context) (any, error) {
		userId, exists := c.Get("user_id")

		if !exists {
			return nil, &helpers.HttpError{Code: http.StatusUnauthorized, Details: "cannot get user id"}
		}

		activeInstance := store.GetActiveInstance()

		if activeInstance == nil {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "no active instance present"}
		}

		cmd, ok := globals.Commands[body.CommandType]

		if !ok {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "command not found"}
		}

		if cmd.IsCoolingDown() {
			return nil, &helpers.HttpError{Code: http.StatusForbidden, Details: "cooling down"}
		}

		if cmd.Prerequisite != nil {
			if !cmd.Prerequisite() {
				return nil, &helpers.HttpError{Code: http.StatusForbidden, Details: "prerequisite not met"}
			}
		}

		ctx, cancel := context.WithTimeout(c, time.Duration(cmd.Timeout)*time.Second)
		defer cancel()

		row, err := globals.Pool.Exec("INSERT INTO command_exec (`type`, `by`, `status`) VALUES (?, ?, ?)", cmd.Type, userId, "created")

		if err != nil {
			return nil, err
		}

		recordId, _ := row.LastInsertId()

		var output []byte

		if cmd.ExecInShell() {
			output, err = remote.RunCommandAsProdSync(ctx, *activeInstance.Ip, cmd.Content)
		}

		if cmd.ExecInServer() {
			if !globals.IsServerRunning {
				return nil, &helpers.HttpError{Code: http.StatusServiceUnavailable, Details: "server not running"}
			}

			rconClient, rconClientErr := rcon.Dial(*activeInstance.Ip, 25575)

			if rconClientErr != nil {
				return nil, &helpers.HttpError{Code: http.StatusServiceUnavailable, Details: "cannot dial server"}
			}

			if err := rconClient.Login("subilan1999"); err != nil {
				return nil, &helpers.HttpError{Code: http.StatusInternalServerError, Details: "cannot login"}
			}

			var messages strings.Builder

			for _, serverCmd := range cmd.Content {
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

		if err != nil {
			_, _ = globals.Pool.Exec("UPDATE `command_exec` SET `status` = ? WHERE id = ?", "error", recordId)
			return helpers.Data(gin.H{"error": err.Error(), "output": string(output)}), nil
		}

		cmd.StartCooldown()

		_, _ = globals.Pool.Exec("UPDATE `command_exec` SET `status` = ? WHERE id = ?", "success", recordId)
		return helpers.Data(gin.H{"error": nil, "output": string(output)}), nil
	})
}
