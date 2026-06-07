package logger

import (
	"sync"
	"testing"
	"time"
)

func ts() time.Time { return time.Now() }

// ─── Podstawowe operacje ────────────────────────────────

func TestRingBuffer_Empty(t *testing.T) {
	r := NewRingBuffer(10)
	if r.Len() != 0 {
		t.Errorf("expected empty buffer to have len 0, got %d", r.Len())
	}
	last := r.Last(5)
	if len(last) != 0 {
		t.Errorf("expected 0 entries from empty buffer, got %d", len(last))
	}
}

func TestRingBuffer_AddAndRetrieve(t *testing.T) {
	r := NewRingBuffer(10)

	r.Add(LogEntry{Timestamp: ts(), Level: INFO, Message: "test 1"})
	r.Add(LogEntry{Timestamp: ts(), Level: WARN, Message: "test 2"})

	if r.Len() != 2 {
		t.Errorf("expected len 2, got %d", r.Len())
	}

	last := r.Last(10)
	if len(last) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(last))
	}
	if last[0].Message != "test 1" {
		t.Errorf("expected first entry 'test 1', got '%s'", last[0].Message)
	}
	if last[1].Message != "test 2" {
		t.Errorf("expected second entry 'test 2', got '%s'", last[1].Message)
	}
}

func TestRingBuffer_ChronologicalOrder(t *testing.T) {
	r := NewRingBuffer(5)

	for i := 0; i < 5; i++ {
		r.Add(LogEntry{Timestamp: ts(), Message: string(rune('A' + i))})
	}

	last := r.Last(5)
	if len(last) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(last))
	}
	for i := 0; i < 5; i++ {
		expected := string(rune('A' + i))
		if last[i].Message != expected {
			t.Errorf("entry %d: expected '%s', got '%s'", i, expected, last[i].Message)
		}
	}
}

// ─── Nadpisywanie (overflow) ────────────────────────────

func TestRingBuffer_Overflow(t *testing.T) {
	r := NewRingBuffer(3)

	for i := 0; i < 5; i++ {
		r.Add(LogEntry{Timestamp: ts(), Message: string(rune('A' + i))})
	}

	if r.Len() != 3 {
		t.Errorf("expected len 3 (overflow), got %d", r.Len())
	}

	last := r.Last(5)
	if len(last) != 3 {
		t.Fatalf("expected 3 entries after overflow, got %d", len(last))
	}

	if last[0].Message != "C" {
		t.Errorf("expected first entry 'C', got '%s'", last[0].Message)
	}
	if last[1].Message != "D" {
		t.Errorf("expected second entry 'D', got '%s'", last[1].Message)
	}
	if last[2].Message != "E" {
		t.Errorf("expected third entry 'E', got '%s'", last[2].Message)
	}
}

func TestRingBuffer_OverflowExact(t *testing.T) {
	r := NewRingBuffer(3)

	r.Add(LogEntry{Timestamp: ts(), Message: "A"})
	r.Add(LogEntry{Timestamp: ts(), Message: "B"})
	r.Add(LogEntry{Timestamp: ts(), Message: "C"})

	last := r.Last(3)
	if len(last) != 3 {
		t.Fatalf("expected 3 entries before wrap, got %d", len(last))
	}

	// 4th entry wraps around, overwrites A
	r.Add(LogEntry{Timestamp: ts(), Message: "D"})

	last = r.Last(3)
	if len(last) != 3 {
		t.Fatalf("expected 3 entries after wrap, got %d", len(last))
	}
	if last[0].Message != "B" {
		t.Errorf("expected 'B' (oldest surviving), got '%s'", last[0].Message)
	}
	if last[1].Message != "C" {
		t.Errorf("expected 'C', got '%s'", last[1].Message)
	}
	if last[2].Message != "D" {
		t.Errorf("expected 'D' (newest), got '%s'", last[2].Message)
	}
}

// ─── Last(n) z różnymi n ─────────────────────────────────

func TestRingBuffer_LastPartial(t *testing.T) {
	r := NewRingBuffer(10)

	for i := 0; i < 5; i++ {
		r.Add(LogEntry{Timestamp: ts(), Message: string(rune('A' + i))})
	}

	// Pos = 5, entries = [A(0), B(1), C(2), D(3), E(4), _, _, _, _, _]

	// Request more than size — capped at size, but only 5 have non-zero ts
	last := r.Last(20)
	if len(last) != 5 {
		t.Fatalf("expected 5 entries (existing only), got %d", len(last))
	}

	// Request 2 — returns the last 2 entries: D, E
	last = r.Last(2)
	if len(last) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(last))
	}
	if last[0].Message != "D" {
		t.Errorf("expected 'D' (4th entry), got '%s'", last[0].Message)
	}
	if last[1].Message != "E" {
		t.Errorf("expected 'E' (5th entry), got '%s'", last[1].Message)
	}
}

func TestRingBuffer_LastZero(t *testing.T) {
	r := NewRingBuffer(10)
	r.Add(LogEntry{Timestamp: ts(), Message: "test"})

	last := r.Last(0)
	if len(last) != 0 {
		t.Errorf("expected 0 entries for n=0, got %d", len(last))
	}
}

// ─── Współbieżność ──────────────────────────────────────

func TestRingBuffer_ConcurrentAdd(t *testing.T) {
	r := NewRingBuffer(1000)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				r.Add(LogEntry{Timestamp: ts(), Message: "test"})
			}
		}()
	}
	wg.Wait()

	if r.Len() != 1000 {
		t.Errorf("expected 1000 entries after concurrent adds, got %d", r.Len())
	}
}

func TestRingBuffer_ConcurrentReadWrite(t *testing.T) {
	r := NewRingBuffer(500)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		r.Add(LogEntry{Timestamp: ts(), Message: "prefill"})
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				r.Last(10)
				r.Len()
			}
		}()
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				r.Add(LogEntry{Timestamp: ts(), Message: "concurrent"})
			}
		}()
	}

	wg.Wait()
}

// ─── Scenariusze brzegowe ───────────────────────────────

func TestRingBuffer_SingleEntry(t *testing.T) {
	r := NewRingBuffer(1)

	r.Add(LogEntry{Timestamp: ts(), Message: "only"})
	if r.Len() != 1 {
		t.Errorf("expected len 1, got %d", r.Len())
	}

	last := r.Last(1)
	if len(last) != 1 || last[0].Message != "only" {
		t.Errorf("expected to retrieve the single entry")
	}

	r.Add(LogEntry{Timestamp: ts(), Message: "second"})
	if r.Len() != 1 {
		t.Errorf("expected len 1 after overwrite, got %d", r.Len())
	}
	last = r.Last(1)
	if last[0].Message != "second" {
		t.Errorf("expected 'second' after overwrite, got '%s'", last[0].Message)
	}
}



// ─── Benchmark ──────────────────────────────────────────

func BenchmarkRingBufferAdd(b *testing.B) {
	r := NewRingBuffer(10000)
	entry := LogEntry{Timestamp: ts(), Message: "benchmark"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Add(entry)
	}
}

func BenchmarkRingBufferLast(b *testing.B) {
	r := NewRingBuffer(10000)
	for i := 0; i < 10000; i++ {
		r.Add(LogEntry{Timestamp: ts(), Message: "prefill"})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Last(200)
	}
}
