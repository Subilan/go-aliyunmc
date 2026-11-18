// Package session 持有 forum 模块独立的登录态。
// 论坛与 lab 共用用户凭据，但 cookie 名称与路径完全独立。
package session

import (
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/session_util"
)

const (
	CookieName = "forum_session"
	CookiePath = "/forum-v1"

	// AdminCookieName 是管理后台独立 session 的 cookie 名。
	AdminCookieName = "forum_admin_session"
	// AdminCookiePath 把后台 cookie 限定在 /forum-v1/admin 下。
	AdminCookiePath = "/forum-v1/admin"

	// ForumRememberLifetime 是论坛“记住我”的登录有效期。
	ForumRememberLifetime = 7 * 24 * time.Hour
	// ForumDefaultLifetime 是论坛默认登录有效期。
	ForumDefaultLifetime = time.Hour
	// AdminRememberLifetime 是后台“记住我”的登录有效期。
	AdminRememberLifetime = 12 * time.Hour
	// AdminDefaultLifetime 是后台默认登录有效期。
	AdminDefaultLifetime = time.Hour
)

// Manager 是 forum 前台专属的 session manager。
var Manager = session_util.New(CookieName, CookiePath)

// AdminManager 是管理后台专属的 session manager。
var AdminManager = session_util.New(AdminCookieName, AdminCookiePath)
