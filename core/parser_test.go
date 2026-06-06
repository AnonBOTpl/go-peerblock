package core

import (
	"bytes"
	"net"
	"strings"
	"testing"
)

// ─── Format Detection Tests ─────────────────────────────

func TestDetect_CIDR(t *testing.T) {
	var d FormatDetector
	data := []byte("1.2.3.0/24\n4.5.6.0/16\n")
	format, combined, err := d.Detect(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != FormatCIDR {
		t.Errorf("expected FormatCIDR (%d), got %d", FormatCIDR, format)
	}
	// Read back the data from the combined reader to verify it works
	var buf bytes.Buffer
	_, err = buf.ReadFrom(combined)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	ranges, err := Parse(buf.Bytes(), format)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(ranges) != 2 {
		t.Errorf("expected 2 ranges, got %d", len(ranges))
	}
}

func TestDetect_P2PText(t *testing.T) {
	var d FormatDetector
	data := []byte("Level1:1.2.3.0-1.2.3.255\nLevel2:4.5.6.0-4.5.6.255\n")
	format, _, err := d.Detect(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != FormatP2PText {
		t.Errorf("expected FormatP2PText (%d), got %d", FormatP2PText, format)
	}
}

func TestDetect_DAT(t *testing.T) {
	var d FormatDetector
	data := []byte("1.2.3.0 - 1.2.3.255 , 100 , Blocklist\n")
	format, _, err := d.Detect(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != FormatDAT {
		t.Errorf("expected FormatDAT (%d), got %d", FormatDAT, format)
	}
}

func TestDetect_EmptyData(t *testing.T) {
	var d FormatDetector
	format, _, err := d.Detect(bytes.NewReader([]byte{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != FormatCIDR {
		t.Errorf("expected FormatCIDR (default), got %d", format)
	}
}

// ─── CIDR Parsing Tests ─────────────────────────────────

func TestParse_CIDR(t *testing.T) {
	input := `1.2.3.0/24
10.0.0.0/8
# comment
192.168.0.0/16
; also a comment
invalid
`
	ranges, err := Parse([]byte(input), FormatCIDR)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranges) != 3 {
		t.Fatalf("expected 3 valid ranges, got %d", len(ranges))
	}

	tests := []struct {
		ip    string
		found bool
	}{
		{"1.2.3.0", true},
		{"1.2.3.255", true},
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"192.168.0.1", true},
		{"192.168.255.255", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}

	db := NewDatabase(ranges)
	for _, tt := range tests {
		ip := IPToUint32(net.ParseIP(tt.ip).To4())
		got := db.Contains(ip)
		if got != tt.found {
			t.Errorf("Contains(%s) = %v, want %v", tt.ip, got, tt.found)
		}
	}
}

func TestParse_CIDR_Empty(t *testing.T) {
	ranges, err := Parse([]byte(""), FormatCIDR)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranges) != 0 {
		t.Errorf("expected 0 ranges for empty input, got %d", len(ranges))
	}
}

// ─── P2P Text Parsing Tests ─────────────────────────────

func TestParse_P2PText(t *testing.T) {
	input := `Level1:1.2.3.0-1.2.3.255
Level2:10.0.0.0-10.255.255.255
# comment
Level3:192.168.0.0-192.168.255.255
`

	ranges, err := Parse([]byte(input), FormatP2PText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranges) != 3 {
		t.Fatalf("expected 3 ranges, got %d", len(ranges))
	}

	if ranges[0].Label != "Level1" {
		t.Errorf("expected label 'Level1', got '%s'", ranges[0].Label)
	}
	if ranges[1].Label != "Level2" {
		t.Errorf("expected label 'Level2', got '%s'", ranges[1].Label)
	}

	db := NewDatabase(ranges)
	if !db.Contains(IPToUint32(net.ParseIP("1.2.3.100").To4())) {
		t.Error("expected 1.2.3.100 to be found")
	}
	if db.Contains(IPToUint32(net.ParseIP("2.2.2.2").To4())) {
		t.Error("expected 2.2.2.2 to NOT be found")
	}
}

func TestParse_P2PText_InvalidLine(t *testing.T) {
	input := `Level1:1.2.3.0-1.2.3.255
bad-line-no-colon
`
	ranges, err := Parse([]byte(input), FormatP2PText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranges) != 1 {
		t.Errorf("expected 1 valid range (skip invalid lines), got %d", len(ranges))
	}
}

// ─── DAT Parsing Tests ──────────────────────────────────

func TestParse_DAT(t *testing.T) {
	input := `1.2.3.0 - 1.2.3.255 , 100 , Level1 Blocklist
10.0.0.0 - 10.255.255.255 , 200 , Level2
# comment
`
	ranges, err := Parse([]byte(input), FormatDAT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranges) != 2 {
		t.Fatalf("expected 2 ranges, got %d", len(ranges))
	}

	db := NewDatabase(ranges)
	if !db.Contains(IPToUint32(net.ParseIP("1.2.3.100").To4())) {
		t.Error("expected 1.2.3.100 to be found in DAT parse result")
	}
}

// ─── Plain Range Parsing Tests ──────────────────────────

func TestParse_PlainRange(t *testing.T) {
	input := `1.2.3.0-1.2.3.255
10.0.0.0-10.255.255.255
# comment
`
	ranges, err := Parse([]byte(input), FormatPlainRange)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranges) != 2 {
		t.Fatalf("expected 2 ranges, got %d", len(ranges))
	}
}

// ─── Parser + Database Integration ──────────────────────

func TestParseAndMerge(t *testing.T) {
	input := `10.0.0.0/24
10.0.0.0/16
10.0.0.0/8
`
	ranges, err := Parse([]byte(input), FormatCIDR)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranges) != 3 {
		t.Fatalf("expected 3 parsed ranges, got %d", len(ranges))
	}

	db := NewDatabase(ranges)
	if len(db.Ranges()) != 1 {
		t.Fatalf("expected 1 merged range, got %d", len(db.Ranges()))
	}

	if !db.Contains(IPToUint32(net.ParseIP("10.1.2.3").To4())) {
		t.Error("expected 10.1.2.3 to be in merged range")
	}
	if !db.Contains(IPToUint32(net.ParseIP("10.255.255.255").To4())) {
		t.Error("expected 10.255.255.255 to be in merged range")
	}
}

// ─── Benchmark ──────────────────────────────────────────

func BenchmarkParseCIDR(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 10000; i++ {
		sb.WriteString("10.0.0.0/24\n")
	}
	data := []byte(sb.String())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(data, FormatCIDR)
	}
}

func BenchmarkParseP2P(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("Level1:1.2.3.0-1.2.3.255\n")
	}
	data := []byte(sb.String())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(data, FormatP2PText)
	}
}

func BenchmarkDetect(b *testing.B) {
	data := []byte("1.2.3.0/24\n4.5.6.0/16\n192.168.0.0/24\n")
	var d FormatDetector
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = d.Detect(bytes.NewReader(data))
	}
}
