// Package service 实现论坛的业务规则与事务边界。
package service

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/session_util"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/common/str_util"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dto"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/perms"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// dummyPasswordHash 在用户不存在时参与一次等价的 bcrypt 比较，减少用户名枚举的时序差异。
var dummyPasswordHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

// Authenticate 校验共享账号的用户名与密码。
// 用户不存在与密码错误返回同一个错误，避免账号枚举。
func Authenticate(username, password string) (*models.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, h.HttpError(http.StatusUnauthorized, "账号或密码错误")
	}

	var user models.User
	if err := store.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
			return nil, h.HttpError(http.StatusUnauthorized, "账号或密码错误")
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, h.HttpError(http.StatusUnauthorized, "账号或密码错误")
	}

	if user.Banned {
		return nil, h.HttpError(http.StatusForbidden, "账号已被封禁")
	}

	return &user, nil
}

// PrepareForumUser 确保论坛资料存在，并按首次登录规则发放默认权限节点。
// 即使管理员撤销了全部授权，也不会在下次登录时自动补回。
func PrepareForumUser(userID uint) error {
	changed := false

	err := store.DB.Transaction(func(tx *gorm.DB) error {
		// 锁定用户行，避免同一用户并发首次登录重复创建资料/授权。
		var user models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return err
		}

		profile, _, err := dao.EnsureProfile(tx, userID)
		if err != nil {
			return err
		}

		if profile.PermissionsInitialized {
			return nil
		}

		var grantCount int64
		if err := tx.Model(&forummodels.UserPermission{}).Where("user_id = ?", userID).Count(&grantCount).Error; err != nil {
			return err
		}

		if grantCount == 0 {
			for _, code := range perms.DefaultCodes() {
				row := forummodels.UserPermission{
					UserID:         userID,
					PermissionCode: code,
					GrantedBy:      0,
				}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
			}
			changed = true
		}

		if err := tx.Model(&forummodels.Profile{}).
			Where("id = ?", profile.ID).
			Update("permissions_initialized", true).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	if changed {
		if err := perms.Reload(); err != nil {
			return err
		}
	}

	return nil
}

// RecordLogin 写入登录记录。
func RecordLogin(userID uint, channel forummodels.LoginChannel, c *gin.Context) error {
	userAgent := str_util.Truncate(c.Request.UserAgent(), 255)
	record := forummodels.LoginRecord{
		UserID:    userID,
		Channel:   channel,
		IP:        c.ClientIP(),
		UserAgent: userAgent,
	}
	return store.DB.Create(&record).Error
}

// LoginResult 组装登录成功响应。
func LoginResult(user *models.User) dto.LoginResult {
	views := dao.LoadUserViews([]uint{user.ID})
	view, ok := views[user.ID]
	if !ok {
		view = dao.UserView{ID: user.ID, Username: user.Username}
	}
	global, boards := perms.PermissionsOf(user)
	return dto.LoginResult{
		User:              dto.SummaryFromView(view),
		GlobalPermissions: global,
		BoardPermissions:  boards,
	}
}

// login 完成论坛/后台登录共用的用户侧副作用。
func login(username, password string, channel forummodels.LoginChannel, requireAdmin bool, c *gin.Context) (*models.User, dto.LoginResult, error) {
	user, err := Authenticate(username, password)
	if err != nil {
		return nil, dto.LoginResult{}, err
	}

	if requireAdmin && !perms.CanAccessAdmin(user) {
		return nil, dto.LoginResult{}, h.HttpError(http.StatusForbidden, "无管理权限")
	}

	if err := PrepareForumUser(user.ID); err != nil {
		return nil, dto.LoginResult{}, err
	}

	if err := RecordLogin(user.ID, channel, c); err != nil {
		return nil, dto.LoginResult{}, err
	}

	return user, LoginResult(user), nil
}

// ForumLogin 完成论坛登录的用户侧副作用。
func ForumLogin(username, password string, c *gin.Context) (*models.User, dto.LoginResult, error) {
	return login(username, password, forummodels.LoginChannelForum, false, c)
}

// AdminLogin 完成后台登录的用户侧副作用。
func AdminLogin(username, password string, c *gin.Context) (*models.User, dto.LoginResult, error) {
	return login(username, password, forummodels.LoginChannelAdmin, true, c)
}

// SetSession 把登录态写入指定 session。
func SetSession(set func(key string, value any), user *models.User, remember bool, rememberLifetime, defaultLifetime time.Duration) time.Duration {
	set(session_util.KeyUserId, user.ID)
	set(session_util.KeyUsername, user.Username)

	lifetime := defaultLifetime
	if remember {
		lifetime = rememberLifetime
	}
	return lifetime
}
