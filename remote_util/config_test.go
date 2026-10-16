package remote_util

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func writeKeyFile(t *testing.T, block *pem.Block) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0600); err != nil {
		t.Fatalf("写入私钥失败: %v", err)
	}
	return path
}

// mustSigner 生成一对临时密钥并返回其 signer。
func mustSigner(t *testing.T) ssh.Signer {
	t.Helper()

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatalf("构造 signer 失败: %v", err)
	}
	return signer
}

// authorizedKeyLine 返回 signer 对应的 authorized_keys 行。
func authorizedKeyLine(signer ssh.Signer) string {
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
}

func TestLoadPrivateKey(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}

	block, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		t.Fatalf("序列化私钥失败: %v", err)
	}

	signer, err := loadPrivateKey(writeKeyFile(t, block))
	if err != nil {
		t.Fatalf("加载私钥失败: %v", err)
	}
	if signer == nil {
		t.Fatal("未返回 signer")
	}
}

func TestLoadPrivateKeyRejectsPassphrase(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}

	block, err := ssh.MarshalPrivateKeyWithPassphrase(privateKey, "", []byte("secret"))
	if err != nil {
		t.Fatalf("序列化私钥失败: %v", err)
	}

	_, err = loadPrivateKey(writeKeyFile(t, block))
	if err == nil {
		t.Fatal("带口令的私钥应当被拒绝")
	}
	if !strings.Contains(err.Error(), "口令") {
		t.Fatalf("错误信息未提示口令问题: %v", err)
	}
}

func TestLoadPrivateKeyMissingFile(t *testing.T) {
	if _, err := loadPrivateKey(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("缺失的私钥应当报错")
	}
}

func TestRsyncSSHCommandPinsIdentity(t *testing.T) {
	original := C.SSHPrivateKeyPath
	C.SSHPrivateKeyPath = "/home/gomc/.ssh/id_gomc"
	t.Cleanup(func() { C.SSHPrivateKeyPath = original })

	command := rsyncSSHCommand()
	if !strings.Contains(command, `-i "/home/gomc/.ssh/id_gomc"`) {
		t.Fatalf("未显式指定身份: %s", command)
	}
	if !strings.Contains(command, "IdentitiesOnly=yes") {
		t.Fatalf("未禁止回落其他身份: %s", command)
	}
}

// withSSHKey 将包级私钥临时替换为 signer，测试结束后恢复。
func withSSHKey(t *testing.T, signer ssh.Signer) {
	t.Helper()

	original := sshKey
	sshKey = signer
	t.Cleanup(func() { sshKey = original })
}

func TestVerifyAuthorizedKey(t *testing.T) {
	signer := mustSigner(t)
	withSSHKey(t, signer)

	line := authorizedKeyLine(signer)
	if err := VerifyAuthorizedKey(line); err != nil {
		t.Fatalf("配对的公钥应当通过: %v", err)
	}

	// authorized_keys 行尾的注释不参与比对
	if err := VerifyAuthorizedKey(line + " gomc@controller"); err != nil {
		t.Fatalf("仅注释不同的公钥应当通过: %v", err)
	}

	err := VerifyAuthorizedKey(authorizedKeyLine(mustSigner(t)))
	if err == nil {
		t.Fatal("不配对的公钥应当被拒绝")
	}
	if !strings.Contains(err.Error(), "指纹") {
		t.Fatalf("错误信息未提示指纹差异: %v", err)
	}
}

func TestVerifyAuthorizedKeyRejectsMalformed(t *testing.T) {
	withSSHKey(t, mustSigner(t))

	// configgen 生成的示例公钥就是这种截断形式
	if err := VerifyAuthorizedKey("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ..."); err == nil {
		t.Fatal("无法解析的公钥应当报错")
	}
}

func TestVerifyAuthorizedKeyWithoutLoadedKey(t *testing.T) {
	withSSHKey(t, nil)

	if err := VerifyAuthorizedKey(authorizedKeyLine(mustSigner(t))); err == nil {
		t.Fatal("未加载私钥时应当报错")
	}
}
