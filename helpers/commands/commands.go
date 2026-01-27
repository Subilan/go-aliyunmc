package commands

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/gctx"
	"github.com/Subilan/go-aliyunmc/helpers/remote"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/Subilan/go-aliyunmc/helpers/templateData"
	"github.com/gin-gonic/gin"
	"github.com/mcstatus-io/mcutil/v4/rcon"
	"github.com/mcstatus-io/mcutil/v4/status"
)

// Commands 是全局指令字典，包含了系统运行时可能用到的所有指令。
var Commands = make(map[consts.CommandType]*Command)

// Command 是对系统代用户在远程机上指定位置执行的预先编写的指令的结构化表示。
type Command struct {
	// Type 是该指令的类型，也可以认为是该指令的标识符。
	Type consts.CommandType

	// ExecuteLocation 表示该指令执行的位置。
	ExecuteLocation consts.CommandExecuteLocation

	// Cooldown 是该指令的冷却时间，单位秒
	Cooldown int

	// Content 是该指令的具体文本内容
	Content []string

	// Timeout 是该指令的推荐超时时间。具体的超时由指令运行的上下文决定，而不是由此字段。
	Timeout int

	// Prerequisite 返回该指令运行的前置条件是否满足。如果该函数返回 false，则指令不可运行。
	Prerequisite func() bool

	// IsQuery 表示该指令是否属于查询类指令。查询类指令永远没有冷却时间，运行时也不会在数据库中记录其过程。
	IsQuery bool

	// Role 表示该指令要求的权限等级
	Role consts.UserRole

	// Whitelisted 表示该指令是否要求用户绑定游戏账号且有白名单
	// 通常，当 Role 设置为高权限等级，如 admin 的时候，不需要设置此项
	Whitelisted bool
}

// DefaultContext 获取该指令用于运行的默认上下文，它是 context.Background 的子上下文，附带了 Command.Timeout 对应的超时时间。
func (c *Command) DefaultContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.TimeoutDuration())
}

// TimeoutDuration 返回 Command.Timeout 的 time.Duration 形式（以秒计）
func (c *Command) TimeoutDuration() time.Duration {
	return time.Duration(c.Timeout) * time.Second
}

// cooldownContexts 用于记录正在运行的冷却计时 goroutine 对应上下文的取消函数，以便重置冷却计时。在实际运行中，该字典中可能包含已经被取消的上下文的取消函数。
var cooldownContexts = make(map[consts.CommandType]context.CancelFunc)

// StartCooldown 开始当前指令的冷却计时。开始计时之前，将尝试取消该指令已经存在的未完成冷却计时（重置冷却计时）。如果该指令没有冷却时间（为0），该方法无效果。
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

// IsCoolingDown 返回该指令是否处于冷却时间内
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

// TestRole 判断 *gin.Context 中携带的权限等级信息是否大于或等于所要求的权限等级
func (c *Command) TestRole(ctx *gin.Context) bool {
	role, exists := ctx.Get("role")

	if !exists {
		return false
	}

	roleInt, ok := role.(consts.UserRole)

	if !ok {
		return false
	}

	return roleInt >= c.Role
}

func (c *Command) TestWhitelisted(ctx *gin.Context) bool {
	if !c.Whitelisted {
		return true
	}

	userId, err := gctx.ShouldGetUserId(ctx)

	if err != nil {
		return false
	}

	gameBound, exists := store.GetGameBound(userId)

	if !exists {
		return false
	}

	return gameBound.Whitelisted
}

