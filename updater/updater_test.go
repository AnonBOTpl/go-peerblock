package updater

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-peerblock/core"
)

// createTestDataFile writes CIDR data to a temp file and returns a file:// URL.
func createTestDataFile(t testing.TB, dir string, name string, data string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("cannot write test data file: %v", err)
	}
	return "file://" + path
}

// ─── Podstawowy cykl aktualizacji ────────────────────────────────

func TestUpdater_UpdateAll_MultipleSources(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")

	data1 := "10.0.0.0/8\n192.168.0.0/16\n"
	data2 := "172.16.0.0/12\n10.0.0.0/8\n" // overlaps with source1

	url1 := createTestDataFile(t, dir, "src1.txt", data1)
	url2 := createTestDataFile(t, dir, "src2.txt", data2)

	fetcher := NewFetcher(cacheDir)
	var mu sync.Mutex
	var reloadedDB *core.IPDatabase
	reloaded := make(chan struct{}, 1)

	u := NewUpdater(
		[]Source{
			{Name: "src1", URL: url1, Format: int(core.FormatCIDR), Enabled: true},
			{Name: "src2", URL: url2, Format: int(core.FormatCIDR), Enabled: true},
		},
		fetcher,
		func(db *core.IPDatabase) {
			mu.Lock()
			reloadedDB = db
			mu.Unlock()
			select {
			case reloaded <- struct{}{}:
			default:
			}
		},
		func(format string, args ...interface{}) {},
		24*time.Hour,
	)

	u.updateAll(false)

	select {
	case <-reloaded:
	case <-time.After(5 * time.Second):
		t.Fatal("onReload was not called within 5s")
	}

	mu.Lock()
	ranges := reloadedDB.Ranges()
	mu.Unlock()

	// src1: 10.0.0.0/8, 192.168.0.0/16 → 2 ranges
	// src2: 172.16.0.0/12, 10.0.0.0/8 → 2 ranges (10.0.0.0/8 overlaps)
	// After merge: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16 → 3 ranges
	if len(ranges) != 3 {
		t.Errorf("expected 3 merged ranges, got %d", len(ranges))
	}
}

// ─── Częściowe niepowodzenie ─────────────────────────────────────

func TestUpdater_UpdateAll_PartialFailure(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")

	goodURL := createTestDataFile(t, dir, "good.txt", "1.2.3.0/24\n")
	badURL := "file://" + filepath.Join(dir, "nonexistent.txt")

	fetcher := NewFetcher(cacheDir)
	reloaded := make(chan struct{}, 1)

	u := NewUpdater(
		[]Source{
			{Name: "good", URL: goodURL, Format: int(core.FormatCIDR), Enabled: true},
			{Name: "bad", URL: badURL, Format: int(core.FormatCIDR), Enabled: true},
		},
		fetcher,
		func(db *core.IPDatabase) {
			if len(db.Ranges()) != 1 {
				t.Errorf("expected 1 range from 'good' source, got %d", len(db.Ranges()))
			}
			select {
			case reloaded <- struct{}{}:
			default:
			}
		},
		func(format string, args ...interface{}) {},
		24*time.Hour,
	)

	u.updateAll(false)

	select {
	case <-reloaded:
	case <-time.After(5 * time.Second):
		t.Fatal("onReload was not called within 5s")
	}
}

// ─── Wszystkie źródła failują ────────────────────────────────────

func TestUpdater_UpdateAll_AllFail(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")

	fetcher := NewFetcher(cacheDir)
	reloaded := make(chan struct{}, 1)

	u := NewUpdater(
		[]Source{
			{Name: "a", URL: "file://" + filepath.Join(dir, "nonexist_a.txt"), Format: int(core.FormatCIDR), Enabled: true},
			{Name: "b", URL: "file://" + filepath.Join(dir, "nonexist_b.txt"), Format: int(core.FormatCIDR), Enabled: true},
		},
		fetcher,
		func(db *core.IPDatabase) {
			if len(db.Ranges()) != 0 {
				t.Errorf("expected 0 ranges when all sources fail, got %d", len(db.Ranges()))
			}
			select {
			case reloaded <- struct{}{}:
			default:
			}
		},
		func(format string, args ...interface{}) {},
		24*time.Hour,
	)

	u.updateAll(false)

	select {
	case <-reloaded:
	case <-time.After(5 * time.Second):
		t.Fatal("onReload was not called within 5s")
	}
}

