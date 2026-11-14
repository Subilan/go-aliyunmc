package store

import (
	"time"

	"github.com/Subilan/go-aliyunmc/internal/store/models"
)

var userSortableFields = map[string]bool{
	"id":            true,
	"username":      true,
	"created_at":    true,
	"updated_at":    true,
	"last_login_at": true,
}

// UserWithLastLogin 是用户列表项。LastLoginAt 不是 users 表的列，而是从
// login_records 联表派生的最近一次登录时间；nil 表示从未登录。
type UserWithLastLogin struct {
	models.User
	LastLoginAt *time.Time `json:"last_login_at"`
}

// ListUsers 获取用户列表，支持按封禁状态和用户名筛选。
// banned 为 nil 时不筛选封禁状态；username 为空时不筛选用户名。
func ListUsers(banned *bool, username string, sort, order string, limit, offset int) ([]UserWithLastLogin, error) {
	var users []UserWithLastLogin
	query := DB.Model(&models.User{})

	if banned != nil {
		query = query.Where("banned = ?", *banned)
	}
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}

	latestLogins := DB.Model(&models.LoginRecord{}).
		Select("user_id, MAX(created_at) AS last_login_at").
		Group("user_id")
	query = query.
		Select("users.*, latest_logins.last_login_at").
		Joins("LEFT JOIN (?) AS latest_logins ON latest_logins.user_id = users.id", latestLogins)

	if userSortableFields[sort] && (order == "asc" || order == "desc") {
		query = query.Order(sort + " " + order)
	} else {
		query = query.Order("id DESC")
	}

	result := query.Limit(limit).Offset(offset).Find(&users)
	return users, result.Error
}

// CountUsers 返回用户总数，支持按封禁状态和用户名筛选。
func CountUsers(banned *bool, username string) (int64, error) {
	var count int64
	query := DB.Model(&models.User{})
	if banned != nil {
		query = query.Where("banned = ?", *banned)
	}
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	result := query.Count(&count)
	return count, result.Error
}
