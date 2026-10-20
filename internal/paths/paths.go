// Package paths 由 STATE_DIR 派生配置与运行时路径。
package paths

import (
	"os"
	"path/filepath"
	"strings"
)

// StateDirEnv 是程序根目录的环境变量名，未设置时使用当前工作目录。
const StateDirEnv = "GO_ALIYUNMC_STATE_DIR"

// Root 返回程序根目录。
func Root() string {
	if dir := strings.TrimSpace(os.Getenv(StateDirEnv)); dir != "" {
		return dir
	}

	return "."
}

// ConfigDir 返回配置目录。
func ConfigDir() string {
	return filepath.Join(Root(), "config")
}

// RuntimeDir 返回运行时数据目录。
func RuntimeDir() string {
	return filepath.Join(Root(), "state")
}

// LogsDir 返回日志目录。
func LogsDir() string {
	return filepath.Join(RuntimeDir(), "logs")
}

// CacheDir 返回本地缓存目录。
func CacheDir() string {
	return filepath.Join(RuntimeDir(), "cache")
}

// EcsCandidatesFile 返回最佳 ECS 候选缓存文件路径。
func EcsCandidatesFile() string {
	return filepath.Join(RuntimeDir(), "ecs_candidates.json")
}
