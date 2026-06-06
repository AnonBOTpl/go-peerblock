package core

import (
	"math/rand"
	"net"
	"testing"
)

// ─── MergeRanges Tests ──────────────────────────────────

func TestMergeRanges_Empty(t *testing.T) {
	result := MergeRanges(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestMergeRanges_NoOverlap(t *testing.T) {
	ranges := []IPRange{
		{Start: 1, End: 10},
		{Start: 20, End: 30},
		{Start: 40, End: 50},
	}
	result := MergeRanges(ranges)
	if len(result) != 3 {
		t.Fatalf("expected 3 ranges, got %d", len(result))
	}
}

func TestMergeRanges_Overlap(t *testing.T) {
	ranges := []IPRange{
		{Start: 1, End: 10},
		{Start: 5, End: 15},
		{Start: 12, End: 20},
	}
	result := MergeRanges(ranges)
	if len(result) != 1 {
		t.Fatalf("expected 1 merged range, got %d", len(result))
	}
	if result[0].Start != 1 || result[0].End != 20 {
		t.Errorf("expected [1,20], got [%d,%d]", result[0].Start, result[0].End)
	}
}

func TestMergeRanges_Adjacent(t *testing.T) {
	ranges := []IPRange{
		{Start: 1, End: 10},
		{Start: 11, End: 20},
	}
	result := MergeRanges(ranges)
	if len(result) != 1 {
		t.Fatalf("expected 1 merged range for adjacent, got %d", len(result))
	}
	if result[0].End != 20 {
		t.Errorf("expected end=20, got %d", result[0].End)
	}
}

func TestMergeRanges_UnsortedInput(t *testing.T) {
	ranges := []IPRange{
		{Start: 30, End: 40},
		{Start: 10, End: 20},
		{Start: 1, End: 5},
	}
	result := MergeRanges(ranges)
	if len(result) != 3 {
		t.Fatalf("expected 3 ranges, got %d", len(result))
	}
	if result[0].Start != 1 || result[1].Start != 10 || result[2].Start != 30 {
		t.Errorf("ranges not sorted correctly")
	}
}

func TestMergeRanges_Contained(t *testing.T) {
	ranges := []IPRange{
		{Start: 1, End: 100},
		{Start: 20, End: 30},
		{Start: 40, End: 50},
	}
	result := MergeRanges(ranges)
	if len(result) != 1 {
		t.Fatalf("expected 1 range, got %d", len(result))
	}
	if result[0].Start != 1 || result[0].End != 100 {
		t.Errorf("expected [1,100], got [%d,%d]", result[0].Start, result[0].End)
	}
}

// ─── CIDRToRange Tests ──────────────────────────────────

func TestCIDRToRange_Simple(t *testing.T) {
	r, err := CIDRToRange("192.168.0.0/24")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 192.168.0.0 = 0xC0A80000
	if r.Start != 0xC0A80000 {
		t.Errorf("expected start 0xC0A80000, got 0x%X", r.Start)
	}
	// 192.168.0.255 = 0xC0A800FF
	if r.End != 0xC0A800FF {
		t.Errorf("expected end 0xC0A800FF, got 0x%X", r.End)
	}
}

func TestCIDRToRange_SingleIP(t *testing.T) {
	r, err := CIDRToRange("10.0.0.1/32")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Start != r.End {
		t.Errorf("single IP should have start == end")
	}
	// 10.0.0.1 = 0x0A000001
	if r.Start != 0x0A000001 {
		t.Errorf("expected 0x0A000001, got 0x%X", r.Start)
	}
}

func TestCIDRToRange_Wildcard(t *testing.T) {
	r, err := CIDRToRange("0.0.0.0/0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Start != 0 {
		t.Errorf("expected start 0, got %d", r.Start)
	}
	if r.End != 0xFFFFFFFF {
		t.Errorf("expected end 0xFFFFFFFF, got 0x%X", r.End)
	}
}

func TestCIDRToRange_Invalid(t *testing.T) {
	_, err := CIDRToRange("not-a-cidr")
	if err == nil {
		t.Error("expected error for invalid CIDR")
	}
}

// ─── IPToUint32 Tests ───────────────────────────────────

func TestIPToUint32(t *testing.T) {
	tests := []struct {
		ip   string
		want uint32
	}{
		{"0.0.0.0", 0},
		{"255.255.255.255", 0xFFFFFFFF},
		{"192.168.1.1", 0xC0A80101},
		{"10.0.0.1", 0x0A000001},
		{"127.0.0.1", 0x7F000001},
		{"8.8.8.8", 0x08080808},
		{"1.1.1.1", 0x01010101},
	}
	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if ip == nil {
			t.Fatalf("invalid test IP: %s", tt.ip)
		}
		got := IPToUint32(ip)
		if got != tt.want {
			t.Errorf("IPToUint32(%s) = 0x%X, want 0x%X", tt.ip, got, tt.want)
		}
	}
}

func TestIPToUint32_Nil(t *testing.T) {
	if got := IPToUint32(nil); got != 0 {
		t.Errorf("expected 0 for nil, got %d", got)
	}
}

// ─── IPDatabase.Contains Tests ──────────────────────────

func TestDatabase_Contains(t *testing.T) {
	ranges := []IPRange{
		{Start: 0x01000000, End: 0x01FFFFFF}, // 1.0.0.0 - 1.255.255.255
		{Start: 0x08080808, End: 0x08080808}, // 8.8.8.8
		{Start: 0xC0A80000, End: 0xC0A8FFFF}, // 192.168.0.0 - 192.168.255.255
	}
	db := NewDatabase(ranges)

	tests := []struct {
		ip    string
		want  bool
		label string
	}{
		{"1.1.1.1", true, "in first range"},
		{"1.255.255.255", true, "end of first range"},
		{"8.8.8.8", true, "single IP"},
		{"192.168.0.1", true, "in last range"},
		{"192.168.255.255", true, "end of last range"},
		{"2.2.2.2", false, "between ranges"},
		{"4.4.4.4", false, "outside any range"},
		{"10.0.0.1", false, "not in db"},
	}
	for _, tt := range tests {
		ip := IPToUint32(net.ParseIP(tt.ip))
		got := db.Contains(ip)
		if got != tt.want {
			t.Errorf("Contains(%s) [%s] = %v, want %v", tt.ip, tt.label, got, tt.want)
		}
	}
}

func TestDatabase_Store_Atomic(t *testing.T) {
	db := NewDatabase([]IPRange{{Start: 1, End: 10}})

	// Initial state
	if !db.Contains(5) {
		t.Error("expected 5 to be in initial db")
	}

	// Atomically replace
	db.Store([]IPRange{{Start: 100, End: 200}})
	if db.Contains(5) {
		t.Error("expected 5 to NOT be in updated db")
	}
	if !db.Contains(150) {
		t.Error("expected 150 to be in updated db")
	}
}

func TestDatabase_Ranges(t *testing.T) {
	ranges := []IPRange{{Start: 1, End: 10}, {Start: 5, End: 15}}
	db := NewDatabase(ranges)
	// Should be merged
	if len(db.Ranges()) != 1 {
		t.Fatalf("expected 1 merged range, got %d", len(db.Ranges()))
	}
}

// ─── Benchmarks ─────────────────────────────────────────

// generateTestDB creates a database with n random non-overlapping ranges.
func generateTestDB(n int) *IPDatabase {
	ranges := make([]IPRange, n)
	start := uint32(0)
	for i := 0; i < n; i++ {
		// Each range is /24 (256 IPs), spaced by /16 (65536 IPs)
		ranges[i] = IPRange{
			Start: start,
			End:   start + 255,
		}
		start += 65536
	}
	return NewDatabase(ranges)
}

func BenchmarkLookup10k(b *testing.B) {
	db := generateTestDB(10_000)
	ips := make([]uint32, 1000)
	for i := range ips {
		ips[i] = uint32(rand.Int31())
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Contains(ips[i%len(ips)])
	}
}

func BenchmarkLookup100k(b *testing.B) {
	db := generateTestDB(100_000)
	ips := make([]uint32, 1000)
	for i := range ips {
		ips[i] = uint32(rand.Int31())
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Contains(ips[i%len(ips)])
	}
}

func BenchmarkLookup500k(b *testing.B) {
	db := generateTestDB(500_000)
	ips := make([]uint32, 10_000)
	for i := range ips {
		ips[i] = uint32(rand.Int31())
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Contains(ips[i%len(ips)])
	}
}

func BenchmarkMergeRanges(b *testing.B) {
	ranges := make([]IPRange, 1000)
	for i := range ranges {
		ranges[i] = IPRange{
			Start: uint32(i * 1000),
			End:   uint32(i*1000 + 500 + i%200),
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MergeRanges(ranges)
	}
}

func BenchmarkCIDRToRange(b *testing.B) {
	cidrs := []string{
		"1.2.3.0/24",
		"10.0.0.0/8",
		"192.168.0.0/16",
		"172.16.0.0/12",
		"8.8.8.0/24",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CIDRToRange(cidrs[i%len(cidrs)])
	}
}

// GenerateRandomIPs generates n random IPs for benchmarking.
// Not used in current benchmarks but available for future use.
var _ = generateRandomIPs

func generateRandomIPs(n int) []uint32 {
	ips := make([]uint32, n)
	for i := range ips {
		ips[i] = uint32(rand.Int31())
	}
	return ips
}

// Ensure net is imported (used in IPToUint32 test).
var _ = net.ParseIP
