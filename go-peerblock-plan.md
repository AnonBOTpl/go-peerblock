# go-peerblock v2 — Kompletny Plan Projektu

## Założenia i cele

Aplikacja dla Windows napisana w Go, blokująca ruch sieciowy na podstawie dynamicznie aktualizowanych list IP.

**Wymagania niefunkcjonalne:**
- Wydajność: lookup < 1 µs, przepustowość > 1 Mpps na jednym wątku
- Niski narzut CPU w trybie idle (< 1%)
- Obsługa setek tysięcy zakresów IP (500k+)
- Atomowe przeładowywanie reguł bez utraty pakietów
- GUI w Wails v2 (Go + WebView2, React frontend)
- Pełne logowanie zdarzeń asynchronicznie
- Automatyczne aktualizacje list IP w tle
- Instalator z obsługą UAC elevation

---

## Podjęte decyzje techniczne

Po przeglądzie planu podjęto następujące decyzje:

| Punkt | Decyzja |
|---|---|
| **WinDivert binding** | Własne minimalne CGO bindings (Open, Recv, Send, Close) — `sbilly/go-windivert2` okazało się niekompatybilne z WinDivert 2.2.2 (zmiany w API helperów) |
| **Format .p2b** | Wycięty — nikt nie używa |
| **Cache TTL** | Konfigurowalny w `config.json` |
| **WebView2 w instalatorze** | Explicitne wywołanie bootstrappera w NSIS |
| **LogView aktualizacje** | Polling co 1s — zostaje (prostsze debugowanie) |

---

## Struktura projektu

```
go-peerblock/
├── main.go                    # Entry point, Wails bootstrap
├── go.mod
├── go.sum
├── app.go                     # Wails App struct, exposed metody
├── wails.json
├── build/
│   ├── windows/
│   │   ├── go-peerblock.exe.manifest   # UAC requireAdministrator
│   │   └── icon.ico
│   └── installer/
│       ├── installer.nsis     # NSIS skrypt instalatora
│       └── install-driver.bat # Instalacja WinDivert SYS
├── WinDivert.dll
├── WinDivert64.sys
├── core/
│   ├── database.go            # IPRange struct, sortowanie, scalanie
│   ├── parser.go              # Parsowanie formatów list IP
│   ├── lookup.go              # Binary search, cache decyzji
│   ├── cache.go               # LRU cache wyników
│   └── allowlist.go           # Allowlista z DNS resolution
├── filter/
│   ├── windivert.go           # Binding CGO do WinDivert
│   ├── pipeline.go            # Wielowątkowy pipeline pakietów
│   └── workerpool.go          # Pool workerów
├── updater/
│   ├── updater.go             # Orchestrator aktualizacji
│   ├── sources.go             # Definicje źródeł list
│   ├── fetcher.go             # HTTP downloader z retry
│   └── formats.go             # Parsowanie różnych formatów
├── logger/
│   ├── logger.go              # Asynchroniczny logger
│   └── ringbuffer.go          # Ring buffer dla GUI log view
├── config/
│   ├── config.go              # Struct konfiguracji
│   └── persistence.go         # Zapis/odczyt config.json
├── systray/
│   └── tray.go                # Ikona w zasobniku systemowym
├── frontend/
│   ├── package.json
│   ├── vite.config.ts
│   ├── src/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   ├── components/
│   │   │   ├── Dashboard.tsx  # Główny widok
│   │   │   ├── LogView.tsx    # Real-time log stream
│   │   │   ├── RuleList.tsx   # Lista źródeł/reguł
│   │   │   ├── Settings.tsx   # Panel ustawień
│   │   │   └── StatusBar.tsx  # Pasek statusu
│   │   └── wailsjs/           # Auto-generowane bindingi
├── data/
│   ├── lists/                 # Pobrane listy IP (cache)
│   └── config.json
└── logs/
    └── peerblock.log
```

---

## Faza 0: Środowisko i UAC

**Cel:** Poprawna obsługa uprawnień administratora od pierwszego uruchomienia.

**Problem:** WinDivert wymaga załadowanego sterownika kernel-mode i uprawnień admina.  
Bez tego aplikacja cicho failuje lub crashuje przy starcie.

**Rozwiązanie — manifest UAC:**

