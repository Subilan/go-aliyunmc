package dao

import (
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
)

// UploadByID 读取上传记录。
func UploadByID(id uint, includeDeleted bool) (*forummodels.Upload, error) {
	db := store.DB
	if includeDeleted {
		db = db.Unscoped()
	}
	var upload forummodels.Upload
	if err := db.First(&upload, id).Error; err != nil {
		return nil, err
	}
	return &upload, nil
}

// UploadBytesByOwner 返回用户未删除上传记录占用的总字节数。
func UploadBytesByOwner(ownerID uint) (int64, error) {
	var total int64
	err := store.DB.Model(&forummodels.Upload{}).
		Where("owner_id = ?", ownerID).
		Select("COALESCE(SUM(size_bytes), 0)").
		Scan(&total).Error
	return total, err
}

// UploadByKey 按 key 读取上传记录。
func UploadByKey(key string, includeDeleted bool) (*forummodels.Upload, error) {
	db := store.DB
	if includeDeleted {
		db = db.Unscoped()
	}
	var upload forummodels.Upload
	if err := db.Where("`key` = ?", key).First(&upload).Error; err != nil {
		return nil, err
	}
	return &upload, nil
}
