# go-peerblock

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Windows](https://img.shields.io/badge/Windows-10%2B-0078D6?logo=windows)](https://www.microsoft.com/windows)
[![Wails](https://img.shields.io/badge/Wails-v2-DF0000?logo=wails)](https://wails.io)

**go-peerblock** is a high-performance Windows application for blocking network traffic based on dynamically updated IP blocklists. Built with Go and Wails v2 (React frontend), it uses the WinDivert driver to capture and filter packets at the network layer with minimal CPU overhead.

> 🇵🇱 [Czytaj po polsku](README.pl.md)

## Features

- **⚡ Blazing fast** — IP lookup in ~186ns on 500,000+ ranges (2× faster than the 400ns target)
- **🛡️ WinDivert-based filtering** — kernel-level packet capture with minimal overhead
- **🎨 Modern GUI** — React + TypeScript frontend via Wails v2 (WebView2)
- **🔄 Auto-updating blocklists** — periodic download and atomic reload of IP lists without packet loss
- **📊 Real-time statistics** — blocked/allowed counters, uptime, live log viewer
- **📈 Blocked packets per second chart** — live line chart with 5m/10m/30m zoom
- **🔍 Block source lookup** — click any blocked IP to see which blocklists contain it
- **📋 Per-list statistics** — view how many ranges each source contributes in the Sources tab
- **🔔 Windows toast notifications** — desktop notification when lists finish updating (toggleable)
- **📊 Range diffs** — after each update, see how many ranges were added/removed per source (▲/▼ badges)
- **🪟 Close dialog** — click X to choose: minimize to tray, close app, or cancel
- **⚙️ "Don't ask again" option** — Settings checkbox to always minimize to tray on close
- **🌍 Multi-language UI** — Polish and English UI with auto-detection of Windows system language
- **🗺️ User-translatable** — add your own language by editing `frontend/src/i18n/{lang}.ts`
- **🔤 Translated source descriptions** — source descriptions in the correct language
- **🔤 Language-aware logs** — backend logs switch between PL and EN based on selected language
- **📦 NSIS installer** — full installer with WinDivert driver setup/cleanup, WebView2 bootstrap, bilingual (PL/EN)
- **📄 MIT License in installer** — license page with AnonBOTpl copyright and GitHub link, requires acceptance
- **🔍 Multi-format parser** — supports PeerGuardian (.p2p), eMule DAT, CIDR, and plain range formats
- **✅ Allowlist support** — DNS-resolveable domain whitelisting
- **⚙️ Configurable cache** — LRU decision cache with adjustable TTL
- **🪟 System tray** — minimize to background, toggle protection from tray menu
- **📝 Async logging** — non-blocking file logger with ring buffer for GUI

## Performance

| Database Size | Lookup Time | Throughput | Target | Status |
|---|---|---|---|---|
| 10,000 ranges | **76 ns** | 13.1M ops/s | < 100 ns | ✅ **Exceeded** |
| 100,000 ranges | **197 ns** | 5.1M ops/s | < 200 ns | ✅ **Exceeded** |
| 500,000 ranges | **186 ns** | 5.4M ops/s | < 400 ns | ✅ **2× faster** |

These benchmarks demonstrate the binary search algorithm's efficiency on sorted, non-overlapping IP ranges. Performance meets or exceeds all targets from the project plan.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Wails v2 (GUI)                        │
│  React + TypeScript ←→ Go App struct (bindingi)         │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────┼───────────────────────────────────┐
│        Core Engine  │                                   │
│  ┌──────────────┐   │   ┌──────────────┐               │
│  │ IP Database  │   │   │   Allowlist  │               │
│  │  (binary     │   │   │  (DNS-based) │               │
│  │   search)    │   │   └──────────────┘               │
│  └──────────────┘   │   ┌──────────────┐               │
│  ┌──────────────┐   │   │  Cache (LRU) │               │
│  │   Parser     │   │   │  + config TTL│               │
│  └──────────────┘   │   └──────────────┘               │
└─────────────────────┼───────────────────────────────────┘
                      │
┌─────────────────────┼───────────────────────────────────┐
│  Filter Pipeline    │                Updater             │
│  ┌──────────┐ ┌────┴─────┐     ┌──────────┐            │
│  │ WinDivert│ │  Worker  │     │ Fetcher  │            │
│  │ Recv (1) │ │  Pool    │     │ (HTTP    │            │
│  └──────────┘ │  (N CPUs)│     │  + retry)│            │
│  ┌──────────┐ └────┬─────┘     └──────────┘            │
│  │ WinDivert│      │            ┌──────────┐            │
│  │ Send (1) │  Decision        │ Sources  │            │
│  └──────────┘  (block/allow)   └──────────┘            │
└─────────────────────────────────────────────────────────┘
```

### Key Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| **WinDivert bindings** | Custom minimal CGO | `go-windivert2` was incompatible with WinDivert 2.2.2 API changes |
| **Build tags** | `//go:build windivert` | Development without CGO, production with WinDivert |
| **Cache** | Simple LRU + RWMutex | 89ns lookup, no external dependencies vs. ristretto (~2MB) |
| **Cache TTL** | Configurable via `config.json` | Critical for fast-rotating lists like Spamhaus DROP |
| **Database** | Sorted slice + binary search | O(log n) with ~19 comparisons for 500k ranges |
| **Concurrency** | `atomic.Pointer` for hot path | Lock-free reads, atomic reload without packet loss |

## Requirements

### Development
- **Go 1.21+** (recommended: 1.23+)
- **Node.js 18+**
- **Wails CLI v2** — `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **GCC** (mingw-w64) — for CGO compilation

### Production
- **Windows 10/11** (64-bit)
- **WebView2 Runtime** (pre-installed on Windows 10+)
- **WinDivert 2.2.2** driver — download from [reqrypt.org](https://reqrypt.org/windivert.html)

## Quick Start

### 1. Clone and prepare
```bash
git clone https://github.com/AnonBOTpl/go-peerblock.git
cd go-peerblock
```

### 2. Install WinDivert SDK (required for production build)
Download [WinDivert-2.2.2-A.zip](https://reqrypt.org/download/WinDivert-2.2.2-A.zip) and extract:
```bash
# Place in project root
cp WinDivert-2.2.2-A/x64/WinDivert.dll .
cp WinDivert-2.2.2-A/x64/WinDivert64.sys .
cp WinDivert-2.2.2-A/include/windivert.h .
# Install header for CGO
cp windivert.h /mingw64/x86_64-w64-mingw32/include/
```

### 3. Install dependencies
```bash
go mod tidy          # Go dependencies
cd frontend && npm install  # Frontend dependencies
cd ..
```

### 4. Development mode (no WinDivert)
```bash
# Frontend + backend hot-reload (no packet capture)
wails dev
```

### 5. Production build (with WinDivert)
```bash
# Full packet capture as Administrator
CGO_ENABLED=1 go build -tags windivert -o go-peerblock.exe .
```
Run `go-peerblock.exe` **as Administrator**.

## Project Structure

```
go-peerblock/
├── main.go                  # Entry point, UAC check, Wails bootstrap
├── app.go                   # Wails App struct, exported binding methods
├── core/                    # IP engine
│   ├── database.go          # IPRange, MergeRanges, binary search
│   ├── parser.go            # Format detection and parsing (CIDR, P2P, DAT)
│   ├── cache.go             # LRU decision cache with configurable TTL
│   └── allowlist.go         # DNS-resolvable allowlist
├── filter/                  # Packet filtering
│   ├── windivert.go         # CGO WinDivert bindings (build tag: windivert)
│   ├── windivert_noop.go    # Noop stub (build tag: !windivert)
│   ├── pipeline.go          # Multi-worker packet pipeline
│   ├── pipeline_noop.go     # Noop pipeline stub
│   ├── shared.go            # Shared types (Packet, Stats, ParseIPHeader)
│   └── workerpool.go        # Worker count calculation
├── i18n/                    # Backend translations (T() helper, EN/PL maps)
│   └── i18n.go
├── updater/                 # IP list updates
│   ├── updater.go           # Update orchestrator
│   ├── sources.go           # Default source definitions
│   └── fetcher.go           # HTTP fetcher with retry + disk cache
├── logger/                  # Async logging
│   ├── logger.go            # Non-blocking file logger
│   └── ringbuffer.go        # Ring buffer for GUI log view
├── config/                  # Configuration
│   ├── config.go            # Config struct with defaults
│   └── persistence.go       # JSON persist to %APPDATA%
├── systray/                 # System tray
│   └── tray.go              # Tray icon and menu
├── frontend/                # React + TypeScript (Wails)
│   └── src/
│       ├── App.tsx          # Main app with routing + event handlers
│       ├── App.css          # Dark theme styling
│       ├── main.tsx         # Entry point
│       ├── components/
│       │   ├── DashboardView.tsx   # Stats overview (counters, uptime)
│       │   ├── ChartsView.tsx      # PPS chart + clickable blocked IP list
│       │   ├── LogView.tsx         # Filterable system/blocked log viewer
│       │   ├── SettingsView.tsx    # Full settings panel
│       │   ├── SourcesView.tsx     # Source management + per-list stats
│       │   ├── AddSourceDialog.tsx # Dialog to add new blocklist source
│       │   ├── SourceDialog.tsx    # Block source lookup results modal
│       │   └── ConfigView.tsx      # Config file viewer
├── build/
│   ├── windows/
│   │   ├── icon.ico         # App icon
│   │   ├── license.txt      # MIT License for installer
│   │   ├── info.json        # Wails metadata
│   │   └── installer/
│   │       └── project.nsi  # NSIS installer script (WinDivert, WebView2)
│   ├── appicon.png           # Icon source for Wails
│   └── README.md
├── windivert.h              # WinDivert C header (needed for CGO)
```

## Tests

```bash
# Run all unit tests with race detector
go test -race ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Build verification (both modes)
go build ./...                          # Noop mode
CGO_ENABLED=1 go build -tags windivert ./...  # Production mode
```

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

Built with ❤️ by [AnonBOTpl](https://github.com/AnonBOTpl)
