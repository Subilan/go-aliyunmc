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

	"github.com/Subilan/go-aliyunmc/config"
	"golang.org/x/crypto/ssh"
)

// RunScriptAsRootAsync 用于在远程服务器上运行指定的脚本，运行过程的信息可通过传入回调函数实现。
// 警告：该函数以 Root 身份运行脚本。禁止用于运行客户端提供的内容。
func RunScriptAsRootAsync(
	ctx context.Context,
	host string,
	templatePath string,
	templateData any,
	sink func([]byte),
	errorHandler func(error),
	okHandler func(),
	finalHandler func(),
) {
	defer finalHandler()
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

		// It's awkward that SIGTERM, SIGINT, SIGKILL do not work as expected. Nothing happens. The process continues
		// and session.Wait just won't return.
		// By calling session.Close here, the process is stopped after a few seconds (detected by ACK) due to pipe close.
		// And according to
		// https://github.com/golang/go/issues/21423#issuecomment-325966525
		// https://github.com/golang/go/issues/21699#issue-254076414
		// session.Close is expected to cause session.Wait to return, which matches the expected path of this function.
		err := session.Close()

		if err != nil {
			log.Println("cannot close session:", err)
		} else {
			log.Println("session closed by context done")
		}
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

		// 如果此处有上下文error，极大可能是因为上下文导致的退出。优先传递此类错误
		if ctx.Err() != nil {
			errorHandler(ctx.Err())
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

// RunCommandAsProdSync 在指定上下文 ctx 下，在远程服务器 host 上运行 commands 指定的指令，并在得到结果或错误时返回。
//
// 该函数以生产身份运行指令，请注意权限设置正确。
func RunCommandAsProdSync(
	ctx context.Context,
	host string,
	commands []string,
	output bool,
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
	if output {
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
	} else {
		go func() {
			_, err := io.Copy(io.Discard, stdout)
			if err != nil {
				log.Println("cannot copy stdout to outBuf:", err)
			}
		}()
		go func() {
			_, err := io.Copy(&outBuf, stderr)
			if err != nil {
				log.Println("cannot copy stderr to outBuf:", err)
			}
		}()
	}

	// Context cancellation
	go func() {
		<-ctx.Done()

		err := session.Close()

		if err != nil {
			log.Println("cannot close session:", err)
		} else {
			log.Println("session closed by context done")
		}
	}()

	if err := session.Start("bash -s"); err != nil {
		return nil, err
	}

	if _, err := stdin.Write([]byte(script)); err != nil {
		return nil, err
	}

	stdin.Close()

	if err := session.Wait(); err != nil {
		if ctx.Err() != nil {
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
