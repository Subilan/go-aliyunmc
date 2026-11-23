package dto

import (
	"strings"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/content"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/perms"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/settings"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/uploads"
)

// Timestamp 把时间格式化为 RFC3339 UTC 字符串。
func Timestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// TimestampPtr 格式化可空时间。
func TimestampPtr(t *time.Time) *string {
	if t == nil || t.IsZero() {
		return nil
	}
	value := Timestamp(*t)
	return &value
}

// AvatarURL 返回用户头像 URL：优先上传头像，否则使用默认头像模板。
func AvatarURL(view dao.UserView) string {
	if view.AvatarKey != "" {
		return uploads.URL(view.AvatarKey)
	}
	template := settings.DefaultAvatarTemplate()
	return strings.ReplaceAll(template, "{username}", view.Username)
}

// avatarShape 返回用户的头像形状，缺失时回退到 circle。
func avatarShape(view dao.UserView) string {
	if view.AvatarShape == "" {
		return string(forummodels.AvatarShapeCircle)
	}
	return view.AvatarShape
}

// SummaryFromView 把用户视图转成 UserSummary。
func SummaryFromView(view dao.UserView) UserSummary {
	return UserSummary{
		ID:          view.ID,
		Username:    view.Username,
		Nickname:    view.Nickname,
		AvatarURL:   AvatarURL(view),
		AvatarShape: avatarShape(view),
	}
}

// SummaryForID 从批量视图中构造作者摘要；缺失或已删除用户返回 nil。
func SummaryForID(id uint, views map[uint]dao.UserView) *UserSummary {
	if id == 0 {
		return nil
	}
	view, ok := views[id]
	if !ok || view.Deleted {
		return nil
	}
	summary := SummaryFromView(view)
	return &summary
}

// ProfileFrom 构造用户主页 DTO。
// likeCount 是内容获赞数，profileLikeCount 是主页收到的点赞数。
func ProfileFrom(view dao.UserView, threads, replies, likes, profileLikes int64, likedByMe, isSelf bool) UserProfile {
	var background *string
	if view.BackgroundKey != "" {
		url := uploads.URL(view.BackgroundKey)
		background = &url
	}

	return UserProfile{
		ID:               view.ID,
		Username:         view.Username,
		Nickname:         view.Nickname,
		Bio:              view.Bio,
		AvatarURL:        AvatarURL(view),
		BackgroundURL:    background,
		AvatarShape:      avatarShape(view),
		JoinedAt:         Timestamp(view.JoinedAt),
		ThreadCount:      threads,
		ReplyCount:       replies,
		LikeCount:        likes,
		ProfileLikeCount: profileLikes,
		LikedByMe:        likedByMe,
		IsSelf:           isSelf,
	}
}

// BoardBriefFrom 构造板块摘要。
func BoardBriefFrom(board forummodels.Board) BoardBrief {
	return BoardBrief{ID: board.ID, Slug: board.Slug, Name: board.Name, Icon: board.Icon, Color: board.Color}
}

// BoardSummaries 构造板块列表。
func BoardSummaries(boards []forummodels.Board, threadCounts, followerCounts map[uint]int64, followed map[uint]bool) []BoardSummary {
	items := make([]BoardSummary, 0, len(boards))
	for _, board := range boards {
		items = append(items, BoardSummary{
			ID:            board.ID,
			Slug:          board.Slug,
			Name:          board.Name,
			Description:   board.Description,
			Icon:          board.Icon,
			Color:         board.Color,
			ThreadCount:   threadCounts[board.ID],
			FollowerCount: followerCounts[board.ID],
			FollowingByMe: followed[board.ID],
		})
	}
	return items
}

// ThreadContext 是批量组装主题 DTO 需要的关联数据。
type ThreadContext struct {
	Views       map[uint]dao.UserView
	Boards      map[uint]forummodels.Board
	Tags        map[uint][]string
	ReplyCounts map[uint]int64
	LikeCounts  map[uint]int64
	Liked       map[uint]bool
}

