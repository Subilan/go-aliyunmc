package store

import "github.com/Subilan/go-aliyunmc/store/models"

var userSortableFields = map[string]bool{
	"id":         true,
	"username":   true,
	"created_at": true,
	"updated_at": true,
}

// ListUsers 获取用户列表，支持按封禁状态和用户名筛选。
// banned 为 nil 时不筛选封禁状态；username 为空时不筛选用户名。
func ListUsers(banned *bool, username string, sort, order string, limit, offset int) ([]models.User, error) {
	var users []models.User
	query := DB.Model(&models.User{})

	if banned != nil {
		query = query.Where("banned = ?", *banned)
	}
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}

	if userSortableFields[sort] && (order == "asc" || order == "desc") {
		query = query.Order(sort + " " + order)
	} else {
		query = query.Order("id DESC")
	}

	result := query.Omit("password_hash").Limit(limit).Offset(offset).Find(&users)
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
