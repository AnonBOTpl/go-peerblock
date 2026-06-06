package core

import (
	"sync"
	"time"
)

// cachedDecision stores a blocking decision with a timestamp.
type cachedDecision struct {
	blocked bool
	ts      time.Time
}

// DecisionCache is a fixed-size LRU cache for IP blocking decisions.
type DecisionCache struct {
	mu      sync.RWMutex
	entries map[uint32]cachedDecision
	lru     []uint32
	maxSize int
	pos     int
	ttl     time.Duration
}

// NewDecisionCache creates a new cache with the given size and TTL.
func NewDecisionCache(maxSize int, ttl time.Duration) *DecisionCache {
	return &DecisionCache{
		entries: make(map[uint32]cachedDecision),
		lru:     make([]uint32, maxSize),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Get returns the cached decision if present and not expired.
func (c *DecisionCache) Get(ip uint32) (blocked bool, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if d, found := c.entries[ip]; found {
		if time.Since(d.ts) < c.ttl {
			return d.blocked, true
		}
	}
	return false, false
}

// Set stores a decision in the cache.
func (c *DecisionCache) Set(ip uint32, blocked bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if old := c.lru[c.pos]; old != 0 {
		delete(c.entries, old)
	}
	c.entries[ip] = cachedDecision{blocked: blocked, ts: time.Now()}
	c.lru[c.pos] = ip
	c.pos = (c.pos + 1) % c.maxSize
}

// Len returns the current number of cached entries.
func (c *DecisionCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Clear removes all entries from the cache.
func (c *DecisionCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[uint32]cachedDecision)
	c.lru = make([]uint32, c.maxSize)
	c.pos = 0
}
