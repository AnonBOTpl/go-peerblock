package main

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"go-peerblock/config"
	"go-peerblock/core"
	"go-peerblock/filter"
	"go-peerblock/logger"
	"go-peerblock/updater"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows/registry"
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
		runtime.LogError(ctx, "Nie można załadować konfiguracji: "+err.Error())
		cfg = config.Defaults()
	}
	a.cfg = cfg

	// Initialize logger
	logDir := filepath.Join(getAppDataDir(), "logs")
	_ = os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, "peerblock.log")
	logger, err := logger.NewLogger(logPath, 5000)
	if err != nil {
		runtime.LogError(ctx, "Nie można utworzyć loggera: "+err.Error())
	}
	a.logger = logger
	a.logger.Info("go-peerblock uruchomiony")

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
			// Clear before Store to prevent workers from caching old DB results
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
				_ = a.configP.Save(a.cfg)
			}
			a.logger.Info("Baza IP przeładowana: %d zakresów", len(newDB.Ranges()))
		},
		func(format string, args ...interface{}) {
			a.logger.Info(format, args...)
		},
		cfg.UpdateInterval,
	)
	go a.updater.Run(ctx)

	// Initialize WinDivert and pipeline (if protection is enabled)
	if cfg.ProtectionEnabled {
		a.startProtection()
	}
}

// shutdown is called when the Wails application exits.
func (a *App) shutdown(ctx context.Context) {
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
		a.logger.Info("Ochrona wyłączona")
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

// MinimizeToTray hides the application window to the system tray.
func (a *App) MinimizeToTray() {
	if a.ctx != nil {
		runtime.WindowHide(a.ctx)
	}
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
		a.logger.Error("Nie można otworzyć WinDivert: %v", err)
		return
	}
	a.logger.Info("WinDivert otwarty: %s", filter.DefaultFilter())

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
	a.logger.Info("Ochrona włączona (%d workerów)", workerCount)
}

// applyAutostart syncs the StartWithSystem config setting to the Windows registry.
// HKCU\Software\Microsoft\Windows\CurrentVersion\Run\go-peerblock
func (a *App) applyAutostart() {
	keyPath := `Software\Microsoft\Windows\CurrentVersion\Run`
	k, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		a.logger.Error("Nie można otworzyć klucza rejestru autostart: %v", err)
		return
	}
	defer k.Close()

	if a.cfg.StartWithSystem {
		exePath, err := os.Executable()
		if err != nil {
			a.logger.Error("Nie można pobrać ścieżki exe dla autostart: %v", err)
			return
		}
		if err := k.SetStringValue("go-peerblock", exePath); err != nil {
			a.logger.Error("Nie można ustawić autostart w rejestrze: %v", err)
		} else {
			a.logger.Info("Autostart włączony: %s", exePath)
		}
	} else {
		if err := k.DeleteValue("go-peerblock"); err != nil && err != registry.ErrNotExist {
			a.logger.Error("Nie można usunąć autostart z rejestru: %v", err)
		} else {
			a.logger.Info("Autostart wyłączony")
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