```xml
<!-- build/windows/go-peerblock.exe.manifest -->
<requestedExecutionLevel level="requireAdministrator" uiAccess="false"/>
```

Wails automatycznie osadza manifest w EXE jeśli plik istnieje w `build/windows/`.

**Instalacja sterownika WinDivert:**

```batch
:: install-driver.bat (uruchamiany przez instalator jako admin)
sc create WinDivert type= kernel start= demand binPath= "%~dp0WinDivert64.sys"
sc start WinDivert
```

**Sprawdzenie przy starcie aplikacji:**

```go
func checkAdminAndDriver() error {
    if !isAdmin() {
        return fmt.Errorf("aplikacja wymaga uprawnień administratora")
    }
    if !isDriverLoaded("WinDivert") {
        if err := installDriver(); err != nil {
            return fmt.Errorf("nie można zainstalować sterownika: %w", err)
        }
    }
    return nil
}
```

---

## Faza 1: Silnik bazy IP (core/database.go)

**Cel:** Struktura danych umożliwiająca lookup < 1 µs na 500k+ zakresach.

### Struktura danych

```go
type IPRange struct {
    Start uint32
    End   uint32
    Label string  // opcjonalnie: nazwa listy źródłowej
}

type IPDatabase struct {
    ranges []IPRange // posortowane po Start, bez nakładek
    mu     sync.RWMutex
}
```

### Algorytmy

**Konwersja CIDR → zakres:**

```go
func CIDRToRange(cidr string) (IPRange, error) {
    _, network, err := net.ParseCIDR(cidr)
    if err != nil {
        return IPRange{}, err
    }
    start := binary.BigEndian.Uint32(network.IP)
    mask := binary.BigEndian.Uint32(network.Mask)
    end := start | ^mask
    return IPRange{Start: start, End: end}, nil
}
```

**Scalanie nakładających się zakresów (merge):**

```go
func MergeRanges(ranges []IPRange) []IPRange {
    if len(ranges) == 0 {
        return nil
    }
    sort.Slice(ranges, func(i, j int) bool {
        return ranges[i].Start < ranges[j].Start
    })
    merged := []IPRange{ranges[0]}
    for _, r := range ranges[1:] {
        last := &merged[len(merged)-1]
        if r.Start <= last.End+1 {
            if r.End > last.End {
                last.End = r.End
            }
        } else {
            merged = append(merged, r)
        }
    }
    return merged
}
```

**Binary search O(log n):**

```go
func (db *IPDatabase) Contains(ip uint32) bool {
    ranges := db.ranges
    lo, hi := 0, len(ranges)-1
    for lo <= hi {
        mid := (lo + hi) / 2
        r := ranges[mid]
        if ip < r.Start {
            hi = mid - 1
        } else if ip > r.End {
            lo = mid + 1
        } else {
            return true
        }
    }
    return false
}
```

### Benchmarki (core/database_bench_test.go)

```go
func BenchmarkLookup500k(b *testing.B) {
    db := generateTestDB(500_000)
    ips := generateRandomIPs(10_000)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        db.Contains(ips[i%len(ips)])
    }
}
// Cel: > 5M ops/s (< 200 ns/op)
```

**Docelowe wyniki benchmarków:**

| Rozmiar bazy | Czas lookupu | Przepustowość |
|---|---|---|
| 10k zakresów | < 100 ns | > 10M ops/s |
| 100k zakresów | < 200 ns | > 5M ops/s |
| 500k zakresów | < 400 ns | > 2.5M ops/s |

---

## Faza 2: Parsowanie formatów list IP (core/parser.go, updater/formats.go)

**Cel:** Obsługa wszystkich popularnych formatów list IP bez zewnętrznych bibliotek.

### Obsługiwane formaty

| Format | Przykład | Popularność |
|---|---|---|
| PeerGuardian Text `.p2p` | `Level1:1.2.3.0-1.2.3.255` | Wysoka (iblocklist.com) |
| DAT (eMule) | `1.2.3.0 - 1.2.3.255 , 100 , Blocklist` | Średnia |
| Plain CIDR | `1.2.3.0/24` (jeden na linię) | Wysoka |
| Plain zakres | `1.2.3.0-1.2.3.255` | Średnia |
| IP360 / GeoIP-style JSON | `{"ip":"1.2.3.0","mask":24}` | Niszowa |

