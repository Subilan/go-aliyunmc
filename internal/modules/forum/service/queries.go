package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/content"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dto"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/perms"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/settings"
)

// 搜索输入上限：避免超长/超多关键词构成超大 SQL 查询。
const (
	maxSearchQueryRunes = 100
	maxSearchKeywords   = 5
)

// SiteInfo 返回公开站点信息。
func SiteInfo() dto.Site {
	values := settings.Get()
	return dto.Site{
		SiteName:        values.SiteName,
		SiteDescription: values.SiteDescription,
		Announcement:    values.Announcement,
		FooterText:      values.FooterText,
		AllowGuestRead:  values.AllowGuestRead,
	}
}

// ListBoards 返回启用板块列表。
func ListBoards(viewer *models.User) ([]dto.BoardSummary, error) {
	boards, err := dao.EnabledBoards()
	if err != nil {
		return nil, err
	}

	ids := make([]uint, 0, len(boards))
	for _, board := range boards {
		ids = append(ids, board.ID)
	}

	var viewerID uint
	if viewer != nil {
		viewerID = viewer.ID
	}

	return dto.BoardSummaries(
		boards,
		dao.BoardThreadCounts(ids),
		dao.BoardFollowerCounts(ids),
		dao.FollowedBoardIDs(viewerID, ids),
	), nil
}

// BoardDetail 返回板块详情。
func BoardDetail(slug string, viewer *models.User) (*dto.BoardSummary, error) {
	board, err := dao.BoardBySlug(slug, false, false)
	if err != nil {
		return nil, h.HttpError(http.StatusNotFound, "板块不存在")
	}

	ids := []uint{board.ID}
	var viewerID uint
	if viewer != nil {
		viewerID = viewer.ID
	}
	summaries := dto.BoardSummaries(
		[]forummodels.Board{*board},
		dao.BoardThreadCounts(ids),
		dao.BoardFollowerCounts(ids),
		dao.FollowedBoardIDs(viewerID, ids),
	)
	return &summaries[0], nil
}

// ListThreads 返回主题流。pinned 决定只取置顶或只取非置顶。
func ListThreads(boardSlug string, pinned bool, page, pageSize int, viewer *models.User) (dto.Page[dto.ThreadSummary], error) {
	var boardID uint
	if boardSlug != "" {
		board, err := dao.BoardBySlug(boardSlug, false, false)
		if err != nil {
			return dto.Page[dto.ThreadSummary]{}, h.HttpError(http.StatusNotFound, "板块不存在")
		}
		boardID = board.ID
	}

	sort := "created"
	if pinned {
		sort = "pinned"
	}

	threads, total, page, pageSize, err := dao.ListThreads(dao.ThreadFilter{
		BoardID:           boardID,
		Pinned:            &pinned,
		OnlyEnabledBoards: true,
		Sort:              sort,
		Page:              page,
		PageSize:          pageSize,
		MaxPageSize:       50,
	})
	if err != nil {
		return dto.Page[dto.ThreadSummary]{}, err
	}

	return dto.NewPage(buildThreadSummaries(threads, viewer), total, page, pageSize), nil
}

func buildThreadSummaries(threads []forummodels.Thread, viewer *models.User) []dto.ThreadSummary {
	if len(threads) == 0 {
		return []dto.ThreadSummary{}
	}
	return dto.ThreadSummaries(threads, threadContext(threads, viewer, nil))
}