// ThreadSummaries 批量构造主题卡片。
func ThreadSummaries(threads []forummodels.Thread, ctx ThreadContext) []ThreadSummary {
	items := make([]ThreadSummary, 0, len(threads))
	for _, thread := range threads {
		items = append(items, ThreadSummaryFrom(thread, ctx))
	}
	return items
}

// ThreadSummaryFrom 构造单个主题卡片。
func ThreadSummaryFrom(thread forummodels.Thread, ctx ThreadContext) ThreadSummary {
	board := ctx.Boards[thread.BoardID]
	tags := ctx.Tags[thread.ID]
	if tags == nil {
		tags = []string{}
	}

	return ThreadSummary{
		ID:         thread.ID,
		Board:      BoardBriefFrom(board),
		Author:     SummaryForID(thread.AuthorID, ctx.Views),
		Title:      thread.Title,
		Excerpt:    content.Excerpt(thread.ContentText, 160),
		Tags:       tags,
		Pinned:     thread.Pinned,
		Locked:     thread.Locked,
		ReplyCount: ctx.ReplyCounts[thread.ID],
		LikeCount:  ctx.LikeCounts[thread.ID],
		LikedByMe:  ctx.Liked[thread.ID],
		CreatedAt:  Timestamp(thread.CreatedAt),
		UpdatedAt:  Timestamp(thread.UpdatedAt),
	}
}

// ThreadDetailFrom 构造主题详情。
func ThreadDetailFrom(thread forummodels.Thread, ctx ThreadContext, canEdit, canDelete, canModerate bool) ThreadDetail {
	return ThreadDetail{
		ThreadSummary: ThreadSummaryFrom(thread, ctx),
		ContentHTML:   thread.ContentHTML,
		CanEdit:       canEdit,
		CanDelete:     canDelete,
		CanModerate:   canModerate,
	}
}

// ReplyContext 是批量组装跟帖 DTO 需要的关联数据与权限回调。
type ReplyContext struct {
	Views        map[uint]dao.UserView
	LikeCount    map[uint]int64
	Liked        map[uint]bool
	ThreadTitles map[uint]string
	CanEdit      func(reply forummodels.Reply) bool
	CanDelete    func(reply forummodels.Reply) bool
}

// Replies 批量构造跟帖 DTO；已删除跟帖返回占位。
func Replies(replies []forummodels.Reply, ctx ReplyContext) []Reply {
	items := make([]Reply, 0, len(replies))
	for _, reply := range replies {
		if reply.DeletedAt.Valid {
			items = append(items, Reply{
				ID:          reply.ID,
				ThreadID:    reply.ThreadID,
				ThreadTitle: ctx.ThreadTitles[reply.ThreadID],
				Floor:       reply.Floor,
				Author:      nil,
				Deleted:     true,
				CreatedAt:   Timestamp(reply.CreatedAt),
			})
			continue
		}

		var updatedAt *string
		if !reply.UpdatedAt.Equal(reply.CreatedAt) {
			updatedAt = TimestampPtr(&reply.UpdatedAt)
		}

		html := reply.ContentHTML
		item := Reply{
			ID:             reply.ID,
			ThreadID:       reply.ThreadID,
			ThreadTitle:    ctx.ThreadTitles[reply.ThreadID],
			Floor:          reply.Floor,
			Author:         SummaryForID(reply.AuthorID, ctx.Views),
			ContentHTML:    &html,
			ReplyToFloor:   reply.ReplyToFloor,
			ReplyToReplyID: reply.ReplyToReplyID,
			LikeCount:      ctx.LikeCount[reply.ID],
			LikedByMe:      ctx.Liked[reply.ID],
			Deleted:        false,
			CreatedAt:      Timestamp(reply.CreatedAt),
			UpdatedAt:      updatedAt,
		}
		if ctx.CanEdit != nil {
			item.CanEdit = ctx.CanEdit(reply)
		}
		if ctx.CanDelete != nil {
			item.CanDelete = ctx.CanDelete(reply)
		}
		items = append(items, item)
	}
	return items
}

