package user_routes

import (
	"bytes"
	"encoding/json"
	"go-aliyunmc/store"
	"go-aliyunmc/store/models"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
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

	code := m.Run()

	_ = os.Remove(testDBPath)
	os.Exit(code)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 配置session中间件
	sessionStore := cookie.NewStore([]byte("secret-key-12345"))
	router.Use(sessions.Sessions("session", sessionStore))
	// 注册用户路由
	Bind(router)
	return router
}

func loginAndGetSessionCookie(t *testing.T, router *gin.Engine, username, password string) *http.Cookie {
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
	router.ServeHTTP(w, req)

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
	router := setupTestRouter()

	// 测试注册新用户
	registerReq := RegisterRequest{
		Username: "testuser",
		Password: "password123",
	}

	reqBody, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/user/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

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
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status code %d for duplicate registration, got %d", http.StatusConflict, w.Code)
	}
}

func TestHandleLogin(t *testing.T) {
	router := setupTestRouter()

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
	router.ServeHTTP(w, req)

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
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d for wrong password, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleDeleteUser(t *testing.T) {
	router := setupTestRouter()

	// 先创建测试用户并登录拿到session
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	testUser := models.User{
		Username:     "deleteuser",
		PasswordHash: string(hashedPassword),
	}
	store.DB.Create(&testUser)
	cookie := loginAndGetSessionCookie(t, router, "deleteuser", "password123")

	req, _ := http.NewRequest("DELETE", "/user", nil)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

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
