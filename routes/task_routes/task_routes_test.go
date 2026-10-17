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

const testDBPath = "task_routes_test.db"

func TestMain(m *testing.M) {
	if err := testutil.ChdirProjectRoot(); err != nil {
		panic(err)
	}
	config.Set(&config.Config{
		Store: config.Store{
			Driver: "sqlite",
			DBName: "task_routes_test",
			Path:   testDBPath,
		},
		Perms: config.Perms{
			ModelPath:  "rbac_model.conf",
			PolicyPath: "rbac_policy.csv",
		},
	})
	tasks.RegisterTaskDefinitions()
	perms.MustInitialize()
	store.MustInitialize()
	if err := store.DB.AutoMigrate(models.Entities()...); err != nil {
		panic(err)
	}

	code := m.Run()
	_ = os.Remove(testDBPath)
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
	whitelistUUID := fmt.Sprintf("wl-%s", username)
	user := models.User{
		Username:      username,
		PasswordHash:  "hash",
		Role:          perms.RoleBasic,
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
	if !strings.Contains(body, "id: 4\n") {
		t.Fatalf("expected task_done id 4:\n%s", body)
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

	// task_done 的事件 ID 是 lastSeq+1 = 2；携带它重连时应返回 204。
	w := requestTaskOutput(t, handler, cookie, task.ID, "2")
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
