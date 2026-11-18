// Package user_routes 提供用户主页、提及候选与用户自助资料接口。
package user_routes

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dto"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/routeutil"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/service"

	"github.com/gin-gonic/gin"
)

type pageQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"pageSize"`
}

type searchQuery struct {
	Query string `form:"q"`
	Limit int    `form:"limit"`
}

// uploadBindingRequest 用 RawMessage 区分“字段缺失”与“显式置空”。
type uploadBindingRequest struct {
	UploadID json.RawMessage `json:"uploadId"`
}

func (req uploadBindingRequest) uploadID() (*uint, error) {
	raw := bytes.TrimSpace(req.UploadID)
	if len(raw) == 0 {
		return nil, h.HttpError(http.StatusBadRequest, "缺少 uploadId 字段")
	}
	if string(raw) == "null" {
		return nil, nil
	}

	var id uint
	if err := json.Unmarshal(raw, &id); err != nil {
		return nil, h.HttpError(http.StatusBadRequest, "uploadId 必须是数字或 null")
	}
	return &id, nil
}

type avatarShapeRequest struct {
	Shape string `json:"shape" binding:"required"`
}

// profilePatch 用 RawMessage 区分“未提供”与“显式置空”。
type profilePatch struct {
	Nickname json.RawMessage `json:"nickname"`
	Bio      json.RawMessage `json:"bio"`
}

// Bind 注册用户路由。
func Bind(router *gin.RouterGroup) {
	router.GET("/users/search", mid.Auth(),
		mid.RateLimitByUser("mention-search", routeutil.ForumRate(config.Get().Forum.RateLimit.MentionSearchPerMinute)),
		h.Q(searchUsers))
	router.GET("/users/:id", mid.OptionalAuth(), mid.OptionalRead(), h.G(getUser))
	router.GET("/users/:id/threads", mid.OptionalAuth(), mid.OptionalRead(), h.Q(getUserThreads))
	router.GET("/users/:id/replies", mid.OptionalAuth(), mid.OptionalRead(), h.Q(getUserReplies))

	router.PUT("/users/:id/like", mid.Auth(), mid.RequirePermission("like.create"),
		mid.RateLimitByUser("user-like", routeutil.ForumRate(config.Get().Forum.RateLimit.LikePerMinute)), h.G(likeUser))
	router.DELETE("/users/:id/like", mid.Auth(), mid.RequirePermission("like.create"),
		mid.RateLimitByUser("user-like", routeutil.ForumRate(config.Get().Forum.RateLimit.LikePerMinute)), h.G(unlikeUser))

	router.PATCH("/me/profile", mid.Auth(), mid.RequirePermission("profile.update_own"), h.B(updateProfile))
	router.PUT("/me/avatar", mid.Auth(), mid.RequirePermission("profile.update_own"), h.B(updateAvatar))
	router.PUT("/me/background", mid.Auth(), mid.RequirePermission("profile.update_own"), h.B(updateBackground))
	router.PUT("/me/avatar-shape", mid.Auth(), mid.RequirePermission("profile.update_own"), h.B(updateAvatarShape))
}

func searchUsers(query searchQuery, c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	items, err := service.MentionCandidates(query.Query, user.ID, query.Limit)
	if err != nil {
		return nil, err
	}
	return gin.H{"items": items}, nil
}

func getUser(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	viewer, _ := mid.CurrentUser(c)
	return service.UserProfile(id, viewer)
}

func getUserThreads(query pageQuery, c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	viewer, _ := mid.CurrentUser(c)
	return service.UserThreads(id, query.Page, query.PageSize, viewer)
}

func getUserReplies(query pageQuery, c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	viewer, _ := mid.CurrentUser(c)
	return service.UserReplies(id, query.Page, query.PageSize, viewer)
}

func likeUser(c *gin.Context) (any, error) {
	return setUserLike(c, true)
}

func unlikeUser(c *gin.Context) (any, error) {
	return setUserLike(c, false)
}

func setUserLike(c *gin.Context, like bool) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	liked, count, err := service.LikeUser(user, id, like)
	if err != nil {
		return nil, err
	}
	return dto.LikeResult{Liked: liked, LikeCount: count}, nil
}

func updateProfile(req profilePatch, c *gin.Context) (any, error) {
	update := service.ProfileUpdate{}
	if req.Nickname != nil {
		update.NicknameSet = true
		if string(req.Nickname) != "null" {
			var nickname string
			if err := json.Unmarshal(req.Nickname, &nickname); err != nil {
				return nil, h.HttpError(http.StatusBadRequest, "nickname 必须是字符串或 null")
			}
			update.Nickname = &nickname
		}
	}
	if req.Bio != nil {
		update.BioSet = true
		if err := json.Unmarshal(req.Bio, &update.Bio); err != nil {
			return nil, h.HttpError(http.StatusBadRequest, "bio 必须是字符串")
		}
	}

	user, _ := mid.CurrentUser(c)
	if err := service.UpdateProfile(user, update); err != nil {
		return nil, err
	}
	return service.UserProfile(user.ID, user)
}

func updateAvatar(req uploadBindingRequest, c *gin.Context) (any, error) {
	uploadID, err := req.uploadID()
	if err != nil {
		return nil, err
	}

	user, _ := mid.CurrentUser(c)
	if err := service.BindAvatar(user, uploadID); err != nil {
		return nil, err
	}
	return service.UserProfile(user.ID, user)
}

func updateBackground(req uploadBindingRequest, c *gin.Context) (any, error) {
	uploadID, err := req.uploadID()
	if err != nil {
		return nil, err
	}

	user, _ := mid.CurrentUser(c)
	if err := service.BindBackground(user, uploadID); err != nil {
		return nil, err
	}
	return service.UserProfile(user.ID, user)
}

func updateAvatarShape(req avatarShapeRequest, c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	if err := service.SetAvatarShape(user, req.Shape); err != nil {
		return nil, err
	}
	return service.UserProfile(user.ID, user)
}
