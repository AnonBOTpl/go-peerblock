package updater

import (
	"context"
	"sync"
	"time"

	"go-peerblock/core"
)

// ReloadFunc is called when the database is updated.
type ReloadFunc func(*core.IPDatabase)

// LogFunc is called for progress messages during updates.
type LogFunc func(format string, args ...interface{})

// Updater orchestrates periodic IP list updates.
type Updater struct {
	sources       []Source
	fetcher       *Fetcher
	onReload      ReloadFunc
	logFn         LogFunc
	interval      time.Duration
	manualTrigger chan struct{}
	mu            sync.Mutex
	running       bool
}

// NewUpdater creates a new Updater.
func NewUpdater(sources []Source, fetcher *Fetcher, onReload ReloadFunc, logFn LogFunc, interval time.Duration) *Updater {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &Updater{
		sources:       sources,
		fetcher:       fetcher,
		onReload:      onReload,
		logFn:         logFn,
		interval:      interval,
		manualTrigger: make(chan struct{}, 1),
	}
}

// Run starts the update loop. Blocks until ctx is cancelled.
func (u *Updater) Run(ctx context.Context) {
	u.mu.Lock()
	u.running = true
	u.mu.Unlock()

	defer func() {
		u.mu.Lock()
		u.running = false
		u.mu.Unlock()
	}()

	u.logf("Rozpoczynam aktualizację list IP...")
	u.updateAll()

	ticker := time.NewTicker(u.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			u.logf("Zaplanowana aktualizacja list IP...")
			u.updateAll()
		case <-u.manualTrigger:
			u.logf("Ręczne wyzwolenie aktualizacji...")
			u.updateAll()
		case <-ctx.Done():
			u.logf("Aktualizator zatrzymany")
			return
		}
	}
}

// TriggerManual triggers an immediate update.
func (u *Updater) TriggerManual() {
	select {
	case u.manualTrigger <- struct{}{}:
	default:
	}
}

// RefreshSources updates the source list (e.g. after config save).
func (u *Updater) RefreshSources(sources []Source) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.sources = sources
}

// GetSources returns a copy of the current source list (with updated LastSync values).
func (u *Updater) GetSources() []Source {
	u.mu.Lock()
	defer u.mu.Unlock()
	result := make([]Source, len(u.sources))
	copy(result, u.sources)
	return result
}

// IsRunning returns whether the updater is active.
func (u *Updater) IsRunning() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.running
}

func (u *Updater) updateAll() {
	// Copy sources under lock, then fetch without holding the lock
	u.mu.Lock()
	sources := make([]Source, len(u.sources))
	copy(sources, u.sources)
	u.mu.Unlock()

	now := time.Now()
	var allRanges []core.IPRange
	for i, src := range sources {
		if !src.Enabled {
			continue
		}
		data, err := u.fetcher.Fetch(src)
		if err != nil {
			u.logf("Nie można pobrać %s: %v", src.Name, err)
			continue
		}
		ranges, err := core.Parse(data, core.Format(src.Format))
		if err != nil {
			u.logf("Błąd parsowania %s: %v", src.Name, err)
			continue
		}
		allRanges = append(allRanges, ranges...)
		sources[i].LastSync = now
		u.logf("Załadowano %d zakresów z %s", len(ranges), src.Name)
	}

	// Lock only for the final merge + reload (fast operation)
	u.mu.Lock()
	// Copy LastSync back to the updater's source list
	for i := range sources {
		if !sources[i].LastSync.IsZero() && i < len(u.sources) {
			u.sources[i].LastSync = sources[i].LastSync
		}
	}
	merged := core.MergeRanges(allRanges)
	newDB := core.NewDatabase(merged)
	u.mu.Unlock()

	if u.onReload != nil {
		u.onReload(newDB)
	}
	u.logf("Baza IP przeładowana: %d zakresów (po merge'u)", len(merged))
}

func (u *Updater) logf(format string, args ...interface{}) {
	if u.logFn != nil {
		u.logFn(format, args...)
	}
}
