// Package dao 提供论坛数据的查询与写入。路由与 service 共享这里的批量查询，
// 避免列表接口出现 N+1。
package dao

import (
	"errors"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"

	"gorm.io/gorm"
)

// maxPage 防止超大页码导致 Offset 溢出或产生极端 OFFSET 查询。
const maxPage = 100000

// Pagination 规范化分页参数。
func Pagination(page, pageSize, defaultSize, maxSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if page > maxPage {
		page = maxPage
	}
	if pageSize <= 0 {
		pageSize = defaultSize
	}
	if maxSize > 0 && pageSize > maxSize {
		pageSize = maxSize
	}
	return page, pageSize
}

// Offset 返回 SQL offset。
func Offset(page, pageSize int) int {
	return (page - 1) * pageSize
}

// UserView 是一个用户 + 论坛资料的只读视图。
type UserView struct {
	ID            uint
	Username      string
	Nickname      *string
	Bio           string
	AvatarShape   string
	AvatarKey     string
	BackgroundKey string
	JoinedAt      time.Time
	Deleted       bool
}

// LoadUserViews 批量加载用户与论坛资料及头像/背景上传。
func LoadUserViews(userIDs []uint) map[uint]UserView {
	views := make(map[uint]UserView, len(userIDs))
	if len(userIDs) == 0 {
		return views
	}

	var users []models.User
	if err := store.DB.Unscoped().Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return views
	}

	var profiles []forummodels.Profile
	if err := store.DB.Where("user_id IN ?", userIDs).Find(&profiles).Error; err != nil {
		return views
	}

	profileByUser := make(map[uint]forummodels.Profile, len(profiles))
	uploadIDs := make([]uint, 0, len(profiles))
	for _, p := range profiles {
		profileByUser[p.UserID] = p
		if p.AvatarUploadID != nil {
			uploadIDs = append(uploadIDs, *p.AvatarUploadID)
		}
		if p.BackgroundUploadID != nil {
			uploadIDs = append(uploadIDs, *p.BackgroundUploadID)
		}
	}

	uploadKeys := make(map[uint]string, len(uploadIDs))
	if len(uploadIDs) > 0 {
		var uploads []forummodels.Upload
		if err := store.DB.Where("id IN ?", uploadIDs).Find(&uploads).Error; err == nil {
			for _, u := range uploads {
				uploadKeys[u.ID] = u.Key
			}
		}
	}

	for _, user := range users {
		view := UserView{
			ID:          user.ID,
			Username:    user.Username,
			AvatarShape: string(forummodels.AvatarShapeCircle),
			JoinedAt:    user.CreatedAt,
			Deleted:     user.DeletedAt.Valid,
		}
		if p, ok := profileByUser[user.ID]; ok {
			view.Nickname = p.Nickname
			view.Bio = p.Bio
			view.AvatarShape = string(p.AvatarShape)
			if p.AvatarUploadID != nil {
				view.AvatarKey = uploadKeys[*p.AvatarUploadID]
			}
			if p.BackgroundUploadID != nil {
				view.BackgroundKey = uploadKeys[*p.BackgroundUploadID]
			}
		}
		views[user.ID] = view
	}

	return views
}

// ForumLoginExistsSQL 是 users 表上的 EXISTS 条件：至少成功登录过一次论坛前台。
// 论坛前台 session 只能由 /forum-v1/auth/login 签发，后台登录（channel=admin）
// 不产生该记录，因此它等价于用户确实进入过论坛。
const ForumLoginExistsSQL = "EXISTS (SELECT 1 FROM forum_login_records flr WHERE flr.user_id = users.id AND flr.channel = ?)"

// HasForumLogin 判断用户是否至少成功登录过一次论坛前台。
func HasForumLogin(userID uint) (bool, error) {
	var count int64
	err := store.DB.Model(&forummodels.LoginRecord{}).
		Where("user_id = ? AND channel = ?", userID, forummodels.LoginChannelForum).
		Count(&count).Error
	return count > 0, err
}