// threadContext 批量组装主题卡片/详情所需的关联数据。
// boards 非 nil 时复用已加载的板块，避免详情接口重复查询。
func threadContext(threads []forummodels.Thread, viewer *models.User, boards map[uint]forummodels.Board) dto.ThreadContext {
	boardIDs := make([]uint, 0, len(threads))
	authorIDs := make([]uint, 0, len(threads))
	threadIDs := make([]uint, 0, len(threads))
	for _, thread := range threads {
		threadIDs = append(threadIDs, thread.ID)
		boardIDs = append(boardIDs, thread.BoardID)
		authorIDs = append(authorIDs, thread.AuthorID)
	}
	boardIDs = uniqueIDs(boardIDs)
	authorIDs = uniqueIDs(authorIDs)

	if boards == nil {
		boards = dao.BoardsByIDs(boardIDs, true)
	}

	var viewerID uint
	if viewer != nil {
		viewerID = viewer.ID
	}

	return dto.ThreadContext{
		Views:       dao.LoadUserViews(authorIDs),
		Boards:      boards,
		Tags:        dao.LoadThreadTags(threadIDs),
		ReplyCounts: dao.ReplyCounts(threadIDs),
		LikeCounts:  dao.ThreadLikeCounts(threadIDs),
		Liked:       dao.LikedThreadIDs(viewerID, threadIDs),
	}
}

// ThreadDetail 返回主题详情与 viewer 权限。
func ThreadDetail(threadID uint, viewer *models.User) (*dto.ThreadDetail, error) {
	thread, err := dao.ThreadByID(threadID, false)
	if err != nil {
		return nil, h.HttpError(http.StatusNotFound, "主题不存在")
	}

	board, err := requireEnabledBoard(thread.BoardID)
	if err != nil {
		return nil, err
	}

	ctx := threadContext(
		[]forummodels.Thread{*thread},
		viewer,
		map[uint]forummodels.Board{board.ID: *board},
	)

	canEdit := CanEditThread(viewer, thread)
	canDelete := CanDeleteThread(viewer, thread)
	canModerate := viewer != nil && (perms.HasBoardPermission(viewer, thread.BoardID, "thread.update_any") ||
		perms.HasBoardPermission(viewer, thread.BoardID, "thread.delete_any") ||
		perms.HasBoardPermission(viewer, thread.BoardID, "thread.pin") ||
		perms.HasBoardPermission(viewer, thread.BoardID, "thread.lock") ||
		perms.HasBoardPermission(viewer, thread.BoardID, "thread.move"))

	detail := dto.ThreadDetailFrom(*thread, ctx, canEdit, canDelete, canModerate)
	return &detail, nil
}

// ThreadReplies 返回主题跟帖（含删除占位）。
func ThreadReplies(threadID uint, page, pageSize int, viewer *models.User) (dto.Page[dto.Reply], error) {
	thread, err := dao.ThreadByID(threadID, false)
	if err != nil {
		return dto.Page[dto.Reply]{}, h.HttpError(http.StatusNotFound, "主题不存在")
	}

	if _, err := requireEnabledBoard(thread.BoardID); err != nil {
		return dto.Page[dto.Reply]{}, err
	}

	replies, total, page, pageSize, err := dao.ListThreadReplies(thread.ID, page, pageSize)
	if err != nil {
		return dto.Page[dto.Reply]{}, err
	}

	authorIDs := make([]uint, 0, len(replies))
	replyIDs := make([]uint, 0, len(replies))
	for _, reply := range replies {
		if reply.DeletedAt.Valid {
			continue
		}
		authorIDs = append(authorIDs, reply.AuthorID)
		replyIDs = append(replyIDs, reply.ID)
	}
	authorIDs = uniqueIDs(authorIDs)

	var viewerID uint
	if viewer != nil {
		viewerID = viewer.ID
	}

	canEdit, canDelete := replyPermissionFuncs(viewer, thread.BoardID)
	ctx := dto.ReplyContext{
		Views:     dao.LoadUserViews(authorIDs),
		LikeCount: dao.ReplyLikeCounts(replyIDs),
		Liked:     dao.LikedReplyIDs(viewerID, replyIDs),
		CanEdit:   canEdit,
		CanDelete: canDelete,
	}

	return dto.NewPage(dto.Replies(replies, ctx), total, page, pageSize), nil
}

