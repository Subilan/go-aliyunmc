package globals

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"text/template"
	"time"

	"github.com/Subilan/go-aliyunmc/helpers/remote"
	"github.com/Subilan/go-aliyunmc/helpers/templateData"
	"github.com/mcstatus-io/mcutil/v4/rcon"
)

var Commands = make(map[CommandType]*Command)

type CommandType string

const (
	CmdTypeStartServer    CommandType = "start_server"
	CmdTypeStopServer     CommandType = "stop_server"
	CmdTypeBackupWorlds   CommandType = "backup_worlds"
	CmdTypeArchiveServer  CommandType = "archive_server"
	CmdTypeScreenfetch    CommandType = "screenfetch"
	CmdTypeGetServerSizes CommandType = "get_server_sizes"
)

type CommandExecuteLocation string

const (
	ExecuteLocationServer CommandExecuteLocation = "server"
	ExecuteLocationShell                         = "shell"
)

type Command struct {
	Type            CommandType
	ExecuteLocation CommandExecuteLocation
	Cooldown        int
	Content         []string
	Timeout         int
	Prerequisite    func() bool
	IsQuery         bool
}

func (c *Command) TimeoutContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.TimeoutDuration())
}

func (c *Command) TimeoutDuration() time.Duration {
	return time.Duration(c.Timeout) * time.Second
}

var cooldownContexts = make(map[CommandType]context.CancelFunc)

// StartCooldown 开始当前指令的冷却计时。
// 冷却计时在一个goroutine中进行。
// 开始计时之前，将尝试取消该指令已经存在的未完成冷却计时。
// 如果该指令没有冷却时间（为0），该方法无效果。
func (c *Command) StartCooldown() {
	if c.Cooldown == 0 {
		return
	}

	previousCancel, ok := cooldownContexts[c.Type]

	if ok {
		previousCancel()
	}

	commandCooldownLeft[c.Type] = c.Cooldown
	ctx, cancel := context.WithCancel(context.Background())
	cooldownContexts[c.Type] = cancel

	go func() {
		for commandCooldownLeft[c.Type] > 0 {
			select {
			case <-ctx.Done():
				return
			default:
				commandCooldownLeft[c.Type] -= 1
				time.Sleep(time.Second)
			}
		}
	}()
}

func (c *Command) IsCoolingDown() bool {
	cd, ok := commandCooldownLeft[c.Type]

	if !ok {
		return false
	}

	return cd > 0
}

// CommandRunOption 表示对指令运行的配置
type CommandRunOption struct {
	// IgnoreCooldown 如果为true，本次执行无需满足冷却时间条件。
	IgnoreCooldown bool

	// DisableResetCooldown 如果为true，本次执行不会重置冷却时间
	DisableResetCooldown bool

	// DisableAudit 如果为true，表示不在数据库中记录执行信息。
	// 对于Query类指令，该选项固定为true，设置无效。
	DisableAudit bool

	// Output 如果为true，表示返回输出的内容（stdout）。
	// 对于Query类指令，该选项固定为true，设置无效。
	Output bool

	// Comment 是本次执行成功后在数据库中填入的备注字段
	Comment string
}

