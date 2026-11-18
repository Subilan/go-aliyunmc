package admin_routes

import (
	"net/http"
	"strconv"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/routeutil"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/service"

	"github.com/gin-gonic/gin"
)

type batchRequest struct {
	IDs     []uint `json:"ids" binding:"required"`
	Action  string `json:"action" binding:"required"`
	BoardID *uint  `json:"boardId"`
	Reason  string `json:"reason"`
}

type updateAdminThreadRequest struct {
	Pinned  *bool `json:"pinned"`
	Locked  *bool `json:"locked"`
	BoardID *uint `json:"boardId"`
}

type updateAdminReplyRequest struct {
	ContentHTML string `json:"contentHtml" binding:"required"`
}

func bindContent(admin *gin.RouterGroup) {
	admin.GET("/threads", h.G(listAdminThreads))
	admin.POST("/threads/batch", h.B(batchAdminThreads))
	admin.GET("/threads/:id", h.G(adminThreadDetail))
	admin.PATCH("/threads/:id", h.B(updateAdminThread))
	admin.DELETE("/threads/:id", h.G(deleteAdminThread))
	admin.POST("/threads/:id/restore", h.G(restoreAdminThread))

	admin.GET("/replies", h.G(listAdminReplies))
	admin.POST("/replies/batch", h.B(batchAdminReplies))
	admin.PATCH("/replies/:id", h.B(updateAdminReply))
	admin.DELETE("/replies/:id", h.G(deleteAdminReply))
	admin.POST("/replies/:id/restore", h.G(restoreAdminReply))
}

func listAdminThreads(c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)

	boardID, _ := strconv.ParseUint(c.DefaultQuery("boardId", "0"), 10, 64)
	authorID, _ := strconv.ParseUint(c.DefaultQuery("authorId", "0"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	pinned, err := optionalBool(c.Query("pinned"))
	if err != nil {
		return nil, h.HttpError(http.StatusBadRequest, "pinned 必须是布尔值")
	}
	locked, err := optionalBool(c.Query("locked"))
	if err != nil {
		return nil, h.HttpError(http.StatusBadRequest, "locked 必须是布尔值")
	}

	return service.AdminListThreads(user, service.AdminThreadFilter{
		BoardID:  uint(boardID),
		AuthorID: uint(authorID),
		Status:   c.Query("status"),
		Pinned:   pinned,
		Locked:   locked,
		Keyword:  c.Query("keyword"),
		Sort:     c.Query("sort"),
		Page:     page,
		PageSize: pageSize,
	})
}

func adminThreadDetail(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	return service.AdminThreadDetail(user, id)
}

func updateAdminThread(req updateAdminThreadRequest, c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	thread, err := service.AdminUpdateThread(user, id, service.AdminUpdateThreadInput{
		Pinned:  req.Pinned,
		Locked:  req.Locked,
		BoardID: req.BoardID,
	}, c)
	if err != nil {
		return nil, err
	}
	return service.AdminThreadDTOFromThread(*thread), nil
}

func deleteAdminThread(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	if err := service.DeleteThread(user, id, c.Query("reason"), true, c); err != nil {
		return nil, err
	}
	return nil, nil
}

func restoreAdminThread(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	thread, err := service.RestoreThread(user, id, c)
	if err != nil {
		return nil, err
	}
	return service.AdminThreadDTOFromThread(*thread), nil
}

func batchAdminThreads(req batchRequest, c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	return service.AdminBatchThreads(user, req.IDs, req.Action, req.BoardID, req.Reason, c)
}

func listAdminReplies(c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)

	threadID, _ := strconv.ParseUint(c.DefaultQuery("threadId", "0"), 10, 64)
	authorID, _ := strconv.ParseUint(c.DefaultQuery("authorId", "0"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	return service.AdminListReplies(user, uint(threadID), uint(authorID), c.Query("status"), c.Query("keyword"), page, pageSize)
}

func updateAdminReply(req updateAdminReplyRequest, c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	reply, err := service.UpdateReply(user, id, req.ContentHTML, true, c)
	if err != nil {
		return nil, err
	}
	return service.AdminReplyDTOFromReply(*reply), nil
}

func deleteAdminReply(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	if err := service.DeleteReply(user, id, c.Query("reason"), true, c); err != nil {
		return nil, err
	}
	return nil, nil
}

func restoreAdminReply(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	reply, err := service.RestoreReply(user, id, c)
	if err != nil {
		return nil, err
	}
	return service.AdminReplyDTOFromReply(*reply), nil
}

func batchAdminReplies(req batchRequest, c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	return service.AdminBatchReplies(user, req.IDs, req.Action, req.Reason, c)
}

func optionalBool(value string) (*bool, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
