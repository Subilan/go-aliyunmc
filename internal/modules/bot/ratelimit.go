package bot

import (
	"net/http"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/h"

	"github.com/gin-gonic/gin"
	"github.com/jellydator/ttlcache/v3"
	"golang.org/x/time/rate"
)

const (
	limiterTTL      = 10 * time.Minute
	limiterCapacity = 16
)

// rateLimiter 按 key_id 做进程内限流。
type rateLimiter struct {
	perMinute int
	limiters  *ttlcache.Cache[string, *rate.Limiter]
}

func newRateLimiter(perMinute int) *rateLimiter {
	limiters := ttlcache.New[string, *rate.Limiter](
		ttlcache.WithTTL[string, *rate.Limiter](limiterTTL),
		ttlcache.WithCapacity[string, *rate.Limiter](limiterCapacity),
	)
	go limiters.Start()

	return &rateLimiter{perMinute: perMinute, limiters: limiters}
}

func (r *rateLimiter) Allow(key string) bool {
	if r == nil || r.perMinute <= 0 {
		return true
	}

	if item := r.limiters.Get(key); item != nil {
		return item.Value().Allow()
	}

	limiter := rate.NewLimiter(rate.Limit(float64(r.perMinute)/60.0), r.perMinute)
	r.limiters.Set(key, limiter, ttlcache.DefaultTTL)
	return limiter.Allow()
}

// Stop 停止后台清理协程，供测试使用。
func (r *rateLimiter) Stop() {
	r.limiters.Stop()
}

// RateLimit 按机器人 key_id 限流。
func RateLimit(rt *botRuntime) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rt.limiter.Allow(rt.keyID) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, h.DetailsF("请求过于频繁，请稍后再试"))
			return
		}
		c.Next()
	}
}
