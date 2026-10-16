package remote_util

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/Subilan/go-aliyunmc/log_util"
	"github.com/Subilan/go-aliyunmc/utils"

	"golang.org/x/crypto/ssh"
)

var C Config

// sshKey 是从配置中加载的私钥，用于以 mc 用户建立 SSH 连接。root 用户不使用密钥。
var sshKey ssh.Signer

type Config struct {
	// SSHPrivateKeyPath 是控制器登录实例所用的私钥文件路径。其公钥部分必须与
	// task-deploy.toml 中的 ssh_gomc_public_key 一致，否则部署后无法以 mc 用户连接。
	SSHPrivateKeyPath string `toml:"ssh_private_key_path" validate:"required" comment:"控制器登录实例所用的私钥文件路径，公钥需与 task-deploy.toml 的 ssh_gomc_public_key 一致"`
}

func MustLoadConfig() {
	utils.MustBindConfigToml(&C, "remote-util")
}

func MustInitialize() {
	signer, err := loadPrivateKey(C.SSHPrivateKeyPath)
	if err != nil {
		log_util.Fatal("加载SSH私钥失败: %v", err)
	}
	sshKey = signer
}

// VerifyAuthorizedKey 校验 authorized_keys 中的一行公钥是否与已加载的 mc 私钥配对。
// 部署脚本会把该行写进 /home/mc/.ssh/authorized_keys：两者不配对时部署会照常成功，
// 但此后所有以 mc 用户发起的操作都会认证失败，而且只表现为连接错误。
func VerifyAuthorizedKey(authorizedKeyLine string) error {
	if sshKey == nil {
		return fmt.Errorf("尚未加载 mc 用户的SSH私钥")
	}

	expected, _, _, _, err := ssh.ParseAuthorizedKey([]byte(authorizedKeyLine))
	if err != nil {
		return fmt.Errorf("无法解析公钥(%s): %w", authorizedKeyLine, err)
	}

	publicKey := sshKey.PublicKey()
	if !bytes.Equal(expected.Marshal(), publicKey.Marshal()) {
		return fmt.Errorf(
			"公钥与 remote-util.toml 中的私钥不配对：配置的公钥指纹为 %s，私钥对应的公钥指纹为 %s",
			ssh.FingerprintSHA256(expected), ssh.FingerprintSHA256(publicKey),
		)
	}

	return nil
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
