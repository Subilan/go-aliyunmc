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

// withSSHKey 将包级私钥临时替换为 signer，测试结束后恢复。
func withSSHKey(t *testing.T, signer ssh.Signer) {
	t.Helper()

	original := sshKey
	sshKey = signer
	t.Cleanup(func() { sshKey = original })
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

func TestAuthorizedKeyLine(t *testing.T) {
	signer := mustSigner(t)
	withSSHKey(t, signer)

	line, err := AuthorizedKeyLine()
	if err != nil {
		t.Fatalf("应当返回公钥行: %v", err)
	}

	// 部署任务把这一行原样写进 authorized_keys，所以它必须能被解析回来，
	// 且与私钥配对——这正是删掉配置里那份公钥后需要守住的唯一不变量。
	parsed, _, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
	if err != nil {
		t.Fatalf("推导出的公钥行无法解析: %v", err)
	}
	if got, want := ssh.FingerprintSHA256(parsed), ssh.FingerprintSHA256(signer.PublicKey()); got != want {
		t.Fatalf("推导出的公钥与私钥不配对: %s != %s", got, want)
	}
}

func TestAuthorizedKeyLineWithoutLoadedKey(t *testing.T) {
	withSSHKey(t, nil)

	if _, err := AuthorizedKeyLine(); err == nil {
		t.Fatal("未加载私钥时应当报错")
	}
}

// 密钥路径由参数传入，因此这个用例不需要碰任何全局状态。
func TestRsyncSSHCommandPinsIdentity(t *testing.T) {
	command := rsyncSSHCommand("/home/gomc/.ssh/id_gomc")

	if !strings.Contains(command, `-i "/home/gomc/.ssh/id_gomc"`) {
		t.Fatalf("未显式指定身份: %s", command)
	}
	if !strings.Contains(command, "IdentitiesOnly=yes") {
		t.Fatalf("未禁止回落其他身份: %s", command)
	}
}
