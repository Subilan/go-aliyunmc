package user_routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/perm_util"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/common/testutil"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/perms"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/session"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func TestMain(m *testing.M) {
	config.Set(&config.Config{})
	perms.MustInitialize()

	cleanup, err := testutil.OpenMySQLTestDB()
	if err != nil {
		panic(err)
	}

	code := m.Run()

	cleanup()
	os.Exit(code)
}

func setupTestRouter() http.Handler {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	Bind(router.Group("/lab-v1"))
	return session.Manager.LoadAndSave(router)
}

func loginAndGetSessionCookie(t *testing.T, handler http.Handler, username, password string) *http.Cookie {
	t.Helper()

	loginReq := LoginRequest{
		Username: username,
		Password: password,
		Remember: true,
	}

	reqBody, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/lab-v1/user/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login failed, status=%d, body=%s", w.Code, w.Body.String())
	}

	for _, c := range w.Result().Cookies() {
		if c.Name == "lab_session" {
			return c
		}
	}

	t.Fatal("session cookie not found")
	return nil
}

func createUserWithRole(t *testing.T, username string, role perm_util.Role) *models.User {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := models.User{
		Username:     username,
		PasswordHash: string(hashedPassword),
		Role:         role,
	}
	if err := store.DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return &user
}

func TestHandleRegister(t *testing.T) {
	handler := setupTestRouter()

	// 测试注册新用户
	registerReq := RegisterRequest{
		Username: "testuser",
		Password: "password123",
	}

	reqBody, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/lab-v1/user/register", bytes.NewBuffer(reqBody))
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

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected created user data, got %v", response["data"])
	}
	if data["username"] != registerReq.Username {
		t.Errorf("Expected username %q, got %v", registerReq.Username, data["username"])
	}
	if data["role"] != string(perm_util.RoleBasic) {
		t.Errorf("Expected role %q, got %v", perm_util.RoleBasic, data["role"])
	}

	var created models.User
	if err := store.DB.Where("username = ?", registerReq.Username).First(&created).Error; err != nil {
		t.Fatalf("failed to find registered user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(created.PasswordHash), []byte(registerReq.Password)); err != nil {
		t.Fatalf("registered user password is invalid: %v", err)
	}

	// 测试重复注册
	req, _ = http.NewRequest("POST", "/lab-v1/user/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status code %d for duplicate registration, got %d", http.StatusConflict, w.Code)
	}

	invalidReq := RegisterRequest{Username: "bad password", Password: "short"}
	reqBody, _ = json.Marshal(invalidReq)
	req, _ = http.NewRequest("POST", "/lab-v1/user/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for invalid credentials, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleCreateUserWithRole(t *testing.T) {
	handler := setupTestRouter()
	suffix := time.Now().UnixNano()
	operator := createUserWithRole(t, fmt.Sprintf("create_operator_%d", suffix), perm_util.RoleOperator)
	cookie := loginAndGetSessionCookie(t, handler, operator.Username, "password123")
	username := fmt.Sprintf("m%08d", suffix%100_000_000)

	body, _ := json.Marshal(CreateUserRequest{
		Username: username,
		Password: "newpass123",
		Role:     perm_util.RoleOperator,
	})
	req, _ := http.NewRequest("POST", "/lab-v1/user/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	var created models.User
	if err := store.DB.Where("username = ?", username).First(&created).Error; err != nil {
		t.Fatalf("failed to find created user: %v", err)
	}
	if created.Role != perm_util.RoleOperator {
		t.Fatalf("expected created role %q, got %q", perm_util.RoleOperator, created.Role)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(created.PasswordHash), []byte("newpass123")); err != nil {
		t.Fatalf("created user password is invalid: %v", err)
	}

	privilegedUsername := fmt.Sprintf("s%08d", suffix%100_000_000)
	body, _ = json.Marshal(CreateUserRequest{
		Username: privilegedUsername,
		Password: "newpass123",
		Role:     perm_util.RoleSuperuser,
	})
	req, _ = http.NewRequest("POST", "/lab-v1/user/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusForbidden, w.Code, w.Body.String())
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
	req, _ := http.NewRequest("POST", "/lab-v1/user/login", bytes.NewBuffer(reqBody))
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
	req, _ = http.NewRequest("POST", "/lab-v1/user/login", bytes.NewBuffer(reqBody))
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

	req, _ := http.NewRequest("DELETE", "/lab-v1/user", nil)
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

func TestHandleChangeUserRole(t *testing.T) {
	handler := setupTestRouter()
	suffix := time.Now().UnixNano()
	current := createUserWithRole(t, fmt.Sprintf("role_superuser_%d", suffix), perm_util.RoleSuperuser)
	target := createUserWithRole(t, fmt.Sprintf("role_target_%d", suffix), perm_util.RoleBasic)
	cookie := loginAndGetSessionCookie(t, handler, current.Username, "password123")

	changeRole := func(userID uint, role perm_util.Role) *httptest.ResponseRecorder {
		t.Helper()
		body, _ := json.Marshal(ChangeUserRoleRequest{UserID: userID, Role: role})
		req, _ := http.NewRequest("POST", "/lab-v1/user/role", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}

	if w := changeRole(target.ID, perm_util.RoleOperator); w.Code != http.StatusOK {
		t.Fatalf("expected role change status %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	var updated models.User
	if err := store.DB.First(&updated, target.ID).Error; err != nil {
		t.Fatalf("failed to reload target user: %v", err)
	}
	if updated.Role != perm_util.RoleOperator {
		t.Fatalf("expected target role %q, got %q", perm_util.RoleOperator, updated.Role)
	}

	if w := changeRole(current.ID, perm_util.RoleBasic); w.Code != http.StatusForbidden {
		t.Fatalf("expected self-change status %d, got %d, body=%s", http.StatusForbidden, w.Code, w.Body.String())
	}

	if w := changeRole(target.ID, perm_util.Role("unknown")); w.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid-role status %d, got %d, body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestHandleChangeUserRoleRequiresSuperuser(t *testing.T) {
	handler := setupTestRouter()
	suffix := time.Now().UnixNano()
	operator := createUserWithRole(t, fmt.Sprintf("role_operator_%d", suffix), perm_util.RoleOperator)
	target := createUserWithRole(t, fmt.Sprintf("role_operator_target_%d", suffix), perm_util.RoleBasic)
	cookie := loginAndGetSessionCookie(t, handler, operator.Username, "password123")

	body, _ := json.Marshal(ChangeUserRoleRequest{UserID: target.ID, Role: perm_util.RoleSuperuser})
	req, _ := http.NewRequest("POST", "/lab-v1/user/role", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusForbidden, w.Code, w.Body.String())
	}
}
