package service

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/content"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/perms"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type processedContent struct {
	HTML       string
	Text       string
	Tags       []string
	MentionIDs []uint
	Floors     []uint
}

// processContent 是主题/跟帖写入前的唯一处理入口。
func processContent(raw string, maxBytes int, wantTags bool) (*processedContent, error) {
	if maxBytes > 0 && len(raw) > maxBytes {
		return nil, h.HttpError(http.StatusBadRequest, fmt.Sprintf("正文超过 %d 字节上限", maxBytes))
	}

	cleaned, err := content.Sanitize(raw)
	if err != nil {
		if errors.Is(err, content.ErrDataImage) {
			return nil, h.HttpError(http.StatusBadRequest, err.Error())
		}
		return nil, h.HttpError(http.StatusBadRequest, "正文包含不允许的内容")
	}

	doc := content.Parse(cleaned)

	mentionIDs := doc.MentionIDs()
	if maxMentions := config.Get().Forum.MaxMentionsPerPost; maxMentions > 0 && len(mentionIDs) > maxMentions {
		return nil, h.HttpError(http.StatusBadRequest, fmt.Sprintf("单篇正文最多提及 %d 个用户", maxMentions))
	}

	// NormalizeMentions 原地更新 doc 并同时返回规范化后仍然有效的提及，无需再次解析。
	validUsers := dao.ValidMentionUsers(mentionIDs)
	cleaned, validIDs := doc.NormalizeMentions(validUsers)

	if maxImages := config.Get().Forum.MaxImagesPerPost; maxImages > 0 && doc.CountImages() > maxImages {
		return nil, h.HttpError(http.StatusBadRequest, fmt.Sprintf("单篇正文最多包含 %d 张图片", maxImages))
	}

	text := doc.PlainText()
	if text == "" && !doc.HasExpandableContent() {
		return nil, h.HttpError(http.StatusBadRequest, "正文不能为空")
	}

	var tags []string
	if wantTags {
		tags = doc.Tags()
		if maxTags := config.Get().Forum.MaxTagsPerPost; maxTags > 0 && len(tags) > maxTags {
			return nil, h.HttpError(http.StatusBadRequest, fmt.Sprintf("单篇主题最多包含 %d 个标签", maxTags))
		}
	}

	floors := doc.Floors()
	if maxQuotes := config.Get().Forum.MaxQuoteRefsPerPost; maxQuotes > 0 && len(floors) > maxQuotes {
		return nil, h.HttpError(http.StatusBadRequest, fmt.Sprintf("单篇正文最多引用 %d 个楼层", maxQuotes))
	}

	return &processedContent{
		HTML:       cleaned,
		Text:       text,
		Tags:       tags,
		MentionIDs: validIDs,
		Floors:     floors,
	}, nil
}

// CanEditThread 判断用户是否可以编辑主题。
func CanEditThread(user *models.User, thread *forummodels.Thread) bool {
	if user == nil || thread == nil {
		return false
	}
	if thread.AuthorID == user.ID && perms.Allow(user, "*", "thread.update_own") {
		return true
	}
	return perms.HasBoardPermission(user, thread.BoardID, "thread.update_any")
}

// CanDeleteThread 判断用户是否可以删除主题。
func CanDeleteThread(user *models.User, thread *forummodels.Thread) bool {
	if user == nil || thread == nil {
		return false
	}
	if thread.AuthorID == user.ID && perms.Allow(user, "*", "thread.delete_own") {
		return true
	}
	return perms.HasBoardPermission(user, thread.BoardID, "thread.delete_any")
}

// replyPermissionFuncs 返回跟帖的可编辑/可删除判断函数，供单条与列表接口复用。
func replyPermissionFuncs(viewer *models.User, boardID uint) (canEdit, canDelete func(forummodels.Reply) bool) {
	canEdit = func(reply forummodels.Reply) bool {
		if viewer == nil || reply.DeletedAt.Valid {
			return false
		}
		if reply.AuthorID == viewer.ID && perms.Allow(viewer, "*", "reply.update_own") {
			return true
		}
		return perms.HasBoardPermission(viewer, boardID, "reply.update_any")
	}

	canDelete = func(reply forummodels.Reply) bool {
		if viewer == nil || reply.DeletedAt.Valid {
			return false
		}
		if reply.AuthorID == viewer.ID && perms.Allow(viewer, "*", "reply.delete_own") {
			return true
		}
		return perms.HasBoardPermission(viewer, boardID, "reply.delete_any")
	}

	return canEdit, canDelete
}

