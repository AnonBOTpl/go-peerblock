# go-peerblock

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://golang.org)
[![Licencja: MIT](https://img.shields.io/badge/Licencja-MIT-yellow.svg)](LICENSE)
[![Windows](https://img.shields.io/badge/Windows-10%2B-0078D6?logo=windows)](https://www.microsoft.com/windows)
[![Wails](https://img.shields.io/badge/Wails-v2-DF0000?logo=wails)](https://wails.io)

**go-peerblock** to wysokowydajna aplikacja dla systemu Windows do blokowania ruchu sieciowego na podstawie dynamicznie aktualizowanych list IP. Zbudowana w języku Go z wykorzystaniem Wails v2 (frontend React), używa sterownika WinDivert do przechwytywania i filtrowania pakietów na poziomie sieciowym z minimalnym obciążeniem CPU.

> 🇬🇧 [Read in English](README.md)

## Funkcje

- **⚡ Błyskawiczna prędkość** — lookup IP w ~186ns dla 500 000+ zakresów (2× szybciej niż zakładany cel 400ns)
- **🛡️ Filtrowanie przez WinDivert** — przechwytywanie pakietów na poziomie jądra z minimalnym narzutem
- **🎨 Nowoczesne GUI** — frontend React + TypeScript przez Wails v2 (WebView2)
- **🔄 Automatyczne aktualizacje** — okresowe pobieranie i atomowe przeładowanie list IP bez utraty pakietów
- **📊 Statystyki w czasie rzeczywistym** — liczniki zablokowanych/przepuszczonych, uptime, podgląd logów
- **🔍 Parser wielu formatów** — obsługa PeerGuardian (.p2p), eMule DAT, CIDR i zakresów
- **✅ Allowlista** — whitelistowanie domen z resolwowaniem DNS
- **⚙️ Konfigurowalny cache** — LRU cache decyzji z regulowanym TTL
- **🪟 System tray** — minimalizacja do tła, przełączanie ochrony z menu w zasobniku
- **📝 Asynchroniczne logowanie** — nieblokujący logger plikowy z ring buffer dla GUI

## Wydajność

| Rozmiar bazy | Czas lookupu | Przepustowość | Cel | Status |
|---|---|---|---|---|
| 10 000 zakresów | **76 ns** | 13.1M ops/s | < 100 ns | ✅ **Przekroczony** |
| 100 000 zakresów | **197 ns** | 5.1M ops/s | < 200 ns | ✅ **Przekroczony** |
| 500 000 zakresów | **186 ns** | 5.4M ops/s | < 400 ns | ✅ **2× szybciej** |

## Architektura

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

### Kluczowe decyzje techniczne

| Decyzja | Wybór | Uzasadnienie |
|---|---|---|
| **Binding WinDivert** | Własne minimalne CGO | `go-windivert2` niekompatybilne z WinDivert 2.2.2 |
| **Build tagi** | `//go:build windivert` | Development bez CGO, produkcja z WinDivert |
| **Cache** | Prosty LRU + RWMutex | 89ns lookup, brak zależności zewnętrznych |
| **TTL cache** | Konfigurowalny w `config.json` | Kluczowe dla szybko rotujących list (Spamhaus DROP) |
| **Baza IP** | Posortowana lista + binary search | O(log n) — ~19 porównań dla 500k zakresów |
| **Współbieżność** | `atomic.Pointer` w hot path | Odczyt bez blokad, atomowy reload bez utraty pakietów |

## Wymagania

### Development
- **Go 1.21+** (zalecane: 1.23+)
- **Node.js 18+**
- **Wails CLI v2** — `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **GCC** (mingw-w64) — do kompilacji CGO

### Produkcja
- **Windows 10/11** (64-bit)
- **WebView2 Runtime** (preinstalowany na Windows 10+)
- **Sterownik WinDivert 2.2.2** — pobierz z [reqrypt.org](https://reqrypt.org/windivert.html)

## Szybki start

### 1. Sklonuj repozytorium
```bash
git clone https://github.com/AnonBOTpl/go-peerblock.git
cd go-peerblock
```

### 2. Zainstaluj WinDivert SDK (wymagane dla builda produkcyjnego)
Pobierz [WinDivert-2.2.2-A.zip](https://reqrypt.org/download/WinDivert-2.2.2-A.zip) i rozpakuj:
```bash
cp WinDivert-2.2.2-A/x64/WinDivert.dll .
cp WinDivert-2.2.2-A/x64/WinDivert64.sys .
cp WinDivert-2.2.2-A/include/windivert.h .
cp windivert.h /mingw64/x86_64-w64-mingw32/include/
```

### 3. Zainstaluj zależności
```bash
go mod tidy
cd frontend && npm install && cd ..
```

### 4. Tryb developerski (bez WinDivert)
```bash
wails dev
```

### 5. Build produkcyjny (z WinDivert)
```bash
CGO_ENABLED=1 go build -tags windivert -o go-peerblock.exe .
```
Uruchom `go-peerblock.exe` **jako Administrator**.

## Struktura projektu

```
go-peerblock/
├── main.go                  # Entry point, sprawdzanie UAC, bootstrap Wails
├── app.go                   # App struct, metody bindingowe
├── core/                    # Silnik IP
│   ├── database.go          # IPRange, MergeRanges, binary search
│   ├── parser.go            # Wykrywanie formatów i parsowanie
│   ├── cache.go             # LRU cache z konfigurowalnym TTL
│   └── allowlist.go         # Allowlista z DNS
├── filter/                  # Filtrowanie pakietów
│   ├── windivert.go         # Binding CGO WinDivert (tag: windivert)
│   ├── windivert_noop.go    # Stub noop (tag: !windivert)
│   ├── pipeline.go          # Wielowątkowy pipeline
│   └── ...
├── updater/                 # Aktualizacje list IP
├── logger/                  # Asynchroniczne logowanie
├── config/                  # Konfiguracja
├── systray/                 # System tray
├── frontend/                # React + TypeScript
├── build/                   # Manifest UAC + instalator NSIS
└── WinDivert.*              # SDK WinDivert
```

## Testy

```bash
# Uruchom testy jednostkowe z race detector
go test -race ./...

# Uruchom benchmarki
go test -bench=. -benchmem ./...

# Weryfikacja builda (oba tryby)
go build ./...
CGO_ENABLED=1 go build -tags windivert ./...
```

## Licencja

Projekt na licencji MIT — szczegóły w pliku [LICENSE](LICENSE).

Zbudowane z ❤️ przez [AnonBOTpl](https://github.com/AnonBOTpl)
