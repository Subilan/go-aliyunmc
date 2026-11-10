// Package cache 提供带 TTL 与 singleflight 的计算结果缓存。
package cache

import (
	"time"

	"github.com/jellydator/ttlcache/v3"
	"golang.org/x/sync/singleflight"
)

// Store 以字符串为键缓存计算结果。取出的值会与缓存共享，调用方不得修改。
type Store[V any] struct {
	items *ttlcache.Cache[string, V]
	group singleflight.Group
}

// New 创建一个 Store。ttl 是条目的存活时间，capacity 是最大条目数，0 表示不限制。
func New[V any](ttl time.Duration, capacity uint64) *Store[V] {
	items := ttlcache.New[string, V](
		ttlcache.WithTTL[string, V](ttl),
		ttlcache.WithCapacity[string, V](capacity),
		ttlcache.WithDisableTouchOnHit[string, V](),
	)
	go items.Start()

	return &Store[V]{items: items}
}

// GetOrLoad 返回 key 对应的缓存值，缺失时调用 load 计算并写入缓存。
// 同一 key 的并发调用只会执行一次 load，其余调用等待并共享结果。load 返回错误时不写入缓存。
func (s *Store[V]) GetOrLoad(key string, load func() (V, error)) (V, error) {
	if item := s.items.Get(key); item != nil {
		return item.Value(), nil
	}

	value, err, _ := s.group.Do(key, func() (any, error) {
		if item := s.items.Get(key); item != nil {
			return item.Value(), nil
		}

		loaded, err := load()
		if err != nil {
			return nil, err
		}

		s.items.Set(key, loaded, ttlcache.DefaultTTL)
		return loaded, nil
	})
	if err != nil {
		var zero V
		return zero, err
	}

	return value.(V), nil
}
