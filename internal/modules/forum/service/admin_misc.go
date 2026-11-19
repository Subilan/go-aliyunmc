package service

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/log_util"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dto"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/settings"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/uploads"

	"github.com/gin-gonic/gin"
)

// AdminGetSettings 返回后台设置（含默认值）。
func AdminGetSettings() dto.SiteSettings {
	values := settings.Get()

	raw := map[string]string{
		settings.KeySiteName:           values.SiteName,
		settings.KeySiteDescription:    values.SiteDescription,
		settings.KeyAnnouncement:       values.Announcement,
		settings.KeyFooterText:         values.FooterText,
		settings.KeyAllowGuestRead:     strconv.FormatBool(values.AllowGuestRead),
		settings.KeyDefaultAvatar:      values.DefaultAvatarURLTemplate,
		settings.KeyRegistrationEnable: strconv.FormatBool(values.RegistrationEnabled),
	}

	return dto.SiteSettings{
		SiteName:                 values.SiteName,
		SiteDescription:          values.SiteDescription,
		Announcement:             values.Announcement,
		FooterText:               values.FooterText,
		AllowGuestRead:           values.AllowGuestRead,
		DefaultAvatarURLTemplate: values.DefaultAvatarURLTemplate,
		RegistrationEnabled:      values.RegistrationEnabled,
		Raw:                      raw,
	}
}

// AdminUpdateSettings 部分更新站点设置。
func AdminUpdateSettings(actor *models.User, updates map[string]string, c *gin.Context) (dto.SiteSettings, error) {
	if len(updates) == 0 {
		return AdminGetSettings(), nil
	}

	// 校验错误属于客户端问题；数据库错误应返回 500，避免把内部细节当作 400 回显。
	if err := settings.Update(updates, actor.ID); err != nil {
		var validation *settings.ValidationError
		if errors.As(err, &validation) {
			return dto.SiteSettings{}, h.HttpError(http.StatusBadRequest, validation.Error())
		}
		log_util.Error("保存论坛站点设置失败: actor=%d error=%v", actor.ID, err)
		return dto.SiteSettings{}, h.HttpError(http.StatusInternalServerError, "保存设置失败")
	}

	dao.Audit(c, actor.ID, "setting.update", "setting", 0, updates)
	return AdminGetSettings(), nil
}

// AdminListUploads 返回上传文件列表。
func AdminListUploads(kind string, ownerID uint, page, pageSize int) (dto.Page[dto.AdminUpload], error) {
	db := store.DB.Model(&forummodels.Upload{})
	if kind != "" {
		db = db.Where("kind = ?", kind)
	}
	if ownerID != 0 {
		db = db.Where("owner_id = ?", ownerID)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return dto.Page[dto.AdminUpload]{}, err
	}

	page, pageSize = dao.Pagination(page, pageSize, 20, 100)

	var rows []forummodels.Upload
	if err := db.Order("created_at DESC, id DESC").
		Offset(dao.Offset(page, pageSize)).
		Limit(pageSize).
		Find(&rows).Error; err != nil {
		return dto.Page[dto.AdminUpload]{}, err
	}

	ownerIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		ownerIDs = append(ownerIDs, row.OwnerID)
	}
	views := dao.LoadUserViews(uniqueIDs(ownerIDs))

	items := make([]dto.AdminUpload, 0, len(rows))
	for _, row := range rows {
		ownerName := ""
		if view, ok := views[row.OwnerID]; ok {
			ownerName = view.Username
		}
		items = append(items, dto.AdminUpload{
			ID:           row.ID,
			OwnerID:      row.OwnerID,
			OwnerName:    ownerName,
			Kind:         string(row.Kind),
			Key:          row.Key,
			URL:          uploads.URL(row.Key),
			OriginalName: row.OriginalName,
			Mime:         row.Mime,
			SizeBytes:    row.SizeBytes,
			Width:        row.Width,
			Height:       row.Height,
			SHA256:       row.SHA256,
			CreatedAt:    dto.Timestamp(row.CreatedAt),
		})
	}

	return dto.NewPage(items, total, page, pageSize), nil
}

// AdminUploadStats 返回按用途统计的上传数量与字节数。
func AdminUploadStats() ([]dto.UploadStats, error) {
	var rows []struct {
		Kind      string
		Count     int64
		SizeBytes int64
	}
	err := store.DB.Model(&forummodels.Upload{}).
		Select("kind, COUNT(*) AS count, COALESCE(SUM(size_bytes), 0) AS size_bytes").
		Group("kind").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	items := make([]dto.UploadStats, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.UploadStats{Kind: row.Kind, Count: row.Count, SizeBytes: row.SizeBytes})
	}
	return items, nil
}