// RecentReplies 返回最近回复侧栏。
func RecentReplies(boardSlug string, limit int) ([]dto.RecentReply, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	db := store.DB.Model(&forummodels.Reply{}).
		Joins("JOIN forum_threads ON forum_threads.id = forum_replies.thread_id AND forum_threads.deleted_at IS NULL").
		Joins("JOIN forum_boards ON forum_boards.id = forum_threads.board_id AND forum_boards.deleted_at IS NULL AND forum_boards.enabled = 1")

	if boardSlug != "" {
		board, err := dao.BoardBySlug(boardSlug, false, false)
		if err != nil {
			return nil, h.HttpError(http.StatusNotFound, "板块不存在")
		}
		db = db.Where("forum_threads.board_id = ?", board.ID)
	}

	var replies []forummodels.Reply
	if err := db.Select("forum_replies.*").Order("forum_replies.created_at DESC, forum_replies.id DESC").
		Limit(limit).
		Find(&replies).Error; err != nil {
		return nil, err
	}

	threadIDs := make([]uint, 0, len(replies))
	authorIDs := make([]uint, 0, len(replies))
	for _, reply := range replies {
		threadIDs = append(threadIDs, reply.ThreadID)
		authorIDs = append(authorIDs, reply.AuthorID)
	}
	threadIDs = uniqueIDs(threadIDs)
	authorIDs = uniqueIDs(authorIDs)

	threads := dao.ThreadsByIDs(threadIDs, false)
	boardIDs := make([]uint, 0, len(threads))
	for _, thread := range threads {
		boardIDs = append(boardIDs, thread.BoardID)
	}
	boards := dao.BoardsByIDs(boardIDs, true)
	views := dao.LoadUserViews(authorIDs)

	items := make([]dto.RecentReply, 0, len(replies))
	for _, reply := range replies {
		thread := threads[reply.ThreadID]
		board := boards[thread.BoardID]
		items = append(items, dto.RecentReply{
			ID:          reply.ID,
			ThreadID:    reply.ThreadID,
			ThreadTitle: thread.Title,
			BoardSlug:   board.Slug,
			Floor:       reply.Floor,
			Excerpt:     content.Excerpt(reply.ContentText, 160),
			Author:      dto.SummaryForID(reply.AuthorID, views),
			CreatedAt:   dto.Timestamp(reply.CreatedAt),
		})
	}
	return items, nil
}

// UserThreads 返回用户发布的主题。
func UserThreads(userID uint, page, pageSize int, viewer *models.User) (dto.Page[dto.ThreadSummary], error) {
	views := dao.LoadUserViews([]uint{userID})
	view, ok := views[userID]
	if !ok || view.Deleted {
		return dto.Page[dto.ThreadSummary]{}, h.HttpError(http.StatusNotFound, "用户不存在")
	}

	threads, total, page, pageSize, err := dao.ListThreads(dao.ThreadFilter{
		AuthorID:          userID,
		OnlyEnabledBoards: true,
		Page:              page,
		PageSize:          pageSize,
		MaxPageSize:       50,
	})
	if err != nil {
		return dto.Page[dto.ThreadSummary]{}, err
	}

	return dto.NewPage(buildThreadSummaries(threads, viewer), total, page, pageSize), nil
}

// UserReplies 返回用户发布的跟帖。
func UserReplies(userID uint, page, pageSize int, viewer *models.User) (dto.Page[dto.Reply], error) {
	views := dao.LoadUserViews([]uint{userID})
	view, ok := views[userID]
	if !ok || view.Deleted {
		return dto.Page[dto.Reply]{}, h.HttpError(http.StatusNotFound, "用户不存在")
	}

	replies, total, page, pageSize, err := dao.ListReplies(dao.ReplyFilter{
		AuthorID:          userID,
		OnlyEnabledBoards: true,
		Page:              page,
		PageSize:          pageSize,
		MaxPageSize:       50,
	})
	if err != nil {
		return dto.Page[dto.Reply]{}, err
	}

	replyIDs := make([]uint, 0, len(replies))
	authorIDs := make([]uint, 0, len(replies))
	for _, reply := range replies {
		replyIDs = append(replyIDs, reply.ID)
		authorIDs = append(authorIDs, reply.AuthorID)
	}
	authorIDs = uniqueIDs(authorIDs)

	var viewerID uint
	if viewer != nil {
		viewerID = viewer.ID
	}

	ctx := dto.ReplyContext{
		Views:     dao.LoadUserViews(authorIDs),
		LikeCount: dao.ReplyLikeCounts(replyIDs),
		Liked:     dao.LikedReplyIDs(viewerID, replyIDs),
	}

	return dto.NewPage(dto.Replies(replies, ctx), total, page, pageSize), nil
}

