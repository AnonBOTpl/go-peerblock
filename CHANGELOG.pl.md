# Rejestr zmian

Wszystkie istotne zmiany w projekcie są dokumentowane w tym pliku.

> 🇬🇧 [English version](CHANGELOG.md)

## [0.1.0] — 2026-06-06

### Dodano

#### Silnik IP
- `core/database.go` — struktura IPRange, MergeRanges (sortowanie+scalanie zakresów), CIDRToRange, wyszukiwanie binarne Contains()
- `core/parser.go` — wykrywanie formatów (CIDR, P2P Text, DAT), parsowanie wszystkich obsługiwanych formatów
- `core/cache.go` — LRU DecisionCache z konfigurowalnym TTL (domyślnie 5 min), ewiktowanie przez ring buffer
- `core/allowlist.go` — Allowlista ze statycznymi IP, zakresami CIDR i domenami z resolwowaniem DNS
- `core/database_test.go` — Testy jednostkowe + benchmarki (500k zakresów: ~186ns lookup)
- `core/cache_test.go` — Testy ewiktowania, TTL, współbieżnego dostępu
- `core/parser_test.go` — Testy wykrywania formatów i poprawności parsowania

#### Filtrowanie pakietów
- `filter/windivert.go` — Własne minimalne bindingi CGO dla WinDivert 2.2.2 (bez zewnętrznych zależności)
- `filter/windivert_noop.go` — Stub noop dla developmentu bez WinDivert
- `filter/pipeline.go` — Wielowątkowy pipeline pakietów (recv → workerzy → send)
- `filter/pipeline_noop.go` — Stub pipeline'a
- `filter/shared.go` — Wspólne typy (Packet, Stats), ParseIPHeader, DefaultFilter
- `filter/workerpool.go` — Obliczanie zalecanej liczby workerów

#### Aktualizacje list IP
- `updater/updater.go` — Orchestrator okresowych aktualizacji z ręcznym wyzwalaniem
- `updater/sources.go` — Domyślne źródła bloklist (Firehol, Spamhaus DROP, iblocklist)
- `updater/fetcher.go` — Pobieranie HTTP z retry, backoff i cache na dysku

#### Logowanie
- `logger/logger.go` — Asynchroniczny, nieblokujący logger plikowy
- `logger/ringbuffer.go` — Wątkowo bezpieczny ring buffer dla podglądu logów w GUI

#### Konfiguracja
- `config/config.go` — Struktura Config z wartościami domyślnymi (CacheTTL, liczba workerów itp.)
- `config/persistence.go` — Zapis/odczyt JSON do `%APPDATA%\go-peerblock\config.json`

#### GUI (Wails v2 + React)
- `frontend/src/App.tsx` — Dashboard z przełącznikiem ochrony, kartami statystyk, podglądem logów, paskiem statusu
- `frontend/src/App.css` — Ciemny motyw

#### System Tray
- `systray/tray.go` — Ikona w zasobniku systemowym z menu (pokaż/przełącz/zamknij)

#### Budowa i wdrożenie
- `main.go` — Punkt wejścia ze sprawdzaniem UAC, bootstrap Wails
- `app.go` — App struct z eksponowanymi metodami bindingowymi Wails
- `build/windows/go-peerblock.exe.manifest` — Manifest UAC requireAdministrator
- `build/installer/install-driver.bat` — Skrypt instalacji sterownika WinDivert

### Zmieniono
- Binding WinDivert: przejście z `go-windivert2` (niekompatybilne z v2.2.2) na własne minimalne CGO
- Format .p2b: usunięty (przestarzały, zerowe użycie)
- TTL cache: konfigurowalny przez `config.json` zamiast twardo ustawionych 5 minut
- WebView2: jawne sprawdzanie bootstrappera w instalatorze NSIS
- Uptime: zmiana z `time.Time` na `int64` (UnixNano) dla czystej serializacji JSON

### Naprawiono
- Build CGO: downgrade mingw-w64 do 13.2.0 (w 16.1.0 brakowało `stddef.h`)
- Bindingi TypeScript: naprawione importy z namespace'ów dla `filter.Stats` i `logger.LogEntry`

### Benchmarki
- Lookup IP (10k zakresów): **76 ns/op** (cel: < 100 ns)
- Lookup IP (100k zakresów): **197 ns/op** (cel: < 200 ns)
- Lookup IP (500k zakresów): **186 ns/op** (cel: < 400 ns)
- Cache set: **242 ns/op**
- Cache get: **89 ns/op**

### Znane problemy
- Sterownik WinDivert może być oznaczony przez Windows Defender (wymaga podpisania kodu)
- Kompilacja CGO wymaga mingw-w64 z kompletnymi nagłówkami (zalecane: 13.x)
- WebView2 wymagany na starszych Windows 10 (bootstrapper dołączony w instalatorze)
