package gateway

import (
	"sync"
	"time"
)

type idempotencyRecord struct {
	response ResponseFrame
	expires  time.Time
}

type idempotencyCache struct {
	ttl   time.Duration
	mu    sync.Mutex
	items map[string]idempotencyRecord
}

func newIdempotencyCache(ttl time.Duration) *idempotencyCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &idempotencyCache{
		ttl:   ttl,
		items: make(map[string]idempotencyRecord),
	}
}

func (c *idempotencyCache) Get(key string, now time.Time) (ResponseFrame, bool) {
	if c == nil || key == "" {
		return ResponseFrame{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cleanupLocked(now)
	item, ok := c.items[key]
	if !ok {
		return ResponseFrame{}, false
	}
	return item.response, true
}

func (c *idempotencyCache) Put(key string, response ResponseFrame, now time.Time) {
	if c == nil || key == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cleanupLocked(now)
	c.items[key] = idempotencyRecord{response: response, expires: now.Add(c.ttl)}
}

func (c *idempotencyCache) cleanupLocked(now time.Time) {
	if now.IsZero() {
		now = time.Now()
	}
	for key, item := range c.items {
		if !item.expires.IsZero() && item.expires.Before(now) {
			delete(c.items, key)
		}
	}
}
