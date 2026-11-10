// Package session 持有 forum 模块独立的登录态。
// 论坛与 lab 共用用户凭据，但 cookie 名称与路径完全独立。
package session

import (
	"github.com/Subilan/go-aliyunmc/internal/common/session_util"
)

const (
	CookieName = "forum_session"
	CookiePath = "/forum-v1"
)

// Manager 是 forum 专属的 session manager。
var Manager = session_util.New(CookieName, CookiePath)
