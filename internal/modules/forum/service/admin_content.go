package service

import (
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dto"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/perms"

	"github.com/gin-gonic/gin"
)

// AdminThreadFilter 是后台主题列表条件。
type AdminThreadFilter struct {
	BoardID  uint
	AuthorID uint
	Status   string
	Pinned   *bool
	Locked   *bool
	Keyword  string
	Sort     string
	Page     int
	PageSize int
}

func contentScope(user *models.User) (all bool, boardIDs []uint, err error) {
	all, boardIDs = perms.BoardScopeFor(user, "content.view_admin")
	if !all && len(boardIDs) == 0 {
		return false, nil, h.HttpError(http.StatusForbidden, "无权访问内容管理")
	}
	return all, boardIDs, nil
}

// AdminListThreads 返回后台主题列表。
func AdminListThreads(user *models.User, filter AdminThreadFilter) (dto.Page[dto.AdminThread], error) {
	all, boardIDs, err := contentScope(user)
	if err != nil {
		return dto.Page[dto.AdminThread]{}, err
	}

	if filter.BoardID != 0 && !all && !slices.Contains(boardIDs, filter.BoardID) {
		return dto.Page[dto.AdminThread]{}, h.HttpError(http.StatusForbidden, "无权访问该板块")
	}

	f := dao.ThreadFilter{
		Status:   filter.Status,
		BoardID:  filter.BoardID,
		AuthorID: filter.AuthorID,
		Pinned:   filter.Pinned,
		Locked:   filter.Locked,
		Keyword:  filter.Keyword,
		Sort:     filter.Sort,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}
	if !all {
		f.BoardIDs = boardIDs
	}

	threads, total, page, pageSize, err := dao.ListThreads(f)
	if err != nil {
		return dto.Page[dto.AdminThread]{}, err
	}

	boardIDsToLoad := make([]uint, 0, len(threads))
	authorIDs := make([]uint, 0, len(threads))
	threadIDs := make([]uint, 0, len(threads))
	for _, thread := range threads {
		boardIDsToLoad = append(boardIDsToLoad, thread.BoardID)
		authorIDs = append(authorIDs, thread.AuthorID)
		threadIDs = append(threadIDs, thread.ID)
	}
	boardIDsToLoad = uniqueIDs(boardIDsToLoad)
	authorIDs = uniqueIDs(authorIDs)

	views := dao.LoadUserViews(authorIDs)
	boards := dao.BoardsByIDs(boardIDsToLoad, true)
	replyCounts := dao.ReplyCounts(threadIDs)
	likeCounts := dao.ThreadLikeCounts(threadIDs)

	items := make([]dto.AdminThread, 0, len(threads))
	for _, thread := range threads {
		items = append(items, dto.AdminThreadFrom(thread, views, boards, replyCounts[thread.ID], likeCounts[thread.ID], false))
	}

	return dto.NewPage(items, total, page, pageSize), nil
}

// AdminThreadDetail 返回后台主题详情（含已删除）。
func AdminThreadDetail(user *models.User, threadID uint) (*dto.AdminThread, error) {
	thread, err := dao.ThreadByID(threadID, true)
	if err != nil {
		return nil, h.HttpError(http.StatusNotFound, "主题不存在")
	}

	all, boardIDs, err := contentScope(user)
	if err != nil {
		return nil, err
	}
	if !all && !slices.Contains(boardIDs, thread.BoardID) {
		return nil, h.HttpError(http.StatusForbidden, "无权访问该板块")
	}

	item := AdminThreadDTOFromThread(*thread)
	return &item, nil
}

// AdminThreadDTOFromThread 组装后台主题详情；调用方负责权限校验。
func AdminThreadDTOFromThread(thread forummodels.Thread) dto.AdminThread {
	views := dao.LoadUserViews([]uint{thread.AuthorID})
	boards := dao.BoardsByIDs([]uint{thread.BoardID}, true)
	replyCounts := dao.ReplyCounts([]uint{thread.ID})
	likeCounts := dao.ThreadLikeCounts([]uint{thread.ID})
	return dto.AdminThreadFrom(thread, views, boards, replyCounts[thread.ID], likeCounts[thread.ID], true)
}

