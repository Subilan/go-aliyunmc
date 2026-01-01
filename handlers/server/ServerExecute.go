package server

import (
	"context"
	"net/http"
	"time"

	"github.com/Subilan/gomc-server/helpers"
	"github.com/Subilan/gomc-server/helpers/remote"
	"github.com/Subilan/gomc-server/helpers/store"
	"github.com/gin-gonic/gin"
)

type ExecuteQuery struct {
	CommandType CommandType `form:"commandType" binding:"required,oneof=start_server"`
}

type CommandType string

func (c CommandType) Prerequisite() bool {
	switch c {
	case CmdTypeStartServer:
		return true
	}

	return true
}

func (c CommandType) Command() []string {
	switch c {
	case CmdTypeStartServer:
		// TODO: 必须先cd到目录下，再执行start.sh，因为start.sh里面写的很可能是相对路径
		return []string{"cd /home/mc/server/archive && ./start.sh && sleep 0.5 && screen -S server -Q select . >/dev/null || echo 'server cannot be started' && exit 1"}
	}

	return nil
}

const (
	CmdTypeStartServer CommandType = "start_server"
)

// HandleServerExecute 尝试在活动实例上运行一个操作，该操作必须在预先固定的有限操作中选取一个。无论运行成功还是失败，该接口总是返回输出的内容。
func HandleServerExecute() gin.HandlerFunc {
	return helpers.QueryHandler[ExecuteQuery](func(body ExecuteQuery, c *gin.Context) (any, error) {
		activeInstance := store.GetActiveInstance()

		if activeInstance == nil {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "no active instance present"}
		}

		if !body.CommandType.Prerequisite() {
			return nil, &helpers.HttpError{Code: http.StatusForbidden, Details: "command prerequisite not met"}
		}

		ctx, cancel := context.WithTimeout(c, 10*time.Second)
		defer cancel()

		cmd := body.CommandType.Command()

		output, err := remote.RunCommandAsProdSync(ctx, activeInstance.Ip, cmd)

		if err != nil {
			return helpers.Data(gin.H{"error": err.Error(), "output": string(output)}), nil
		}

		return helpers.Data(gin.H{"error": nil, "output": string(output)}), nil
	})
}
