package remote_util

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPathExistsCommandHandlesSpecialCharacters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "player's data")
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	cmd := exec.Command("sh", "-c", pathExistsCommand(path))
	if err := cmd.Run(); err != nil {
		t.Fatalf("存在性检查未识别带空格和引号的路径: %v", err)
	}
}