// Notifications 批量构造通知 DTO。
func Notifications(rows []forummodels.Notification, views map[uint]dao.UserView, threadTitles map[uint]string) []Notification {
	items := make([]Notification, 0, len(rows))
	for _, row := range rows {
		items = append(items, Notification{
			ID:          row.ID,
			Type:        row.Type,
			Actor:       SummaryForID(row.ActorID, views),
			ThreadID:    row.ThreadID,
			ThreadTitle: threadTitles[row.ThreadID],
			ReplyID:     row.ReplyID,
			QuotedFloor: row.QuotedFloor,
			Excerpt:     row.Excerpt,
			Read:        row.ReadAt != nil,
			CreatedAt:   Timestamp(row.CreatedAt),
		})
	}
	return items
}

// AdminUserFrom 构造后台用户 DTO。
func AdminUserFrom(user models.User, view dao.UserView, threads, replies, likes int64, lastForum, lastAdmin *time.Time, grants []perms.Grant) AdminUser {
	if grants == nil {
		grants = []perms.Grant{}
	}
	return AdminUser{
		ID:             user.ID,
		Username:       user.Username,
		Nickname:       view.Nickname,
		Bio:            view.Bio,
		AvatarURL:      AvatarURL(view),
		AvatarShape:    avatarShape(view),
		Banned:         user.Banned,
		LabRole:        string(user.Role),
		CreatedAt:      Timestamp(user.CreatedAt),
		LastForumLogin: TimestampPtr(lastForum),
		LastAdminLogin: TimestampPtr(lastAdmin),
		ThreadCount:    threads,
		ReplyCount:     replies,
		LikeCount:      likes,
		Permissions:    grants,
	}
}

// AdminThreadFrom 构造后台主题 DTO。
func AdminThreadFrom(thread forummodels.Thread, views map[uint]dao.UserView, boards map[uint]forummodels.Board, replyCount, likeCount int64, withContent bool) AdminThread {
	board := boards[thread.BoardID]
	var deletedAt *string
	if thread.DeletedAt.Valid {
		deletedAt = TimestampPtr(&thread.DeletedAt.Time)
	}

	item := AdminThread{
		ID:          thread.ID,
		BoardID:     thread.BoardID,
		Board:       BoardBriefFrom(board),
		Author:      SummaryForID(thread.AuthorID, views),
		Title:       thread.Title,
		Excerpt:     content.Excerpt(thread.ContentText, 160),
		Pinned:      thread.Pinned,
		Locked:      thread.Locked,
		Deleted:     thread.DeletedAt.Valid,
		DeletedAt:   deletedAt,
		DeletedBy:   thread.DeletedBy,
		ReplyCount:  replyCount,
		LikeCount:   likeCount,
		CreatedAt:   Timestamp(thread.CreatedAt),
		UpdatedAt:   Timestamp(thread.UpdatedAt),
		LastReplyAt: TimestampPtr(thread.LastReplyAt),
	}
	if withContent {
		item.ContentHTML = thread.ContentHTML
	}
	return item
}

// AdminReplyFrom 构造后台跟帖 DTO。
func AdminReplyFrom(reply forummodels.Reply, views map[uint]dao.UserView, threadTitle string, likeCount int64, withContent bool) AdminReply {
	var deletedAt *string
	if reply.DeletedAt.Valid {
		deletedAt = TimestampPtr(&reply.DeletedAt.Time)
	}

	item := AdminReply{
		ID:          reply.ID,
		ThreadID:    reply.ThreadID,
		ThreadTitle: threadTitle,
		Floor:       reply.Floor,
		Author:      SummaryForID(reply.AuthorID, views),
		Excerpt:     content.Excerpt(reply.ContentText, 160),
		Deleted:     reply.DeletedAt.Valid,
		DeletedAt:   deletedAt,
		DeletedBy:   reply.DeletedBy,
		LikeCount:   likeCount,
		CreatedAt:   Timestamp(reply.CreatedAt),
		UpdatedAt:   Timestamp(reply.UpdatedAt),
	}
	if withContent {
		item.ContentHTML = reply.ContentHTML
	}
	return item
}