### Autodetekcja formatu

```go
type FormatDetector struct{}

func (d FormatDetector) Detect(r io.Reader) (Format, io.Reader, error) {
    // Czytamy pierwsze 512 bajtów do bufora, nie konsumując readera
    buf := make([]byte, 512)
    n, _ := r.Read(buf)
    header := buf[:n]
    combined := io.MultiReader(bytes.NewReader(header), r)

    switch {
    case bytes.HasPrefix(header, []byte{0xFF, 0xFF, 0xFF, 0xFF}):
        return FormatP2PBinary, combined, nil
    case bytes.Contains(header, []byte(" - ")) && bytes.Contains(header, []byte(" , ")):
        return FormatDAT, combined, nil
    case bytes.Contains(header, []byte(":")):
        return FormatP2PText, combined, nil
    default:
        return FormatCIDR, combined, nil
    }
}
```

---

## Faza 3: Cache decyzji (core/cache.go)

**Cel:** Unikanie binary search dla powtarzających się IP.

### Dlaczego prosty LRU zamiast ristretto

- Ristretto to ~2MB dodatkowej zależności z generycznym GC-aware cache
- Przy lookupu < 400 ns, prosty LRU z RWMutex ma porównywalną latencję
- Hit rate dla ruchu sieciowego jest wysoki (wiele połączeń do tych samych IP)

### Implementacja

```go
type DecisionCache struct {
    mu      sync.RWMutex
    entries map[uint32]cachedDecision
    lru     []uint32  // ring buffer kolejności dostępu
    maxSize int
    pos     int
}

type cachedDecision struct {
    blocked bool
    ts      time.Time
}

func (c *DecisionCache) Get(ip uint32) (blocked bool, ok bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    if d, found := c.entries[ip]; found {
        if time.Since(d.ts) < 5*time.Minute {
            return d.blocked, true
        }
    }
    return false, false
}

func (c *DecisionCache) Set(ip uint32, blocked bool) {
    c.mu.Lock()
    defer c.mu.Unlock()
    if old := c.lru[c.pos]; old != 0 {
        delete(c.entries, old)
    }
    c.entries[ip] = cachedDecision{blocked: blocked, ts: time.Now()}
    c.lru[c.pos] = ip
    c.pos = (c.pos + 1) % c.maxSize
}
```

Rekomendowany rozmiar cache: 65536 wpisów (~1 MB pamięci).

**TTL konfigurowalny:** Zamiast twardego `5*time.Minute`, cache TTL będzie odczytywany z `config.json` (pole `CacheTTL`). Domyślnie 5 minut, ale użytkownik może skrócić (np. dla Spamhaus DROP) lub wydłużyć.

---

## Faza 4: Binding WinDivert (filter/windivert.go)

**Cel:** Przechwytywanie pakietów z wykorzystaniem `sbilly/go-windivert2` — gotowych, vendorowanych bindingów. Unikamy własnego CGO (ryzyko memory leaków, problemy z kompilacją).

### Zależność

```go
import (
    windivert "github.com/sbilly/go-windivert2"
    "github.com/sbilly/go-windivert2/windivert"
)
```

Biblioteka vendorowana do projektu (`go mod vendor`), co zapewnia powtarzalność buildów i kontrolę nad kodem.

### Warstwa abstrakcji (filter/windivert.go)

```go
type Packet struct {
    Data    []byte
    Addr    windivert.Address
    SrcIP   uint32
    DstIP   uint32
    SrcPort uint16
    DstPort uint16
    Proto   uint8
}

func ParseIPHeader(data []byte) (srcIP, dstIP uint32, proto uint8) {
    if len(data) < 20 || (data[0]>>4) != 4 {
        return 0, 0, 0
    }
    proto = data[9]
    srcIP = binary.BigEndian.Uint32(data[12:16])
    dstIP = binary.BigEndian.Uint32(data[16:20])
    return
}
```

### Zarządzanie uchwytem przez go-windivert2

