# Changelog

All notable changes to this project will be documented in this file.

> 🇵🇱 [Polska wersja](CHANGELOG.pl.md)

## [0.4.1] — 2026-06-08

### Fixed

#### WinDivert driver not starting after reboot

- `main.go` — `installDriver()` rewritten: instead of running a non-existent batch file from `build/installer/install-driver.bat`, now calls `sc` commands directly
- `main.go` — added `isDriverInstalled()`: checks if the WinDivert service entry exists (vs. just checking if it's running)
- `main.go` — added `findSysPath()`: searches for `WinDivert64.sys` in the executable directory and current working directory, handling both installed runtime and development builds
- `main.go` — added `removeDriverService()`: cleans up broken WinDivert service entries (e.g., when the old `binPath` pointed to a temp directory that was cleared on reboot)
- `installDriver()` now handles 3 scenarios: (1) driver running → skip, (2) driver exists but stopped → `sc start`, (3) driver service broken → `sc delete` + `sc create` + `sc start`

#### Build dependency fix

- `updater/updater_test.go` — added missing 6th `"en"` parameter to all 10 `NewUpdater()` calls (the function was updated in v0.4.0 to accept a `lang` string)

### Changed

#### NSIS installer — driver auto-start

- `build/windows/installer/project.nsi` — WinDivert service registration changed from `start= demand` to `start= auto`
- On fresh installs, the driver will now start automatically when Windows boots, eliminating the "app won't start after reboot" issue permanently

### Added

#### WinDivert troubleshooting docs

- `README.md` — new **Troubleshooting** section with 4 subsections:
  - "App won't start after reboot" — cause and step-by-step recovery with `sc` commands
  - "Check driver status" — how to verify WinDivert is running (`sc query`)
  - "Driver doesn't auto-start after fresh install" — how to fix with `sc config start= auto`
  - "Run as Administrator" — permission requirement
- `README.pl.md` — same content translated to Polish in **Rozwiązywanie problemów** section

## [0.4.0] — 2026-06-08

### Added

#### Multi-language UI (PL/EN)
- `frontend/src/i18n/index.tsx` — `I18nProvider` + `useT()` hook with React context, params interpolation, fallback to English
- `frontend/src/i18n/pl.ts`, `frontend/src/i18n/en.ts` — ~120 translation keys each for the entire UI
- All components: `App.tsx`, `Dashboard.tsx`, `SourcesView.tsx`, `SettingsView.tsx`, `LogView.tsx`, `ChartsView.tsx`, `AddSourceDialog.tsx`, `SourceDialog.tsx` — migrated from hardcoded strings to `t()` calls
- `config/config.go` — `Language string` field, defaults to `""` (triggers autodetection)
- `app.go` — `detectSystemLanguage()` using `windows.GetUserPreferredUILanguages()`, auto-detects PL system language
- `SettingsView.tsx` — language selector dropdown (PL/EN) with backend save

#### Backend i18n (language-aware logs)
- `i18n/i18n.go` — new package: `T(lang, key, args...)` with EN/PL maps (~25 keys each)
- `app.go` — all `Info/Warn/Error/Debug` calls use `i18n.T(a.GetLanguage(), ...)`
- `updater/updater.go` — `logf()` translates via `i18n.T(u.lang, ...)`, accepts `lang` parameter from caller
- `main.go` — startup error messages in English (before app initialization)
- `systray/tray.go` — tray menu uses `i18n.T(lang, key)` instead of hardcoded if-else branching

#### Custom user rules (I7)
- `SettingsView.tsx` — new textarea in Ustawienia for custom CIDR/IP/range rules
- `app.go` — `parseCustomRuleLines()` parses and merges custom rules into the IP database
- Config saved to `config.json` and reloaded on startup

#### NSIS installer with WinDivert
- `build/windows/installer/project.nsi` — custom installer script with WinDivert driver handling
- `build/windows/license.txt` — MIT License displayed in installer with Copyright (c) 2026 AnonBOTpl + GitHub link
- Optional desktop shortcut via Components page (unchecked by default)
- Start menu shortcut created always
- Driver installed: `sc create` + `sc start` on install
- Driver removed: `sc stop` + `sc delete` on uninstall
- WebView2 Runtime bootstrap via Wails
- AppData (`%APPDATA%`) preserved on uninstall
- Bilingual installer (English + Polish)

### Changed

#### Source descriptions always in English
- `updater/sources.go` — all 9 default source descriptions translated to English
- `frontend/src/i18n/pl.ts`, `frontend/src/i18n/en.ts` — 10 `source.desc.{name}` keys for translated display in GUI
- `SourcesView.tsx` — `getSourceDesc()` helper shows translated description or falls back to stored value for custom sources

#### Install directory fixed
- `build/windows/installer/project.nsi` — changed from `$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}` to `$PROGRAMFILES64\${INFO_PRODUCTNAME}`
- Resolves double directory issue: now `C:\Program Files\go-peerblock\` instead of `C:\Program Files\go-peerblock\go-peerblock\`

#### Git tracking cleanup
- `.gitignore` — new patterns for `build/windows/installer/tmp/`, `build/windows/*.manifest`, `build/darwin/`, `build/installer/`, audit/plan files
- Removed from git: `WinDivert.dll`, `WinDivert64.sys`, `build/installer/`, `build/windows/*.manifest`, `build/darwin/`, audit/plan files, `test-blocklist.txt`, `frontend/package.json.md5`
- Kept: `windivert.h` (C header), `build/windows/info.json` (Wails metadata)
- Repository now contains only source code, config, documentation, and essential build resources (~81 files)

### Fixed

#### Audit fixes (all 9 items from go-peerblock-audit-final.md)

| # | File | Fix |
|---|---|---|
| 1 | `updater/updater.go` | `NewDatabase(allRanges)` moved outside `u.mu.Lock()` — lock held for less time |
| 2 | `app.go` | `a.sourceRanges` changed from raw map to `atomic.Pointer` — eliminates race condition |
| 3 | `logger/logger.go` | `rotateIfNeeded()` called every 100 writes instead of every log entry |
| 4 | `systray/tray.go` + `i18n/i18n.go` | Systray menu now uses `i18n.T()` with proper language keys |
| 5 | `core/allowlist.go` | `isIPString()` now checks `ip.To4() != nil` — rejects IPv6 addresses, safe nil check |
| 6 | `app.go` | `syscall.NewLazyDLL` → `windows.GetUserPreferredUILanguages()` |
| 7 | `updater/updater.go` | `logf()` removed redundant `"%s"` wrapper — passes formatted message directly |
| 8 | `updater/fetcher.go` | `io.LimitReader(resp.Body, 100MB)` limits HTTP download size |
| 9 | `config/config.go` | `Defaults()` sets `Language: ""` sentinel — triggers autodetection on clean install |

## [0.3.0] — 2026-06-07

### Added

#### Charts tab with live line chart
- `frontend/src/components/ChartsView.tsx` — new **📈 Wykresy** tab with Chart.js line chart showing blocked (red) vs allowed (green) packets/s over time
- Time range switcher: 5m / 10m / 30m with animated button group
- Data collection auto-pauses when the tab isn't active (`collectingRef` approach — no wasted CPU)
- Empty state with "Zbieranie danych..." hint while samples accumulate
- Tab placed before Settings in navigation

#### Packets-per-second in status bar
- `frontend/App.tsx` — PPS calculated inline from `stats.started_at` (UnixNano → elapsed seconds), displayed as "Pakiety: X (Y/s)" in the footer

#### Range count per source
- `updater/sources.go` — `RangeCount int` field tracks how many IP ranges each blocklist contributed
- `frontend/src/components/SourcesView.tsx` — each source shows a green badge with "X zakresów" after update
- Value synced back to `u.sources` during the sync-back loop alongside `LastSync`

#### Custom application icon
- `frontend/src/assets/ikona.png` — 500×500 custom icon replaces placeholder text in header
- `frontend/index.html` — favicon linked to `ikona.png`
- `build/appicon.png` — source icon for Wails .exe icon generation

### Changed

#### Tab order
- Wykresy tab moved before Ustawienia: Dashboard → Źródła → **Wykresy** → Ustawienia

#### Window title
- `main.go` + `frontend/index.html` — title changed from "go-peerblock" to **"GO PeerBlock - IP Filter"**

#### Merged ideas.md into fixes.md
- All items from `ideas.md` mapped as I1–I8 into existing audit categories
- Duplicate entries merged (A17 + I8 → single "Statystyki historyczne")

### Fixed

#### A12 — Double MergeRanges in updater
- `updater/updater.go`: `updateAll()` was calling `MergeRanges` before `NewDatabase()`, which calls it again internally. Removed the redundant call — now passes `allRanges` directly.

#### Ghost icon in system tray
- `systray/tray.go`: added `time.Sleep(200ms)` in `onExit()` before `systray.Quit()` — ensures the icon is fully removed from the notification area before the process exits.

#### Systray tooltip consistency
- `systray/tray.go`: tooltip updated from "go-peerblock - IP Blocker" to **"GO PeerBlock - IP Filter"** matching the window title.

### Dependencies

- Added `chart.js` + `react-chartjs-2` for the Charts tab

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
