package service

import (
	"net/http"
	"strings"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dto"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
)

// ProfileUpdate 描述一次资料修改。Set 字段区分“未提供”与“显式置空”。
type ProfileUpdate struct {
	NicknameSet bool
	Nickname    *string

	BioSet bool
	Bio    string
}

// UpdateProfile 修改昵称与简介。
func UpdateProfile(user *models.User, update ProfileUpdate) error {
	profile, _, err := dao.EnsureProfile(store.DB, user.ID)
	if err != nil {
		return err
	}

	updates := map[string]any{}

	if update.NicknameSet {
		nickname, err := normalizeNickname(update.Nickname)
		if err != nil {
			return err
		}
		updates["nickname"] = nickname
	}

	if update.BioSet {
		bio, err := normalizeBio(update.Bio)
		if err != nil {
			return err
		}
		updates["bio"] = bio
	}

	if len(updates) == 0 {
		return nil
	}

	return store.DB.Model(&forummodels.Profile{}).Where("id = ?", profile.ID).Updates(updates).Error
}

func normalizeNickname(nickname *string) (*string, error) {
	if nickname == nil {
		return nil, nil
	}

	value := strings.TrimSpace(*nickname)
	if value == "" {
		return nil, nil
	}

	count := 0
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return nil, h.HttpError(http.StatusBadRequest, "昵称不能包含控制字符")
		}
		count++
	}
	if count > 32 {
		return nil, h.HttpError(http.StatusBadRequest, "昵称不能超过 32 个字符")
	}

	return &value, nil
}

func normalizeBio(bio string) (string, error) {
	bio = strings.TrimSpace(bio)

	count := 0
	for _, r := range bio {
		if r < 0x20 || r == 0x7f {
			return "", h.HttpError(http.StatusBadRequest, "简介不能包含控制字符")
		}
		count++
	}
	if count > 200 {
		return "", h.HttpError(http.StatusBadRequest, "简介不能超过 200 个字符")
	}

	return bio, nil
}

// BindAvatar 绑定或清除头像。
func BindAvatar(user *models.User, uploadID *uint) error {
	return bindProfileUpload(user, uploadID, forummodels.UploadKindAvatar, "avatar_upload_id")
}

// BindBackground 绑定或清除主页背景。
func BindBackground(user *models.User, uploadID *uint) error {
	return bindProfileUpload(user, uploadID, forummodels.UploadKindBackground, "background_upload_id")
}

func bindProfileUpload(user *models.User, uploadID *uint, kind forummodels.UploadKind, column string) error {
	profile, _, err := dao.EnsureProfile(store.DB, user.ID)
	if err != nil {
		return err
	}

	if uploadID != nil {
		upload, err := dao.UploadByID(*uploadID, false)
		if err != nil {
			return h.HttpError(http.StatusNotFound, "上传文件不存在")
		}
		if upload.OwnerID != user.ID {
			return h.HttpError(http.StatusForbidden, "不能使用他人的上传文件")
		}
		if upload.Kind != kind {
			return h.HttpError(http.StatusBadRequest, "上传文件用途不匹配")
		}
	}

	return store.DB.Model(&forummodels.Profile{}).
		Where("id = ?", profile.ID).
		Update(column, uploadID).Error
}

// SetAvatarShape 设置头像形状。
func SetAvatarShape(user *models.User, shape string) error {
	if shape != string(forummodels.AvatarShapeCircle) && shape != string(forummodels.AvatarShapeSquare) {
		return h.HttpError(http.StatusBadRequest, "头像形状只能是 circle 或 square")
	}

	profile, _, err := dao.EnsureProfile(store.DB, user.ID)
	if err != nil {
		return err
	}

	return store.DB.Model(&forummodels.Profile{}).
		Where("id = ?", profile.ID).
		Update("avatar_shape", shape).Error
}

// UserProfile 返回用户主页资料。
func UserProfile(targetUserID uint, viewer *models.User) (*dto.UserProfile, error) {
	views := dao.LoadUserViews([]uint{targetUserID})
	view, ok := views[targetUserID]
	if !ok || view.Deleted {
		return nil, h.HttpError(http.StatusNotFound, "用户不存在")
	}

	counts := dao.UserStatsBatch([]uint{targetUserID})[targetUserID]
	profileLikes := dao.UserLikeCount(targetUserID)
	likedByMe := false
	isSelf := false
	if viewer != nil {
		likedByMe = dao.HasUserLike(targetUserID, viewer.ID)
		isSelf = viewer.ID == targetUserID
	}

	profile := dto.ProfileFrom(view, counts.Threads, counts.Replies, counts.Likes, profileLikes, likedByMe, isSelf)
	return &profile, nil
}
