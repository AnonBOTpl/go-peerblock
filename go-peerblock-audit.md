# go-peerblock — Audit Kodu

## Ogólna ocena

Kod jest dobrej jakości jak na pierwszy szkielet. Architektura zgodna z planem, testy solidne, benchmarki sensowne. Poniżej lista problemów pogrupowanych według priorytetu.

---

## 🔴 Krytyczne

### 1. Race condition w stats pipeline (`filter/pipeline.go`)

```go
// PROBLEM: Load + modyfikacja + Store to nie jest atomowa operacja
s := p.stats.Load().(Stats)
s.Blocked++
p.stats.Store(s)
```

Przy N workerach pracujących równolegle, każdy może wczytać tę samą wartość zanim któryś zdąży zapisać. Wynik: utracone zliczenia.

**Fix:** Użyj `sync/atomic` bezpośrednio na polach `uint64`:

```go
// W Stats zmień na:
type Stats struct {
    Allowed   atomic.Uint64
    Blocked   atomic.Uint64
    Dropped   atomic.Uint64
    StartedAt int64
}

// W worker:
p.stats.Blocked.Add(1)
```

Lub zachowaj `atomic.Value` ale chroń przez mutex przy update — wtedy sens `atomic.Value` odpada. Pierwsze rozwiązanie jest lepsze.

### 2. `isDriverLoaded` zawsze zwraca `false` (`main.go`)

```go
func isDriverLoaded(name string) bool {
    return false  // ← zawsze false!
}
```

Przy każdym starcie aplikacja próbuje instalować sterownik od nowa. Na systemie gdzie sterownik już działa — `installDriver()` wywoła `sc create` które zwróci błąd "service already exists", a `installDriver()` zwróci `nil` (ignoruje błąd), więc wszystko "działa" — ale przypadkowo. Na systemie z inną konfiguracją może crashować.

**Fix:** Zaimplementuj prawdziwy check przez Windows SCM:

```go
func isDriverLoaded(name string) bool {
    out, err := exec.Command("sc", "query", name).Output()
    if err != nil {
        return false
    }
    return bytes.Contains(out, []byte("RUNNING"))
}
```

### 3. `installDriver` ignoruje błędy (`main.go`)

```go
func installDriver() error {
    return nil  // ← zawsze sukces!
}
```

Powinno uruchamiać `install-driver.bat` i propagować błąd jeśli sterownik nie wystartował.

**Fix:**

```go
func installDriver() error {
    batPath := filepath.Join(execDir(), "build", "installer", "install-driver.bat")
    cmd := exec.Command("cmd", "/C", batPath)
    cmd.Dir = execDir()
    if out, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("install-driver failed: %w\nOutput: %s", err, out)
    }
    return nil
}
```

---

## 🟠 Poważne

### 4. `updateAll` trzyma lock podczas I/O sieciowego (`updater/updater.go`)

```go
func (u *Updater) updateAll() {
    u.mu.Lock()
    defer u.mu.Unlock()  // ← lock przez cały czas pobierania HTTP!

    for _, src := range u.sources {
        data, err := u.fetcher.Fetch(src)  // ← może trwać 30s × 3 retry × N źródeł
```

`IsRunning()` też chce ten lock — przez cały czas aktualizacji (może być kilka minut) GUI nie może sprawdzić statusu.

**Fix:** Lock tylko przy sprawdzaniu `running` flag, nie przy pobieraniu:

```go
func (u *Updater) updateAll() {
    // Zbierz ranges bez locka
    var allRanges []core.IPRange
    for _, src := range u.sources {
        if !src.Enabled {
            continue
        }
        data, err := u.fetcher.Fetch(src)  // bez locka
        // ...
        allRanges = append(allRanges, ranges...)
    }

    // Lock tylko przy zapisie wyniku
    u.mu.Lock()
    merged := core.MergeRanges(allRanges)
    newDB := core.NewDatabase(merged)
    u.mu.Unlock()

    if u.onReload != nil {
        u.onReload(newDB)
    }
}
```

