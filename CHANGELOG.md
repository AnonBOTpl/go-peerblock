# Changelog

All notable changes to this project will be documented in this file.

> 🇵🇱 [Polska wersja](CHANGELOG.pl.md)

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
