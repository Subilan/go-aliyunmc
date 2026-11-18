// Package upload_routes 提供图片上传与静态文件服务。
package upload_routes

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/routeutil"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/service"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/uploads"

	"github.com/gin-gonic/gin"
	"github.com/jellydator/ttlcache/v3"
)

// fileMeta 是静态文件路由的缓存条目；Found=false 是负缓存，避免随机 key 反复查库。
type fileMeta struct {
	Mime  string
	Found bool
}

var fileMetaCache = ttlcache.New[string, fileMeta](
	ttlcache.WithTTL[string, fileMeta](10*time.Minute),
	ttlcache.WithCapacity[string, fileMeta](16384),
)

func init() {
	go fileMetaCache.Start()
}

func lookupFileMeta(key string) fileMeta {
	if item := fileMetaCache.Get(key); item != nil {
		return item.Value()
	}

	var meta fileMeta
	if upload, err := dao.UploadByKey(key, false); err == nil {
		meta = fileMeta{Mime: upload.Mime, Found: true}
	}
	fileMetaCache.Set(key, meta, ttlcache.DefaultTTL)
	return meta
}

// Bind 注册上传与文件路由。
func Bind(router *gin.RouterGroup) {
	router.POST("/uploads", mid.Auth(), mid.RequirePermission("upload.create"),
		mid.RateLimitByUser("upload", routeutil.ForumRate(config.Get().Forum.RateLimit.UploadPerMinute)), h.G(createUpload))
	router.GET("/files/*key", serveFile)
}

func createUpload(c *gin.Context) (any, error) {
	limit := uploads.MaxRequestBytes()
	defer func() {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
	}()

	// 先按最大可能的请求体限制读取，具体用途的上限在 service 内再校验。
	bodyLimit := limit + 1<<20
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, bodyLimit)

	if err := c.Request.ParseMultipartForm(bodyLimit); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return nil, h.HttpError(http.StatusRequestEntityTooLarge, "上传内容过大")
		}
		return nil, h.HttpError(http.StatusBadRequest, "无法解析上传请求")
	}

	kind := c.PostForm("kind")
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return nil, h.HttpError(http.StatusBadRequest, "缺少 file 字段")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, h.HttpError(http.StatusBadRequest, "无法读取上传文件")
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, h.HttpError(http.StatusRequestEntityTooLarge, "上传内容过大")
	}

	user, _ := mid.CurrentUser(c)
	return service.CreateUpload(user, kind, fileHeader.Filename, data)
}

func serveFile(c *gin.Context) {
	key := strings.TrimPrefix(c.Param("key"), "/")
	// Path 同时完成 key 正则校验与路径穿越防护。
	path, err := uploads.Path(key)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	meta := lookupFileMeta(key)
	if !meta.Found {
		c.Status(http.StatusNotFound)
		return
	}

	c.Header("Content-Type", meta.Mime)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", "inline")
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.File(path)
}