### 5. `allowlist.Contains` zakłada posortowane zakresy, ale nie gwarantuje sortowania (`core/allowlist.go`)

```go
func (a *Allowlist) Contains(ip uint32) bool {
    // binary search — wymaga posortowanych ranges
    lo, hi := 0, len(ranges)-1
    for lo <= hi {
        mid := (lo + hi) / 2
```

`ResolveAndRefresh` woła `MergeRanges` (które sortuje), ale konstruktor `NewAllowlist` buduje `a.ranges = a.staticRanges` bez sortowania. Jeśli konfiguracja ma zakresy w złej kolejności, binary search zwróci błędne wyniki.

**Fix:** W `NewAllowlist` przed przypisaniem:

```go
a.staticRanges = MergeRanges(a.staticRanges)  // sortuje i scala
a.ranges = a.staticRanges
```

### 6. Brak obsługi błędu przy `a.logger.Close()` w shutdown (`app.go`)

```go
func (a *App) shutdown(ctx context.Context) {
    if a.logger != nil {
        _ = a.logger.Close()  // błąd ignorowany
    }
}
```

Przy zamknięciu może nie zostać zapisana część logów. Nie blokujące, ale warto przynajmniej wylogować:

```go
if err := a.logger.Close(); err != nil {
    runtime.LogError(ctx, "Logger close error: "+err.Error())
}
```

### 7. `CacheTTL` przekazywane jako `time.Duration` po konwersji z nanosekund (`app.go`)

```go
cacheTTL := cfg.CacheTTL
if cacheTTL <= 0 {
    cacheTTL = 5 * 60 * 1000000000 // 5 minutes default
}
```

Magic number zamiast `5 * time.Minute`. Ale ważniejsze: `Config.CacheTTL` jest `time.Duration` (nanosekund jako int64 w JSON), więc wartość zapisana w `config.json` jako `"cache_ttl": 300000000000` jest nieczytelna dla ludzi. Rozważ zapisywanie jako sekundy z custom marshalerem, albo przynajmniej komentarz w README.

**Fix (minimalny):**
```go
cacheTTL = 5 * time.Minute  // zamiast magic number
```

---

## 🟡 Umiarkowane

### 8. `pipeline.go` — brak DB reload przy zmianie bazy (`filter/pipeline.go`)

Pipeline trzyma referencję `db *core.IPDatabase` przekazaną przy konstruktorze. Gdy updater woła `a.db.Store(newDB)` w `app.go`, pipeline nadal używa starego `db`. `atomic.Pointer` w `App` nie jest widoczny dla pipeline.

**Fix:** Pipeline powinien trzymać `*atomic.Pointer[core.IPDatabase]` zamiast `*core.IPDatabase`:

```go
type Pipeline struct {
    db *atomic.Pointer[core.IPDatabase]  // zamiast *core.IPDatabase
    // ...
}

// W shouldBlock:
db := p.db.Load()
```

I przekazywać `&a.db` zamiast `a.db.Load()` przy tworzeniu pipeline.

### 9. `recvLoop` — blokujące `Recv` bez timeout (`filter/pipeline.go`)

```go
func (p *Pipeline) recvLoop() {
    for {
        select {
        case <-p.done:
            return
        default:
        }
        n, addr, err := p.wd.Recv(buf)  // blokuje na WinDivert
```

`select { case <-p.done: }` sprawdzany PRZED `Recv`, ale po wejściu w `Recv` goroutine blokuje do następnego pakietu. Przy `Close()`, jeśli ruch sieciowy zatrzyma się (np. brak pakietów), `recvLoop` może wisieć w `Recv` w nieskończoność.

**Fix:** Ustaw timeout na WinDivert handle:

```go
// Po otwarciu handle:
C.WinDivertSetParam(w.handle, C.WINDIVERT_PARAM_QUEUE_TIME, 1000) // 1s timeout
```

Lub w `Close()` wymuś wyjście z blokującego Recv przez zamknięcie handle przed close(done).

### 10. Format detektorowy — potencjalny false positive dla P2P vs CIDR (`core/parser.go`)