// AdminUpdateThreadInput 是后台主题审核字段。
type AdminUpdateThreadInput struct {
	Pinned  *bool `json:"pinned"`
	Locked  *bool `json:"locked"`
	BoardID *uint `json:"boardId"`
}

// AdminUpdateThread 置顶/锁定/移动主题，每个字段单独校验权限。
func AdminUpdateThread(user *models.User, threadID uint, input AdminUpdateThreadInput, c *gin.Context) (*forummodels.Thread, error) {
	thread, err := dao.ThreadByID(threadID, false)
	if err != nil {
		return nil, h.HttpError(http.StatusNotFound, "主题不存在")
	}

	updates := map[string]any{}

	if input.Pinned != nil {
		if !perms.HasBoardPermission(user, thread.BoardID, "thread.pin") {
			return nil, h.HttpError(http.StatusForbidden, "无权限置顶该板块的主题")
		}
		updates["pinned"] = *input.Pinned
		if *input.Pinned {
			if !thread.Pinned {
				now := time.Now()
				updates["pinned_at"] = &now
			}
		} else {
			updates["pinned_at"] = nil
		}
	}

	if input.Locked != nil {
		if !perms.HasBoardPermission(user, thread.BoardID, "thread.lock") {
			return nil, h.HttpError(http.StatusForbidden, "无权限锁定该板块的主题")
		}
		updates["locked"] = *input.Locked
	}

	if input.BoardID != nil && *input.BoardID != thread.BoardID {
		if !perms.HasBoardPermission(user, thread.BoardID, "thread.move") {
			return nil, h.HttpError(http.StatusForbidden, "无权限移动该板块的主题")
		}
		target, err := dao.BoardByID(*input.BoardID, false)
		if err != nil || !target.Enabled {
			return nil, h.HttpError(http.StatusBadRequest, "目标板块不存在或已禁用")
		}
		if !perms.HasBoardPermission(user, target.ID, "thread.move") {
			return nil, h.HttpError(http.StatusForbidden, "无权限把主题移动到目标板块")
		}
		updates["board_id"] = target.ID
	}

	if len(updates) > 0 {
		if err := store.DB.Model(&forummodels.Thread{}).Where("id = ?", thread.ID).Updates(updates).Error; err != nil {
			return nil, err
		}
		dao.Audit(c, user.ID, "thread.update", "thread", thread.ID, updates)
	}

	return dao.ThreadByID(thread.ID, true)
}

// AdminListReplies 返回后台跟帖列表。
func AdminListReplies(user *models.User, threadID uint, authorID uint, status, keyword string, page, pageSize int) (dto.Page[dto.AdminReply], error) {
	all, boardIDs, err := contentScope(user)
	if err != nil {
		return dto.Page[dto.AdminReply]{}, err
	}

	f := dao.ReplyFilter{
		Status:   status,
		ThreadID: threadID,
		AuthorID: authorID,
		Keyword:  keyword,
		Page:     page,
		PageSize: pageSize,
	}
	if !all {
		f.BoardIDs = boardIDs
	}

	replies, total, page, pageSize, err := dao.ListReplies(f)
	if err != nil {
		return dto.Page[dto.AdminReply]{}, err
	}

	threadIDs := make([]uint, 0, len(replies))
	authorIDs := make([]uint, 0, len(replies))
	replyIDs := make([]uint, 0, len(replies))
	for _, reply := range replies {
		threadIDs = append(threadIDs, reply.ThreadID)
		authorIDs = append(authorIDs, reply.AuthorID)
		replyIDs = append(replyIDs, reply.ID)
	}
	threadIDs = uniqueIDs(threadIDs)
	authorIDs = uniqueIDs(authorIDs)

	titles := dao.ThreadTitles(threadIDs)
	views := dao.LoadUserViews(authorIDs)
	likeCounts := dao.ReplyLikeCounts(replyIDs)

	items := make([]dto.AdminReply, 0, len(replies))
	for _, reply := range replies {
		items = append(items, dto.AdminReplyFrom(reply, views, titles[reply.ThreadID], likeCounts[reply.ID], false))
	}

	return dto.NewPage(items, total, page, pageSize), nil
}

