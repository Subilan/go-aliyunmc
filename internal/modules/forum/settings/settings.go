// Package settings 提供论坛站点设置的默认值、读取与更新。
package settings

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"

	"github.com/Subilan/go-aliyunmc/internal/common/store"

	"gorm.io/gorm"
)

// 站点设置键。
const (
	KeySiteName           = "site_name"
	KeySiteDescription    = "site_description"
	KeyAnnouncement       = "announcement"
	KeyFooterText         = "footer_text"
	KeyAllowGuestRead     = "allow_guest_read"
	KeyDefaultAvatar      = "default_avatar_url_template"
	KeyRegistrationEnable = "registration_enabled"
)

var knownKeys = map[string]bool{
	KeySiteName:           true,
	KeySiteDescription:    true,
	KeyAnnouncement:       true,
	KeyFooterText:         true,
	KeyAllowGuestRead:     true,
	KeyDefaultAvatar:      true,
	KeyRegistrationEnable: true,
}

// Values 是完整的站点设置。
type Values struct {
	SiteName        string
	SiteDescription string
	Announcement    string
	FooterText      string
	AllowGuestRead  bool

	DefaultAvatarURLTemplate string
	RegistrationEnabled      bool
}

var defaults = Values{
	SiteName:                 "Tide",
	SiteDescription:          "一个简单的论坛。",
	Announcement:             "",
	FooterText:               "",
	AllowGuestRead:           true,
	DefaultAvatarURLTemplate: "https://minotar.net/helm/{username}/200.png",
	RegistrationEnabled:      false,
}

// 设置读取频率高，用进程内缓存减少查询；写入后失效。
var (
	mu    sync.RWMutex
	cache *Values
)

// Invalidate 清除缓存，下次读取时重新加载。
func Invalidate() {
	mu.Lock()
	cache = nil
	mu.Unlock()
}

// Get 返回当前设置（含默认值）。
// 冷加载持有写锁，避免与 Invalidate 竞态时把旧值重新写回缓存。
func Get() Values {
	mu.RLock()
	if cache != nil {
		values := *cache
		mu.RUnlock()
		return values
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()
	if cache != nil {
		return *cache
	}

	values := load()
	cache = &values
	return values
}

func load() Values {
	values := defaults

	var rows []forummodels.Setting
	if err := store.DB.Find(&rows).Error; err != nil {
		return values
	}

	for _, row := range rows {
		switch row.Key {
		case KeySiteName:
			values.SiteName = row.Value
		case KeySiteDescription:
			values.SiteDescription = row.Value
		case KeyAnnouncement:
			values.Announcement = row.Value
		case KeyFooterText:
			values.FooterText = row.Value
		case KeyAllowGuestRead:
			if parsed, err := strconv.ParseBool(row.Value); err == nil {
				values.AllowGuestRead = parsed
			}
		case KeyDefaultAvatar:
			if strings.TrimSpace(row.Value) != "" {
				values.DefaultAvatarURLTemplate = row.Value
			}
		case KeyRegistrationEnable:
			if parsed, err := strconv.ParseBool(row.Value); err == nil {
				values.RegistrationEnabled = parsed
			}
		}
	}

	return values
}

// AllowGuestRead 是读取站点设置的最常用入口。
func AllowGuestRead() bool {
	return Get().AllowGuestRead
}

// Known 判断 key 是否在代码登记的键中。
func Known(key string) bool {
	return knownKeys[key]
}

// Update 部分更新设置；未知键返回错误。
func Update(updates map[string]string, updatedBy uint) error {
	if err := Validate(updates); err != nil {
		return err
	}

	err := store.DB.Transaction(func(tx *gorm.DB) error {
		for key, value := range updates {
			row := forummodels.Setting{Key: key, Value: value, UpdatedBy: updatedBy}
			if err := tx.Save(&row).Error; err != nil {
				return fmt.Errorf("保存设置 %s 失败: %w", key, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	Invalidate()
	return nil
}

// ValidationError 表示站点设置值不合法。
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Validate 校验未知键与布尔值。
func Validate(updates map[string]string) error {
	for key, value := range updates {
		if !Known(key) {
			return &ValidationError{Message: fmt.Sprintf("未知的设置键: %s", key)}
		}
		if key == KeyAllowGuestRead || key == KeyRegistrationEnable {
			if _, err := strconv.ParseBool(value); err != nil {
				return &ValidationError{Message: fmt.Sprintf("设置 %s 必须是布尔值", key)}
			}
		}
	}
	return nil
}

// DefaultAvatarTemplate 返回默认头像模板。
func DefaultAvatarTemplate() string {
	return Get().DefaultAvatarURLTemplate
}
