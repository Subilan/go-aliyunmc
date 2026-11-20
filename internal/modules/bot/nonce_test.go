package bot

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNonceCacheAcceptsOnce(t *testing.T) {
	cache := newNonceCache(time.Minute, 16)
	defer cache.Stop()

	if !cache.CheckAndStore("default:abcdefghijklmnop") {
		t.Fatal("首次使用应当通过")
	}
	if cache.CheckAndStore("default:abcdefghijklmnop") {
		t.Fatal("重复使用应当被拒绝")
	}
}

func TestNonceCacheExpires(t *testing.T) {
	cache := newNonceCache(20*time.Millisecond, 16)
	defer cache.Stop()

	if !cache.CheckAndStore("key") {
		t.Fatal("首次使用应当通过")
	}
	time.Sleep(50 * time.Millisecond)
	if !cache.CheckAndStore("key") {
		t.Fatal("TTL 过期后应当重新可用")
	}
}

func TestNonceCacheConcurrent(t *testing.T) {
	cache := newNonceCache(time.Minute, 16)
	defer cache.Stop()

	var accepted atomic.Int64
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			if cache.CheckAndStore("same") {
				accepted.Add(1)
			}
		})
	}
	wg.Wait()

	if accepted.Load() != 1 {
		t.Fatalf("并发请求应只有一个被接受，实际 %d 个", accepted.Load())
	}
}
