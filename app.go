package main

import (
	"context"
	"encoding/binary"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"go-peerblock/config"
	"go-peerblock/core"
	"go-peerblock/filter"
	"go-peerblock/i18n"
	"go-peerblock/logger"
	"go-peerblock/updater"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows/registry"
	"syscall"
)

// App is the main application struct, binding Go methods to the Wails frontend.
type App struct {
	ctx           context.Context
	pipeline       *filter.Pipeline
	updater        *updater.Updater
	logger         *logger.Logger
	cfg            *config.Config
	configP        *config.Persistence
	db             atomic.Pointer[core.IPDatabase]
	cache          *core.DecisionCache
	allowlist      *core.Allowlist
	allowlistDone  chan struct{}
	logSubCh      chan logger.LogEntry
	eventsDone    chan struct{}
	sourceRanges  map[string][]core.IPRange
	customRanges  []core.IPRange
	quitting      atomic.Bool
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// startup is called when the Wails application starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize configuration
	a.configP = config.NewPersistence()
	cfg, err := a.configP.Load()
	if err != nil {
		runtime.LogError(ctx, i18n.T("en", "app.config.load.error", err.Error()))
		cfg = config.Defaults()
	}
	a.cfg = cfg

	// Auto-detect system language if not set
	if a.cfg.Language == "" || (a.cfg.Language != "pl" && a.cfg.Language != "en") {
		a.cfg.Language = a.detectSystemLanguage()
	}

	// Parse custom rules from config
	a.customRanges = parseCustomRuleLines(a.cfg.CustomRules)

	// Initialize logger
	logDir := filepath.Join(getAppDataDir(), "logs")
	_ = os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, "peerblock.log")
	logger, err := logger.NewLogger(logPath, 5000, cfg.LogMaxSizeMB)
	if err != nil {
		runtime.LogError(ctx, i18n.T("en", "app.logger.create.error", err.Error()))
	}
	a.logger = logger
	a.logger.Info(i18n.T(a.GetLanguage(), "app.started"))

	// Forward log entries to frontend in real-time via Wails events
	a.logSubCh = a.logger.Subscribe()
	go func() {
		for entry := range a.logSubCh {
			runtime.EventsEmit(a.ctx, "log", entry)
		}
	}()

	// Apply autostart setting (sync config state to registry)
	a.applyAutostart()

	// Initialize IP database (empty initially)
	db := core.NewDatabase(nil)
	a.db.Store(db)

	// Initialize cache with configurable TTL
	cacheTTL := cfg.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}
	a.cache = core.NewDecisionCache(cfg.CacheSize, cacheTTL)

	// Initialize allowlist
	a.allowlist = core.NewAllowlist(cfg.Allowlist)
	a.allowlistDone = make(chan struct{})
	go a.allowlist.StartRefreshLoop(30*time.Minute, a.allowlistDone)

	// Initialize updater
	fetcher := updater.NewFetcher(filepath.Join(getAppDataDir(), "cache"))
	a.updater = updater.NewUpdater(cfg.Sources, fetcher,
		func(newDB *core.IPDatabase) {
			// Get per-source ranges first (needed for custom rules merge)
		a.sourceRanges = a.updater.GetSourceRanges()

			// Merge custom rules into the database
		if len(a.customRanges) > 0 {
			var allRanges []core.IPRange
			for _, ranges := range a.sourceRanges {
				allRanges = append(allRanges, ranges...)
			}
			allRanges = append(allRanges, a.customRanges...)
			newDB = core.NewDatabase(allRanges)
		}

		a.cache.Clear() // O(1) version increment — stale entries become invisible
		a.db.Store(newDB)
			// Sync LastSync from updater back to config so GUI sees correct dates
			if upSources := a.updater.GetSources(); len(upSources) > 0 {
				for i := range a.cfg.Sources {
					for _, us := range upSources {
						if us.Name == a.cfg.Sources[i].Name && !us.LastSync.IsZero() {
							a.cfg.Sources[i].LastSync = us.LastSync
							break
						}
					}
				}
				// Backup config before saving (I5 — auto-backup przed aktualizacją)
				if err := a.configP.Backup(); err != nil {
					a.logger.Warn(i18n.T(a.GetLanguage(), "app.config.backup.error", err))
				}
				_ = a.configP.Save(a.cfg)
			}
			a.logger.Info(i18n.T(a.GetLanguage(), "app.db.reloaded", len(newDB.Ranges())))
			// Source ranges already saved at the top of this callback
			// Notify frontend about the database and cache changes
			runtime.EventsEmit(a.ctx, "db-info", a.GetDatabaseInfo())
			runtime.EventsEmit(a.ctx, "cache-info", a.GetCacheInfo())
			// Signal update completion so frontend can re-enable the button
			diffs := a.updater.GetRangeDiffs()
			runtime.EventsEmit(a.ctx, "update-status", map[string]interface{}{
				"ok":     true,
				"ranges": len(newDB.Ranges()),
				"diffs":  diffs,
			})
		},
		func(format string, args ...interface{}) {
			a.logger.Debug(format, args...)
		},
		cfg.UpdateInterval,
		a.GetLanguage(),
	)
	go a.updater.Run(ctx)

	// Emit initial db-info and cache-info
	runtime.EventsEmit(a.ctx, "db-info", a.GetDatabaseInfo())
	runtime.EventsEmit(a.ctx, "cache-info", a.GetCacheInfo())

	// Start the stats event emitter (every 1 second)
	a.eventsDone = make(chan struct{})
	go a.eventsEmitter()

	// Initialize WinDivert and pipeline (if protection is enabled)
	if cfg.ProtectionEnabled {
		a.startProtection()
	}
}

