package session

import (
	"context"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
)

var manager *scs.SessionManager

const (
	KeyUserId   = "user_id"
	KeyUsername = "username"
)

func init() {
	manager = scs.New()
	manager.Lifetime = 1 * time.Hour
	manager.Cookie.HttpOnly = true
	manager.Cookie.SameSite = http.SameSiteLaxMode
	manager.Cookie.Secure = false
	manager.Cookie.Path = "/"
}

func LoadAndSave(next http.Handler) http.Handler {
	return manager.LoadAndSave(next)
}

func gtx(c *gin.Context) context.Context {
	return c.Request.Context()
}

func Remove(c *gin.Context, key string) {
	manager.Remove(gtx(c), key)
}

func Get(c *gin.Context, key string) (result any) {
	result = manager.Get(gtx(c), key)
	return
}

func GetUsername(c *gin.Context) (string, bool) {
	username, ok := Get(c, KeyUsername).(string)
	return username, ok
}

func GetUserID(c *gin.Context) (uint, bool) {
	userID, ok := Get(c, KeyUserId).(uint)
	return userID, ok
}

func Set(c *gin.Context, key string, value any) {
	manager.Put(gtx(c), key, value)
}

func SetDeadline(c *gin.Context, duration time.Duration) {
	manager.SetDeadline(c.Request.Context(), time.Now().Add(duration))
}
