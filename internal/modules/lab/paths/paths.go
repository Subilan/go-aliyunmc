// Package paths 提供 lab 模块专属的运行时路径。
package paths

import (
	"path/filepath"

	"github.com/Subilan/go-aliyunmc/internal/common/paths"
)

// EcsCandidatesFile 返回最佳 ECS 候选缓存文件路径。
func EcsCandidatesFile() string {
	return filepath.Join(paths.RuntimeDir(), "ecs_candidates.json")
}
