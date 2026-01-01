package remote

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"text/template"
	"time"

	"github.com/Subilan/gomc-server/config"
	"golang.org/x/crypto/ssh"
)

func RunScriptOnHostAsync(
	ctx context.Context,
	host string,
	templatePath string,
	templateData any,
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

	err = scriptsTempl.Execute(&scriptBuf, templateData)

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

	stdout, _ := session.StdoutPipe()
	stderr, _ := session.StderrPipe()
	stdin, _ := session.StdinPipe()

	// 3. Stream output in real time
	go relayWithContext(ctx, stdout, sink)
	go relayWithContext(ctx, stderr, sink)

	// Run killer gorountine
	go func() {
		<-ctx.Done()
		_ = session.Signal(ssh.SIGKILL)
	}()

	// 4. Start shell
	if err := session.Start("bash -s"); err != nil {
		errorHandler(fmt.Errorf("start shell: %w", err))
		return
	}

	log.Println(scriptBuf.String())

	// 5. Send scripts
	if _, err := stdin.Write(scriptBuf.Bytes()); err != nil {
		errorHandler(fmt.Errorf("copy script: %w", err))
		return
	}
	stdin.Close()

	// 6. Wait for completion or be killed due to ctx.Done() received
	if err := session.Wait(); err != nil {
		var exitErr *ssh.ExitError

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			errorHandler(fmt.Errorf("context deadline exceeded: %w", err))
			return
		}

		if errors.As(err, &exitErr) {
			errorHandler(fmt.Errorf("exit code %d: %w", exitErr.ExitStatus(), exitErr))
			return
		}

		errorHandler(fmt.Errorf("ssh command: %w", err))
		return
	}

	okHandler()
}

func relayWithContext(ctx context.Context, r io.Reader, sink func([]byte)) {
	buf := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := r.Read(buf)
			if n > 0 {
				sink(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}
}

func RunCommandsOnHostSync(
	ctx context.Context,
	host string,
	commands []string,
) ([]byte, error) {
	script := strings.Join(commands, "\n") + "\n"

	cfg := &ssh.ClientConfig{
		User: "mc",
		Auth: []ssh.AuthMethod{
			ssh.Password(config.Cfg.Aliyun.Ecs.ProdPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", host+":22", cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	stdout, _ := session.StdoutPipe()
	stderr, _ := session.StderrPipe()
	stdin, _ := session.StdinPipe()

	var outBuf bytes.Buffer
	mw := io.MultiWriter(&outBuf)

	// Read output concurrently
	go func() {
		_, err := io.Copy(mw, stdout)
		if err != nil {
			log.Println("cannot copy stdout to outBuf:", err)
		}
	}()
	go func() {
		_, err := io.Copy(mw, stderr)
		if err != nil {
			log.Println("cannot copy stderr to outBuf:", err)
		}
	}()

	// Context cancellation
	go func() {
		<-ctx.Done()
		_ = session.Signal(ssh.SIGKILL)
	}()

	if err := session.Start("bash -s"); err != nil {
		return nil, err
	}

	if _, err := stdin.Write([]byte(script)); err != nil {
		return nil, err
	}
	_ = stdin.Close()

	if err := session.Wait(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return outBuf.Bytes(), ctx.Err()
		}

		var exitErr *ssh.ExitError
		if errors.As(err, &exitErr) {
			return outBuf.Bytes(), fmt.Errorf("exit code %d", exitErr.ExitStatus())
		}

		return outBuf.Bytes(), err
	}

	return outBuf.Bytes(), nil
}