// Search 按多关键词 AND 搜索主题。
func Search(query, boardSlug string, page, pageSize int, viewer *models.User) (dto.Page[dto.ThreadSummary], error) {
	if len([]rune(query)) > maxSearchQueryRunes {
		return dto.Page[dto.ThreadSummary]{}, h.HttpError(http.StatusBadRequest, fmt.Sprintf("搜索词不能超过 %d 个字符", maxSearchQueryRunes))
	}

	keywords := strings.Fields(strings.TrimSpace(query))
	if len(keywords) == 0 {
		return dto.NewPage([]dto.ThreadSummary{}, 0, 1, 20), nil
	}
	if len(keywords) > maxSearchKeywords {
		return dto.Page[dto.ThreadSummary]{}, h.HttpError(http.StatusBadRequest, fmt.Sprintf("搜索词最多包含 %d 个关键词", maxSearchKeywords))
	}

	db := store.DB.Model(&forummodels.Thread{}).
		Joins("JOIN forum_boards ON forum_boards.id = forum_threads.board_id AND forum_boards.deleted_at IS NULL AND forum_boards.enabled = 1").
		Joins("LEFT JOIN users ON users.id = forum_threads.author_id AND users.deleted_at IS NULL").
		Joins("LEFT JOIN forum_profiles ON forum_profiles.user_id = forum_threads.author_id AND forum_profiles.deleted_at IS NULL").
		Where("forum_threads.deleted_at IS NULL")

	if boardSlug != "" {
		board, err := dao.BoardBySlug(boardSlug, false, false)
		if err != nil {
			return dto.Page[dto.ThreadSummary]{}, h.HttpError(http.StatusNotFound, "板块不存在")
		}
		db = db.Where("forum_threads.board_id = ?", board.ID)
	}

	effectiveKeywords := 0
	for _, keyword := range keywords {
		if strings.HasPrefix(keyword, "#") {
			tag := strings.TrimPrefix(keyword, "#")
			if tag == "" {
				continue
			}
			effectiveKeywords++
			db = db.Where(`EXISTS (SELECT 1 FROM forum_thread_tags tt JOIN forum_tags tg ON tg.id = tt.tag_id
				WHERE tt.thread_id = forum_threads.id AND INSTR(tg.name, ?) > 0)`, tag)
			continue
		}

		effectiveKeywords++
		db = db.Where(`INSTR(forum_threads.title, ?) > 0
			OR INSTR(forum_threads.content_text, ?) > 0
			OR INSTR(users.username, ?) > 0
			OR INSTR(forum_profiles.nickname, ?) > 0
			OR INSTR(forum_boards.name, ?) > 0
			OR EXISTS (SELECT 1 FROM forum_replies r WHERE r.thread_id = forum_threads.id AND r.deleted_at IS NULL AND INSTR(r.content_text, ?) > 0)
			OR EXISTS (SELECT 1 FROM forum_thread_tags tt JOIN forum_tags tg ON tg.id = tt.tag_id WHERE tt.thread_id = forum_threads.id AND INSTR(tg.name, ?) > 0)`,
			keyword, keyword, keyword, keyword, keyword, keyword, keyword)
	}

	if effectiveKeywords == 0 {
		// 只有裸 # 之类的无效关键词，不应退化成“返回全部主题”。
		return dto.NewPage([]dto.ThreadSummary{}, 0, 1, 20), nil
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return dto.Page[dto.ThreadSummary]{}, err
	}

	page, pageSize = dao.Pagination(page, pageSize, 20, 50)

	var threads []forummodels.Thread
	if err := db.Select("forum_threads.*").Order("forum_threads.created_at DESC, forum_threads.id DESC").
		Offset(dao.Offset(page, pageSize)).
		Limit(pageSize).
		Find(&threads).Error; err != nil {
		return dto.Page[dto.ThreadSummary]{}, err
	}

	return dto.NewPage(buildThreadSummaries(threads, viewer), total, page, pageSize), nil
}

