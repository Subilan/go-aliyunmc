package mid

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/context_util"
	"github.com/Subilan/go-aliyunmc/internal/common/h"

	"github.com/gin-gonic/gin"
	"github.com/jellydator/ttlcache/v3"
	"golang.org/x/time/rate"
)

// 限流器按 key 缓存，空闲 10 分钟后过期；容量上限防止轮换 key 时无限增长。
const (
	limiterTTL      = 10 * time.Minute
	limiterCapacity = 16384
)

var limiters = ttlcache.New[string, *rate.Limiter](
	ttlcache.WithTTL[string, *rate.Limiter](limiterTTL),
	ttlcache.WithCapacity[string, *rate.Limiter](limiterCapacity),
)

func init() {
	go limiters.Start()
}

// Hit 记一次限流事件。perMinute<=0 表示不限制。
func Hit(name string, perMinute int, key string) bool {
	if perMinute <= 0 {
		return true
	}

	fullKey := name + ":" + key
	if item := limiters.Get(fullKey); item != nil {
		return item.Value().Allow()
	}

	limiter := rate.NewLimiter(rate.Limit(float64(perMinute)/60.0), perMinute)
	limiters.Set(fullKey, limiter, ttlcache.DefaultTTL)
	return limiter.Allow()
}

// rateLimitKey 已登录时按用户限流，否则按客户端 IP。
func rateLimitKey(c *gin.Context) string {
	if userID, ok := context_util.GetUserID(c); ok {
		return "user:" + strconv.FormatUint(uint64(userID), 10)
	}
	return c.ClientIP()
}

// RateLimitByUser 按当前用户限流；未登录时按客户端 IP。
func RateLimitByUser(name string, perMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !Hit(name, perMinute, rateLimitKey(c)) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, h.DetailsF("请求过于频繁，请稍后再试"))
			return
		}
		c.Next()
	}
}

// RateLimitAdminWrites 仅对后台写操作限流。
func RateLimitAdminWrites(perMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		if !Hit("admin-write", perMinute, rateLimitKey(c)) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, h.DetailsF("请求过于频繁，请稍后再试"))
			return
		}
		c.Next()
	}
}