// shutdown is called when the Wails application exits.
func (a *App) shutdown(ctx context.Context) {
	a.eventsStop()
	if a.pipeline != nil {
		a.pipeline.Close()
		a.pipeline = nil
	}
	if a.allowlistDone != nil {
		close(a.allowlistDone)
	}
	if a.logger != nil {
		if err := a.logger.Close(); err != nil {
			runtime.LogError(ctx, "Logger close error: "+err.Error())
		}
	}
}

// GetCtx returns the application context (used by systray).
func (a *App) GetCtx() context.Context {
	return a.ctx
}

// --- Wails exported methods ---

// GetStats returns current pipeline statistics.
func (a *App) GetStats() filter.Stats {
	if a.pipeline == nil {
		return filter.Stats{}
	}
	return a.pipeline.GetStats()
}

// eventsEmitter sends real-time stats to the frontend every second.
func (a *App) eventsEmitter() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			stats := a.GetStats()
			runtime.EventsEmit(a.ctx, "stats", stats)
		case <-a.eventsDone:
			return
		}
	}
}

// eventsStop cleanly shuts down event emitters and log subscription.
func (a *App) eventsStop() {
	if a.eventsDone != nil {
		close(a.eventsDone)
	}
	if a.logSubCh != nil {
		a.logger.Unsubscribe(a.logSubCh)
		a.logSubCh = nil
	}
}

// GetLogs returns the last n log entries.
func (a *App) GetLogs(n int) []logger.LogEntry {
	if a.logger == nil {
		return nil
	}
	return a.logger.Ring().Last(n)
}

// TriggerUpdate triggers a manual update of IP lists.
func (a *App) TriggerUpdate() {
	if a.updater != nil {
		go a.updater.TriggerManual()
	}
}

// IsProtectionEnabled returns whether packet filtering is active.
func (a *App) IsProtectionEnabled() bool {
	if a.pipeline == nil {
		return false
	}
	return a.pipeline.IsRunning()
}