// HotTags 返回热门标签。
func HotTags(boardSlug string, limit int) ([]dto.TagSummary, error) {
	if limit <= 0 {
		limit = 8
	}
	if limit > 50 {
		limit = 50
	}

	db := store.DB.Model(&forummodels.ThreadTag{}).
		Select("forum_tags.name AS name, COUNT(DISTINCT forum_thread_tags.thread_id) AS count").
		Joins("JOIN forum_tags ON forum_tags.id = forum_thread_tags.tag_id").
		Joins("JOIN forum_threads ON forum_threads.id = forum_thread_tags.thread_id AND forum_threads.deleted_at IS NULL").
		Joins("JOIN forum_boards ON forum_boards.id = forum_threads.board_id AND forum_boards.deleted_at IS NULL AND forum_boards.enabled = 1").
		Group("forum_tags.id, forum_tags.name").
		Order("count DESC, forum_tags.name ASC").
		Limit(limit)

	if boardSlug != "" {
		board, err := dao.BoardBySlug(boardSlug, false, false)
		if err != nil {
			return nil, h.HttpError(http.StatusNotFound, "板块不存在")
		}
		db = db.Where("forum_threads.board_id = ?", board.ID)
	}

	var rows []struct {
		Name  string
		Count int64
	}
	if err := db.Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]dto.TagSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.TagSummary{Name: row.Name, Count: row.Count})
	}
	return items, nil
}

// MentionCandidates 返回 @ 提及候选用户。
func MentionCandidates(query string, excludeUserID uint, limit int) ([]dto.UserSummary, error) {
	query = strings.TrimSpace(query)
	if query == "" || len([]rune(query)) > 32 {
		return nil, h.HttpError(http.StatusBadRequest, "搜索词长度必须为 1~32 个字符")
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	like := "%" + query + "%"
	var users []models.User
	err := store.DB.
		Joins("JOIN forum_profiles ON forum_profiles.user_id = users.id AND forum_profiles.deleted_at IS NULL").
		Where("users.deleted_at IS NULL AND users.banned = ? AND users.id <> ?", false, excludeUserID).
		Where("users.username LIKE ? OR forum_profiles.nickname LIKE ?", like, like).
		Order("users.username ASC").
		Limit(limit).
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	ids := make([]uint, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	views := dao.LoadUserViews(ids)

	items := make([]dto.UserSummary, 0, len(users))
	for _, user := range users {
		if view, ok := views[user.ID]; ok {
			items = append(items, dto.SummaryFromView(view))
		}
	}
	return items, nil
}

// ReplyDTO 返回单个跟帖 DTO（用于创建/编辑后的响应）。
func ReplyDTO(reply *forummodels.Reply, viewer *models.User) (*dto.Reply, error) {
	thread, err := dao.ThreadByID(reply.ThreadID, false)
	if err != nil {
		return nil, h.HttpError(http.StatusNotFound, "主题不存在")
	}

	var viewerID uint
	if viewer != nil {
		viewerID = viewer.ID
	}

	canEdit, canDelete := replyPermissionFuncs(viewer, thread.BoardID)
	ctx := dto.ReplyContext{
		Views:     dao.LoadUserViews([]uint{reply.AuthorID}),
		LikeCount: dao.ReplyLikeCounts([]uint{reply.ID}),
		Liked:     dao.LikedReplyIDs(viewerID, []uint{reply.ID}),
		CanEdit:   canEdit,
		CanDelete: canDelete,
	}

	items := dto.Replies([]forummodels.Reply{*reply}, ctx)
	return &items[0], nil
}
