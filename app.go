package main

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"

	"go-peerblock/config"
	"go-peerblock/core"
	"go-peerblock/filter"
	"go-peerblock/logger"
	"go-peerblock/updater"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the main application struct, binding Go methods to the Wails frontend.
type App struct {
	ctx       context.Context
	pipeline  *filter.Pipeline
	updater   *updater.Updater
	logger    *logger.Logger
	cfg       *config.Config
	configP   *config.Persistence
	db        atomic.Pointer[core.IPDatabase]
	cache     *core.DecisionCache
	allowlist *core.Allowlist
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// startup is called when the Wails application starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	appCtx = ctx

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

	// Initialize IP database (empty initially)
	db := core.NewDatabase(nil)
	a.db.Store(db)

	// Initialize cache with configurable TTL
	cacheTTL := cfg.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 5 * 60 * 1000000000 // 5 minutes default
	}
	a.cache = core.NewDecisionCache(cfg.CacheSize, cacheTTL)

	// Initialize allowlist
	a.allowlist = core.NewAllowlist(cfg.Allowlist)
	go a.allowlist.StartRefreshLoop(30*60*1000000000, make(chan struct{})) // 30 min refresh

	// Initialize updater
	fetcher := updater.NewFetcher(filepath.Join(getAppDataDir(), "cache"))
	a.updater = updater.NewUpdater(cfg.Sources, fetcher, func(newDB *core.IPDatabase) {
		a.db.Store(newDB)
		a.logger.Info("Baza IP przeładowana: %d zakresów", len(newDB.Ranges()))
	})
	go a.updater.Run(ctx)

	// Initialize WinDivert and pipeline (if protection is enabled)
	if cfg.ProtectionEnabled {
		a.startProtection()
	}
}

// shutdown is called when the Wails application exits.
func (a *App) shutdown(ctx context.Context) {
	if a.pipeline != nil {
		a.pipeline.Stop()
	}
	if a.logger != nil {
		_ = a.logger.Close()
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
	if a.pipeline == nil {
		a.startProtection()
		return
	}
	if a.pipeline.IsRunning() {
		a.pipeline.Stop()
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
			a.pipeline.Stop()
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

// --- Internal helpers ---

func (a *App) startProtection() {
	db := a.db.Load()
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

	a.pipeline = filter.NewPipeline(wd, db, a.cache, a.allowlist, workerCount)
	a.pipeline.Start()
	a.logger.Info("Ochrona włączona (%d workerów)", workerCount)
}

func getAppDataDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(appData, "go-peerblock")
}
