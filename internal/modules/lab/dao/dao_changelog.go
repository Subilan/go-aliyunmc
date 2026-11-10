package dao

import (
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/models"
)

// ChangelogItem 是查询结果中每条 changelog 的表示，包含点赞信息
type ChangelogItem struct {
	ID        uint              `json:"id"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Category  labmodels.LogType `json:"category"`
	LikeCount int64             `json:"like_count"`
	Liked     bool              `json:"liked"`
}

// CreateChangelog 创建一条新的 changelog
func CreateChangelog(title, body string, category labmodels.LogType) (*labmodels.Changelog, error) {
	c := &labmodels.Changelog{
		Title:    title,
		Body:     body,
		Category: category,
	}
	return c, store.DB.Create(c).Error
}

// QueryChangelogs 分页查询 changelog，支持排序和分类过滤，返回带点赞信息的结果
func QueryChangelogs(sortBy string, page, pageSize int, category labmodels.LogType, userID uint) ([]ChangelogItem, int64, error) {
	var items []labmodels.Changelog
	var total int64

	query := store.DB.Model(&labmodels.Changelog{})
	if category != "" {
		query = query.Where("category = ?", category)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := "created_at DESC"
	if sortBy == "asc" {
		order = "created_at ASC"
	}

	offset := (page - 1) * pageSize
	if err := query.Order(order).Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return buildChangelogItems(items, userID), total, nil
}

// ToggleLike 切换用户对某条 changelog 的点赞状态，返回是否已点赞和最新点赞数
func ToggleLike(changelogID, userID uint) (bool, int64, error) {
	var cl labmodels.Changelog
	if err := store.DB.First(&cl, changelogID).Error; err != nil {
		return false, 0, err
	}

	var user models.User
	if err := store.DB.First(&user, userID).Error; err != nil {
		return false, 0, err
	}

	var existing int64
	store.DB.Table("changelog_likes").Where("changelog_id = ? AND user_id = ?", changelogID, userID).Count(&existing)

	if existing > 0 {
		if err := store.DB.Model(&cl).Association("LikedBy").Delete(&user); err != nil {
			return false, 0, err
		}
	} else {
		if err := store.DB.Model(&cl).Association("LikedBy").Append(&user); err != nil {
			return false, 0, err
		}
	}

	var newCount int64
	store.DB.Table("changelog_likes").Where("changelog_id = ?", changelogID).Count(&newCount)

	return existing == 0, newCount, nil
}

// UpdateChangelog 更新一条 changelog 的指定字段
func UpdateChangelog(id uint, updates map[string]any) error {
	return store.DB.Model(&labmodels.Changelog{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteChangelog 软删除一条 changelog
func DeleteChangelog(id uint) error {
	return store.DB.Delete(&labmodels.Changelog{}, id).Error
}

func buildChangelogItems(items []labmodels.Changelog, userID uint) []ChangelogItem {
	if len(items) == 0 {
		return nil
	}

	ids := make([]uint, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}

	type likeCountRow struct {
		ChangelogID uint
		Count       int64
	}
	var counts []likeCountRow
	store.DB.Table("changelog_likes").Select("changelog_id, count(*) as count").
		Where("changelog_id IN ?", ids).Group("changelog_id").Find(&counts)

	countMap := make(map[uint]int64, len(counts))
	for _, c := range counts {
		countMap[c.ChangelogID] = c.Count
	}

	var likedIDs []uint
	if userID > 0 {
		store.DB.Table("changelog_likes").Select("changelog_id").
			Where("changelog_id IN ? AND user_id = ?", ids, userID).Find(&likedIDs)
	}
	likedSet := make(map[uint]bool, len(likedIDs))
	for _, id := range likedIDs {
		likedSet[id] = true
	}

	result := make([]ChangelogItem, len(items))
	for i, item := range items {
		result[i] = ChangelogItem{
			ID:        item.ID,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
			Title:     item.Title,
			Body:      item.Body,
			Category:  item.Category,
			LikeCount: countMap[item.ID],
			Liked:     likedSet[item.ID],
		}
	}
	return result
}
