package tasks

import (
	"errors"
	"testing"

	"github.com/Subilan/go-aliyunmc/perms"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"
)

func TestExclusiveKey(t *testing.T) {
	if got := (&TaskDefinition{Type: "plain"}).exclusiveKey(); got != "" {
		t.Fatalf("non-exclusive task should have empty key, got %q", got)
	}
	if got := (&TaskDefinition{Type: "same_type", Exclusive: true}).exclusiveKey(); got != "same_type" {
		t.Fatalf("exclusive task should use its type as key, got %q", got)
	}
	if got := (&TaskDefinition{
		Type:           "grouped",
		Exclusive:      true,
		ExclusiveGroup: TaskExclusiveGroupDeployChain,
	}).exclusiveKey(); got != TaskExclusiveGroupDeployChain {
		t.Fatalf("grouped task should use the shared group key, got %q", got)
	}
	// ExclusiveGroup 只有在 Exclusive 为 true 时才生效。
	if got := (&TaskDefinition{Type: "grouped", ExclusiveGroup: "group"}).exclusiveKey(); got != "" {
		t.Fatalf("ExclusiveGroup without Exclusive should not reserve, got %q", got)
	}
}

func TestOneClickStartGameTaskDefinitionRegistered(t *testing.T) {
	RegisterTaskDefinitions()

	def, ok := TaskDefinitions[models.TaskTypeOneClickStartGame]
	if !ok {
		t.Fatalf("oneclick_start_game task definition is not registered")
	}
	if !def.Exclusive {
		t.Fatalf("oneclick_start_game should be exclusive")
	}
	if def.ExclusiveGroup != TaskExclusiveGroupDeployChain {
		t.Fatalf("oneclick_start_game should share %s, got %s", TaskExclusiveGroupDeployChain, def.ExclusiveGroup)
	}
	if def.F == nil || def.E == nil {
		t.Fatalf("oneclick_start_game should define F and E")
	}
}

// 共享 ExclusiveGroup 的不同任务类型之间也必须互斥；任务结束后释放占位。
func TestExclusiveGroupBlocksDifferentTypes(t *testing.T) {
	release := make(chan struct{})
	group := models.TaskType("exclusive_group_test")

	blockingDef := &TaskDefinition{
		Type:           "exclusive_group_test_a",
		Exclusive:      true,
		ExclusiveGroup: group,
		F: func(tc *TaskContext, _ map[string]any) error {
			select {
			case <-release:
				return nil
			case <-tc.Context().Done():
				return tc.Context().Err()
			}
		},
	}
	otherDef := &TaskDefinition{
		Type:           "exclusive_group_test_b",
		Exclusive:      true,
		ExclusiveGroup: group,
		F:              func(*TaskContext, map[string]any) error { return nil },
	}

	first := NewExecutor(blockingDef)
	if _, err := first.RunTask(nil, nil); err != nil {
		t.Fatalf("first RunTask failed: %v", err)
	}

	second := NewExecutor(otherDef)
	if _, err := second.RunTask(nil, nil); !errors.Is(err, ErrTaskTypeExecuting) {
		close(release)
		<-first.Done()
		t.Fatalf("expected ErrTaskTypeExecuting, got %v", err)
	}

	// 第一个任务结束后，同组占位应被释放。
	close(release)
	<-first.Done()

	third := NewExecutor(otherDef)
	if _, err := third.RunTask(nil, nil); err != nil {
		t.Fatalf("group key should be released after task completion: %v", err)
	}
	<-third.Done()
}

func TestNormalizeOneClickStartGameArgs(t *testing.T) {
	// 完全不带参数时使用默认交换机，并强制启动实例。
	normalized := normalizeOneClickStartGameArgs(nil)
	if normalized["startWhenCreated"] != true {
		t.Fatalf("startWhenCreated should be forced to true, got %v", normalized["startWhenCreated"])
	}
	if normalized["useDefaultVSwitch"] != true {
		t.Fatalf("useDefaultVSwitch should default to true, got %v", normalized["useDefaultVSwitch"])
	}

	// 显式指定的交换机不能被默认值覆盖。
	normalized = normalizeOneClickStartGameArgs(map[string]any{"vSwitchId": "vsw-test"})
	if _, ok := normalized["useDefaultVSwitch"]; ok {
		t.Fatalf("useDefaultVSwitch should not be added when vSwitchId is provided")
	}

	// 调用方的显式选择保留，只覆盖 startWhenCreated。
	normalized = normalizeOneClickStartGameArgs(map[string]any{"useDefaultVSwitch": false, "startWhenCreated": false})
	if normalized["useDefaultVSwitch"] != false {
		t.Fatalf("explicit useDefaultVSwitch=false should be preserved")
	}
	if normalized["startWhenCreated"] != true {
		t.Fatalf("startWhenCreated should be forced to true")
	}
}

func TestEnforceOneClickStartGameTaskSkipsCreateCheckWhenInstanceExists(t *testing.T) {
	instance := models.Instance{
		InstanceId:   "test-oneclick-start-game-resume",
		InstanceType: "ecs.test",
		RegionId:     "test-region",
		ZoneId:       "test-zone",
		VSwitchId:    "test-vswitch",
	}
	if err := store.DB.Create(&instance).Error; err != nil {
		t.Fatalf("create instance failed: %v", err)
	}
	defer store.DB.Unscoped().Delete(&instance)

	// 续跑不会创建实例，因此不应再执行 create-custom-instance 的权限检查。
	if err := enforceOneClickStartGameTask(perms.RoleBasic, map[string]any{"zoneId": "custom-zone"}); err != nil {
		t.Fatalf("expected no create permission check on resume, got %v", err)
	}
}
