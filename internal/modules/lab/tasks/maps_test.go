package tasks

import (
	"testing"

	"github.com/Subilan/go-aliyunmc/internal/modules/lab/models"
)

func TestGetRunningTaskIDByType(t *testing.T) {
	const taskID = uint(4242)
	executor := NewExecutor(&TaskDefinition{Type: labmodels.TaskTypeOneClickStartGame})
	SetExecutor(taskID, executor)
	t.Cleanup(func() { DeleteExecutor(taskID) })

	got, ok := GetRunningTaskIDByType(labmodels.TaskTypeOneClickStartGame)
	if !ok || got != taskID {
		t.Fatalf("GetRunningTaskIDByType() = (%d, %v), want (%d, true)", got, ok, taskID)
	}

	if _, ok := GetRunningTaskIDByType(labmodels.TaskTypeStartServer); ok {
		t.Fatal("其他任务类型不应被匹配")
	}
}