// CreateThread 创建主题并写入标签、提及通知。
func CreateThread(user *models.User, boardID uint, title, rawHTML string) (*forummodels.Thread, error) {
	board, err := requireEnabledBoard(boardID)
	if err != nil {
		return nil, err
	}

	title, err = content.ValidateTitle(title)
	if err != nil {
		return nil, h.HttpError(http.StatusBadRequest, err.Error())
	}

	processed, err := processContent(rawHTML, config.Get().Forum.MaxThreadHTMLBytes, true)
	if err != nil {
		return nil, err
	}

	thread := &forummodels.Thread{
		BoardID:     board.ID,
		AuthorID:    user.ID,
		Title:       title,
		ContentHTML: processed.HTML,
		ContentText: processed.Text,
	}

	var created []forummodels.Notification
	err = store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(thread).Error; err != nil {
			return err
		}
		if err := syncThreadTags(tx, thread.ID, processed.Tags); err != nil {
			return err
		}

		excerpt := content.Excerpt(processed.Text, 160)
		seen := map[uint]bool{user.ID: true}

		// 提及优先于板块关注通知。
		for _, recipient := range processed.MentionIDs {
			if seen[recipient] {
				continue
			}
			n, err := addNotification(tx, notificationInput{
				RecipientID: recipient,
				ActorID:     user.ID,
				Type:        forummodels.NotificationTypeMention,
				ThreadID:    thread.ID,
				Excerpt:     excerpt,
			})
			if err != nil {
				return err
			}
			seen[recipient] = true
			if n != nil {
				created = append(created, *n)
			}
		}

		followers, err := dao.BoardFollowerIDs(tx, board.ID)
		if err != nil {
			return err
		}
		for _, follower := range followers {
			if seen[follower] {
				continue
			}
			n, err := addNotification(tx, notificationInput{
				RecipientID: follower,
				ActorID:     user.ID,
				Type:        forummodels.NotificationTypeBoardThread,
				ThreadID:    thread.ID,
				Excerpt:     excerpt,
			})
			if err != nil {
				return err
			}
			seen[follower] = true
			if n != nil {
				created = append(created, *n)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	publishNotifications(created)
	return thread, nil
}

// UpdateThread 编辑主题。title/contentHTML 为 nil 表示不修改。
func UpdateThread(user *models.User, threadID uint, title *string, rawHTML *string) (*forummodels.Thread, error) {
	thread, err := dao.ThreadByID(threadID, false)
	if err != nil {
		return nil, h.HttpError(http.StatusNotFound, "主题不存在")
	}
	if !CanEditThread(user, thread) {
		return nil, h.HttpError(http.StatusForbidden, "无权编辑该主题")
	}

	var newTitle string
	titleChanged := false
	if title != nil {
		newTitle, err = content.ValidateTitle(*title)
		if err != nil {
			return nil, h.HttpError(http.StatusBadRequest, err.Error())
		}
		titleChanged = newTitle != thread.Title
	}

	var processed *processedContent
	if rawHTML != nil {
		processed, err = processContent(*rawHTML, config.Get().Forum.MaxThreadHTMLBytes, true)
		if err != nil {
			return nil, err
		}
	}

	var created []forummodels.Notification
	err = store.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{}
		if titleChanged {
			updates["title"] = newTitle
		}

		if processed != nil {
			oldMentions := content.MentionIDs(thread.ContentHTML)
			updates["content_html"] = processed.HTML
			updates["content_text"] = processed.Text
			if err := syncThreadTags(tx, thread.ID, processed.Tags); err != nil {
				return err
			}

			excerpt := content.Excerpt(processed.Text, 160)
			if err := tx.Model(&forummodels.Notification{}).
				Where("thread_id = ? AND reply_id = 0", thread.ID).
				Update("excerpt", excerpt).Error; err != nil {
				return err
			}

			for _, recipient := range newMentionIDs(oldMentions, processed.MentionIDs) {
				n, err := addNotification(tx, notificationInput{
					RecipientID: recipient,
					ActorID:     user.ID,
					Type:        forummodels.NotificationTypeMention,
					ThreadID:    thread.ID,
					Excerpt:     excerpt,
				})
				if err != nil {
					return err
				}
				if n != nil {
					created = append(created, *n)
				}
			}
		}

		if len(updates) > 0 {
			if err := tx.Model(&forummodels.Thread{}).Where("id = ?", thread.ID).Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	publishNotifications(created)

	return dao.ThreadByID(thread.ID, false)
}

// maxAuditReasonRunes 限制删除原因长度，避免审计 detail 过大。
const maxAuditReasonRunes = 500

func normalizeReason(reason string) string {
	reason = strings.TrimSpace(reason)
	runes := []rune(reason)
	if len(runes) > maxAuditReasonRunes {
		return string(runes[:maxAuditReasonRunes]) + "…"
	}
	return reason
}

// DeleteThread 软删除主题及其跟帖；通知刻意保留，指向已删除内容时由前端展示 404。
// asAdmin 表示操作来自管理入口，即使操作者是作者本人也需要审计。
func DeleteThread(user *models.User, threadID uint, reason string, asAdmin bool, c *gin.Context) error {
	thread, err := dao.ThreadByID(threadID, false)
	if err != nil {
		if existing, findErr := dao.ThreadByID(threadID, true); findErr == nil && existing.DeletedAt.Valid {
			return h.HttpError(http.StatusConflict, "主题已删除")
		}
		return h.HttpError(http.StatusNotFound, "主题不存在")
	}
	if asAdmin {
		if !perms.HasBoardPermission(user, thread.BoardID, "thread.delete_any") {
			return h.HttpError(http.StatusForbidden, "无权删除该主题")
		}
	} else if !CanDeleteThread(user, thread) {
		return h.HttpError(http.StatusForbidden, "无权删除该主题")
	}

	err = store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&forummodels.Thread{}).
			Where("id = ?", thread.ID).
			Updates(map[string]any{"deleted_at": time.Now(), "deleted_by": user.ID}).Error; err != nil {
			return err
		}
		return tx.Where("thread_id = ?", thread.ID).Delete(&forummodels.Reply{}).Error
	})
	if err != nil {
		return err
	}

	if asAdmin || thread.AuthorID != user.ID {
		dao.Audit(c, user.ID, "thread.delete", "thread", thread.ID, map[string]any{"reason": normalizeReason(reason)})
	}
	return nil
}