// maxBatchIDs 限制单次批量操作的对象数量，避免超长循环与超大 SQL。
const maxBatchIDs = 100

// AdminBatchThreads 批量执行主题操作，每个 ID 单独校验权限。
func AdminBatchThreads(user *models.User, ids []uint, action string, boardID *uint, reason string, c *gin.Context) (dto.BatchResult, error) {
	if len(ids) == 0 || len(ids) > maxBatchIDs {
		return dto.BatchResult{}, h.HttpError(http.StatusBadRequest, fmt.Sprintf("单次批量操作需要 1~%d 个 ID", maxBatchIDs))
	}
	result := dto.BatchResult{Succeeded: []uint{}, Failed: []dto.BatchFailure{}}

	if action == "move" && boardID == nil {
		return result, h.HttpError(http.StatusBadRequest, "移动操作需要 boardId")
	}

	// 每个 ID 的权限校验与加载都由对应 service 负责，避免批量层重复查询与双检。
	for _, id := range ids {
		var err error
		switch action {
		case "pin", "unpin":
			pinned := action == "pin"
			_, err = AdminUpdateThread(user, id, AdminUpdateThreadInput{Pinned: &pinned}, c)
		case "lock", "unlock":
			locked := action == "lock"
			_, err = AdminUpdateThread(user, id, AdminUpdateThreadInput{Locked: &locked}, c)
		case "delete":
			err = DeleteThread(user, id, reason, true, c)
		case "restore":
			_, err = RestoreThread(user, id, c)
		case "move":
			_, err = AdminUpdateThread(user, id, AdminUpdateThreadInput{BoardID: boardID}, c)
		default:
			return result, h.HttpError(http.StatusBadRequest, "不支持的批量操作: "+action)
		}

		if err != nil {
			result.Failed = append(result.Failed, dto.BatchFailure{ID: id, Reason: err.Error()})
			continue
		}
		result.Succeeded = append(result.Succeeded, id)
	}

	return result, nil
}

// AdminBatchReplies 批量删除/恢复跟帖。
func AdminBatchReplies(user *models.User, ids []uint, action, reason string, c *gin.Context) (dto.BatchResult, error) {
	if len(ids) == 0 || len(ids) > maxBatchIDs {
		return dto.BatchResult{}, h.HttpError(http.StatusBadRequest, fmt.Sprintf("单次批量操作需要 1~%d 个 ID", maxBatchIDs))
	}
	result := dto.BatchResult{Succeeded: []uint{}, Failed: []dto.BatchFailure{}}

	for _, id := range ids {
		var err error
		switch action {
		case "delete":
			err = DeleteReply(user, id, reason, true, c)
		case "restore":
			_, err = RestoreReply(user, id, c)
		default:
			return result, h.HttpError(http.StatusBadRequest, "不支持的批量操作: "+action)
		}

		if err != nil {
			result.Failed = append(result.Failed, dto.BatchFailure{ID: id, Reason: err.Error()})
			continue
		}
		result.Succeeded = append(result.Succeeded, id)
	}

	return result, nil
}

// AdminReplyDTOFromReply 组装后台跟帖详情；调用方负责权限校验。
func AdminReplyDTOFromReply(reply forummodels.Reply) dto.AdminReply {
	titles := dao.ThreadTitles([]uint{reply.ThreadID})
	views := dao.LoadUserViews([]uint{reply.AuthorID})
	likeCounts := dao.ReplyLikeCounts([]uint{reply.ID})
	return dto.AdminReplyFrom(reply, views, titles[reply.ThreadID], likeCounts[reply.ID], true)
}