// Run 运行该指令。
// option可以填nil表示使用默认值。
// 传入的上下文只会影响该指令的执行过程，不会影响数据库的记录过程。
// 如果by参数填nil，表示该运行是自动发起。
// 如果运行的指令为查询类，则始终不会记录过程，始终会返回输出内容。
func (c *Command) Run(ctx context.Context, host string, by *int, option *CommandRunOption) (string, error) {
	if option == nil {
		option = &CommandRunOption{}
	}

	if c.IsCoolingDown() && !option.IgnoreCooldown {
		return "", errors.New("cooling down")
	}

	if c.Prerequisite != nil && !c.Prerequisite() {
		return "", errors.New("prerequisite not met")
	}

	doRecord := !option.DisableAudit && !c.IsQuery
	doOutput := option.Output || c.IsQuery

	var recordId int64

	if doRecord {
		row, err := Pool.Exec("INSERT INTO command_exec (`type`, `by`, `status`, `auto`) VALUES (?, ?, ?, ?)", c.Type, by, "created", by == nil)

		if err != nil {
			return "", err
		}

		recordId, err = row.LastInsertId()

		if err != nil {
			return "", err
		}
	}

	var output []byte
	var err error

	if c.ExecuteLocation == ExecuteLocationShell {
		output, err = remote.RunCommandAsProdSync(ctx, host, c.Content, doOutput)
	}

	if c.ExecuteLocation == ExecuteLocationServer {
		rconClient, rconClientErr := rcon.Dial(host, 25575)

		if rconClientErr != nil {
			return "", rconClientErr
		}

		if err := rconClient.Login("subilan1999"); err != nil {
			return "", err
		}

		var messages strings.Builder

		for _, serverCmd := range c.Content {
			if err := rconClient.Run(serverCmd); err != nil {
				return "", err
			}

			if doOutput {
				messages.WriteString(<-rconClient.Messages)
			}
		}

		if doOutput {
			output = []byte(messages.String())
		} else {
			output = []byte{}
		}
	}

	outputStr := string(output)

	if err == nil && !option.DisableResetCooldown {
		c.StartCooldown()
	}

	if doRecord {
		if err != nil {
			_, _ = Pool.Exec("UPDATE `command_exec` SET `status` = ?, `comment` = ? WHERE id = ?", "error", err.Error(), recordId)
			return outputStr, err
		} else {
			_, _ = Pool.Exec("UPDATE `command_exec` SET `status` = ?, `comment` = ? WHERE id = ?", "success", option.Comment, recordId)
			return outputStr, nil
		}
	} else {
		return outputStr, err
	}
}

var commandCooldownLeft = make(map[CommandType]int)

func LoadCommands() {
	Commands[CmdTypeStartServer] = &Command{
		Type:            CmdTypeStartServer,
		ExecuteLocation: ExecuteLocationShell,
		Cooldown:        60,
		Content:         []string{"cd /home/mc/server/archive && ./start.sh && sleep 0.5 && screen -S server -Q select . >/dev/null || echo 'server cannot be started'"},
		Timeout:         5,
		Prerequisite: func() bool {
			return !IsServerRunning
		},
	}
	Commands[CmdTypeStopServer] = &Command{
		Type:            CmdTypeStopServer,
		ExecuteLocation: ExecuteLocationServer,
		Cooldown:        60,
		Content:         []string{"stop"},
		Timeout:         5,
		Prerequisite: func() bool {
			return IsServerRunning
		},
	}
	Commands[CmdTypeGetServerSizes] = &Command{
		Type:            CmdTypeGetServerSizes,
		ExecuteLocation: ExecuteLocationShell,
		Cooldown:        0,
		Content:         []string{"du -sh /home/mc/server/archive", "du -sh /home/mc/server/archive/world", "du -sh /home/mc/server/archive/world_nether", "du -sh /home/mc/server/archive/world_the_end"},
		Timeout:         5,
		IsQuery:         true,
	}
	Commands[CmdTypeScreenfetch] = &Command{
		Type:            CmdTypeScreenfetch,
		ExecuteLocation: ExecuteLocationShell,
		Cooldown:        0,
		Content:         []string{"screenfetch"},
		Timeout:         5,
		IsQuery:         true,
	}

	for _, typ := range []CommandType{CmdTypeBackupWorlds, CmdTypeArchiveServer} {
		var filename string
		if typ == CmdTypeBackupWorlds {
			filename = "backup.tmpl.sh"
		} else {
			filename = "archive.tmpl.sh"
		}

		parsed, err := template.ParseFiles(filename)

		if err != nil {
			log.Fatalf("Error parsing template '%s': %s", filename, err)
		}

		var buf bytes.Buffer
		err = parsed.Execute(&buf, templateData.Archive())

		if err != nil {
			log.Fatalf("Error parsing template '%s': %s", filename, err)
		}

		Commands[typ] = &Command{
			Type:            typ,
			ExecuteLocation: ExecuteLocationShell,
			Cooldown:        30,
			Content:         []string{buf.String()},
			Timeout:         60,
		}
	}

	log.Printf("Loaded %d commands\n", len(Commands))
}

func MustGetCommand(commandType CommandType) *Command {
	command, ok := Commands[commandType]

	if !ok {
		log.Fatalf("Unknown command type: %s", commandType)
	}

	return command
}

func ShouldGetCommand(commandType CommandType) (*Command, bool) {
	command, ok := Commands[commandType]
	return command, ok
}