// RestoreThread 恢复被软删除的主题；不自动恢复其跟帖。
func RestoreThread(user *models.User, threadID uint, c *gin.Context) (*forummodels.Thread, error) {
	thread, err := dao.ThreadByID(threadID, true)
	if err != nil {
		return nil, h.HttpError(http.StatusNotFound, "主题不存在")
	}
	if !thread.DeletedAt.Valid {
		return nil, h.HttpError(http.StatusConflict, "主题未被删除")
	}
	if !perms.HasBoardPermission(user, thread.BoardID, "content.restore") {
		return nil, h.HttpError(http.StatusForbidden, "无权恢复该主题")
	}

	err = store.DB.Transaction(func(tx *gorm.DB) error {
		return tx.Unscoped().Model(&forummodels.Thread{}).
			Where("id = ?", thread.ID).
			Updates(map[string]any{"deleted_at": nil, "deleted_by": nil}).Error
	})
	if err != nil {
		return nil, err
	}

	dao.Audit(c, user.ID, "thread.restore", "thread", thread.ID, nil)
	return dao.ThreadByID(thread.ID, false)
}

// CreateReply 在主题下创建跟帖，事务内锁主题行分配稳定楼层。
func CreateReply(user *models.User, threadID uint, rawHTML string) (*forummodels.Reply, error) {
	thread, err := dao.ThreadByID(threadID, false)
	if err != nil {
		return nil, h.HttpError(http.StatusNotFound, "主题不存在")
	}
	if thread.Locked {
		return nil, h.HttpError(http.StatusForbidden, "主题已锁定，禁止回复")
	}

	if _, err := requireEnabledBoard(thread.BoardID); err != nil {
		return nil, err
	}

	processed, err := processContent(rawHTML, config.Get().Forum.MaxReplyHTMLBytes, false)
	if err != nil {
		return nil, err
	}

	var (
		reply   *forummodels.Reply
		created []forummodels.Notification
	)
	err = store.DB.Transaction(func(tx *gorm.DB) error {
		locked, err := dao.ThreadByIDForUpdate(tx, threadID)
		if err != nil {
			return err
		}
		if locked.Locked {
			return h.HttpError(http.StatusForbidden, "主题已锁定，禁止回复")
		}

		maxFloor, err := dao.MaxFloor(tx, threadID)
		if err != nil {
			return err
		}

		reply = &forummodels.Reply{
			ThreadID:    threadID,
			AuthorID:    user.ID,
			Floor:       maxFloor + 1,
			ContentHTML: processed.HTML,
			ContentText: processed.Text,
		}
		resolveReplyReference(tx, reply, processed.Floors, maxFloor)

		if err := tx.Create(reply).Error; err != nil {
			return err
		}

		if err := tx.Model(&forummodels.Thread{}).
			Where("id = ?", threadID).
			Update("last_reply_at", reply.CreatedAt).Error; err != nil {
			return err
		}

		excerpt := content.Excerpt(processed.Text, 160)
		plans, err := planReplyRecipients(tx, locked.AuthorID, user.ID, threadID, processed.Floors, processed.MentionIDs)
		if err != nil {
			return err
		}
		for recipient, plan := range plans {
			n, err := addNotification(tx, notificationInput{
				RecipientID: recipient,
				ActorID:     user.ID,
				Type:        plan.Type,
				ThreadID:    threadID,
				ReplyID:     reply.ID,
				QuotedFloor: plan.QuotedFloor,
				Excerpt:     excerpt,
			})
			if err != nil {
				return err
			}
			if n != nil {
				created = append(created, *n)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	publishNotifications(created)
	return reply, nil
}

// UpdateReply 编辑跟帖正文，并同步引用、摘要与新增提及通知。
// asAdmin 为 true 时要求板块级 reply.update_any，并写审计；否则按作者所有权校验。
func UpdateReply(user *models.User, replyID uint, rawHTML string, asAdmin bool, c *gin.Context) (*forummodels.Reply, error) {
	reply, err := dao.ReplyByID(replyID, false)
	if err != nil {
		return nil, h.HttpError(http.StatusNotFound, "跟帖不存在")
	}

	thread, err := dao.ThreadByID(reply.ThreadID, false)
	if err != nil {
		return nil, h.HttpError(http.StatusNotFound, "主题不存在")
	}

	if asAdmin {
		if !perms.HasBoardPermission(user, thread.BoardID, "reply.update_any") {
			return nil, h.HttpError(http.StatusForbidden, "无权编辑该跟帖")
		}
	} else {
		canEdit, _ := replyPermissionFuncs(user, thread.BoardID)
		if !canEdit(*reply) {
			return nil, h.HttpError(http.StatusForbidden, "无权编辑该跟帖")
		}
	}

	processed, err := processContent(rawHTML, config.Get().Forum.MaxReplyHTMLBytes, false)
	if err != nil {
		return nil, err
	}

	var created []forummodels.Notification
	err = store.DB.Transaction(func(tx *gorm.DB) error {
		maxFloor, err := dao.MaxFloor(tx, reply.ThreadID)
		if err != nil {
			return err
		}

		updated := *reply
		updated.ContentHTML = processed.HTML
		updated.ContentText = processed.Text
		updated.ReplyToFloor = nil
		updated.ReplyToReplyID = nil
		resolveReplyReference(tx, &updated, processed.Floors, maxFloor)

		if err := tx.Model(&forummodels.Reply{}).
			Where("id = ?", reply.ID).
			Updates(map[string]any{
				"content_html":      updated.ContentHTML,
				"content_text":      updated.ContentText,
				"reply_to_floor":    updated.ReplyToFloor,
				"reply_to_reply_id": updated.ReplyToReplyID,
			}).Error; err != nil {
			return err
		}

		excerpt := content.Excerpt(processed.Text, 160)
		if err := tx.Model(&forummodels.Notification{}).
			Where("reply_id = ?", reply.ID).
			Update("excerpt", excerpt).Error; err != nil {
			return err
		}

		oldQuotes, err := quoteRecipients(tx, reply.ThreadID, content.ReplyFloors(reply.ContentHTML))
		if err != nil {
			return err
		}
		newQuotes, err := quoteRecipients(tx, reply.ThreadID, processed.Floors)
		if err != nil {
			return err
		}

		// 只补发新增引用；已有任意内容类通知的接收者不重复提醒。
		for recipient, floor := range newQuotes {
			if recipient == user.ID {
				continue
			}
			if _, existed := oldQuotes[recipient]; existed {
				continue
			}
			exists, err := dao.HasContentNotification(tx, recipient, reply.ThreadID, reply.ID)
			if err != nil {
				return err
			}
			if exists {
				continue
			}
			n, err := addNotification(tx, notificationInput{
				RecipientID: recipient,
				ActorID:     user.ID,
				Type:        forummodels.NotificationTypeQuote,
				ThreadID:    reply.ThreadID,
				ReplyID:     reply.ID,
				QuotedFloor: floor,
				Excerpt:     excerpt,
			})
			if err != nil {
				return err
			}
			if n != nil {
				created = append(created, *n)
			}
		}

		oldMentions := content.MentionIDs(reply.ContentHTML)
		for _, recipient := range newMentionIDs(oldMentions, processed.MentionIDs) {
			if recipient == user.ID {
				continue
			}
			if _, quoted := newQuotes[recipient]; quoted {
				continue
			}
			exists, err := dao.HasContentNotification(tx, recipient, reply.ThreadID, reply.ID)
			if err != nil {
				return err
			}
			if exists {
				continue
			}
			n, err := addNotification(tx, notificationInput{
				RecipientID: recipient,
				ActorID:     user.ID,
				Type:        forummodels.NotificationTypeMention,
				ThreadID:    reply.ThreadID,
				ReplyID:     reply.ID,
				Excerpt:     excerpt,
			})
			if err != nil {
				return err
			}
			if n != nil {
				created = append(created, *n)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	publishNotifications(created)

	if asAdmin {
		dao.Audit(c, user.ID, "reply.update", "reply", reply.ID, map[string]any{"threadId": reply.ThreadID})
	}

	return dao.ReplyByID(reply.ID, false)
}

// DeleteReply 软删除跟帖并保留楼层占位；通知刻意保留。
// asAdmin 表示操作来自管理入口，即使操作者是作者本人也需要审计。
func DeleteReply(user *models.User, replyID uint, reason string, asAdmin bool, c *gin.Context) error {
	reply, err := dao.ReplyByID(replyID, false)
	if err != nil {
		if existing, findErr := dao.ReplyByID(replyID, true); findErr == nil && existing.DeletedAt.Valid {
			return h.HttpError(http.StatusConflict, "跟帖已删除")
		}
		return h.HttpError(http.StatusNotFound, "跟帖不存在")
	}

	thread, err := dao.ThreadByID(reply.ThreadID, true)
	if err != nil {
		return h.HttpError(http.StatusNotFound, "主题不存在")
	}

	if asAdmin {
		if !perms.HasBoardPermission(user, thread.BoardID, "reply.delete_any") {
			return h.HttpError(http.StatusForbidden, "无权删除该跟帖")
		}
	} else {
		_, canDelete := replyPermissionFuncs(user, thread.BoardID)
		if !canDelete(*reply) {
			return h.HttpError(http.StatusForbidden, "无权删除该跟帖")
		}
	}

	err = store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&forummodels.Reply{}).
			Where("id = ?", reply.ID).
			Updates(map[string]any{"deleted_at": time.Now(), "deleted_by": user.ID}).Error; err != nil {
			return err
		}
		return dao.RecalculateLastReplyAt(tx, reply.ThreadID)
	})
	if err != nil {
		return err
	}

	if asAdmin || reply.AuthorID != user.ID {
		dao.Audit(c, user.ID, "reply.delete", "reply", reply.ID, map[string]any{"reason": normalizeReason(reason), "threadId": reply.ThreadID})
	}
	return nil
}

// RestoreReply 恢复被软删除的跟帖。
func RestoreReply(user *models.User, replyID uint, c *gin.Context) (*forummodels.Reply, error) {
	reply, err := dao.ReplyByID(replyID, true)
	if err != nil {
		return nil, h.HttpError(http.StatusNotFound, "跟帖不存在")
	}
	if !reply.DeletedAt.Valid {
		return nil, h.HttpError(http.StatusConflict, "跟帖未被删除")
	}

	thread, err := dao.ThreadByID(reply.ThreadID, true)
	if err != nil {
		return nil, h.HttpError(http.StatusNotFound, "主题不存在")
	}
	if !perms.HasBoardPermission(user, thread.BoardID, "content.restore") {
		return nil, h.HttpError(http.StatusForbidden, "无权恢复该跟帖")
	}

	err = store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Model(&forummodels.Reply{}).
			Where("id = ?", reply.ID).
			Updates(map[string]any{"deleted_at": nil, "deleted_by": nil}).Error; err != nil {
			return err
		}
		return dao.RecalculateLastReplyAt(tx, reply.ThreadID)
	})
	if err != nil {
		return nil, err
	}

	dao.Audit(c, user.ID, "reply.restore", "reply", reply.ID, map[string]any{"threadId": reply.ThreadID})
	return dao.ReplyByID(reply.ID, false)
}

// resolveReplyReference 取第一个合法楼层引用，并解析为目标回复。
func resolveReplyReference(tx *gorm.DB, reply *forummodels.Reply, floors []uint, maxFloor uint) {
	for _, floor := range floors {
		if floor == 0 || floor > maxFloor {
			continue
		}
		reply.ReplyToFloor = &floor
		if target, err := dao.ReplyByFloor(tx, reply.ThreadID, floor); err == nil {
			id := target.ID
			reply.ReplyToReplyID = &id
		}
		return
	}
}

func newMentionIDs(oldIDs, newIDs []uint) []uint {
	old := make(map[uint]bool, len(oldIDs))
	for _, id := range oldIDs {
		old[id] = true
	}

	result := make([]uint, 0, len(newIDs))
	seen := make(map[uint]bool, len(newIDs))
	for _, id := range newIDs {
		if old[id] || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
}

// syncThreadTags 在事务内同步主题标签关联，只增删差异。孤立标签保留。
func syncThreadTags(tx *gorm.DB, threadID uint, tags []string) error {
	desired := make(map[string]string, len(tags))
	order := make([]string, 0, len(tags))
	for _, name := range tags {
		normalized := content.NormalizeTag(name)
		if normalized == "" {
			continue
		}
		if _, ok := desired[normalized]; ok {
			continue
		}
		desired[normalized] = name
		order = append(order, normalized)
	}

	type linkRow struct {
		TagID      uint
		Normalized string
	}
	var rows []linkRow
	if err := tx.Model(&forummodels.ThreadTag{}).
		Select("forum_thread_tags.tag_id, forum_tags.normalized").
		Joins("JOIN forum_tags ON forum_tags.id = forum_thread_tags.tag_id").
		Where("forum_thread_tags.thread_id = ?", threadID).
		Scan(&rows).Error; err != nil {
		return err
	}

	existing := make(map[string]uint, len(rows))
	remove := make([]uint, 0, len(rows))
	for _, row := range rows {
		existing[row.Normalized] = row.TagID
		if _, ok := desired[row.Normalized]; !ok {
			remove = append(remove, row.TagID)
		}
	}
	if len(remove) > 0 {
		if err := tx.Where("thread_id = ? AND tag_id IN ?", threadID, remove).
			Delete(&forummodels.ThreadTag{}).Error; err != nil {
			return err
		}
	}

	for _, normalized := range order {
		if _, ok := existing[normalized]; ok {
			continue
		}

		name := desired[normalized]
		var tag forummodels.Tag
		err := tx.Where("normalized = ?", normalized).First(&tag).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tag = forummodels.Tag{Name: name, Normalized: normalized}
			if err := tx.Create(&tag).Error; err != nil {
				if !errors.Is(err, gorm.ErrDuplicatedKey) {
					return err
				}
				if err := tx.Where("normalized = ?", normalized).First(&tag).Error; err != nil {
					return err
				}
			}
		} else if err != nil {
			return err
		}

		link := forummodels.ThreadTag{ThreadID: threadID, TagID: tag.ID}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&link).Error; err != nil {
			return err
		}
	}

	return nil
}

// notificationInput 是写入一条通知所需的字段。
type notificationInput struct {
	RecipientID uint
	ActorID     uint
	Type        forummodels.NotificationType
	ThreadID    uint
	ReplyID     uint
	QuotedFloor uint
	Excerpt     string
}

// addNotification 写入一条通知；自触发或唯一键冲突时返回 nil。
// excerpt 由调用方负责截断（同一段文本会被多个接收者复用，避免重复计算）。
func addNotification(tx *gorm.DB, in notificationInput) (*forummodels.Notification, error) {
	if in.RecipientID == 0 || in.RecipientID == in.ActorID {
		return nil, nil
	}

	notification := forummodels.Notification{
		RecipientID: in.RecipientID,
		ActorID:     in.ActorID,
		Type:        string(in.Type),
		ThreadID:    in.ThreadID,
		ReplyID:     in.ReplyID,
		QuotedFloor: in.QuotedFloor,
		Excerpt:     in.Excerpt,
		CreatedAt:   time.Now(),
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&notification)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 || notification.ID == 0 {
		return nil, nil
	}
	return &notification, nil
}

// quoteRecipients 按文档顺序返回被引用楼层作者 -> 首个楼层号。
func quoteRecipients(tx *gorm.DB, threadID uint, floors []uint) (map[uint]uint, error) {
	targets, err := dao.QuoteTargets(tx, threadID, floors)
	if err != nil {
		return nil, err
	}

	byFloor := make(map[uint]dao.QuoteTarget, len(targets))
	for _, target := range targets {
		byFloor[target.Floor] = target
	}

	result := make(map[uint]uint)
	seenFloor := make(map[uint]bool, len(floors))
	for _, floor := range floors {
		if floor == 0 || seenFloor[floor] {
			continue
		}
		seenFloor[floor] = true

		target, ok := byFloor[floor]
		if !ok || target.AuthorID == 0 {
			continue
		}
		if _, exists := result[target.AuthorID]; !exists {
			result[target.AuthorID] = floor
		}
	}
	return result, nil
}

type recipientPlan struct {
	Type        forummodels.NotificationType
	QuotedFloor uint
}

// planReplyRecipients 实现 quote > reply(主题作者) > mention 的优先级，
// 每个接收者最多产生一条内容类通知。
func planReplyRecipients(
	tx *gorm.DB,
	threadAuthorID, actorID, threadID uint,
	floors, mentionIDs []uint,
) (map[uint]recipientPlan, error) {
	plans := make(map[uint]recipientPlan)

	quotes, err := quoteRecipients(tx, threadID, floors)
	if err != nil {
		return nil, err
	}
	for recipient, floor := range quotes {
		if recipient == 0 || recipient == actorID {
			continue
		}
		plans[recipient] = recipientPlan{
			Type:        forummodels.NotificationTypeQuote,
			QuotedFloor: floor,
		}
	}

	if threadAuthorID != 0 && threadAuthorID != actorID {
		if _, exists := plans[threadAuthorID]; !exists {
			plans[threadAuthorID] = recipientPlan{Type: forummodels.NotificationTypeReply}
		}
	}

	for _, recipient := range mentionIDs {
		if recipient == 0 || recipient == actorID {
			continue
		}
		if _, exists := plans[recipient]; exists {
			continue
		}
		plans[recipient] = recipientPlan{Type: forummodels.NotificationTypeMention}
	}

	return plans, nil
}
