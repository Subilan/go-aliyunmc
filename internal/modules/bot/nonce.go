package bot

import (
	"time"

	"github.com/jellydator/ttlcache/v3"
)

// nonceCache 使用 TTL 缓存做防重放：同一 nonce 在有效期内只能出现一次。
type nonceCache struct {
	items *ttlcache.Cache[string, struct{}]
}

func newNonceCache(ttl time.Duration, capacity int) *nonceCache {
	items := ttlcache.New[string, struct{}](
		ttlcache.WithTTL[string, struct{}](ttl),
		ttlcache.WithCapacity[string, struct{}](uint64(capacity)),
		ttlcache.WithDisableTouchOnHit[string, struct{}](),
	)
	go items.Start()

	return &nonceCache{items: items}
}

// CheckAndStore 接受首次出现的 key，拒绝重复使用。
func (c *nonceCache) CheckAndStore(key string) bool {
	_, found := c.items.GetOrSet(key, struct{}{})
	return !found
}

// Stop 停止后台清理协程，供测试使用。
func (c *nonceCache) Stop() {
	c.items.Stop()
}