// EnsureProfile 确保用户存在论坛资料，返回资料与是否新建。
func EnsureProfile(tx *gorm.DB, userID uint) (*forummodels.Profile, bool, error) {
	var profile forummodels.Profile
	err := tx.Where("user_id = ?", userID).First(&profile).Error
	if err == nil {
		return &profile, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	profile = forummodels.Profile{
		UserID:      userID,
		AvatarShape: forummodels.AvatarShapeCircle,
	}
	if err := tx.Create(&profile).Error; err != nil {
		return nil, false, err
	}
	return &profile, true, nil
}

// UserCounts 是用户的主题、回复与获赞计数。
type UserCounts struct {
	Threads int64
	Replies int64
	Likes   int64
}

// UserStatsBatch 批量返回用户计数，避免列表接口逐用户查询。
func UserStatsBatch(userIDs []uint) map[uint]UserCounts {
	stats := make(map[uint]UserCounts, len(userIDs))
	if len(userIDs) == 0 {
		return stats
	}
	for _, id := range userIDs {
		stats[id] = UserCounts{}
	}

	type countRow struct {
		UserID uint
		Count  int64
	}

	var rows []countRow
	store.DB.Model(&forummodels.Thread{}).
		Select("author_id AS user_id, COUNT(*) AS count").
		Where("author_id IN ?", userIDs).
		Group("author_id").
		Scan(&rows)
	for _, row := range rows {
		value := stats[row.UserID]
		value.Threads = row.Count
		stats[row.UserID] = value
	}

	rows = nil
	store.DB.Model(&forummodels.Reply{}).
		Select("author_id AS user_id, COUNT(*) AS count").
		Where("author_id IN ?", userIDs).
		Group("author_id").
		Scan(&rows)
	for _, row := range rows {
		value := stats[row.UserID]
		value.Replies = row.Count
		stats[row.UserID] = value
	}

	rows = nil
	store.DB.Model(&forummodels.ThreadLike{}).
		Select("forum_threads.author_id AS user_id, COUNT(*) AS count").
		Joins("JOIN forum_threads ON forum_threads.id = forum_thread_likes.thread_id AND forum_threads.deleted_at IS NULL").
		Where("forum_threads.author_id IN ?", userIDs).
		Group("forum_threads.author_id").
		Scan(&rows)
	for _, row := range rows {
		value := stats[row.UserID]
		value.Likes += row.Count
		stats[row.UserID] = value
	}

	rows = nil
	store.DB.Model(&forummodels.ReplyLike{}).
		Select("forum_replies.author_id AS user_id, COUNT(*) AS count").
		Joins("JOIN forum_replies ON forum_replies.id = forum_reply_likes.reply_id AND forum_replies.deleted_at IS NULL").
		Where("forum_replies.author_id IN ?", userIDs).
		Group("forum_replies.author_id").
		Scan(&rows)
	for _, row := range rows {
		value := stats[row.UserID]
		value.Likes += row.Count
		stats[row.UserID] = value
	}

	return stats
}

// LoginTimes 是用户最近一次论坛/后台登录时间。
type LoginTimes struct {
	Forum *time.Time
	Admin *time.Time
}

// LastLoginTimes 批量返回用户最近一次论坛/后台登录时间。
func LastLoginTimes(userIDs []uint) map[uint]LoginTimes {
	result := make(map[uint]LoginTimes, len(userIDs))
	if len(userIDs) == 0 {
		return result
	}

	type loginRow struct {
		UserID    uint
		Channel   forummodels.LoginChannel
		CreatedAt time.Time
	}
	var rows []loginRow
	store.DB.Model(&forummodels.LoginRecord{}).
		Select("user_id, channel, MAX(created_at) AS created_at").
		Where("user_id IN ?", userIDs).
		Group("user_id, channel").
		Scan(&rows)

	for _, row := range rows {
		value := result[row.UserID]
		createdAt := row.CreatedAt
		switch row.Channel {
		case forummodels.LoginChannelForum:
			value.Forum = &createdAt
		case forummodels.LoginChannelAdmin:
			value.Admin = &createdAt
		}
		result[row.UserID] = value
	}

	return result
}

// UserLikeCount 返回用户主页收到的点赞数。
func UserLikeCount(userID uint) int64 {
	var count int64
	store.DB.Model(&forummodels.UserLike{}).Where("target_user_id = ?", userID).Count(&count)
	return count
}

// HasUserLike 判断 liker 是否点赞了目标用户。
func HasUserLike(targetUserID, likerID uint) bool {
	if likerID == 0 {
		return false
	}
	var count int64
	store.DB.Model(&forummodels.UserLike{}).
		Where("target_user_id = ? AND liker_id = ?", targetUserID, likerID).
		Count(&count)
	return count > 0
}

// ValidMentionUsers 返回可用于 @ 提及的用户 ID → username。
// 只保留存在、未删除、未封禁且有论坛资料的用户。
func ValidMentionUsers(ids []uint) map[uint]string {
	result := make(map[uint]string, len(ids))
	if len(ids) == 0 {
		return result
	}

	var users []models.User
	store.DB.
		Joins("JOIN forum_profiles ON forum_profiles.user_id = users.id AND forum_profiles.deleted_at IS NULL").
		Where("users.id IN ? AND users.deleted_at IS NULL AND users.banned = ?", ids, false).
		Find(&users)

	for _, user := range users {
		result[user.ID] = user.Username
	}
	return result
}