// AdminDeleteUpload 删除未被引用的上传文件。
func AdminDeleteUpload(actor *models.User, uploadID uint, c *gin.Context) error {
	upload, err := dao.UploadByID(uploadID, false)
	if err != nil {
		return h.HttpError(http.StatusNotFound, "上传文件不存在")
	}

	var profileRefs int64
	if err := store.DB.Model(&forummodels.Profile{}).
		Where("avatar_upload_id = ? OR background_upload_id = ?", upload.ID, upload.ID).
		Count(&profileRefs).Error; err != nil {
		return err
	}
	if profileRefs > 0 {
		return h.HttpError(http.StatusConflict, "上传文件仍被用户资料引用")
	}

	var threadRefs, replyRefs int64
	if err := store.DB.Unscoped().Model(&forummodels.Thread{}).
		Where("INSTR(content_html, ?) > 0", upload.Key).
		Count(&threadRefs).Error; err != nil {
		return err
	}
	if err := store.DB.Unscoped().Model(&forummodels.Reply{}).
		Where("INSTR(content_html, ?) > 0", upload.Key).
		Count(&replyRefs).Error; err != nil {
		return err
	}
	if threadRefs > 0 || replyRefs > 0 {
		return h.HttpError(http.StatusConflict, "上传文件仍被正文引用")
	}

	if err := store.DB.Delete(&forummodels.Upload{}, upload.ID).Error; err != nil {
		return err
	}
	if err := uploads.Delete(upload.Key); err != nil {
		log_util.Warn("删除上传文件失败: uploadId=%d key=%s error=%v", upload.ID, upload.Key, err)
	}

	dao.Audit(c, actor.ID, "upload.delete", "upload", upload.ID, map[string]any{"key": upload.Key})
	return nil
}

// AdminDashboard 返回数据看板。
func AdminDashboard(days int) (*dto.Dashboard, error) {
	if days <= 0 {
		days = 7
	}
	if days > 30 {
		days = 30
	}

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	trendStart := startOfDay.AddDate(0, 0, -(days - 1))

	totalThreads := countRows(&forummodels.Thread{}, "deleted_at IS NULL")
	totalReplies := countRows(&forummodels.Reply{}, "deleted_at IS NULL")
	totalUsers := countRows(&models.User{}, "deleted_at IS NULL")
	totalBoards := countRows(&forummodels.Board{}, "deleted_at IS NULL")

	todayThreads := countSince(&forummodels.Thread{}, startOfDay, "deleted_at IS NULL")
	todayReplies := countSince(&forummodels.Reply{}, startOfDay, "deleted_at IS NULL")
	todayUsers := countSince(&models.User{}, startOfDay, "deleted_at IS NULL")
	todayBoards := countSince(&forummodels.Board{}, startOfDay, "deleted_at IS NULL")

	threadTrend := dailyCounts(&forummodels.Thread{}, trendStart, "deleted_at IS NULL")
	replyTrend := dailyCounts(&forummodels.Reply{}, trendStart, "deleted_at IS NULL")
	userTrend := dailyCounts(&models.User{}, trendStart, "deleted_at IS NULL")

	trend := make([]dto.DashboardTrend, 0, days)
	for i := 0; i < days; i++ {
		day := trendStart.AddDate(0, 0, i).Format("2006-01-02")
		trend = append(trend, dto.DashboardTrend{
			Date:    day,
			Threads: threadTrend[day],
			Replies: replyTrend[day],
			Users:   userTrend[day],
		})
	}

	return &dto.Dashboard{
		Totals: dto.DashboardTotals{
			Threads: totalThreads,
			Replies: totalReplies,
			Users:   totalUsers,
			Boards:  totalBoards,
		},
		Today: dto.DashboardTotals{
			Threads: todayThreads,
			Replies: todayReplies,
			Users:   todayUsers,
			Boards:  todayBoards,
		},
		Trend: trend,
	}, nil
}

func countRows(model any, condition string) int64 {
	var count int64
	db := store.DB.Model(model)
	if condition != "" {
		db = db.Where(condition)
	}
	_ = db.Count(&count).Error
	return count
}

func countSince(model any, since time.Time, condition string) int64 {
	var count int64
	db := store.DB.Model(model).Where("created_at >= ?", since)
	if condition != "" {
		db = db.Where(condition)
	}
	_ = db.Count(&count).Error
	return count
}

func dailyCounts(model any, since time.Time, condition string) map[string]int64 {
	result := make(map[string]int64)

	var rows []struct {
		Day   string
		Count int64
	}

	db := store.DB.Model(model).
		// parseTime=True 时 DATE() 会被驱动解析为 time.Time，再扫描进 string 会得到
		// "2026-11-19 00:00:00 +0000 UTC" 这类值，无法与 "2006-01-02" 对齐。
		// 直接用 DATE_FORMAT 返回字符串，保证趋势表的日期键一致。
		Select("DATE_FORMAT(created_at, '%Y-%m-%d') AS day, COUNT(*) AS count").
		Where("created_at >= ?", since)
	if condition != "" {
		db = db.Where(condition)
	}

	if err := db.Group("DATE_FORMAT(created_at, '%Y-%m-%d')").Scan(&rows).Error; err != nil {
		return result
	}
	for _, row := range rows {
		result[row.Day] = row.Count
	}
	return result
}
