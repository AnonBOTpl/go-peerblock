package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ─── Podstawowe tworzenie i zamykanie ────────────────────────────

func TestLogger_NewAndClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	l, err := NewLogger(path, 100, 0) // no rotation
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	l.Info("test message")
	l.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read log file: %v", err)
	}
	if !strings.Contains(string(data), "test message") {
		t.Errorf("log file does not contain expected message")
	}
}

// ─── Wiele wpisów ───────────────────────────────────────────────

func TestLogger_MultipleEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.log")

	l, err := NewLogger(path, 100, 0)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	for i := 0; i < 50; i++ {
		l.Info("entry %d", i)
	}
	l.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read log file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 50 {
		t.Errorf("expected 50 lines, got %d", len(lines))
	}
}

// ─── Rotacja plików ─────────────────────────────────────────────

func TestLogger_Rotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rotate.log")

	// maxSizeMB=1 → 1 MB. Każdy wpis ~2000 B → 1000 wpisów = ~2 MB (rotacja ok. 1 raz).
	// Kanał loggera ma buffer 1024, a log() dropuje gdy pełny — dodajemy 1ms delay
	// między każdym wpisem, żeby gorutyna zdążyła przetworzyć.
	l, err := NewLogger(path, 100, 1)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	for i := 0; i < 1000; i++ {
		l.Info("linia %04d: %s", i, strings.Repeat("X", 1950))
		time.Sleep(time.Millisecond)
	}
	l.Close()

	// Po rotacji powinny być co najmniej 2 pliki: rotate.log + rotate.log.*
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read dir: %v", err)
	}

	var rotatedFiles int
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "rotate.log") {
			rotatedFiles++
		}
	}

	if rotatedFiles < 2 {
		t.Errorf("expected at least 2 log files after rotation (original + backups), got %d: %v", rotatedFiles, entries)
	}

	// Oryginalny plik powinien istnieć i nie być pusty
	origData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read current log file: %v", err)
	}
	if len(origData) == 0 {
		t.Error("current log file should not be empty")
	}
}

// ─── Brak rotacji gdy maxSize=0 ─────────────────────────────────

func TestLogger_NoRotationWhenDisabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "norotate.log")

	l, err := NewLogger(path, 100, 0)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	// Wiele wpisów — rozmiar przekroczy 1 KB, ale maxSize=0 więc brak rotacji
	for i := 0; i < 100; i++ {
		l.Info("linia %d", i)
	}
	l.Close()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read dir: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected exactly 1 log file (no rotation), got %d", len(entries))
	}
}

// ─── Ring buffer jest zasilany przez logger ─────────────────────

func TestLogger_RingBufferPopulated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ring.log")

	l, err := NewLogger(path, 50, 0)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	for i := 0; i < 10; i++ {
		l.Info("msg %d", i)
	}
	l.Close()

	ring := l.Ring()
	if ring.Len() != 10 {
		t.Errorf("expected 10 entries in ring buffer, got %d", ring.Len())
	}

	last := ring.Last(10)
	if len(last) != 10 {
		t.Fatalf("expected 10 entries from Last, got %d", len(last))
	}
	if last[0].Message != "msg 0" {
		t.Errorf("expected first message 'msg 0', got '%s'", last[0].Message)
	}
	if last[9].Message != "msg 9" {
		t.Errorf("expected last message 'msg 9', got '%s'", last[9].Message)
	}
}

// ─── Subscribe/Unsubscribe ──────────────────────────────────────

func TestLogger_Subscribe(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub.log")

	l, err := NewLogger(path, 100, 0)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer l.Close()

	ch := l.Subscribe()
	l.Info("hello from subscribe test")
	msg := <-ch
	if !strings.Contains(msg.Message, "hello from subscribe") {
		t.Errorf("unexpected message: %s", msg.Message)
	}
}

// ─── Poziomy logowania ──────────────────────────────────────────

func TestLogger_LogLevels(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "levels.log")

	l, err := NewLogger(path, 100, 0)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	l.Debug("debug msg")
	l.Info("info msg")
	l.Warn("warn msg")
	l.Error("error msg")
	l.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read log file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "DEBUG") {
		t.Error("missing DEBUG")
	}
	if !strings.Contains(content, "INFO") {
		t.Error("missing INFO")
	}
	if !strings.Contains(content, "WARN") {
		t.Error("missing WARN")
	}
	if !strings.Contains(content, "ERROR") {
		t.Error("missing ERROR")
	}
}
