package task_routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/internal/testutil"
	"github.com/Subilan/go-aliyunmc/perms"
	"github.com/Subilan/go-aliyunmc/session"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"
	"github.com/Subilan/go-aliyunmc/tasks"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	if err := testutil.ChdirProjectRoot(); err != nil {
		panic(err)
	}
	config.Set(&config.Config{
		Perms: config.Perms{
			ModelPath:  "rbac_model.conf",
			PolicyPath: "rbac_policy.csv",
		},
	})
	tasks.RegisterTaskDefinitions()
	perms.MustInitialize()

	cleanup, err := testutil.OpenMySQLTestDB()
	if err != nil {
		panic(err)
	}

	code := m.Run()

	cleanup()
	os.Exit(code)
}

func setupTaskRouter() http.Handler {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 测试辅助路由：将 user_id 写入 session，便于通过 Auth 中间件。
	router.GET("/_test/login/:id", func(c *gin.Context) {
		var uid uint
		_, _ = fmt.Sscanf(c.Param("id"), "%d", &uid)
		session.Set(c, session.KeyUserId, uid)
		c.Status(http.StatusNoContent)
	})

	Bind(router)
	return session.LoadAndSave(router)
}

func createTestUser(t *testing.T, username string) models.User {
	t.Helper()
	return createTestUserWithRole(t, username, perms.RoleBasic)
}

func createTestUserWithRole(t *testing.T, username string, role perms.Role) models.User {
	t.Helper()
	whitelistUUID := fmt.Sprintf("wl-%s", username)
	user := models.User{
		Username:      username,
		PasswordHash:  "hash",
		Role:          role,
		WhitelistUUID: &whitelistUUID,
	}
	if err := store.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	return user
}

func loginCookie(t *testing.T, handler http.Handler, userID uint) *http.Cookie {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/_test/login/%d", userID), nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("login helper failed: status=%d", w.Code)
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	t.Fatal("session cookie not found")
	return nil
}

