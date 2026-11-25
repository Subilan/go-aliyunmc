package service

import (
	"errors"
	"net/http"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/log_util"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/common/str_util"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dto"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/uploads"
)

// CreateUpload 校验并保存图片，写入元数据；数据库失败时删除磁盘文件。
func CreateUpload(user *models.User, kindValue, originalName string, data []byte) (*dto.UploadResult, error) {
	kind, ok := uploads.ParseKind(kindValue)
	if !ok {
		return nil, h.HttpError(http.StatusBadRequest, "kind 只能是 post、avatar 或 background")
	}

	// 单账号磁盘配额，避免一个账号无限写满上传目录。
	if quota := config.Get().Forum.UserQuotaBytes; quota > 0 {
		used, err := dao.UploadBytesByOwner(user.ID)
		if err != nil {
			return nil, err
		}
		if used+int64(len(data)) > quota {
			return nil, h.HttpError(http.StatusRequestEntityTooLarge, "上传空间不足")
		}
	}

	// 全站磁盘配额，避免整个论坛写满服务器磁盘。
	if quota := config.Get().Forum.StorageQuotaBytes; quota > 0 {
		used, err := dao.UploadBytesTotal()
		if err != nil {
			return nil, err
		}
		if used+int64(len(data)) > quota {
			return nil, h.HttpError(http.StatusRequestEntityTooLarge, "上传空间不足")
		}
	}

	saved, err := uploads.Save(data, kind)
	if err != nil {
		switch {
		case errors.Is(err, uploads.ErrTooLarge):
			return nil, h.HttpError(http.StatusRequestEntityTooLarge, err.Error())
		case errors.Is(err, uploads.ErrUnsupportedType):
			return nil, h.HttpError(http.StatusUnsupportedMediaType, err.Error())
		case errors.Is(err, uploads.ErrInvalidImage):
			return nil, h.HttpError(http.StatusBadRequest, "图片校验失败: "+err.Error())
		default:
			// 编码/落盘等内部错误不应回显文件系统细节。
			log_util.Error("保存论坛上传文件失败: userId=%d kind=%s error=%v", user.ID, kind, err)
			return nil, h.HttpError(http.StatusInternalServerError, "图片保存失败")
		}
	}

	originalName = str_util.Truncate(originalName, 255)

	upload := &forummodels.Upload{
		OwnerID:      user.ID,
		Kind:         kind,
		Key:          saved.Key,
		OriginalName: originalName,
		Mime:         saved.Mime,
		SizeBytes:    saved.SizeBytes,
		Width:        saved.Width,
		Height:       saved.Height,
		SHA256:       saved.SHA256,
	}

	if err := store.DB.Create(upload).Error; err != nil {
		_ = uploads.Delete(saved.Key)
		return nil, err
	}

	return &dto.UploadResult{
		ID:     upload.ID,
		URL:    uploads.URL(upload.Key),
		Width:  upload.Width,
		Height: upload.Height,
		Mime:   upload.Mime,
		Size:   upload.SizeBytes,
	}, nil
}
