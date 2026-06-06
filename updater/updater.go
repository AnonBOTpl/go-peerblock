package updater

import (
	"context"
	"log"
	"sync"
	"time"

	"go-peerblock/core"
)

// ReloadFunc is called when the database is updated.
type ReloadFunc func(*core.IPDatabase)

// Updater orchestrates periodic IP list updates.
type Updater struct {
	sources      []Source
	fetcher      *Fetcher
	parser       interface{} // placeholder for parser
	onReload     ReloadFunc
	logger       *log.Logger
	manualTrigger chan struct{}
	mu           sync.Mutex
	running      bool
}

// NewUpdater creates a new Updater.
func NewUpdater(sources []Source, fetcher *Fetcher, onReload ReloadFunc) *Updater {
	return &Updater{
		sources:       sources,
		fetcher:       fetcher,
		onReload:      onReload,
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

	u.updateAll()

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			u.updateAll()
		case <-u.manualTrigger:
			u.updateAll()
		case <-ctx.Done():
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

// IsRunning returns whether the updater is active.
func (u *Updater) IsRunning() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.running
}

func (u *Updater) updateAll() {
	u.mu.Lock()
	defer u.mu.Unlock()

	var allRanges []core.IPRange
	for _, src := range u.sources {
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
		u.logf("Załadowano %d zakresów z %s", len(ranges), src.Name)
	}

	merged := core.MergeRanges(allRanges)
	newDB := core.NewDatabase(merged)
	if u.onReload != nil {
		u.onReload(newDB)
	}
	u.logf("Baza IP przeładowana: %d zakresów (po merge'u)", len(merged))
}

func (u *Updater) logf(format string, args ...interface{}) {
	if u.logger != nil {
		u.logger.Printf(format, args...)
	}
}
