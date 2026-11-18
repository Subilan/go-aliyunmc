package auth_routes

import (
	"net/http"
	"strings"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/log_util"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dto"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/perms"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/service"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/session"

	"github.com/gin-gonic/gin"
)

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Remember bool   `json:"remember"`
}

// Bind 注册论坛前台认证路由。
func Bind(router *gin.RouterGroup) {
	group := router.Group("/auth")
	group.POST("/login", h.B(handleLogin))
	group.POST("/logout", h.G(handleLogout))
	group.GET("/me", mid.OptionalAuth(), h.G(handleMe))
}

// BindAdmin 注册管理后台认证路由（router 为 /admin 组）。
func BindAdmin(router *gin.RouterGroup) {
	group := router.Group("/auth")
	group.POST("/login", h.B(handleAdminLogin))
	group.POST("/logout", h.G(handleAdminLogout))
	group.GET("/me", mid.AdminAuth(), h.G(handleAdminMe))
}

func handleLogin(req loginRequest, c *gin.Context) (any, error) {
	ip := c.ClientIP()
	if !mid.Hit("forum-login-ip", loginIPLimit(), ip) {
		return nil, h.HttpError(http.StatusTooManyRequests, "登录尝试过于频繁，请稍后再试")
	}
	usernameKey, ok := loginKey(req.Username)
	if !ok {
		return nil, h.HttpError(http.StatusUnauthorized, "账号或密码错误")
	}
	if !mid.Hit("forum-login", loginLimit(), ip+":"+usernameKey) {
		return nil, h.HttpError(http.StatusTooManyRequests, "登录尝试过于频繁，请稍后再试")
	}

	user, result, err := service.ForumLogin(req.Username, req.Password, c)
	if err != nil {
		log_util.Warn("forum 登录失败: username=%s ip=%s error=%v", req.Username, ip, err)
		return nil, err
	}

	// 登录成功后轮换 session token，防止会话固定。
	if err := session.Manager.RenewToken(c); err != nil {
		return nil, err
	}

	lifetime := service.SetSession(func(key string, value any) {
		session.Manager.Set(c, key, value)
	}, user, req.Remember, session.ForumRememberLifetime, session.ForumDefaultLifetime)
	session.Manager.SetDeadline(c, lifetime)

	return result, nil
}

func handleLogout(c *gin.Context) (any, error) {
	if err := session.Manager.Destroy(c); err != nil {
		return nil, err
	}
	return nil, nil
}

func handleMe(c *gin.Context) (any, error) {
	user, ok := mid.CurrentUser(c)
	if !ok {
		// session 存在但用户已删除/封禁时，清掉无效登录态。
		if _, has := session.Manager.GetUserID(c); has {
			_ = session.Manager.Destroy(c)
		}
		return dto.Me{
			User:              nil,
			GlobalPermissions: []string{},
			BoardPermissions:  map[string][]string{},
		}, nil
	}

	global, boards := perms.PermissionsOf(user)
	summary := dto.SummaryFromView(dao.LoadUserViews([]uint{user.ID})[user.ID])
	return dto.Me{
		User:              &summary,
		GlobalPermissions: global,
		BoardPermissions:  boards,
	}, nil
}

func handleAdminLogin(req loginRequest, c *gin.Context) (any, error) {
	ip := c.ClientIP()
	if !mid.Hit("admin-login-ip", loginIPLimit(), ip) {
		return nil, h.HttpError(http.StatusTooManyRequests, "登录尝试过于频繁，请稍后再试")
	}
	usernameKey, ok := loginKey(req.Username)
	if !ok {
		return nil, h.HttpError(http.StatusUnauthorized, "账号或密码错误")
	}
	if !mid.Hit("admin-login", loginLimit(), ip+":"+usernameKey) {
		return nil, h.HttpError(http.StatusTooManyRequests, "登录尝试过于频繁，请稍后再试")
	}

	user, result, err := service.AdminLogin(req.Username, req.Password, c)
	if err != nil {
		log_util.Warn("forum 后台登录失败: username=%s ip=%s error=%v", req.Username, ip, err)
		return nil, err
	}

	// 登录成功后轮换 session token，防止会话固定。
	if err := session.AdminManager.RenewToken(c); err != nil {
		return nil, err
	}

	lifetime := service.SetSession(func(key string, value any) {
		session.AdminManager.Set(c, key, value)
	}, user, req.Remember, session.AdminRememberLifetime, session.AdminDefaultLifetime)
	session.AdminManager.SetDeadline(c, lifetime)

	return result, nil
}

func handleAdminLogout(c *gin.Context) (any, error) {
	if err := session.AdminManager.Destroy(c); err != nil {
		return nil, err
	}
	return nil, nil
}

func handleAdminMe(c *gin.Context) (any, error) {
	user, ok := mid.CurrentUser(c)
	if !ok {
		return nil, h.HttpError(http.StatusUnauthorized, "未登录")
	}

	global, boards := perms.PermissionsOf(user)
	summary := dto.SummaryFromView(dao.LoadUserViews([]uint{user.ID})[user.ID])

	return dto.AdminMe{
		User:              &summary,
		IsSuperuser:       perms.IsSuperuser(user),
		GlobalPermissions: global,
		BoardPermissions:  boards,
	}, nil
}

// loginKey 规范化限流用的用户名，与 Authenticate 的 TrimSpace 行为保持一致，
// 并限制长度，避免超长用户名生成巨大的限流器 key。
func loginKey(username string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(username))
	if normalized == "" || len(normalized) > 191 {
		return "", false
	}
	return normalized, true
}

func loginLimit() int {
	cfg := config.Get().Forum.RateLimit
	if !cfg.Enabled {
		return 0
	}
	return cfg.LoginPerMinute
}

func loginIPLimit() int {
	cfg := config.Get().Forum.RateLimit
	if !cfg.Enabled {
		return 0
	}
	return cfg.LoginIPPerMinute
}
