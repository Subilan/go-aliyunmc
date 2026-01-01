package helpers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"text/template"
	"time"

	"github.com/Subilan/gomc-server/config"
	"golang.org/x/crypto/ssh"
)

type DeployTemplateData struct {
	Username        string
	Password        string
	SSHPublicKey    string
	Packages        []string
	RegionId        string
	AccessKeyId     string
	AccessKeySecret string
	JavaVersion     uint
	DataDiskSize    int
	ArchiveOSSPath  string
}

func stripAnsiColorCodes(text []byte) []byte {
	re := regexp.MustCompile("\x1b\\[[0-9;]*m")
	return re.ReplaceAll(text, nil)
}

func RunRemoteScripts(
	host string,
	templatePath string,
	sink func([]byte),
	errorHandler func(error),
	okHandler func(),
) {
	// 1. Read scripts in order
	scriptsTempl, err := template.ParseFiles(templatePath)

	if err != nil {
		errorHandler(err)
		return
	}

	var scriptBuf bytes.Buffer

	err = scriptsTempl.Execute(&scriptBuf, DeployTemplateData{
		Username:        "mc",
		Password:        config.Cfg.Aliyun.Ecs.ProdPassword,
		SSHPublicKey:    config.Cfg.Deploy.SSHPublicKey,
		Packages:        config.Cfg.Deploy.Packages,
		RegionId:        config.Cfg.Aliyun.RegionId,
		AccessKeyId:     config.Cfg.Aliyun.AccessKeyId,
		AccessKeySecret: config.Cfg.Aliyun.AccessKeySecret,
		JavaVersion:     config.Cfg.Deploy.JavaVersion,
		DataDiskSize:    config.Cfg.Aliyun.Ecs.DataDisk.Size,
		ArchiveOSSPath:  config.Cfg.Deploy.ArchiveOSSPath,
	})

	if err != nil {
		errorHandler(err)
		return
	}

	// 2. SSH configuration
	cfg := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password(config.Cfg.Aliyun.Ecs.RootPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // replace in prod
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", host+":22", cfg)
	if err != nil {
		errorHandler(fmt.Errorf("ssh dial: %w", err))
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		errorHandler(fmt.Errorf("new session: %w", err))
		return
	}
	defer session.Close()

	// 3. Request PTY for real-time output
	//modes := ssh.TerminalModes{
	//	ssh.ECHO:          0,
	//	ssh.TTY_OP_ISPEED: 14400,
	//	ssh.TTY_OP_OSPEED: 14400,
	//}

	stdout, _ := session.StdoutPipe()
	stderr, _ := session.StderrPipe()
	stdin, _ := session.StdinPipe()

	// 4. Stream output in real time
	go relay(stdout, sink)
	go relay(stderr, sink)

	// 5. Start shell
	if err := session.Start("bash -s"); err != nil {
		errorHandler(fmt.Errorf("start shell: %w", err))
		return
	}

	log.Println(scriptBuf.String())

	// 6. Send scripts
	if _, err := stdin.Write(scriptBuf.Bytes()); err != nil {
		errorHandler(fmt.Errorf("copy script: %w", err))
		return
	}
	stdin.Close()

	// 7. Wait for completion
	if err := session.Wait(); err != nil {
		var exitErr *ssh.ExitError

		if errors.As(err, &exitErr) {
			errorHandler(fmt.Errorf("exit code %d: %w", exitErr.ExitStatus(), exitErr))
			return
		}

		errorHandler(fmt.Errorf("ssh command: %w", err))
		return
	}

	okHandler()
}

func relay(r io.Reader, sink func([]byte)) {
	buf := make([]byte, 4096)

	for {
		n, err := r.Read(buf)
		if n > 0 {
			sink(stripAnsiColorCodes(buf[:n]))
		}
		if err != nil {
			return
		}
	}
}