// ToggleProtection toggles packet filtering on/off.
func (a *App) ToggleProtection() {
	if a.pipeline != nil && a.pipeline.IsRunning() {
		a.pipeline.Close()
		a.pipeline = nil
		a.logger.Info(i18n.T(a.GetLanguage(), "app.protection.disabled"))
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "protection", false)
		}
	} else {
		a.startProtection()
	}
}

// SetProtectionEnabled enables or disables protection.
func (a *App) SetProtectionEnabled(enabled bool) {
	if enabled {
		a.startProtection()
	} else {
		if a.pipeline != nil {
			a.pipeline.Close()
			a.pipeline = nil
		}
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "protection", false)
		}
	}
	a.cfg.ProtectionEnabled = enabled
	_ = a.configP.Save(a.cfg)
}

// GetConfig returns the current configuration.
func (a *App) GetConfig() config.Config {
	return *a.cfg
}

// SaveConfig saves a new configuration.
func (a *App) SaveConfig(cfg config.Config) error {
	*a.cfg = cfg

	// Re-parse custom rules and rebuild database
	a.customRanges = parseCustomRuleLines(cfg.CustomRules)
	a.rebuildDB()

	if a.updater != nil {
		a.updater.RefreshSources(cfg.Sources)
	}
	a.applyAutostart()
	return a.configP.Save(a.cfg)
}

// GetDatabaseInfo returns information about the IP database.
func (a *App) GetDatabaseInfo() map[string]interface{} {
	db := a.db.Load()
	if db == nil {
		return map[string]interface{}{
			"ranges": 0,
		}
	}
	return map[string]interface{}{
		"ranges": len(db.Ranges()),
	}
}

// GetCacheInfo returns cache usage information.
func (a *App) GetCacheInfo() map[string]interface{} {
	if a.cache == nil {
		return map[string]interface{}{
			"entries": 0,
			"max":     65536,
		}
	}
	return map[string]interface{}{
		"entries": a.cache.Len(),
		"max":     a.cfg.CacheSize,
	}
}

// ResetAllowlist resets the allowlist to the default values and saves config.
func (a *App) ResetAllowlist() error {
	defaults := config.Defaults()
	a.cfg.Allowlist = defaults.Allowlist
	a.allowlist = core.NewAllowlist(a.cfg.Allowlist)
	return a.configP.Save(a.cfg)
}

// LookupBlockSource returns which source lists contain the given IP address.
func (a *App) LookupBlockSource(ipStr string) []string {
	ip := net.ParseIP(ipStr).To4()
	if ip == nil {
		return nil
	}
	ipU32 := binary.BigEndian.Uint32(ip)

	if a.sourceRanges == nil {
		return nil
	}

	var sources []string
	for name, ranges := range a.sourceRanges {
		for _, r := range ranges {
			if ipU32 >= r.Start && ipU32 <= r.End {
				sources = append(sources, name)
				break
			}
		}
	}
	return sources
}

// MinimizeToTray hides the application window to the system tray.
func (a *App) MinimizeToTray() {
	if a.ctx != nil {
		runtime.WindowHide(a.ctx)
	}
}

// QuitApp closes the application entirely.
func (a *App) QuitApp() {
	if a.ctx != nil {
		a.quitting.Store(true)
		runtime.Quit(a.ctx)
	}
}

// isQuitting returns true if the app is in the process of shutting down.
func (a *App) isQuitting() bool {
	return a.quitting.Load()
}

// GetLanguage returns the current interface language.
func (a *App) GetLanguage() string {
	if a.cfg == nil {
		return "en"
	}
	return a.cfg.Language
}

// detectSystemLanguage detects the Windows UI language.
func (a *App) detectSystemLanguage() string {
	// Kernel32.GetUserDefaultUILanguage returns the default UI language ID
	mod := syscall.NewLazyDLL("kernel32.dll")
	proc := mod.NewProc("GetUserDefaultUILanguage")
	ret, _, _ := proc.Call()
	langID := uint16(ret)
	primary := langID & 0x3FF
	if primary == 0x15 {
		return "pl"
	}
	return "en"
}

