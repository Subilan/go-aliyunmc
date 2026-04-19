package task_routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-aliyunmc/perms"
	"go-aliyunmc/store"
	"go-aliyunmc/store/models"
	"go-aliyunmc/tasks"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

const testDBPath = "task_routes_test.db"

func TestMain(m *testing.M) {
	store.C = store.Config{
		Driver: "sqlite",
		DBName: "task_routes_test",
		Path:   testDBPath,
	}
	perms.MustInitialize()
	store.MustInitialize()
	store.AutoMigrate()

	code := m.Run()
	_ = os.Remove(testDBPath)
	os.Exit(code)
}

func setupTaskRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	sessionStore := cookie.NewStore([]byte("task-routes-test-secret"))
	router.Use(sessions.Sessions("session", sessionStore))

	// 测试辅助路由：将 user_id 写入 session，便于通过 Auth 中间件。
	router.GET("/_test/login/:id", func(c *gin.Context) {
		var uid uint
		_, _ = fmt.Sscanf(c.Param("id"), "%d", &uid)
		s := sessions.Default(c)
		s.Set("user_id", uid)
		_ = s.Save()
		c.Status(http.StatusNoContent)
	})

	Bind(router)
	return router
}

func createTestUser(t *testing.T, username string) models.User {
	t.Helper()
	user := models.User{
		Username:     username,
		PasswordHash: "hash",
		Role:         perms.RoleBasic,
	}
	if err := store.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	return user
}

func loginCookie(t *testing.T, router *gin.Engine, userID uint) *http.Cookie {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/_test/login/%d", userID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
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

func postTrigger(t *testing.T, router *gin.Engine, cookie *http.Cookie, taskType string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"type": taskType})
	req, _ := http.NewRequest(http.MethodPost, "/task/trigger", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
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
	router := setupTaskRouter()
	w := postTrigger(t, router, nil, string(models.TaskTypeTest))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleTriggerTaskExecution_InvalidType(t *testing.T) {
	router := setupTaskRouter()
	user := createTestUser(t, fmt.Sprintf("task_user_invalid_%d", time.Now().UnixNano()))
	cookie := loginCookie(t, router, user.ID)

	w := postTrigger(t, router, cookie, "not_exist_type")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d, body=%s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestTaskRoutes_TriggerThenGetTask(t *testing.T) {
	router := setupTaskRouter()
	user := createTestUser(t, fmt.Sprintf("task_user_ok_%d", time.Now().UnixNano()))
	cookie := loginCookie(t, router, user.ID)

	triggerResp := postTrigger(t, router, cookie, string(models.TaskTypeTest))
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
	router.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get task expected %d, got %d, body=%s", http.StatusOK, getResp.Code, getResp.Body.String())
	}
}

func TestHandleGetTaskOutput_NotFoundExecutor(t *testing.T) {
	router := setupTaskRouter()
	user := createTestUser(t, fmt.Sprintf("task_user_output_%d", time.Now().UnixNano()))
	cookie := loginCookie(t, router, user.ID)

	req, _ := http.NewRequest(http.MethodGet, "/task/999999/output", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d, body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestHandleTriggerTaskExecution_TypeExclusiveMutex(t *testing.T) {
	router := setupTaskRouter()
	user := createTestUser(t, fmt.Sprintf("task_user_exclusive_%d", time.Now().UnixNano()))
	cookie := loginCookie(t, router, user.ID)

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

	first := postTrigger(t, router, cookie, string(taskType))
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

	second := postTrigger(t, router, cookie, string(taskType))
	if second.Code != http.StatusConflict {
		t.Fatalf("second trigger expected %d for exclusive lock, got %d, body=%s", http.StatusConflict, second.Code, second.Body.String())
	}
}
