# Changelog

All notable changes to this project will be documented in this file.

> 🇵🇱 [Polska wersja](CHANGELOG.pl.md)

## [0.2.0] — 2026-06-07

### Fixed

#### 🔴 Critical — SrcIP blocking everything
- `filter/pipeline.go`: `shouldBlock` was checking **both SrcIP and DstIP** against the DB. The user's local IP (172.16.3.206) fell within firehol-level1's `172.16.0.0/12` range, causing **every outgoing packet to be blocked**. Fixed: only DstIP is checked (source IP is the user's local interface, never a malicious target).

#### 🟠 Race conditions & cache poisoning
- `filter/pipeline.go`: removed re-verification of cached `blocked=true` entries against the DB on every hit — was defeating cache purpose. Replaced with **cache versioning** (see Changed).
- `app.go`: removed duplicate `cache.Clear()` workaround in `onReload` — no longer needed after versioning.

#### Other fixes
- `app.go`: `LastSync` now properly synced from updater back to config after each update (GUI showed stale dates).
- `frontend/App.tsx`: Update buttons in header and SourcesView now share a single `updating` state — no more desync.
- `filter/pipeline_noop.go`: added missing method signatures to match the windivert build.
- `main.go`: removed `init()` with `runtime.LockOSThread()` and global `appCtx` — Wails and systray manage threads themselves.

### Changed

#### Cache versioning (O(1) invalidation)
- `core/cache.go`: `Clear()` now increments an `atomic.Uint64` version counter instead of rebuilding the map (O(n)). Entries with stale versions are ignored by `Get()`. `Set()` stores the current version with each entry.
- This eliminates the race condition where a worker could cache a decision from the old DB after `Clear()` but before `Store()`.

#### Minimize-to-tray
- `main.go`: systray now starts in a goroutine **before** `wails.Run()`, keeping the process alive when the window is hidden.
- `app.go`: added `MinimizeToTray()` → `runtime.WindowHide()`.
- `systray/tray.go`: "Zamknij" now calls `runtime.Quit(ctx)` before `systray.Quit()` for clean shutdown.
- `frontend/App.tsx`: ⬇ button in header hides the window to system tray. Restore via tray icon → "Pokaż okno".

#### Autostart with Windows
- `app.go`: added `applyAutostart()` — writes/deletes `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\go-peerblock` using `golang.org/x/sys/windows/registry`.
- Called on startup and whenever `SaveConfig()` is invoked.
- `frontend/App.tsx`: new "System" section in Settings with toggle "Uruchamiaj z systemem Windows".

### Added

#### Settings panel (GUI)
- `frontend/App.tsx`: new **⚙️ Ustawienia** tab with editable fields:
  - Allowlist (textarea, one entry per line, `#` comments stripped)
  - Worker count (0 = auto/NumCPU)
  - Cache size (number of entries)
  - Cache TTL (in minutes, converts to/from nanoseconds for Go's `time.Duration`)
  - Update interval (in hours)
  - Log level (dropdown: DEBUG/INFO/WARN/ERROR)
  - "Uruchamiaj z systemem Windows" toggle
  - "Przywróć domyślną allowlistę" button (with confirm dialog)

#### Cache usage indicator
- `frontend/App.tsx`: new **Cache** stat card on Dashboard showing entries/max (e.g. "128 / 65,536"), styled in subtle slate color.
- `app.go`: added `GetCacheInfo()` exposing cache entries count and max capacity.

#### Multicast in default allowlist
- `config/config.go`: added `"224.0.0.0/4"` (multicast: SSDP, mDNS, BitTorrent LPD) to default allowlist.

### Benchmarks

- Cache Clear: **O(1)** instead of O(n) — no measurable allocation cost
- Cache Get/Set: unchanged (~89ns / ~242ns)
- Binary search: unchanged (~186ns on 500k ranges)

## [0.1.0] — 2026-06-06

### Added

#### Core Engine
- `core/database.go` — IPRange struct, MergeRanges (sort+merge adjacent/overlapping ranges), CIDRToRange, binary search Contains()
- `core/parser.go` — Format detection (CIDR, P2P Text, DAT), parsing for all supported formats
- `core/cache.go` — LRU DecisionCache with configurable TTL (default 5 min), ring buffer eviction
- `core/allowlist.go` — Allowlist with static IP, CIDR ranges, and DNS-resolvable domain entries
- `core/database_test.go` — Unit tests + benchmarks (500k ranges: ~186ns lookup)
- `core/cache_test.go` — Unit tests for eviction, TTL expiry, concurrent access
- `core/parser_test.go` — Tests for format detection and parsing correctness

#### Packet Filtering
- `filter/windivert.go` — Custom minimal CGO bindings for WinDivert 2.2.2 (no external dependency)
- `filter/windivert_noop.go` — Noop stub for development without WinDivert
- `filter/pipeline.go` — Multi-worker packet pipeline (recv → workers → send)
- `filter/pipeline_noop.go` — Noop pipeline stub
- `filter/shared.go` — Packet/Stats structs, ParseIPHeader, DefaultFilter
- `filter/workerpool.go` — RecommendedWorkerCount calculation

#### IP List Updater
- `updater/updater.go` — Periodic update orchestrator with manual trigger
- `updater/sources.go` — Default blocklist sources (Firehol, Spamhaus DROP, iblocklist)
- `updater/fetcher.go` — HTTP fetcher with retry, backoff, and disk cache fallback

#### Logging
- `logger/logger.go` — Async non-blocking file logger
- `logger/ringbuffer.go` — Thread-safe ring buffer for GUI log view

#### Configuration
- `config/config.go` — Config struct with defaults (CacheTTL, worker count, etc.)
- `config/persistence.go` — JSON save/load to `%APPDATA%\go-peerblock\config.json`

#### GUI (Wails v2 + React)
- `frontend/src/App.tsx` — Dashboard with protection toggle, stats cards, log viewer, status bar
- `frontend/src/App.css` — Dark theme styling

#### System Tray
- `systray/tray.go` — System tray icon with show/toggle/quit menu

#### Build & Deployment
- `main.go` — Entry point with UAC check, Wails bootstrap
- `app.go` — App struct with exported Wails binding methods
- `build/windows/go-peerblock.exe.manifest` — UAC requireAdministrator manifest
- `build/installer/install-driver.bat` — WinDivert driver installation script

### Changed
- WinDivert bindings: switched from `go-windivert2` (incompatible with v2.2.2) to custom minimal CGO
- Format .p2b: removed (obsolete, zero usage)
- Cache TTL: made configurable via `config.json` instead of hardcoded 5 minutes
- WebView2: explicit bootstrapper check in NSIS installer
- Uptime: changed from `time.Time` to `int64` (UnixNano) for clean JSON serialization

### Added
- GUI Sources tab: list of blocklist sources with enable/disable toggle switches
- CIDR parser: inline comment stripping (after `;` or `#`) for Spamhaus DROP format (`1.2.3.0/24 ; SBL123`)
- Updater logger: per-source progress messages visible in GUI log panel
- Configurable update interval from config.json
- Fetcher: User-Agent header and automatic gzip decompression

### Changed
- Default sources: replaced dead iblocklist-level1 with working firehol-level2
- Updater API: `NewUpdater` now accepts `LogFunc` callback and configurable `interval`

### Fixed
- WinDivert infinite loop: added Impostor flag check — re-injected packets bypass pipeline (prevents capture loop, restoring internet connectivity)
- WinDivert handle leak: `ToggleProtection` and `SetProtectionEnabled` now use `Close()` instead of `Stop()`, properly closing the WinDivert handle
- Pipeline goroutine leak: worker/sendLoop now use `select` with `<-p.done` for clean shutdown
- `startProtection()`: closes existing pipeline before creating a new one (prevents duplicate WinDivert handles)
- `isAdmin()`: removed incorrect `os.IsPermission(err)` check which falsely reported non-admin users as admin
- CGO build: downgraded mingw-w64 to 13.2.0 (16.1.0 was missing `stddef.h`)
- TypeScript bindings: fixed namespace imports for `filter.Stats` and `logger.LogEntry`

### Benchmarks
- IP lookup (10k ranges): **76 ns/op** (target: < 100 ns)
- IP lookup (100k ranges): **197 ns/op** (target: < 200 ns)
- IP lookup (500k ranges): **186 ns/op** (target: < 400 ns)
- Cache set: **242 ns/op**
- Cache get: **89 ns/op**

### Known Issues
- WinDivert kernel driver may be flagged by Windows Defender (requires code signing)
- CGO compilation requires mingw-w64 with complete headers (recommended: 13.x)
- WebView2 required on older Windows 10 builds (bootstrapper included in installer)
