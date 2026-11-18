package service

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dto"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/perms"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	slugRe  = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)
	colorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

	reservedSlugs = map[string]bool{"admin": true, "new": true, "api": true, "files": true}
	allowedIcons  = map[string]bool{"discussion": true, "code": true, "life": true, "feedback": true}
)

// BoardInput 是创建/修改板块的输入。
type BoardInput struct {
	Slug        *string `json:"slug"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	Color       *string `json:"color"`
	SortOrder   *int    `json:"sortOrder"`
	Enabled     *bool   `json:"enabled"`
}

// AdminListBoards 返回管理员可见的板块列表。
func AdminListBoards(user *models.User, includeDisabled, includeDeleted bool, page, pageSize int) (dto.Page[dto.AdminBoard], error) {
	page, pageSize = dao.Pagination(page, pageSize, 20, 100)

	all, boardIDs := perms.BoardScopeFor(user, "admin.access")

	db := store.DB.Model(&forummodels.Board{})
	if includeDeleted {
		db = db.Unscoped()
	}
	if !includeDisabled {
		db = db.Where("enabled = ?", true)
	}
	if !all {
		if len(boardIDs) == 0 {
			return dto.NewPage([]dto.AdminBoard{}, 0, page, pageSize), nil
		}
		db = db.Where("id IN ?", boardIDs)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return dto.Page[dto.AdminBoard]{}, err
	}

	var boards []forummodels.Board
	if err := db.Order("sort_order ASC, id ASC").
		Offset(dao.Offset(page, pageSize)).
		Limit(pageSize).
		Find(&boards).Error; err != nil {
		return dto.Page[dto.AdminBoard]{}, err
	}

	ids := make([]uint, 0, len(boards))
	for _, board := range boards {
		ids = append(ids, board.ID)
	}
	counts := dao.BoardThreadCounts(ids)

	items := make([]dto.AdminBoard, 0, len(boards))
	for _, board := range boards {
		items = append(items, adminBoardFrom(board, counts[board.ID]))
	}

	return dto.NewPage(items, total, page, pageSize), nil
}

// CreateBoard 创建板块。slug 创建后不可修改，软删除后仍占用。
func CreateBoard(user *models.User, input BoardInput, c *gin.Context) (*forummodels.Board, error) {
	if !perms.Allow(user, "*", "board.create") {
		return nil, h.HttpError(http.StatusForbidden, "权限不足")
	}

	if input.Slug == nil || input.Name == nil {
		return nil, h.HttpError(http.StatusBadRequest, "slug 与 name 为必填项")
	}

	slug := strings.TrimSpace(*input.Slug)
	if !slugRe.MatchString(slug) {
		return nil, h.HttpError(http.StatusBadRequest, "slug 只能包含小写字母、数字和连字符，长度 1~64 且不能以连字符开头或结尾")
	}
	if reservedSlugs[slug] {
		return nil, h.HttpError(http.StatusBadRequest, "slug 是保留字")
	}

	name, err := validateBoardName(*input.Name)
	if err != nil {
		return nil, err
	}
	description, err := validateBoardDescription(valueOr(input.Description, ""))
	if err != nil {
		return nil, err
	}
	icon, err := validateIcon(valueOr(input.Icon, "discussion"))
	if err != nil {
		return nil, err
	}
	color, err := validateColor(valueOr(input.Color, "#6366f1"))
	if err != nil {
		return nil, err
	}

	board := &forummodels.Board{
		Slug:        slug,
		Name:        name,
		Description: description,
		Icon:        icon,
		Color:       color,
		Enabled:     true,
	}
	if input.SortOrder != nil {
		board.SortOrder = *input.SortOrder
	}
	if input.Enabled != nil {
		board.Enabled = *input.Enabled
	}

	var existing forummodels.Board
	if err := store.DB.Unscoped().Where("slug = ?", slug).First(&existing).Error; err == nil {
		if existing.DeletedAt.Valid {
			return nil, h.HttpError(http.StatusConflict, "slug 已被已删除的板块占用，请恢复原板块")
		}
		return nil, h.HttpError(http.StatusConflict, "slug 已被占用")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if err := store.DB.Create(board).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, h.HttpError(http.StatusConflict, "slug 已被占用")
		}
		return nil, err
	}

	dao.Audit(c, user.ID, "board.create", "board", board.ID, map[string]any{"slug": board.Slug, "name": board.Name})
	return board, nil
}

// UpdateBoard 修改板块设置。slug 不可修改。
func UpdateBoard(user *models.User, boardID uint, input BoardInput, c *gin.Context) (*forummodels.Board, error) {
	board, err := dao.BoardByID(boardID, false)
	if err != nil {
		return nil, h.HttpError(http.StatusNotFound, "板块不存在")
	}
	if !perms.HasBoardPermission(user, board.ID, "board.update") {
		return nil, h.HttpError(http.StatusForbidden, "无权修改该板块")
	}

	if input.Slug != nil {
		return nil, h.HttpError(http.StatusBadRequest, "slug 创建后不可修改")
	}

	updates := map[string]any{}
	if input.Name != nil {
		name, err := validateBoardName(*input.Name)
		if err != nil {
			return nil, err
		}
		updates["name"] = name
	}
	if input.Description != nil {
		description, err := validateBoardDescription(*input.Description)
		if err != nil {
			return nil, err
		}
		updates["description"] = description
	}
	if input.Icon != nil {
		icon, err := validateIcon(*input.Icon)
		if err != nil {
			return nil, err
		}
		updates["icon"] = icon
	}
	if input.Color != nil {
		color, err := validateColor(*input.Color)
		if err != nil {
			return nil, err
		}
		updates["color"] = color
	}
	if input.SortOrder != nil {
		updates["sort_order"] = *input.SortOrder
	}
	if input.Enabled != nil {
		updates["enabled"] = *input.Enabled
	}

	if len(updates) > 0 {
		if err := store.DB.Model(&forummodels.Board{}).Where("id = ?", board.ID).Updates(updates).Error; err != nil {
			return nil, err
		}
		dao.Audit(c, user.ID, "board.update", "board", board.ID, updates)
	}

	return dao.BoardByID(board.ID, true)
}

// DeleteBoard 软删除板块；要求板块下没有未删除主题。
func DeleteBoard(user *models.User, boardID uint, c *gin.Context) error {
	board, err := dao.BoardByID(boardID, false)
	if err != nil {
		return h.HttpError(http.StatusNotFound, "板块不存在")
	}
	if !perms.Allow(user, "*", "board.delete") {
		return h.HttpError(http.StatusForbidden, "权限不足")
	}

	var count int64
	if err := store.DB.Model(&forummodels.Thread{}).Where("board_id = ?", board.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return h.HttpError(http.StatusConflict, "板块下仍有主题，请先迁移或停用板块")
	}

	if err := store.DB.Delete(&forummodels.Board{}, board.ID).Error; err != nil {
		return err
	}

	dao.Audit(c, user.ID, "board.delete", "board", board.ID, map[string]any{"slug": board.Slug})
	return nil
}

// ReorderBoards 按给定 ID 顺序重写 sort_order。
func ReorderBoards(user *models.User, ids []uint, c *gin.Context) error {
	if !perms.Allow(user, "*", "board.reorder") {
		return h.HttpError(http.StatusForbidden, "权限不足")
	}
	if len(ids) == 0 {
		return h.HttpError(http.StatusBadRequest, "ids 不能为空")
	}

	return store.DB.Transaction(func(tx *gorm.DB) error {
		for index, id := range ids {
			result := tx.Model(&forummodels.Board{}).Where("id = ?", id).Update("sort_order", index)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return h.HttpError(http.StatusBadRequest, "板块不存在: "+strconv.FormatUint(uint64(id), 10))
			}
		}
		dao.Audit(c, user.ID, "board.reorder", "board", 0, map[string]any{"ids": ids})
		return nil
	})
}

// RestoreBoard 恢复被软删除的板块。
func RestoreBoard(user *models.User, boardID uint, c *gin.Context) (*forummodels.Board, error) {
	board, err := dao.BoardByID(boardID, true)
	if err != nil {
		return nil, h.HttpError(http.StatusNotFound, "板块不存在")
	}
	if !board.DeletedAt.Valid {
		return nil, h.HttpError(http.StatusConflict, "板块未被删除")
	}
	if !perms.Allow(user, "*", "board.delete") {
		return nil, h.HttpError(http.StatusForbidden, "权限不足")
	}

	if err := store.DB.Unscoped().Model(&forummodels.Board{}).Where("id = ?", board.ID).
		Update("deleted_at", nil).Error; err != nil {
		return nil, err
	}

	dao.Audit(c, user.ID, "board.restore", "board", board.ID, map[string]any{"slug": board.Slug})
	return dao.BoardByID(board.ID, false)
}

func validateBoardName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", h.HttpError(http.StatusBadRequest, "板块名称不能为空")
	}
	if len([]rune(value)) > 64 {
		return "", h.HttpError(http.StatusBadRequest, "板块名称不能超过 64 个字符")
	}
	return value, nil
}

func validateBoardDescription(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > 255 {
		return "", h.HttpError(http.StatusBadRequest, "板块描述不能超过 255 个字符")
	}
	return value, nil
}

func validateIcon(value string) (string, error) {
	if !allowedIcons[value] {
		return "", h.HttpError(http.StatusBadRequest, "图标只能是 discussion、code、life 或 feedback")
	}
	return value, nil
}

func validateColor(value string) (string, error) {
	if !colorRe.MatchString(value) {
		return "", h.HttpError(http.StatusBadRequest, "颜色必须是 #RRGGBB 格式")
	}
	return value, nil
}

func valueOr(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}

// AdminBoardDTO 把板块模型转换为后台 DTO。
func AdminBoardDTO(board *forummodels.Board) dto.AdminBoard {
	counts := dao.BoardThreadCounts([]uint{board.ID})
	return adminBoardFrom(*board, counts[board.ID])
}

func adminBoardFrom(board forummodels.Board, threadCount int64) dto.AdminBoard {
	return dto.AdminBoard{
		ID:          board.ID,
		Slug:        board.Slug,
		Name:        board.Name,
		Description: board.Description,
		Icon:        board.Icon,
		Color:       board.Color,
		SortOrder:   board.SortOrder,
		Enabled:     board.Enabled,
		Deleted:     board.DeletedAt.Valid,
		ThreadCount: threadCount,
	}
}