// Run 运行该指令。
// option 可以填 nil 表示使用默认值。
// 传入的上下文只会影响该指令的执行过程，不会影响数据库的记录过程。
// 如果 by 参数填 nil，表示该运行是自动发起。
// 注意：如果运行的指令为查询类，则行为有所差异，详见 Command.IsQuery。
func (c *Command) Run(ctx context.Context, host string, by *int64, option *CommandRunOption) (string, error) {
	if option == nil {
		option = &CommandRunOption{}
	}

	if c.IsCoolingDown() && !option.IgnoreCooldown {
		return "", &helpers.HttpError{Code: http.StatusTooManyRequests, Details: "该指令仍在冷却中"}
	}

	if c.Prerequisite != nil && !c.Prerequisite() {
		return "", &helpers.HttpError{Code: http.StatusServiceUnavailable, Details: "该指令前置条件未满足"}
	}

	doRecord := !option.DisableAudit && !c.IsQuery
	doOutput := option.Output || c.IsQuery

	var recordId int64

	if doRecord {
		row, err := db.Pool.Exec("INSERT INTO command_exec (`type`, `by`, `status`, `auto`) VALUES (?, ?, ?, ?)", c.Type, by, "created", by == nil)

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

	if c.ExecuteLocation == consts.ExecuteLocationShell {
		output, err = remote.RunCommandAsProdSync(ctx, host, c.Content, doOutput)
	}

	if c.ExecuteLocation == consts.ExecuteLocationServer {
		rconClient, rconClientErr := rcon.Dial(host, config.Cfg.GetGameRconPort())
		if rconClientErr != nil {
			return "", rconClientErr
		}
		defer rconClient.Close()

		if err := rconClient.Login(config.Cfg.Server.RconPassword); err != nil {
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
			_, _ = db.Pool.Exec("UPDATE `command_exec` SET `status` = ?, `comment` = ? WHERE id = ?", "error", err.Error(), recordId)
			return outputStr, err
		}

		_, _ = db.Pool.Exec("UPDATE `command_exec` SET `status` = ?, `comment` = ? WHERE id = ?", "success", option.Comment, recordId)
		return outputStr, nil
	}

	return outputStr, err
}

// RunWithoutCooldown 以无冷却时间相关考虑运行该指令。这样，指令的执行不会考虑冷却时间，亦不会重置冷却时间。仍然可以传入其它选项。
func (c *Command) RunWithoutCooldown(ctx context.Context, host string, by *int64, option *CommandRunOption) (string, error) {
	if option == nil {
		option = &CommandRunOption{
			IgnoreCooldown:       true,
			DisableResetCooldown: true,
		}
	} else {
		option.IgnoreCooldown = true
		option.DisableResetCooldown = true
	}

	return c.Run(ctx, host, by, option)
}

var commandCooldownLeft = make(map[consts.CommandType]int)

// Load 加载系统的所有指令并写入到 Commands 字典中，用于调用。应当在系统初始化时调用，且在所有依赖指令的操作开始之前调用。
func Load() {
	Commands[consts.CmdTypeStartServer] = &Command{
		Type:            consts.CmdTypeStartServer,
		ExecuteLocation: consts.ExecuteLocationShell,
		Cooldown:        60,
		Content:         []string{"cd /home/mc/server/archive && ./start.sh && sleep 0.5 && screen -S server -Q select . >/dev/null || echo 'server cannot be started'"},
		Timeout:         5,
		Prerequisite: func() bool {
			activeInstance, err := store.GetDeployedActiveInstance()

			if err != nil {
				return false
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err = status.Modern(ctx, *activeInstance.Ip, config.Cfg.GetGamePort())

			// 需要服务器不在线
			return err != nil
		},
		Whitelisted: true,
	}

	Commands[consts.CmdTypeStopServer] = &Command{
		Type:            consts.CmdTypeStopServer,
		ExecuteLocation: consts.ExecuteLocationServer,
		Cooldown:        60,
		Content:         []string{"stop"},
		Timeout:         5,
		Prerequisite: func() bool {
			activeInstance, err := store.GetDeployedActiveInstance()

			if err != nil {
				return false
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err = status.Modern(ctx, *activeInstance.Ip, config.Cfg.GetGamePort())

			// 需要服务器在线
			return err == nil
		},
		Role: consts.UserRoleAdmin,
	}
	Commands[consts.CmdTypeGetServerSizes] = &Command{
		Type:            consts.CmdTypeGetServerSizes,
		ExecuteLocation: consts.ExecuteLocationShell,
		Cooldown:        0,
		Content:         []string{"du -sh /home/mc/server/archive", "du -sh /home/mc/server/archive/world", "du -sh /home/mc/server/archive/world_nether", "du -sh /home/mc/server/archive/world_the_end", "du -sh /home/mc/server/archive/bluemap"},
		Timeout:         5,
		IsQuery:         true,
		Prerequisite: func() bool {
			_, err := store.GetDeployedActiveInstance()
			return err == nil
		},
	}
	Commands[consts.CmdTypeScreenfetch] = &Command{
		Type:            consts.CmdTypeScreenfetch,
		ExecuteLocation: consts.ExecuteLocationShell,
		Cooldown:        0,
		Content:         []string{"screenfetch -N"},
		Timeout:         5,
		IsQuery:         true,
	}
	Commands[consts.CmdTypeGetCachedPlayers] = &Command{
		Type:            consts.CmdTypeGetCachedPlayers,
		ExecuteLocation: consts.ExecuteLocationShell,
		Cooldown:        0,
		Content:         []string{"cat /home/mc/server/archive/usercache.json"},
		Timeout:         5,
		IsQuery:         true,
		Prerequisite: func() bool {
			_, err := store.GetDeployedActiveInstance()
			return err == nil
		},
	}
	Commands[consts.CmdTypeGetServerProperties] = &Command{
		Type:            consts.CmdTypeGetServerProperties,
		ExecuteLocation: consts.ExecuteLocationShell,
		Cooldown:        0,
		Content:         []string{"cat /home/mc/server/archive/server.properties | grep -E '^(white-list|view-distance|simulation-distance|spawn-protection|online-mode|difficulty|max-players).*='"},
		Timeout:         5,
		IsQuery:         true,
		Prerequisite: func() bool {
			_, err := store.GetDeployedActiveInstance()
			return err == nil
		},
	}

	Commands[consts.CmdTypeGetWhitelist] = &Command{
		Type:            consts.CmdTypeGetWhitelist,
		ExecuteLocation: consts.ExecuteLocationShell,
		Cooldown:        0,
		Content:         []string{"cat /home/mc/server/archive/whitelist.json"},
		Timeout:         5,
		IsQuery:         true,
		Prerequisite: func() bool {
			_, err := store.GetDeployedActiveInstance()
			return err == nil
		},
		Role: consts.UserRoleAdmin,
	}

	Commands[consts.CmdTypeGetOps] = &Command{
		Type:            consts.CmdTypeGetOps,
		ExecuteLocation: consts.ExecuteLocationShell,
		Cooldown:        0,
		Content:         []string{"cat /home/mc/server/archive/ops.json"},
		Timeout:         5,
		IsQuery:         true,
		Prerequisite: func() bool {
			_, err := store.GetDeployedActiveInstance()
			return err == nil
		},
	}

	for _, typ := range []consts.CommandType{consts.CmdTypeBackupWorlds, consts.CmdTypeArchiveServer} {
		var filename string
		if typ == consts.CmdTypeBackupWorlds {
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
			ExecuteLocation: consts.ExecuteLocationShell,
			Cooldown:        30,
			Content:         []string{buf.String()},
			Timeout:         300,
			Role:            consts.UserRoleAdmin,
		}
	}

	log.Printf("Loaded %d commands\n", len(Commands))
}

// MustGetCommand 一定获取到 commandType 对应的指令，否则将会导致程序退出。
func MustGetCommand(commandType consts.CommandType) *Command {
	command, ok := Commands[commandType]

	if !ok {
		log.Fatalf("Unknown command type: %s", commandType)
	}

	return command
}

// ShouldGetCommand 尝试获取 commandType 对应的指令。
func ShouldGetCommand(commandType consts.CommandType) (*Command, bool) {
	command, ok := Commands[commandType]
	return command, ok
}