```go
type WinDivert struct {
    handle windivert.WinDivertHandle
    filter string
}

func Open(filter string, layer windivert.Layer, priority int16) (*WinDivert, error) {
    handle, err := windivert.Open(filter, layer, priority, 0)
    if handle == windivert.InvalidHandle {
        return nil, fmt.Errorf("WinDivertOpen failed: %w", err)
    }
    return &WinDivert{handle: handle, filter: filter}, nil
}

func (w *WinDivert) Recv(buf []byte) (int, windivert.Address, error) {
    return w.handle.Recv(buf)
}

func (w *WinDivert) Send(buf []byte, addr windivert.Address) (int, error) {
    return w.handle.Send(buf, addr)
}

func (w *WinDivert) Close() error {
    return w.handle.Close()
}
```

### Filtr WinDivert

```
// Przechwytuj tylko pakiety IP (nie ICMP wewnętrzny, nie loopback)
"ip and (ip.DstAddr != 127.0.0.1) and (ip.SrcAddr != 127.0.0.1)"
```

---

## Faza 5: Wielowątkowy pipeline (filter/pipeline.go)

**Cel:** Maksymalna przepustowość przy minimalnym CPU.

### Architektura

```
WinDivertRecv (1 goroutine)
        │
        ▼ (buforowany channel, 4096)
   Packet Channel
        │
        ├──▶ Worker 0 ──▶ Decision Queue (allowed)
        ├──▶ Worker 1 ──▶ Decision Queue (allowed)
        ├──▶ Worker 2 ──▶  [dropped]
        └──▶ Worker N ──▶ Decision Queue (allowed)
                                 │
                                 ▼
                        WinDivertSend (1 goroutine)
```

### Implementacja

```go
type Pipeline struct {
    wd          *WinDivert
    db          *atomic.Pointer[core.IPDatabase]
    cache       *core.DecisionCache
    allowlist   *core.Allowlist
    packetCh    chan Packet
    sendCh      chan Packet
    workerCount int
    stats       Stats
    done        chan struct{}
}

func (p *Pipeline) recvLoop() {
    buf := make([]byte, 65535)
    for {
        n, addr, err := p.wd.Recv(buf)
        if err != nil {
            select {
            case <-p.done:
                return
            default:
                continue
            }
        }
        pkt := Packet{Data: make([]byte, n), Addr: addr}
        copy(pkt.Data, buf[:n])
        pkt.SrcIP, pkt.DstIP, pkt.Proto = ParseIPHeader(pkt.Data)
        p.packetCh <- pkt
    }
}

func (p *Pipeline) worker() {
    for pkt := range p.packetCh {
        if p.shouldBlock(pkt) {
            atomic.AddUint64(&p.stats.Blocked, 1)
            continue  // upuszczamy pakiet
        }
        atomic.AddUint64(&p.stats.Allowed, 1)
        p.sendCh <- pkt
    }
}

func (p *Pipeline) shouldBlock(pkt Packet) bool {
    // 1. Sprawdź allowlistę (zawsze przepuść)
    if p.allowlist.Contains(pkt.SrcIP) || p.allowlist.Contains(pkt.DstIP) {
        return false
    }
    // 2. Cache lookup
    db := p.db.Load()
    for _, ip := range []uint32{pkt.SrcIP, pkt.DstIP} {
        if blocked, ok := p.cache.Get(ip); ok {
            if blocked {
                return true
            }
            continue
        }
        blocked := db.Contains(ip)
        p.cache.Set(ip, blocked)
        if blocked {
            return true
        }
    }
    return false
}
```

### Konfiguracja liczby workerów

```go
workerCount := runtime.NumCPU()
if workerCount > 8 {
    workerCount = 8  // WinDivert i tak jest bottleneckiem powyżej ~8
}
```

---

## Faza 6: Allowlista z DNS resolution (core/allowlist.go)

**Cel:** Plik allowlist.txt obsługujący IP, zakresy CIDR i domeny.

### Format pliku

```
# Komentarze są ignorowane
8.8.8.8
8.8.4.4
1.1.1.0/24
github.com
*.cloudflare.com  # wildcard (resolwuje A records)
192.168.0.0/16    # sieci lokalne
```

### Implementacja z DNS