// ─── sourceRanges per-źródło ────────────────────────────────────

func TestUpdater_UpdateAll_SourceRanges(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")

	url1 := createTestDataFile(t, dir, "alpha.txt", "10.0.0.0/8\n")
	url2 := createTestDataFile(t, dir, "beta.txt", "192.168.0.0/16\n")

	fetcher := NewFetcher(cacheDir)

	u := NewUpdater(
		[]Source{
			{Name: "alpha", URL: url1, Format: int(core.FormatCIDR), Enabled: true},
			{Name: "beta", URL: url2, Format: int(core.FormatCIDR), Enabled: true},
		},
		fetcher,
		func(db *core.IPDatabase) {},
		func(format string, args ...interface{}) {},
		24*time.Hour,
	)

	u.updateAll(false)

	sr := u.GetSourceRanges()
	if len(sr) != 2 {
		t.Fatalf("expected 2 source ranges, got %d", len(sr))
	}

	if len(sr["alpha"]) != 1 {
		t.Errorf("expected 1 range from alpha, got %d", len(sr["alpha"]))
	}
	if len(sr["beta"]) != 1 {
		t.Errorf("expected 1 range from beta, got %d", len(sr["beta"]))
	}

	tenNet, _ := core.CIDRToRange("10.0.0.0/8")
	r := sr["alpha"][0]
	if r.Start != tenNet.Start || r.End != tenNet.End {
		t.Errorf("alpha range mismatch: got %d-%d, want %d-%d", r.Start, r.End, tenNet.Start, tenNet.End)
	}
}

// ─── Wyłączone źródła ────────────────────────────────────────────

func TestUpdater_UpdateAll_DisabledSources(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")

	enabledURL := createTestDataFile(t, dir, "enabled.txt", "1.2.3.0/24\n")
	disabledURL := createTestDataFile(t, dir, "disabled.txt", "10.0.0.0/8\n")

	fetcher := NewFetcher(cacheDir)
	reloaded := make(chan struct{}, 1)

	u := NewUpdater(
		[]Source{
			{Name: "disabled", URL: disabledURL, Format: int(core.FormatCIDR), Enabled: false},
			{Name: "enabled", URL: enabledURL, Format: int(core.FormatCIDR), Enabled: true},
		},
		fetcher,
		func(db *core.IPDatabase) {
			ranges := db.Ranges()
			if len(ranges) != 1 {
				t.Errorf("expected 1 range from enabled source only, got %d", len(ranges))
			}
			expected, _ := core.CIDRToRange("1.2.3.0/24")
			if len(ranges) > 0 && ranges[0].Start != expected.Start {
				t.Errorf("expected range starting at %d (1.2.3.0), got %d", expected.Start, ranges[0].Start)
			}
			select {
			case reloaded <- struct{}{}:
			default:
			}
		},
		func(format string, args ...interface{}) {},
		24*time.Hour,
	)

	u.updateAll(false)

	select {
	case <-reloaded:
	case <-time.After(5 * time.Second):
		t.Fatal("onReload was not called within 5s")
	}
}

// ─── GetSources zwraca skopiowane LastSync ───────────────────────

func TestUpdater_UpdateAll_LastSyncUpdated(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")

	url := createTestDataFile(t, dir, "src.txt", "1.2.3.0/24\n")
	fetcher := NewFetcher(cacheDir)

	u := NewUpdater(
		[]Source{
			{Name: "src", URL: url, Format: int(core.FormatCIDR), Enabled: true},
		},
		fetcher,
		func(db *core.IPDatabase) {},
		func(format string, args ...interface{}) {},
		24*time.Hour,
	)

	u.updateAll(false)

	sources := u.GetSources()
	if sources[0].LastSync.IsZero() {
		t.Error("LastSync should be set after update")
	}
	if sources[0].RangeCount != 1 {
		t.Errorf("RangeCount should be 1, got %d", sources[0].RangeCount)
	}
}

// ─── Współbieżny TriggerManual ──────────────────────────────────

