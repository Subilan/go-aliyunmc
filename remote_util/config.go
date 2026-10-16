package remote_util

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/log_util"

	"golang.org/x/crypto/ssh"
)

// sshKey 是从配置中加载的私钥，用于以 mc 用户建立 SSH 连接。root 用户不使用密钥。
var sshKey ssh.Signer

func MustInitialize() {
	signer, err := loadPrivateKey(config.Get().SSH.PrivateKeyPath)
	if err != nil {
		log_util.Fatal("加载SSH私钥失败: %v", err)
	}
	sshKey = signer
}

// AuthorizedKeyLine 返回已加载私钥对应的 authorized_keys 行。
func AuthorizedKeyLine() (string, error) {
	if sshKey == nil {
		return "", fmt.Errorf("尚未加载 mc 用户的SSH私钥")
	}

	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshKey.PublicKey()))), nil
}

func loadPrivateKey(path string) (ssh.Signer, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取私钥文件失败(%s): %w", path, err)
	}

	signer, err := ssh.ParsePrivateKey(content)
	if err != nil {
		var passphraseMissing *ssh.PassphraseMissingError
		if errors.As(err, &passphraseMissing) {
			return nil, fmt.Errorf("私钥 %s 带有口令，而服务运行在无 agent 的环境中，请改用无口令密钥", path)
		}
		return nil, fmt.Errorf("解析私钥失败(%s): %w", path, err)
	}

	return signer, nil
}