```go
type Allowlist struct {
    ranges  []core.IPRange  // statyczne IP i CIDR
    domains []string        // domeny do resolwowania
    mu      sync.RWMutex
}

func (a *Allowlist) ResolveAndRefresh() {
    var newRanges []core.IPRange

    // Dodaj statyczne zakresy
    newRanges = append(newRanges, a.staticRanges...)

    // Resolwuj domeny
    for _, domain := range a.domains {
        addrs, err := net.LookupHost(domain)
        if err != nil {
            continue
        }
        for _, addr := range addrs {
            ip := net.ParseIP(addr).To4()
            if ip == nil {
                continue
            }
            n := binary.BigEndian.Uint32(ip)
            newRanges = append(newRanges, core.IPRange{Start: n, End: n})
        }
    }

    // Sortuj i scal
    merged := core.MergeRanges(newRanges)
    a.mu.Lock()
    a.ranges = merged
    a.mu.Unlock()
}
```

Odświeżanie DNS co 30 minut w tle (goroutine z tickerem).

---

## Faza 7: Atomowy reload bazy (core/database.go + app.go)

**Cel:** Podmiana bazy IP bez zatrzymania filtrowania, bez race conditions.

```go
// W App struct (app.go)
type App struct {
    db        atomic.Pointer[core.IPDatabase]
    pipeline  *filter.Pipeline
    // ...
}

// Atomowa podmiana po aktualizacji
func (a *App) reloadDatabase(newDB *core.IPDatabase) {
    a.db.Store(newDB)
    // Pipeline czyta db.Load() przy każdym pakiecie — natychmiastowy efekt
    a.logger.Info("Baza IP przeładowana: %d zakresów", len(newDB.Ranges()))
}
```

Brak mutexów w hot path — `atomic.Pointer` gwarantuje memory ordering bez blokowania.

---

## Faza 8: Automatyczny updater (updater/)

**Cel:** Pobieranie i aktualizowanie list IP w tle z obsługą błędów.

### Definicje źródeł (updater/sources.go)

```go
type Source struct {
    Name     string
    URL      string
    Format   Format  // autodetekcja jeśli Unknown
    Enabled  bool
    LastSync time.Time
}

var DefaultSources = []Source{
    {
        Name:    "iblocklist-level1",
        URL:     "https://list.iblocklist.com/?list=ydxerpxkpcfqjaybcssw&fileformat=p2p&archiveformat=gz",
        Format:  FormatP2PText,
        Enabled: true,
    },
    {
        Name:    "firehol-level1",
        URL:     "https://raw.githubusercontent.com/firehol/blocklist-ipsets/master/firehol_level1.netset",
        Format:  FormatCIDR,
        Enabled: true,
    },
    {
        Name:    "spamhaus-drop",
        URL:     "https://www.spamhaus.org/drop/drop.txt",
        Format:  FormatCIDR,
        Enabled: true,
    },
}
```

### Fetcher z retry i walidacją (updater/fetcher.go)

```go
type Fetcher struct {
    client    *http.Client
    cacheDir  string
    maxRetry  int
    backoff   time.Duration
}

func (f *Fetcher) Fetch(src Source) ([]byte, error) {
    var lastErr error
    for i := 0; i < f.maxRetry; i++ {
        if i > 0 {
            time.Sleep(f.backoff * time.Duration(i))
        }
        data, err := f.fetchOnce(src)
        if err == nil {
            f.saveToCache(src.Name, data)
            return data, nil
        }
        lastErr = err
    }
    // Fallback: użyj cache z dysku
    if cached, err := f.loadFromCache(src.Name); err == nil {
        return cached, nil
    }
    return nil, fmt.Errorf("fetch failed after %d retries: %w", f.maxRetry, lastErr)
}
```

### Orchestrator (updater/updater.go)

```go
func (u *Updater) Run(ctx context.Context) {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    u.updateAll()  // aktualizacja przy starcie
    for {
        select {
        case <-ticker.C:
            u.updateAll()
        case <-u.manualTrigger:
            u.updateAll()
        case <-ctx.Done():
            return
        }
    }
}

func (u *Updater) updateAll() {
    var allRanges []core.IPRange
    for _, src := range u.sources {
        if !src.Enabled {
            continue
        }
        data, err := u.fetcher.Fetch(src)
        if err != nil {
            u.logger.Warn("Nie można pobrać %s: %v", src.Name, err)
            continue
        }
        ranges, err := u.parser.Parse(data, src.Format)
        if err != nil {
            u.logger.Warn("Błąd parsowania %s: %v", src.Name, err)
            continue
        }
        allRanges = append(allRanges, ranges...)
        u.logger.Info("Załadowano %d zakresów z %s", len(ranges), src.Name)
    }
    merged := core.MergeRanges(allRanges)
    newDB := core.NewDatabase(merged)
    u.onReload(newDB)
}
```

