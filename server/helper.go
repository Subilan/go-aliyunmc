package server

import (
	"github.com/Subilan/go-aliyunmc/h"
	"net/http"
	"strings"

	"github.com/mcstatus-io/mcutil/v4/rcon"
)

// RunSingleCommand 在指定服务器上运行单条指令，并返回指令的输出。
func RunSingleCommand(ip string, command string) (string, error) {
	return RunCommand(ip, []string{command})
}

// RunCommand 在指定服务器上按顺序运行指令列表中的指令，并合并返回指令的输出。
//  - ip: 服务器IP地址。注意：服务器的端口由配置文件内容指定。
//  - commands: 要执行的指令列表
func RunCommand(ip string, commands []string) (string, error) {
	rconClient, err := rcon.Dial(ip, C.RconPort)

	if err != nil {
		return "", h.HttpError(http.StatusServiceUnavailable, "无法连接到服务器")
	}

	defer rconClient.Close()

	if err := rconClient.Login(C.RconPassword); err != nil {
		return "", h.HttpError(http.StatusServiceUnavailable, "无法登录到服务器")
	}

	var messages strings.Builder

	for _, cmd := range commands {
		if err := rconClient.Run(cmd); err != nil {
			return "", h.HttpError(http.StatusInternalServerError, "执行指令失败")
		}

		messages.WriteString(<-rconClient.Messages)
	}
	
	return messages.String(), nil
}
