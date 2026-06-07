package core

import (
	"sync"
	"sync/atomic"
	"time"
)

// cachedDecision stores a blocking decision with a timestamp and version.
type cachedDecision struct {
	blocked bool
	ts      time.Time
	version uint64
}

// DecisionCache is a fixed-size LRU cache for IP blocking decisions.
// Uses versioning for O(1) invalidation: Clear() increments a version counter
// instead of rebuilding the map. Entries with stale versions are ignored by Get().
type DecisionCache struct {
	mu      sync.RWMutex
	entries map[uint32]cachedDecision
	lru     []uint32
	maxSize int
	pos     int
	ttl     time.Duration
	version atomic.Uint64
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

// Get returns the cached decision if present, not expired, and from the current version.
func (c *DecisionCache) Get(ip uint32) (blocked bool, ok bool) {
	currentVersion := c.version.Load()
	c.mu.RLock()
	defer c.mu.RUnlock()
	if d, found := c.entries[ip]; found {
		if d.version == currentVersion && time.Since(d.ts) < c.ttl {
			return d.blocked, true
		}
	}
	return false, false
}

// Set stores a decision in the cache with the current version.
func (c *DecisionCache) Set(ip uint32, blocked bool) {
	currentVersion := c.version.Load()
	c.mu.Lock()
	defer c.mu.Unlock()
	if old := c.lru[c.pos]; old != 0 {
		delete(c.entries, old)
	}
	c.entries[ip] = cachedDecision{
		blocked: blocked,
		ts:      time.Now(),
		version: currentVersion,
	}
	c.lru[c.pos] = ip
	c.pos = (c.pos + 1) % c.maxSize
}

// Len returns the current number of cached entries.
func (c *DecisionCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Clear invalidates all entries in O(1) by incrementing the version counter.
// Stale entries are ignored by Get() and get naturally overwritten by Set() via LRU.
func (c *DecisionCache) Clear() {
	c.version.Add(1)
	// Map and LRU slice are not cleared — old entries will be ignored by Get()
	// because their version doesn't match the current one. They'll be evicted
	// naturally as new entries are added via the LRU mechanism.
}