func postTrigger(t *testing.T, handler http.Handler, cookie *http.Cookie, taskType string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"type": taskType})
	req, _ := http.NewRequest(http.MethodPost, "/task/trigger", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func cleanupExecutor(taskID uint) {
	executor, ok := tasks.GetExecutor(taskID)
	if !ok {
		return
	}
	executor.Interrupt()
	select {
	case <-executor.Done():
	case <-time.After(2 * time.Second):
	}
}

func TestHandleTriggerTaskExecution_Unauthorized(t *testing.T) {
	handler := setupTaskRouter()
	w := postTrigger(t, handler, nil, string(models.TaskTypeTest))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleTriggerTaskExecution_InvalidType(t *testing.T) {
	handler := setupTaskRouter()
	user := createTestUser(t, fmt.Sprintf("task_user_invalid_%d", time.Now().UnixNano()))
	cookie := loginCookie(t, handler, user.ID)

	w := postTrigger(t, handler, cookie, "not_exist_type")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d, body=%s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestTaskRoutes_TriggerThenGetTask(t *testing.T) {
	handler := setupTaskRouter()
	user := createTestUser(t, fmt.Sprintf("task_user_ok_%d", time.Now().UnixNano()))
	cookie := loginCookie(t, handler, user.ID)

	triggerResp := postTrigger(t, handler, cookie, string(models.TaskTypeTest))
	if triggerResp.Code != http.StatusOK {
		t.Fatalf("trigger expected %d, got %d, body=%s", http.StatusOK, triggerResp.Code, triggerResp.Body.String())
	}

	var payload struct {
		Data struct {
			ID uint `json:"ID"`
		} `json:"data"`
	}
	if err := json.Unmarshal(triggerResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal trigger response failed: %v", err)
	}
	if payload.Data.ID == 0 {
		t.Fatalf("expected non-zero task id, response=%s", triggerResp.Body.String())
	}
	defer cleanupExecutor(payload.Data.ID)

	getReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/task/%d", payload.Data.ID), nil)
	getReq.AddCookie(cookie)
	getResp := httptest.NewRecorder()
	handler.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get task expected %d, got %d, body=%s", http.StatusOK, getResp.Code, getResp.Body.String())
	}
}

func TestHandleGetTaskOutput_NotFoundExecutor(t *testing.T) {
	handler := setupTaskRouter()
	user := createTestUser(t, fmt.Sprintf("task_user_output_%d", time.Now().UnixNano()))
	cookie := loginCookie(t, handler, user.ID)

	req, _ := http.NewRequest(http.MethodGet, "/task/999999/output", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d, body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestHandleTriggerTaskExecution_TypeExclusiveMutex(t *testing.T) {
	handler := setupTaskRouter()
	user := createTestUser(t, fmt.Sprintf("task_user_exclusive_%d", time.Now().UnixNano()))
	cookie := loginCookie(t, handler, user.ID)

	taskType := models.TaskType(fmt.Sprintf("exclusive_test_%d", time.Now().UnixNano()))
	release := make(chan struct{})

	// 在测试内注册自定义任务定义，不依赖项目内置的任务定义。
	tasks.TaskDefinitions[taskType] = &tasks.TaskDefinition{
		Type:      taskType,
		Exclusive: true,
		Timeout:   10 * time.Second,
		F: func(tc *tasks.TaskContext, _ map[string]any) error {
			select {
			case <-release:
				return nil
			case <-tc.Context().Done():
				return tc.Context().Err()
			}
		},
	}
	defer delete(tasks.TaskDefinitions, taskType)

	first := postTrigger(t, handler, cookie, string(taskType))
	if first.Code != http.StatusOK {
		t.Fatalf("first trigger expected %d, got %d, body=%s", http.StatusOK, first.Code, first.Body.String())
	}

	var payload struct {
		Data struct {
			ID uint `json:"ID"`
		} `json:"data"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal first trigger response failed: %v", err)
	}
	if payload.Data.ID == 0 {
		t.Fatalf("expected non-zero task id, response=%s", first.Body.String())
	}

	defer func() {
		close(release)
		cleanupExecutor(payload.Data.ID)
	}()

	second := postTrigger(t, handler, cookie, string(taskType))
	if second.Code != http.StatusConflict {
		t.Fatalf("second trigger expected %d for exclusive lock, got %d, body=%s", http.StatusConflict, second.Code, second.Body.String())
	}
}

func postInterrupt(t *testing.T, handler http.Handler, cookie *http.Cookie, taskID uint) *httptest.ResponseRecorder {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/task/%d/interrupt", taskID), nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

// registerBlockingTask 注册一个阻塞到 ctx 取消或 release 关闭的任务定义。
func registerBlockingTask(t *testing.T, taskType models.TaskType, release <-chan struct{}) {
	t.Helper()
	tasks.TaskDefinitions[taskType] = &tasks.TaskDefinition{
		Type:    taskType,
		Timeout: 10 * time.Second,
		F: func(tc *tasks.TaskContext, _ map[string]any) error {
			select {
			case <-release:
				return nil
			case <-tc.Context().Done():
				return tc.Context().Err()
			}
		},
	}
	t.Cleanup(func() { delete(tasks.TaskDefinitions, taskType) })
}

func triggerAndGetID(t *testing.T, handler http.Handler, cookie *http.Cookie, taskType models.TaskType) uint {
	t.Helper()
	resp := postTrigger(t, handler, cookie, string(taskType))
	if resp.Code != http.StatusOK {
		t.Fatalf("trigger expected %d, got %d, body=%s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload struct {
		Data struct {
			ID uint `json:"ID"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal trigger response failed: %v", err)
	}
	if payload.Data.ID == 0 {
		t.Fatalf("expected non-zero task id, response=%s", resp.Body.String())
	}
	return payload.Data.ID
}

// operator 中止任务后返回 202；任务在执行器异步收尾后标记为中断失败。
func TestHandleInterruptTask_OperatorSucceeds(t *testing.T) {
	handler := setupTaskRouter()
	user := createTestUserWithRole(t, fmt.Sprintf("task_user_interrupt_%d", time.Now().UnixNano()), perms.RoleOperator)
	cookie := loginCookie(t, handler, user.ID)

	release := make(chan struct{})
	defer close(release)
	taskType := models.TaskType(fmt.Sprintf("interrupt_route_test_%d", time.Now().UnixNano()))
	registerBlockingTask(t, taskType, release)

	taskID := triggerAndGetID(t, handler, cookie, taskType)

	w := postInterrupt(t, handler, cookie, taskID)
	if w.Code != http.StatusAccepted {
		t.Fatalf("interrupt expected %d, got %d, body=%s", http.StatusAccepted, w.Code, w.Body.String())
	}

	// 202 只表示中断请求已受理，终态由 monitor 异步写入。
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, ok := tasks.GetExecutor(taskID); !ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("executor should be unregistered after the task stopped")
		}
		time.Sleep(10 * time.Millisecond)
	}

	task, err := store.GetTaskRecord(taskID)
	if err != nil {
		t.Fatalf("get task failed: %v", err)
	}
	if task.Status != models.TaskStatusFailed || task.Error != "__INTERRUPTED__" {
		t.Fatalf("expected interrupted failure, got status=%s error=%q", task.Status, task.Error)
	}
}

// basic 角色不能中止任务。
func TestHandleInterruptTask_BasicForbidden(t *testing.T) {
	handler := setupTaskRouter()
	user := createTestUser(t, fmt.Sprintf("task_user_interrupt_basic_%d", time.Now().UnixNano()))
	cookie := loginCookie(t, handler, user.ID)

	release := make(chan struct{})
	defer close(release)
	taskType := models.TaskType(fmt.Sprintf("interrupt_route_basic_%d", time.Now().UnixNano()))
	registerBlockingTask(t, taskType, release)

	taskID := triggerAndGetID(t, handler, cookie, taskType)
	defer cleanupExecutor(taskID)

	w := postInterrupt(t, handler, cookie, taskID)
	if w.Code != http.StatusForbidden {
		t.Fatalf("interrupt expected %d, got %d, body=%s", http.StatusForbidden, w.Code, w.Body.String())
	}

	if _, ok := tasks.GetExecutor(taskID); !ok {
		t.Fatalf("task should still be running after a forbidden interrupt")
	}
}

// 已结束的任务再次中止应返回冲突。
func TestHandleInterruptTask_NotRunning(t *testing.T) {
	handler := setupTaskRouter()
	user := createTestUserWithRole(t, fmt.Sprintf("task_user_interrupt_done_%d", time.Now().UnixNano()), perms.RoleOperator)
	cookie := loginCookie(t, handler, user.ID)

	task := createFinishedTask(t)
	w := postInterrupt(t, handler, cookie, task.ID)
	if w.Code != http.StatusConflict {
		t.Fatalf("interrupt expected %d, got %d, body=%s", http.StatusConflict, w.Code, w.Body.String())
	}
}

// createFinishedTask 直接构造一个已结束任务及其输出行，用于验证数据库回放。
func createFinishedTask(t *testing.T, outputs ...string) models.Task {
	t.Helper()

	task := models.Task{
		Type:   models.TaskTypeTest,
		Status: models.TaskStatusSuccess,
	}
	if err := store.DB.Create(&task).Error; err != nil {
		t.Fatalf("create task failed: %v", err)
	}

	for i, line := range outputs {
		output := models.TaskOutputLine{
			TaskID: task.ID,
			Seq:    uint64(i + 1),
			Step:   0,
			Output: line,
		}
		if err := store.DB.Create(&output).Error; err != nil {
			t.Fatalf("create task output line failed: %v", err)
		}
	}

	return task
}

func requestTaskOutput(t *testing.T, handler http.Handler, cookie *http.Cookie, taskID uint, lastEventID string) *httptest.ResponseRecorder {
	t.Helper()

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/task/%d/output", taskID), nil)
	req.AddCookie(cookie)
	if lastEventID != "" {
		req.Header.Set("Last-Event-ID", lastEventID)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func TestHandleGetTaskOutput_FinishedTaskReplaysAll(t *testing.T) {
	handler := setupTaskRouter()
	user := createTestUser(t, fmt.Sprintf("task_user_replay_all_%d", time.Now().UnixNano()))
	cookie := loginCookie(t, handler, user.ID)

	task := createFinishedTask(t, "line one", "line two", "line three")

	w := requestTaskOutput(t, handler, cookie, task.ID, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	body := w.Body.String()
	for _, line := range []string{"line one", "line two", "line three"} {
		if !strings.Contains(body, fmt.Sprintf(`"output":%q`, line)) {
			t.Fatalf("expected %q in replay body:\n%s", line, body)
		}
	}
	if !strings.Contains(body, "event: task_done") {
		t.Fatalf("expected task_done event:\n%s", body)
	}
	if !strings.Contains(body, "id: done\n") {
		t.Fatalf("expected task_done event id:\n%s", body)
	}
}

func TestHandleGetTaskOutput_ResumesAfterLastEventID(t *testing.T) {
	handler := setupTaskRouter()
	user := createTestUser(t, fmt.Sprintf("task_user_replay_some_%d", time.Now().UnixNano()))
	cookie := loginCookie(t, handler, user.ID)

	task := createFinishedTask(t, "line one", "line two", "line three")

	w := requestTaskOutput(t, handler, cookie, task.ID, "1")
	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	body := w.Body.String()
	if strings.Contains(body, `"output":"line one"`) {
		t.Fatalf("line one should not be replayed:\n%s", body)
	}
	if !strings.Contains(body, `"output":"line two"`) || !strings.Contains(body, `"output":"line three"`) {
		t.Fatalf("expected remaining lines in replay body:\n%s", body)
	}
}

func TestHandleGetTaskOutput_StopsAfterTaskDone(t *testing.T) {
	handler := setupTaskRouter()
	user := createTestUser(t, fmt.Sprintf("task_user_replay_done_%d", time.Now().UnixNano()))
	cookie := loginCookie(t, handler, user.ID)

	task := createFinishedTask(t, "line one")

	// 客户端已经收到 task_done，携带该 ID 重连时应返回 204。
	w := requestTaskOutput(t, handler, cookie, task.ID, tasks.TaskDoneEventID)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d, body=%s", http.StatusNoContent, w.Code, w.Body.String())
	}
}

func TestHandleGetTask_IncludesReconstructedOutput(t *testing.T) {
	handler := setupTaskRouter()
	user := createTestUser(t, fmt.Sprintf("task_user_output_text_%d", time.Now().UnixNano()))
	cookie := loginCookie(t, handler, user.ID)

	task := createFinishedTask(t, "first", "second")

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/task/%d", task.ID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	var payload struct {
		Data struct {
			Output string `json:"output"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if payload.Data.Output != "first\nsecond\n" {
		t.Fatalf("unexpected output: %q", payload.Data.Output)
	}
}