---

## Faza 9: Logger asynchroniczny (logger/)

**Cel:** Logowanie zdarzeń bez blokowania hot path filtrowania.

### Ring buffer dla GUI

```go
type RingBuffer struct {
    entries []LogEntry
    pos     int
    size    int
    mu      sync.Mutex
}

type LogEntry struct {
    Timestamp time.Time
    Level     LogLevel
    Message   string
}

func (r *RingBuffer) Add(e LogEntry) {
    r.mu.Lock()
    r.entries[r.pos] = e
    r.pos = (r.pos + 1) % r.size
    r.mu.Unlock()
}

func (r *RingBuffer) Last(n int) []LogEntry {
    r.mu.Lock()
    defer r.mu.Unlock()
    // zwraca ostatnie n wpisów w chronologicznej kolejności
}
```

### Asynchroniczny zapis do pliku

```go
type Logger struct {
    ch     chan LogEntry
    file   *os.File
    ring   *RingBuffer
    done   chan struct{}
}

func (l *Logger) Info(format string, args ...any) {
    select {
    case l.ch <- LogEntry{Timestamp: time.Now(), Level: INFO, Message: fmt.Sprintf(format, args...)}:
    default:
        // channel pełny — dropujemy log (nie blokujemy)
    }
}

func (l *Logger) run() {
    for entry := range l.ch {
        l.ring.Add(entry)
        fmt.Fprintf(l.file, "[%s] %s %s\n",
            entry.Timestamp.Format("2006-01-02 15:04:05"),
            entry.Level,
            entry.Message,
        )
    }
}
```

---

## Faza 10: GUI — Wails v2 + React

**Cel:** Natywne okno Windows z WebView2, bez Electron overhead.

### Dlaczego Wails zamiast Fyne

| Cecha | Fyne | Wails |
|---|---|---|
| Wygląd | Własny renderer, niestandardowy | Natywne WebView2, HTML/CSS |
| Technologia frontend | Go/Canvas | React/Vue/Svelte |
| Rozmiar exe | ~30 MB | ~10 MB + WebView2 (preinstalowany na Win10/11) |
| Hot reload dev | Nie | Tak (`wails dev`) |
| Dostępność UI | Ograniczona | Pełna (ARIA, czytniki ekranu) |
| Stylowanie | Własne API | Dowolny CSS |

### Setup Wails

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails init -n go-peerblock -t react-ts
```

### Eksponowane metody (app.go)

```go
type App struct {
    ctx      context.Context
    pipeline *filter.Pipeline
    updater  *updater.Updater
    logger   *logger.Logger
    config   *config.Config
    db       atomic.Pointer[core.IPDatabase]
}

// Metody dostępne z frontendu przez wails.Call()

func (a *App) GetStats() Stats {
    return a.pipeline.GetStats()
}

func (a *App) GetLogs(n int) []logger.LogEntry {
    return a.logger.Ring().Last(n)
}

func (a *App) TriggerUpdate() {
    go a.updater.TriggerManual()
}

func (a *App) SetProtectionEnabled(enabled bool) {
    if enabled {
        a.pipeline.Start()
    } else {
        a.pipeline.Stop()
    }
    a.config.ProtectionEnabled = enabled
    a.config.Save()
}

func (a *App) GetConfig() config.Config {
    return *a.config
}

