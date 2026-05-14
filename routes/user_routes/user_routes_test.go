package user_routes

import (
	"bytes"
	"encoding/json"
	"github.com/Subilan/go-aliyunmc/perms"
	"github.com/Subilan/go-aliyunmc/session"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const testDBPath = "user_routes_test.db"

func TestMain(m *testing.M) {
	store.C = store.Config{
		Driver: "sqlite",
		DBName: "user_routes_test",
		Path:   testDBPath,
	}

	_ = os.Remove(testDBPath)
	store.MustInitialize()
	store.AutoMigrate()
	perms.MustInitialize()

	code := m.Run()

	_ = os.Remove(testDBPath)

	os.Exit(code)
}

func setupTestRouter() http.Handler {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	Bind(router)
	return session.LoadAndSave(router)
}

func loginAndGetSessionCookie(t *testing.T, handler http.Handler, username, password string) *http.Cookie {
	t.Helper()

	loginReq := LoginRequest{
		Username: username,
		Password: password,
		Remember: true,
	}

	reqBody, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/user/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login failed, status=%d, body=%s", w.Code, w.Body.String())
	}

	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			return c
		}
	}

	t.Fatal("session cookie not found")
	return nil
}

func TestHandleRegister(t *testing.T) {
	handler := setupTestRouter()

	// 测试注册新用户
	registerReq := RegisterRequest{
		Username: "testuser",
		Password: "password123",
	}

	reqBody, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/user/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if response["data"] != nil {
		t.Errorf("Expected data to be nil, got %v", response["data"])
	}

	// 测试重复注册
	req, _ = http.NewRequest("POST", "/user/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status code %d for duplicate registration, got %d", http.StatusConflict, w.Code)
	}
}

func TestHandleLogin(t *testing.T) {
	handler := setupTestRouter()

	// 先创建测试用户
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	testUser := models.User{
		Username:     "loginuser",
		PasswordHash: string(hashedPassword),
	}
	store.DB.Create(&testUser)

	// 测试登录
	loginReq := LoginRequest{
		Username: "loginuser",
		Password: "password123",
		Remember: true,
	}

	reqBody, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/user/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if response["data"] != nil {
		t.Errorf("Expected data to be nil, got %v", response["data"])
	}

	// 测试错误密码
	loginReq.Password = "wrongpassword"
	reqBody, _ = json.Marshal(loginReq)
	req, _ = http.NewRequest("POST", "/user/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d for wrong password, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleDeleteUser(t *testing.T) {
	handler := setupTestRouter()

	// 先创建测试用户并登录拿到session
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	testUser := models.User{
		Username:     "deleteuser",
		PasswordHash: string(hashedPassword),
	}
	store.DB.Create(&testUser)
	cookie := loginAndGetSessionCookie(t, handler, "deleteuser", "password123")

	req, _ := http.NewRequest("DELETE", "/user", nil)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if response["data"] != nil {
		t.Errorf("Expected data to be nil, got %v", response["data"])
	}
}
