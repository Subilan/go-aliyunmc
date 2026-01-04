package globals

import (
	"bytes"
	"log"
	"text/template"
	"time"

	"github.com/Subilan/go-aliyunmc/helpers/templateData"
)

var Commands = make(map[CommandType]*Command)

type CommandType string

const (
	CmdTypeStartServer   CommandType = "start_server"
	CmdTypeStopServer    CommandType = "stop_server"
	CmdTypeBackupWorlds  CommandType = "backup_worlds"
	CmdTypeArchiveServer CommandType = "archive_server"
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
}

func (c *Command) ExecInShell() bool {
	return c.ExecuteLocation == ExecuteLocationShell
}

func (c *Command) ExecInServer() bool {
	return c.ExecuteLocation == ExecuteLocationServer
}

func (c *Command) StartCooldown() {
	if c.Cooldown == 0 {
		return
	}

	commandCooldownLeft[c.Type] = c.Cooldown

	go func() {
		for commandCooldownLeft[c.Type] > 0 {
			commandCooldownLeft[c.Type] -= 1
			time.Sleep(time.Second)
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