// rebuildDB rebuilds the IP database from source ranges + custom rules.
// Called after SaveConfig (custom rules changed) or when needed.
func (a *App) rebuildDB() {
	var allRanges []core.IPRange
	for _, ranges := range a.sourceRanges {
		allRanges = append(allRanges, ranges...)
	}
	allRanges = append(allRanges, a.customRanges...)
	db := core.NewDatabase(allRanges)
	a.cache.Clear()
	a.db.Store(db)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "db-info", a.GetDatabaseInfo())
	}
}

// parseCustomRuleLines parses custom rule strings (CIDR, bare IP, range) into a merged IPRange slice.
func parseCustomRuleLines(lines []string) []core.IPRange {
	if len(lines) == 0 {
		return nil
	}
	var buf strings.Builder
	for _, line := range lines {
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	ranges, err := core.Parse([]byte(buf.String()), core.FormatCIDR)
	if err != nil {
		return nil
	}
	return core.MergeRanges(ranges)
}

// --- Internal helpers ---

func (a *App) startProtection() {
	// Close any existing pipeline first (closes old WinDivert handle)
	if a.pipeline != nil {
		a.pipeline.Close()
		a.pipeline = nil
	}

	workerCount := a.cfg.WorkerCount
	if workerCount <= 0 {
		workerCount = filter.RecommendedWorkerCount()
	}

	wd, err := filter.Open(filter.DefaultFilter(), 0, 0) // layer=0 (Network), priority=0
	if err != nil {
		a.logger.Error(i18n.T(a.GetLanguage(), "app.windivert.open.error", err))
		return
	}
	a.logger.Debug(i18n.T(a.GetLanguage(), "app.windivert.opened", filter.DefaultFilter()))

	a.pipeline = filter.NewPipeline(wd, &a.db, a.cache, a.allowlist, workerCount)
	a.pipeline.SetOnBlock(func(srcIP, dstIP uint32, proto uint8) {
		src := core.Uint32ToIP(srcIP)
		dst := core.Uint32ToIP(dstIP)
		protoName := "?"
		switch proto {
		case 6:
			protoName = "TCP"
		case 17:
			protoName = "UDP"
		case 1:
			protoName = "ICMP"
		}
		a.logger.Info("BLOCK %s → %s [%s]", src, dst, protoName)
	})
	a.pipeline.Start()
	a.logger.Info(i18n.T(a.GetLanguage(), "app.protection.enabled", workerCount))
	// Notify frontend
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "protection", true)
	}
}

// applyAutostart syncs the StartWithSystem config setting to the Windows registry.
// HKCU\Software\Microsoft\Windows\CurrentVersion\Run\go-peerblock
func (a *App) applyAutostart() {
	keyPath := `Software\Microsoft\Windows\CurrentVersion\Run`
	k, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		a.logger.Error(i18n.T(a.GetLanguage(), "app.autostart.open.error", err))
		return
	}
	defer k.Close()

	if a.cfg.StartWithSystem {
		exePath, err := os.Executable()
		if err != nil {
			a.logger.Error(i18n.T(a.GetLanguage(), "app.autostart.path.error", err))
			return
		}
		if err := k.SetStringValue("go-peerblock", exePath); err != nil {
			a.logger.Error(i18n.T(a.GetLanguage(), "app.autostart.set.error", err))
		} else {
			a.logger.Debug(i18n.T(a.GetLanguage(), "app.autostart.enabled", exePath))
		}
	} else {
		if err := k.DeleteValue("go-peerblock"); err != nil && err != registry.ErrNotExist {
			a.logger.Error(i18n.T(a.GetLanguage(), "app.autostart.delete.error", err))
		} else {
			a.logger.Debug(i18n.T(a.GetLanguage(), "app.autostart.disabled"))
		}
	}
}

func getAppDataDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(appData, "go-peerblock")
}
