package logger

import "sync"

// RingBuffer is a fixed-size ring buffer for log entries.
// Thread-safe: supports concurrent Add and Last calls.
type RingBuffer struct {
	entries []LogEntry
	pos     int
	size    int
	mu      sync.Mutex
}

// NewRingBuffer creates a ring buffer with the given capacity.
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		entries: make([]LogEntry, size),
		size:    size,
	}
}

// Add appends an entry to the buffer, overwriting the oldest if full.
func (r *RingBuffer) Add(e LogEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[r.pos] = e
	r.pos = (r.pos + 1) % r.size
}

// Last returns the last n entries in chronological order.
func (r *RingBuffer) Last(n int) []LogEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	if n > r.size {
		n = r.size
	}

	result := make([]LogEntry, 0, n)
	start := (r.pos - n + r.size) % r.size
	for i := 0; i < n; i++ {
		idx := (start + i) % r.size
		if r.entries[idx].Timestamp.IsZero() {
			continue
		}
		result = append(result, r.entries[idx])
	}
	return result
}

// Len returns the number of entries currently in the buffer.
func (r *RingBuffer) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for i := 0; i < r.size; i++ {
		if !r.entries[i].Timestamp.IsZero() {
			count++
		}
	}
	return count
}
