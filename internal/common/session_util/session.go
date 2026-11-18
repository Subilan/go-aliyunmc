package session_util

import (
	"net/http"
	"time"

	"github.com/alexedwards/scs/gormstore"
	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	KeyUserId   = "user_id"
	KeyUsername = "username"
)

// Manager 包装 scs.SessionManager，使不同模块可以各自持有独立的 cookie 与登录态。
type Manager struct {
	scs *scs.SessionManager
}

// New 创建一个模块专属的 session manager。
// cookieName 与 cookiePath 让不同模块的 cookie 互不发送，从而保持登录态独立。
func New(cookieName, cookiePath string) *Manager {
	manager := scs.New()
	manager.Lifetime = 1 * time.Hour
	manager.Cookie.Name = cookieName
	manager.Cookie.HttpOnly = true
	manager.Cookie.SameSite = http.SameSiteLaxMode
	manager.Cookie.Secure = false
	manager.Cookie.Path = cookiePath

	return &Manager{scs: manager}
}

// InitStore 创建一次数据库 session 存储并安装到所有 manager。
// 需要在 store.MustInitialize() 之后调用。
func InitStore(db *gorm.DB, managers ...*Manager) error {
	s, err := gormstore.New(db)
	if err != nil {
		return err
	}

	for _, manager := range managers {
		manager.scs.Store = s
	}
	return nil
}

// LoadAndSave 返回使用该 manager 的 session 中间件。
func (m *Manager) LoadAndSave(next http.Handler) http.Handler {
	return m.scs.LoadAndSave(next)
}

// SetSecure 设置 cookie 的 Secure 属性。生产环境启用 TLS 后应为 true。
func (m *Manager) SetSecure(secure bool) {
	m.scs.Cookie.Secure = secure
}

// RenewToken 为当前 session 生成新的 token，登录成功后必须调用，避免会话固定攻击。
func (m *Manager) RenewToken(c *gin.Context) error {
	return m.scs.RenewToken(c.Request.Context())
}

// Destroy 删除当前 session 并清除 cookie。
func (m *Manager) Destroy(c *gin.Context) error {
	return m.scs.Destroy(c.Request.Context())
}

func (m *Manager) Remove(c *gin.Context, key string) {
	m.scs.Remove(c.Request.Context(), key)
}

func (m *Manager) Get(c *gin.Context, key string) (result any) {
	result = m.scs.Get(c.Request.Context(), key)
	return
}

func (m *Manager) GetUsername(c *gin.Context) (string, bool) {
	username, ok := m.Get(c, KeyUsername).(string)
	return username, ok
}

func (m *Manager) GetUserID(c *gin.Context) (uint, bool) {
	userID, ok := m.Get(c, KeyUserId).(uint)
	return userID, ok
}

func (m *Manager) Set(c *gin.Context, key string, value any) {
	m.scs.Put(c.Request.Context(), key, value)
}

func (m *Manager) SetDeadline(c *gin.Context, duration time.Duration) {
	m.scs.SetDeadline(c.Request.Context(), time.Now().Add(duration))
}
