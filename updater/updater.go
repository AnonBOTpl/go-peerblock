package updater

import (
	"context"
	"sync"
	"time"

	"go-peerblock/core"
	"go-peerblock/i18n"
)

// ReloadFunc is called when the database is updated.
type ReloadFunc func(*core.IPDatabase)

// LogFunc is called for progress messages during updates.
type LogFunc func(format string, args ...interface{})

// Updater orchestrates periodic IP list updates.
type Updater struct {
	sources         []Source
	fetcher          *Fetcher
	onReload         ReloadFunc
	logFn            LogFunc
	interval         time.Duration
	lang             string
	manualTrigger    chan struct{}
	mu               sync.Mutex
	running          bool
	sourceRanges     map[string][]core.IPRange
	prevRangeCounts  map[string]int
	rangeDiffs       map[string]int
}

// NewUpdater creates a new Updater.
func NewUpdater(sources []Source, fetcher *Fetcher, onReload ReloadFunc, logFn LogFunc, interval time.Duration, lang string) *Updater {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &Updater{
		sources:       sources,
		fetcher:       fetcher,
		onReload:      onReload,
		logFn:         logFn,
		interval:      interval,
		lang:          lang,
		manualTrigger: make(chan struct{}, 1),
	}
}

// GetSourceRanges returns per-source IP ranges (pre-merge) for source lookup.
func (u *Updater) GetSourceRanges() map[string][]core.IPRange {
	u.mu.Lock()
	defer u.mu.Unlock()
	result := make(map[string][]core.IPRange, len(u.sourceRanges))
	for name, ranges := range u.sourceRanges {
		cpy := make([]core.IPRange, len(ranges))
		copy(cpy, ranges)
		result[name] = cpy
	}
	return result
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

	// Pierwsza aktualizacja przy starcie — cicha (nie zaśmieca logów)
	u.updateAll(true)

	ticker := time.NewTicker(u.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			u.logf("updater.scheduled")
			u.updateAll(false)
		case <-u.manualTrigger:
			u.logf("updater.manual")
			u.updateAll(false)
		case <-ctx.Done():
			u.logf("updater.stopped")
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

// GetRangeDiffs returns the difference in range counts from the last update.
func (u *Updater) GetRangeDiffs() map[string]int {
	u.mu.Lock()
	defer u.mu.Unlock()
	result := make(map[string]int, len(u.rangeDiffs))
	for k, v := range u.rangeDiffs {
		result[k] = v
	}
	return result
}

func (u *Updater) updateAll(silent bool) {
	// Copy sources under lock, then fetch without holding the lock
	u.mu.Lock()
	sources := make([]Source, len(u.sources))
	copy(sources, u.sources)

	// Save previous range counts before updating
	u.prevRangeCounts = make(map[string]int)
	for _, s := range u.sources {
		u.prevRangeCounts[s.Name] = s.RangeCount
	}
	u.mu.Unlock()

	now := time.Now()
	var allRanges []core.IPRange
	perSource := make(map[string][]core.IPRange)
	for i, src := range sources {
		if !src.Enabled {
			continue
		}
		data, err := u.fetcher.Fetch(src)
		if err != nil {
			if !silent {
				u.logf("updater.fetch.error", src.Name, err)
			}
			continue
		}
		ranges, err := core.Parse(data, core.Format(src.Format))
		if err != nil {
			if !silent {
				u.logf("updater.parse.error", src.Name, err)
			}
			continue
		}
		perSource[src.Name] = ranges
		allRanges = append(allRanges, ranges...)
		sources[i].LastSync = now
		sources[i].RangeCount = len(ranges)
		if !silent {				u.logf("updater.loaded", len(ranges), src.Name)
		}
	}

	// Store per-source ranges before merge (for source lookup I2)
	u.mu.Lock()
	u.sourceRanges = perSource

	// Compute range diffs per source
	u.rangeDiffs = make(map[string]int)
	for i := range sources {
		if sources[i].LastSync.IsZero() && !silent {
			// Source failed — skip diff
			continue
		}
		prev := u.prevRangeCounts[sources[i].Name]
		u.rangeDiffs[sources[i].Name] = sources[i].RangeCount - prev
	}

	u.mu.Unlock()

	// Copy LastSync back to the updater's source list
	u.mu.Lock()
	for i := range sources {
		if !sources[i].LastSync.IsZero() && i < len(u.sources) {
			u.sources[i].LastSync = sources[i].LastSync
			u.sources[i].RangeCount = sources[i].RangeCount
		}
	}
	u.mu.Unlock()

	newDB := core.NewDatabase(allRanges)

	if u.onReload != nil {
		u.onReload(newDB)
	}
	if !silent {
		u.logf("updater.db.reloaded", len(newDB.Ranges()))
	}
}

func (u *Updater) logf(key string, args ...interface{}) {
	if u.logFn != nil {
		u.logFn(i18n.T(u.lang, key, args...))
	}
}
