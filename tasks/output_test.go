package tasks

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/internal/testutil"
	"github.com/Subilan/go-aliyunmc/sse"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	config.Set(&config.Config{})

	cleanup, err := testutil.OpenMySQLTestDB()
	if err != nil {
		panic(err)
	}

	code := m.Run()

	cleanup()
	os.Exit(code)
}

func TestExecutorPersistsOutputLines(t *testing.T) {
	def := &TaskDefinition{
		Type: "output_persist_test",
		F: func(tc *TaskContext, _ map[string]any) error {
			tc.println("first")
			tc.nextStep()
			tc.println("second")
			return nil
		},
	}

	executor := NewExecutor(def)
	task, err := executor.RunTaskAndWait(nil, nil)
	if err != nil {
		t.Fatalf("run task failed: %v", err)
	}

	lines, err := store.GetTaskOutputLinesRange(task.ID, 0, nil, 0)
	if err != nil {
		t.Fatalf("read output lines failed: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d", len(lines))
	}
	if lines[0].Seq != 1 || lines[0].Step != 0 || lines[0].Output != "first" {
		t.Fatalf("unexpected first line: %+v", lines[0])
	}
	if lines[1].Seq != 2 || lines[1].Step != 1 || lines[1].Output != "second" {
		t.Fatalf("unexpected second line: %+v", lines[1])
	}

	got, err := store.GetTask(task.ID)
	if err != nil {
		t.Fatalf("get task failed: %v", err)
	}
	if got.Output != "first\nsecond\n" {
		t.Fatalf("unexpected reconstructed output: %q", got.Output)
	}
	if got.Status != models.TaskStatusSuccess {
		t.Fatalf("expected success status, got %q", got.Status)
	}
	if got.EndAt == nil {
		t.Fatal("expected end_at to be persisted")
	}

	// 旧 tasks.output 列只为兼容保留，新任务不应再往里面写内容。
	var legacyOutput string
	if err := store.DB.Table("tasks").Select("output").Where("id = ?", task.ID).Scan(&legacyOutput).Error; err != nil {
		t.Fatalf("read legacy output column failed: %v", err)
	}
	if legacyOutput != "" {
		t.Fatalf("legacy output column should stay empty, got %q", legacyOutput)
	}
}

func TestSubscribeFromReplaysHistoryThenLive(t *testing.T) {
	release := make(chan struct{})

	def := &TaskDefinition{
		Type: "output_replay_test",
		F: func(tc *TaskContext, _ map[string]any) error {
			tc.println("history")
			<-release
			tc.println("live")
			return nil
		},
	}

	executor := NewExecutor(def)
	if _, err := executor.RunTask(nil, nil); err != nil {
		t.Fatalf("run task failed: %v", err)
	}
	defer func() {
		executor.Interrupt()
		<-executor.Done()
	}()

	deadline := time.Now().Add(2 * time.Second)
	for executor.lastSeq.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if executor.lastSeq.Load() < 1 {
		t.Fatal("task output was not assigned a sequence in time")
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	client, err := sse.NewClient(c)
	if err != nil {
		t.Fatalf("create sse client failed: %v", err)
	}

	unsubscribe, ok := executor.SubscribeFrom(client, 0)
	if !ok {
		t.Fatal("subscribe from failed")
	}
	defer unsubscribe()

	listenDone := make(chan struct{})
	go func() {
		client.Listen()
		close(listenDone)
	}()

	close(release)

	select {
	case <-listenDone:
	case <-time.After(3 * time.Second):
		t.Fatal("sse listen did not finish")
	}

	body := recorder.Body.String()
	if count := strings.Count(body, `"output":"history"`); count != 1 {
		t.Fatalf("expected history replayed exactly once, got %d\nbody:\n%s", count, body)
	}
	if !strings.Contains(body, `"output":"live"`) {
		t.Fatalf("expected live output in stream\nbody:\n%s", body)
	}
	if !strings.Contains(body, "event: task_done") {
		t.Fatalf("expected task_done event\nbody:\n%s", body)
	}
	if !strings.Contains(body, "id: 1\n") || !strings.Contains(body, "id: 2\n") {
		t.Fatalf("expected output event ids\nbody:\n%s", body)
	}
	if !strings.Contains(body, "id: done\n") {
		t.Fatalf("expected task_done event id\nbody:\n%s", body)
	}
}

func TestReplayTaskOutputResumesAfterSeq(t *testing.T) {
	def := &TaskDefinition{
		Type: "output_dedup_test",
		F: func(tc *TaskContext, _ map[string]any) error {
			for i := 0; i < 3; i++ {
				tc.println(fmt.Sprintf("line-%d", i))
			}
			return nil
		},
	}

	executor := NewExecutor(def)
	task, err := executor.RunTaskAndWait(nil, nil)
	if err != nil {
		t.Fatalf("run task failed: %v", err)
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	client, err := sse.NewClient(c)
	if err != nil {
		t.Fatalf("create sse client failed: %v", err)
	}

	// 任务已经结束，执行器不再接受订阅；这里直接验证数据库回放路径。
	lastSeq, err := store.GetLastTaskOutputSeq(task.ID)
	if err != nil {
		t.Fatalf("get last seq failed: %v", err)
	}
	if lastSeq != 3 {
		t.Fatalf("expected last seq 3, got %d", lastSeq)
	}

	ReplayTaskOutput(client, task, 2)
	client.Listen()

	body := recorder.Body.String()
	if strings.Contains(body, `"output":"line-0"`) {
		t.Fatalf("line-0 should not be replayed after seq 2\nbody:\n%s", body)
	}
	if !strings.Contains(body, `"output":"line-2"`) {
		t.Fatalf("expected line-2 in replay\nbody:\n%s", body)
	}
	if strings.Count(body, "event: task_done") != 1 {
		t.Fatalf("expected exactly one task_done event\nbody:\n%s", body)
	}
}
