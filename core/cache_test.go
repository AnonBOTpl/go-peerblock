package core

import (
	"sync"
	"testing"
	"time"
)

// ─── Basic Set/Get Tests ───────────────────────────────

func TestCache_SetAndGet(t *testing.T) {
	c := NewDecisionCache(100, 5*time.Minute)

	// Set a value
	c.Set(0x01010101, true)
	blocked, ok := c.Get(0x01010101)
	if !ok {
		t.Error("expected entry to be found")
	}
	if !blocked {
		t.Error("expected blocked=true")
	}

	// Set another value
	c.Set(0x08080808, false)
	blocked, ok = c.Get(0x08080808)
	if !ok {
		t.Error("expected entry to be found")
	}
	if blocked {
		t.Error("expected blocked=false")
	}
}

func TestCache_MissingEntry(t *testing.T) {
	c := NewDecisionCache(100, 5*time.Minute)
	_, ok := c.Get(0xDEADBEEF)
	if ok {
		t.Error("expected missing entry to return ok=false")
	}
}

// ─── LRU Eviction Tests ────────────────────────────────

func TestCache_LRUEviction(t *testing.T) {
	c := NewDecisionCache(3, 5*time.Minute)

	// Fill the cache
	c.Set(1, true)
	c.Set(2, false)
	c.Set(3, true)

	// All should be present
	for _, ip := range []uint32{1, 2, 3} {
		if _, ok := c.Get(ip); !ok {
			t.Errorf("expected entry %d to exist before eviction", ip)
		}
	}

	// Add a 4th entry, should evict IP 1 (oldest)
	c.Set(4, false)

	if _, ok := c.Get(1); ok {
		t.Error("expected entry 1 to be evicted (oldest)")
	}
	if _, ok := c.Get(2); !ok {
		t.Error("expected entry 2 to still exist")
	}
	if _, ok := c.Get(3); !ok {
		t.Error("expected entry 3 to still exist")
	}
	if _, ok := c.Get(4); !ok {
		t.Error("expected entry 4 to exist")
	}
}

func TestCache_EvictionOrder(t *testing.T) {
	c := NewDecisionCache(2, 5*time.Minute)

	c.Set(1, true)
	c.Set(2, true)
	c.Set(3, true) // evicts 1

	// 2 should still be there (second added)
	if _, ok := c.Get(2); !ok {
		t.Error("expected entry 2 to survive eviction")
	}
}

// ─── TTL Tests ──────────────────────────────────────────

func TestCache_TTLExpiry(t *testing.T) {
	c := NewDecisionCache(100, 50*time.Millisecond)

	c.Set(42, true)

	// Should be found immediately
	if _, ok := c.Get(42); !ok {
		t.Error("expected entry to be found immediately")
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	if _, ok := c.Get(42); ok {
		t.Error("expected entry to be expired after TTL")
	}
}

func TestCache_ZeroTTL(t *testing.T) {
	c := NewDecisionCache(100, 0)

	c.Set(42, true)

	// With zero TTL, time.Since(d.ts) < 0 is always false
	if _, ok := c.Get(42); ok {
		t.Error("expected entry with zero TTL to be considered expired")
	}
}

// ─── Len/Clear Tests ────────────────────────────────────

func TestCache_Len(t *testing.T) {
	c := NewDecisionCache(100, 5*time.Minute)

	if c.Len() != 0 {
		t.Errorf("expected empty cache to have len 0, got %d", c.Len())
	}

	c.Set(1, true)
	c.Set(2, false)
	if c.Len() != 2 {
		t.Errorf("expected len 2, got %d", c.Len())
	}
}

func TestCache_Clear(t *testing.T) {
	c := NewDecisionCache(100, 5*time.Minute)

	c.Set(1, true)
	c.Set(2, false)
	c.Clear()

	// After Clear() entries are invisible (different version), but map is not cleared
	if _, ok := c.Get(1); ok {
		t.Error("expected no entry visible after Clear()")
	}
	if _, ok := c.Get(2); ok {
		t.Error("expected no entry visible after Clear()")
	}

	// New entries after Clear() work normally
	c.Set(3, true)
	blocked, ok := c.Get(3)
	if !ok {
		t.Error("expected new entry to be visible after Clear()")
	}
	if !blocked {
		t.Error("expected blocked=true for new entry")
	}
}

func TestCache_ClearVersioning(t *testing.T) {
	c := NewDecisionCache(100, 5*time.Minute)

	c.Set(1, true)
	c.Clear()

	// Stale entry invisible after Clear()
	if _, ok := c.Get(1); ok {
		t.Error("expected stale entry to be invisible after Clear()")
	}

	// New entry after Clear() works normally
	c.Set(1, false)
	blocked, ok := c.Get(1)
	if !ok {
		t.Error("expected new entry to be visible after Clear()")
	}
	if blocked {
		t.Error("expected blocked=false for new entry")
	}
}

// ─── Concurrent Access Tests ────────────────────────────

func TestCache_ConcurrentAccess(t *testing.T) {
	c := NewDecisionCache(1000, 5*time.Minute)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n uint32) {
			defer wg.Done()
			for j := uint32(0); j < 100; j++ {
				c.Set(n*1000+j, j%2 == 0)
			}
		}(uint32(i))
	}
	wg.Wait()

	if c.Len() == 0 {
		t.Error("expected some entries after concurrent writes")
	}
}

func TestCache_ConcurrentReadWrite(t *testing.T) {
	c := NewDecisionCache(1000, 5*time.Minute)

	// Pre-fill some entries
	for i := uint32(0); i < 100; i++ {
		c.Set(i, true)
	}

	var wg sync.WaitGroup

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := uint32(0); j < 1000; j++ {
				c.Get(j)
			}
		}()
	}

	// Concurrent writes
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := uint32(0); j < 500; j++ {
				c.Set(j, j%2 == 0)
			}
		}()
	}

	wg.Wait()
}

// ─── Edge Cases ─────────────────────────────────────────

func TestCache_OverwriteExistingEntry(t *testing.T) {
	c := NewDecisionCache(100, 5*time.Minute)

	c.Set(1, true)
	c.Set(1, false) // overwrite

	blocked, ok := c.Get(1)
	if !ok {
		t.Error("expected entry to exist after overwrite")
	}
	if blocked {
		t.Error("expected blocked=false after overwrite")
	}
}

func TestCache_SmallCache(t *testing.T) {
	c := NewDecisionCache(1, 5*time.Minute)

	c.Set(1, true)
	c.Set(2, false) // evicts 1

	if _, ok := c.Get(1); ok {
		t.Error("expected entry 1 to be evicted from size-1 cache")
	}
	blocked, ok := c.Get(2)
	if !ok {
		t.Error("expected entry 2 to exist")
	}
	if blocked {
		t.Error("expected blocked=false for entry 2")
	}
}

// ─── Benchmark ──────────────────────────────────────────

func BenchmarkCacheSet(b *testing.B) {
	c := NewDecisionCache(65536, 5*time.Minute)
	ips := generateRandomIPs(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set(ips[i%len(ips)], i%2 == 0)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	c := NewDecisionCache(65536, 5*time.Minute)
	ips := generateRandomIPs(10000)
	for i := 0; i < 10000; i++ {
		c.Set(ips[i], i%2 == 0)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(ips[i%len(ips)])
	}
}