func (a *App) SaveConfig(cfg config.Config) error {
    *a.config = cfg
    return a.config.Save()
}
```

### Widoki frontendu

**Dashboard.tsx** — główny ekran:
- Duży przełącznik ochrony (ON/OFF) z animacją
- 4 karty statystyk: Reguły aktywne / Zablokowane / Przepuszczone / Czas ostatniej aktualizacji
- Wykres blokad w czasie (sparkline, ostatnie 60 minut)
- Przycisk "Aktualizuj teraz"

**LogView.tsx** — log w czasie rzeczywistym:
- Polling co 1s przez `window.go.main.App.GetLogs(100)`
- Wirtualizowana lista (tylko widoczne wpisy renderowane w DOM)
- Filtry: ALL / BLOCKED / ALLOWED / WARN / ERROR
- Wyszukiwarka (filtr po IP lub tekście)
- Auto-scroll z możliwością pauzowania

**RuleList.tsx** — zarządzanie listami:
- Tabela źródeł: nazwa, URL, liczba zakresów, data synca, toggle
- Dodawanie własnych URL
- Podgląd zakresów dla danej listy (virtualized, bo 500k+)

**Settings.tsx** — ustawienia:
- Allowlista (textarea z walidacją IP/CIDR/domen)
- Liczba workerów
- Częstotliwość aktualizacji
- Logowanie: poziom, rotacja pliku
- Autostart z systemem (rejestr Windows)
- Reset do domyślnych

**StatusBar.tsx** — pasek na dole:
- Aktualny status: "Ochrona aktywna" / "Ochrona wyłączona" / "Aktualizuję..."
- FPS pakietów (pakietów/s, odświeżane co 1s)
- Użycie pamięci przez bazę IP

### Real-time updates przez Wails events

```go
// Backend emituje zdarzenia
runtime.EventsEmit(a.ctx, "stats:update", a.pipeline.GetStats())
runtime.EventsEmit(a.ctx, "log:new", entry)
```

```typescript
// Frontend nasłuchuje
import { EventsOn } from "../wailsjs/runtime/runtime"

EventsOn("stats:update", (stats) => setStats(stats))
EventsOn("log:new", (entry) => appendLog(entry))
```

---

## Faza 11: Systray (systray/tray.go)

**Cel:** Działanie w tle bez widocznego okna.

```go
import "github.com/getlantern/systray"

func RunTray(app *App) {
    systray.Run(func() {
        systray.SetIcon(iconData)
        systray.SetTitle("go-peerblock")

        mShow := systray.AddMenuItem("Pokaż okno", "")
        mToggle := systray.AddMenuItem("Wyłącz ochronę", "")
        systray.AddSeparator()
        mQuit := systray.AddMenuItem("Zamknij", "")

        for {
            select {
            case <-mShow.ClickedCh:
                runtime.WindowShow(app.ctx)
            case <-mToggle.ClickedCh:
                app.ToggleProtection()
                if app.IsProtectionEnabled() {
                    mToggle.SetTitle("Wyłącz ochronę")
                } else {
                    mToggle.SetTitle("Włącz ochronę")
                }
            case <-mQuit.ClickedCh:
                systray.Quit()
            }
        }
    }, nil)
}
```

Przy zamknięciu okna głównego (X) — chowanie do systray zamiast zamykania.

---

## Faza 12: Konfiguracja (config/)

```go
type Config struct {
    ProtectionEnabled bool          `json:"protection_enabled"`
    StartMinimized    bool          `json:"start_minimized"`
    StartWithSystem   bool          `json:"start_with_system"`
    WorkerCount       int           `json:"worker_count"`
    CacheSize         int           `json:"cache_size"`
    UpdateInterval    time.Duration `json:"update_interval"`
    LogLevel          string        `json:"log_level"`
    LogMaxSizeMB      int           `json:"log_max_size_mb"`
    Sources           []updater.Source `json:"sources"`
    Allowlist         []string      `json:"allowlist"`
}

var Defaults = Config{
    ProtectionEnabled: true,
    StartMinimized:    false,
    StartWithSystem:   false,
    WorkerCount:       0,  // 0 = auto (NumCPU)
    CacheSize:         65536,
    UpdateInterval:    24 * time.Hour,
    LogLevel:          "info",
    LogMaxSizeMB:      10,
    Sources:           updater.DefaultSources,
}
```

Zapis do `%APPDATA%\go-peerblock\config.json`.

---

## Faza 13: Instalator (build/installer/)

**Cel:** Profesjonalny instalator obsługujący cały lifecycle.

### NSIS skrypt (installer.nsis)

```nsis
Name "go-peerblock"
OutFile "go-peerblock-setup.exe"
RequestExecutionLevel admin

