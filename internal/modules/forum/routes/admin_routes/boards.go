package admin_routes

import (
	"strconv"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/routeutil"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/service"

	"github.com/gin-gonic/gin"
)

type boardBody struct {
	Slug        *string `json:"slug"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	Color       *string `json:"color"`
	SortOrder   *int    `json:"sortOrder"`
	Enabled     *bool   `json:"enabled"`
}

type reorderRequest struct {
	IDs []uint `json:"ids" binding:"required"`
}

func bindBoards(admin *gin.RouterGroup) {
	// 板块级/全局权限统一由 service 检查，避免 middleware 与 service 双检。
	admin.GET("/boards", h.G(listBoards))
	admin.POST("/boards", h.B(createBoard))
	admin.POST("/boards/reorder", h.B(reorderBoards))
	admin.PATCH("/boards/:id", h.B(updateBoard))
	admin.DELETE("/boards/:id", h.G(deleteBoard))
	admin.POST("/boards/:id/restore", h.G(restoreBoard))
}

func listBoards(c *gin.Context) (any, error) {
	includeDisabled := c.Query("includeDisabled") == "true"
	includeDeleted := c.Query("includeDeleted") == "true"
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	user, _ := mid.CurrentUser(c)
	return service.AdminListBoards(user, includeDisabled, includeDeleted, page, pageSize)
}

func createBoard(req boardBody, c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	board, err := service.CreateBoard(user, toBoardInput(req), c)
	if err != nil {
		return nil, err
	}
	item := service.AdminBoardDTO(board)
	return item, nil
}

func updateBoard(req boardBody, c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	board, err := service.UpdateBoard(user, id, toBoardInput(req), c)
	if err != nil {
		return nil, err
	}
	item := service.AdminBoardDTO(board)
	return item, nil
}

func deleteBoard(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	if err := service.DeleteBoard(user, id, c); err != nil {
		return nil, err
	}
	return nil, nil
}

func restoreBoard(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	board, err := service.RestoreBoard(user, id, c)
	if err != nil {
		return nil, err
	}
	item := service.AdminBoardDTO(board)
	return item, nil
}

func reorderBoards(req reorderRequest, c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	if err := service.ReorderBoards(user, req.IDs, c); err != nil {
		return nil, err
	}
	return nil, nil
}

func toBoardInput(req boardBody) service.BoardInput {
	return service.BoardInput{
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		Color:       req.Color,
		SortOrder:   req.SortOrder,
		Enabled:     req.Enabled,
	}
}