func TestUpdater_ConcurrentTriggerManual(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")

	url := createTestDataFile(t, dir, "src.txt", "10.0.0.0/8\n")
	fetcher := NewFetcher(cacheDir)

	var reloadCount atomic.Int32
	ctx := t.Context()

	u := NewUpdater(
		[]Source{
			{Name: "src", URL: url, Format: int(core.FormatCIDR), Enabled: true},
		},
		fetcher,
		func(db *core.IPDatabase) {
			reloadCount.Add(1)
		},
		func(format string, args ...interface{}) {},
		24*time.Hour,
	)

	// Run the updater (does initial updateAll)
	go u.Run(ctx)
	time.Sleep(200 * time.Millisecond)

	initial := reloadCount.Load()
	if initial == 0 {
		t.Fatal("initial update should have triggered reload")
	}

	// Fire multiple triggers concurrently
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u.TriggerManual()
		}()
	}
	wg.Wait()

	// Give it time to process (only 1 trigger fires since channel is buffered 1)
	time.Sleep(500 * time.Millisecond)

	after := reloadCount.Load()
	if after <= initial {
		t.Error("TriggerManual should have caused at least one additional reload")
	}
}

// ─── Empty sources ───────────────────────────────────────────────

func TestUpdater_UpdateAll_NoSources(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")

	fetcher := NewFetcher(cacheDir)
	reloaded := make(chan struct{}, 1)

	u := NewUpdater(
		[]Source{},
		fetcher,
		func(db *core.IPDatabase) {
			if len(db.Ranges()) != 0 {
				t.Errorf("expected 0 ranges with no sources, got %d", len(db.Ranges()))
			}
			select {
			case reloaded <- struct{}{}:
			default:
			}
		},
		func(format string, args ...interface{}) {},
		24*time.Hour,
	)

	u.updateAll(false)

	select {
	case <-reloaded:
	case <-time.After(5 * time.Second):
		t.Fatal("onReload was not called within 5s")
	}
}

// ─── P2P format ─────────────────────────────────────────────────

func TestUpdater_UpdateAll_P2PFormat(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")

	data := "Level1:10.0.0.0-10.255.255.255\nLevel2:192.168.0.0-192.168.255.255\n"
	url := createTestDataFile(t, dir, "p2p.txt", data)
	fetcher := NewFetcher(cacheDir)
	reloaded := make(chan struct{}, 1)

	u := NewUpdater(
		[]Source{
			{Name: "mylist", URL: url, Format: int(core.FormatP2PText), Enabled: true},
		},
		fetcher,
		func(db *core.IPDatabase) {
			ranges := db.Ranges()
			if len(ranges) != 2 {
				t.Errorf("expected 2 ranges after merge, got %d", len(ranges))
			}
			select {
			case reloaded <- struct{}{}:
			default:
			}
		},
		func(format string, args ...interface{}) {},
		24*time.Hour,
	)

	u.updateAll(false)

	select {
	case <-reloaded:
	case <-time.After(5 * time.Second):
		t.Fatal("onReload was not called within 5s")
	}

	sr := u.GetSourceRanges()
	if ranges, ok := sr["mylist"]; ok {
		if len(ranges) != 2 {
			t.Errorf("expected 2 source ranges for mylist, got %d", len(ranges))
		}
	} else {
		t.Error("mylist not found in sourceRanges")
	}
}

// ─── Benchmark updatera ──────────────────────────────────────────

func BenchmarkUpdater_UpdateAll(b *testing.B) {
	dir := b.TempDir()
	cacheDir := filepath.Join(dir, "cache")

	createFile := func(name, data string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(data), 0644); err != nil {
			b.Fatalf("cannot write: %v", err)
		}
		return "file://" + path
	}

	u := NewUpdater(
		[]Source{
			{Name: "a", URL: createFile("bench_a.txt", "10.0.0.0/8\n192.168.0.0/16\n"), Format: int(core.FormatCIDR), Enabled: true},
			{Name: "b", URL: createFile("bench_b.txt", "172.16.0.0/12\n"), Format: int(core.FormatCIDR), Enabled: true},
		},
		NewFetcher(cacheDir),
		func(db *core.IPDatabase) {},
		func(format string, args ...interface{}) {},
		24*time.Hour,
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		u.updateAll(false)
	}
}