Section "Główna aplikacja"
  SetOutPath "$INSTDIR"
  File "go-peerblock.exe"
  File "WinDivert.dll"
  File "WinDivert64.sys"

  ; Instalacja sterownika
  ExecWait '"$INSTDIR\install-driver.bat"'

  ; Sprawdzenie / instalacja WebView2
  IfFileExists "$LOCALAPPDATA\Microsoft\Edge\Application\*.*" SkipWebView2
  DetailPrint "Instalowanie WebView2..."
  File "MicrosoftEdgeWebView2RuntimeInstaller.exe"
  ExecWait '"$INSTDIR\MicrosoftEdgeWebView2RuntimeInstaller.exe" /silent /install'
  SkipWebView2:

  ; Skrót w menu Start
  CreateShortCut "$SMPROGRAMS\go-peerblock.lnk" "$INSTDIR\go-peerblock.exe"

  ; Wpis w rejestrze (Add/Remove Programs)
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\go-peerblock" \
    "DisplayName" "go-peerblock"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\go-peerblock" \
    "UninstallString" "$INSTDIR\uninstall.exe"

  WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
  ; Zatrzymaj sterownik
  ExecWait 'sc stop WinDivert'
  ExecWait 'sc delete WinDivert'

  Delete "$INSTDIR\*.*"
  RMDir "$INSTDIR"
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\go-peerblock"
SectionEnd
```

### Build script (Makefile)

```makefile
.PHONY: build dist clean

build:
	wails build -platform windows/amd64

dist: build
	makensis build/installer/installer.nsis
	@echo "Instalator: go-peerblock-setup.exe"

clean:
	rm -rf build/bin/ dist/
```

---

## Faza 14: Testy

### Testy jednostkowe

```
core/database_test.go      — parsowanie, scalanie, lookup correctness
core/cache_test.go         — eviction, TTL, concurrent access
updater/formats_test.go    — parsowanie wszystkich formatów
filter/pipeline_test.go    — mock WinDivert, sprawdzenie decyzji
```

### Testy integracyjne

```
e2e/blocking_test.go       — realne pakiety przez WinDivert (wymaga admina)
```

### Race detector

```bash
go test -race ./...
```

Wszystkie testy muszą przechodzić bez data race warnings.

---

## Harmonogram (orientacyjny)

| Faza | Szacowany czas | Zależności |
|---|---|---|
| 0 — UAC + środowisko | 1 dzień | — |
| 1 — Silnik IP | 2 dni | — |
| 2 — Parsowanie formatów | 2 dni | Faza 1 |
| 3 — Cache | 1 dzień | Faza 1 |
| 4 — WinDivert binding | 2 dni | Faza 1 |
| 5 — Pipeline | 2 dni | Fazy 1-4 |
| 6 — Allowlista | 1 dzień | Faza 1 |
| 7 — Atomowy reload | 0.5 dnia | Fazy 1, 5 |
| 8 — Updater | 2 dni | Fazy 2, 7 |
| 9 — Logger | 1 dzień | — |
| 10 — GUI Wails | 4 dni | Fazy 5-9 |
| 11 — Systray | 1 dzień | Faza 10 |
| 12 — Config | 1 dzień | — |
| 13 — Instalator | 1 dzień | Wszystkie |
| 14 — Testy | ciągłe | — |
| **Razem** | **~3 tygodnie** | |

---

## Znane ryzyka i mitigacje

| Ryzyko | Prawdopodobieństwo | Mitigacja |
|---|---|---|
| WinDivert odrzucony przez Windows Defender | Wysokie | Podpisanie kodu (code signing cert), instrukcja ręcznego whitelistowania |
| WebView2 nieobecny na starych Win10 | Niskie | Wails bundluje bootstrapper WebView2 |
| CGO kompilacja na CI/CD | Średnie | Użyć cross-kompilacji z mingw-w64 na Linux |
| Memory leak w CGO (WinDivert bufory) | Średnie | Valgrind-style analiza z `go tool pprof`, testy soak |
| Fałszywe pozytywy blokujące ważne serwisy | Wysokie | Domyślna allowlista (8.8.8.8, 1.1.1.1, lokalne sieci), łatwe dodawanie wyjątków |