```go
case bytes.Contains(header, []byte(":")): // ← P2P Text
```

Adresy IPv6 zawierają `:`. Lista CIDR z komentarzem `# source: firehol` też zawiera `:`. Detektor błędnie sklasyfikuje taką listę jako P2P Text.

**Fix:** Sprawdzaj dokładniejszy wzorzec P2P:

```go
// Szukaj wzorca "tekst:IP-IP" zamiast samego ":"
case p2pRegexp.Match(header):  // regexp: `\w+:\d+\.\d+`
```

### 11. `RingBuffer.Len()` jest O(n) (`logger/ringbuffer.go`)

```go
func (r *RingBuffer) Len() int {
    count := 0
    for i := 0; i < r.size; i++ {
        if !r.entries[i].Timestamp.IsZero() {
            count++
        }
    }
    return count
}
```

Przy ring buffer 5000 elementów wywołanym z GUI co 1s — 5000 iteracji + lock co sekundę. Mało, ale bezprzyczynowo.

**Fix:** Trzymaj osobny licznik:

```go
type RingBuffer struct {
    count int  // dodaj pole
}

func (r *RingBuffer) Add(e LogEntry) {
    if r.entries[r.pos].Timestamp.IsZero() {
        r.count++
    }
    // ...
}

func (r *RingBuffer) Len() int {
    r.mu.Lock()
    defer r.mu.Unlock()
    return r.count
}
```

### 12. `windivert_noop.go` — niezgodna sygnatura `Open` (`filter/windivert_noop.go`)

```go
// noop:
func Open(filter string, layer interface{}, priority int16) (*WinDivert, error) {

// real (windivert.go):
func Open(filter string, layer int32, priority int16) (*WinDivert, error) {
```

`layer interface{}` vs `layer int32`. Przy buildzie bez tagu wywołanie z `int32` skompiluje się (Go automatycznie dopasuje), ale to nieczyste API i może powodować zaskoczenie. Ujednolić do `int32`.

---

## 🟢 Drobne / styl

### 13. `go.mod` deklaruje `go 1.26.4` — nieistniejąca wersja

```
go 1.26.4
```

Go 1.26 nie istnieje (aktualne to 1.24.x). Wails prawdopodobnie wygenerował go automatycznie z jakiegoś template'u. Zmień na `go 1.23` lub `go 1.24`.

### 14. `sources.go` — Format jako magiczne int zamiast stałych

```go
Format: 3, // FormatCIDR
```

Jeśli ktoś zmieni kolejność stałych `Format` w `parser.go`, te liczby będą błędne bez żadnego błędu kompilacji.

**Fix:**

```go
import "go-peerblock/core"
Format: int(core.FormatCIDR),
```

### 15. `app.go` — `StartRefreshLoop` dostaje hardkodowany `done` channel

```go
go a.allowlist.StartRefreshLoop(30*60*1000000000, make(chan struct{}))
```

Ten `done` channel nigdy nie jest zamykany — goroutina żyje do końca procesu bez możliwości zatrzymania. Przy `shutdown()` allowlist refresh nadal działa.

**Fix:** Przechowuj `done` channel w `App` i zamykaj w `shutdown()`.

### 16. Brak `go.sum` w gitignore check

`go.sum` jest w repo (dobrze!), ale warto upewnić się że nie ma w `.gitignore`.

---

## Podsumowanie

| Priorytet | Liczba | Najważniejsze |
|---|---|---|
| 🔴 Krytyczne | 3 | Race w stats, stub driver functions |
| 🟠 Poważne | 4 | Mutex podczas I/O, brak DB reload w pipeline, allowlist sort |
| 🟡 Umiarkowane | 5 | Blokujący Recv, format detektor, niezgodna sygnatura noop |
| 🟢 Drobne | 4 | go.mod wersja, magic numbers, done channel |

**Najważniejszy fix do zrobienia teraz:** Problem #8 (pipeline nie widzi nowego DB po aktualizacji) — bez tego cała funkcja automatycznych aktualizacji list jest niedziałająca, bo pipeline blokuje stare IP po reloadzie.
