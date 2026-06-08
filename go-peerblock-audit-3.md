# go-peerblock — Audit #3

## Ogólna ocena

Znacząca poprawa względem poprzednich wersji. Wszystkie bugi z Auditu #1 i #2 naprawione.
Cache versioning działa poprawnie. Nowe funkcje (LookupBlockSource, Subscribe/Unsubscribe,
file:// URL handler, autostart przez rejestr) zaimplementowane czysto.
Poniżej tylko drobniejsze rzeczy.

---

## 🟠 Poważne

### 1. `updateAll` — podwójny lock i `newDB` budowany pod lockiem

```go
// Pierwsze zamknięcie pod lockiem
u.mu.Lock()
u.sourceRanges = perSource
u.mu.Unlock()

// Drugie zamknięcie pod lockiem — core.NewDatabase() może być wolne (MergeRanges na 500k+)
u.mu.Lock()
for i := range sources { ... }
newDB := core.NewDatabase(allRanges)  // ← wolna operacja pod lockiem!
u.mu.Unlock()
```

`core.NewDatabase(allRanges)` woła `MergeRanges` który sortuje i scala setki tysięcy zakresów.
To może trwać dziesiątki ms. Przez ten czas `GetSources()`, `IsRunning()`, `RefreshSources()` — wszystkie blokują.

**Fix:** Przenieś `core.NewDatabase` przed lock:

```go
// Zaktualizuj LastSync w lokalnej kopii
for i := range sources { ... }

// Zbuduj DB bez locka (wolna operacja)
newDB := core.NewDatabase(allRanges)

// Lock tylko do zapisu wyników
u.mu.Lock()
u.sourceRanges = perSource
for i := range u.sources { ... } // kopiuj LastSync
u.mu.Unlock()

if u.onReload != nil {
    u.onReload(newDB)
}
```

### 2. `LookupBlockSource` — O(n) na każde kliknięcie w GUI (`app.go`)

```go
for name, ranges := range a.sourceRanges {
    for _, r := range ranges {         // ← iteracja przez wszystkie zakresy źródła
        if ipU32 >= r.Start && ipU32 <= r.End {
```

Przy np. blocklist.de (50k+ zakresów) każde kliknięcie robi liniowe przeszukiwanie.
Nieodczuwalne dla pojedynczego IP, ale przy wielu kliknięciach z rzędu może blokować goroutine GUI na chwilę.

**Fix (opcjonalny):** Posortuj zakresy per-source i użyj binary search zamiast linear scan. Alternatywnie — zostawić tak jak jest, bo to tylko GUI path, nie hot path.

---

## 🟡 Umiarkowane

### 3. `pipeline.go` — `Stop()` zamyka handle WinDivert przed close(done)

```go
func (p *Pipeline) Stop() {
    if !p.started.Load() {
        return
    }
    p.started.Store(false)
    if p.wd != nil && p.wd.IsOpen() {
        p.wd.Close()        // ← zamyka handle
    }
    close(p.done)           // ← sygnał dla goroutines
}
```

Po `p.wd.Close()` ale przed `close(p.done)`:
- `sendLoop` może próbować `p.wd.Send()` na zamkniętym handle → błąd ignorowany (dropped++)  
- `worker` może wstawić do `sendCh` pakiet który `sendLoop` wyśle przez zamknięty handle

To nie crashuje (błędy są ignorowane), ale powoduje niepotrzebne dropped++ przy każdym Stop().

**Fix:** Odwróć kolejność — najpierw `close(p.done)`, potem `p.wd.Close()` z małym delay:

```go
func (p *Pipeline) Stop() {
    if !p.started.CompareAndSwap(true, false) {
        return
    }
    close(p.done)              // najpierw sygnał
    time.Sleep(10 * time.Millisecond) // daj goroutinom chwilę na exit
    if p.wd != nil && p.wd.IsOpen() {
        p.wd.Close()           // potem zamknij handle
    }
}
```

### 4. `logger.run()` — duplikacja kodu w drain loop

Blok `case <-l.done:` zawiera identyczny kod jak `case entry := <-l.ch:` — zapis do pliku i notyfikacja subscriberów. Przy zmianie logiki (np. dodanie rotacji pliku) trzeba pamiętać o zmianie w dwóch miejscach.

**Fix:** Wydziel helper:

```go
func (l *Logger) writeEntry(entry LogEntry) {
    l.ring.Add(entry)
    _, _ = fmt.Fprintf(l.file, "[%s] %s %s\n",
        entry.Timestamp.Format("2006-01-02 15:04:05"),
        entry.Level,
        entry.Message,
    )
    l.subMu.Lock()
    for _, s := range l.subscribers {
        select {
        case s.ch <- entry:
        default:
        }
    }
    l.subMu.Unlock()
}

func (l *Logger) run() {
    defer l.wg.Done()
    for {
        select {
        case entry := <-l.ch:
            l.writeEntry(entry)
        case <-l.done:
            for {
                select {
                case entry := <-l.ch:
                    l.writeEntry(entry)
                default:
                    return
                }
            }
        }
    }
}
```

### 5. `isAdmin()` — nieintuicyjna metoda detekcji admina (`main.go`)

```go
func isAdmin() bool {
    _, err := os.Open("\\\\.\\PHYSICALDRIVE0")
    return err == nil
}
```

Otwieranie `PHYSICALDRIVE0` to obejście — działa, ale jest podatne na fałszywe negatywy
(np. dysk zaszyfrowany, brak dysku fizycznego, maszyna wirtualna). Standardowe podejście na Windows to sprawdzenie tokenu procesu przez WinAPI.

**Fix z `golang.org/x/sys/windows` (już masz tę zależność przez rejestr):**

```go
func isAdmin() bool {
    token := windows.Token(0)
    member, err := token.IsMember(windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid, nil))
    return err == nil && member
}
```

Alternatywnie prostszy check przez `net.Interfaces()` który też wymaga admina na Windows, ale PHYSICALDRIVE0 jest bardziej niezawodny niż IsMember w praktyce — więc to kwestia gustu.

### 6. `config.go` — `CacheTTL` jako `time.Duration` w JSON

```json
"cache_ttl": 300000000000
```

Nanosekund y jako int64 w JSON — nieczytelne dla człowieka. Jeśli użytkownik chce ręcznie edytować `config.json`, nie wie co wpisać.

**Fix (opcjonalny):** Custom marshaler zapisujący jako sekundy:

```go
type durationSeconds time.Duration

func (d durationSeconds) MarshalJSON() ([]byte, error) {
    return json.Marshal(int64(time.Duration(d).Seconds()))
}

func (d *durationSeconds) UnmarshalJSON(data []byte) error {
    var secs int64
    if err := json.Unmarshal(data, &secs); err != nil {
        return err
    }
    *d = durationSeconds(time.Duration(secs) * time.Second)
    return nil
}
```

Albo po prostu dodaj komentarz w README że `cache_ttl` jest w nanosekund ach i podaj przykłady.

---

## 🟢 Drobne / styl

### 7. `sources.go` — `APIKey` w struct widoczny w JSON

```go
APIKey string `json:"api_key"`
```

`api_key` będzie zapisywany do `config.json` w plaintext. Niegroźne jeśli plik jest chroniony
przez Windows ACL, ale warto dodać notatkę w komentarzu że to ograniczenie.

### 8. `allowlist.go` — `isIPString` sprawdza IPv6 też

```go
func isIPString(s string) bool {
    return net.ParseIP(s) != nil  // ← ParseIP obsługuje też IPv6
}
```

Jeśli użytkownik wpisze IPv6 w allowliście, `isIPString` zwróci `true`, ale `To4()` zwróci `nil`
i IP zostanie zignorowane bez żadnego błędu. Cicha utrata wpisu.

**Fix:** Wyraźnie sprawdź IPv4:

```go
func isIPString(s string) bool {
    return net.ParseIP(s).To4() != nil
}
```

### 9. `fetcher.go` — brak walidacji rozmiaru odpowiedzi

Przy pobieraniu list nie ma limitu rozmiaru odpowiedzi. Złośliwy lub błędny serwer mógłby zwrócić
gigabajtową odpowiedź którą `io.ReadAll` załaduje do pamięci.

**Fix:**

```go
data, err := io.ReadAll(io.LimitReader(resp.Body, 100*1024*1024)) // max 100MB
```

### 10. `pipeline_noop.go` — brak eksportu `RecommendedWorkerCount` w noop

Nie dotyczy bezpośrednio — `workerpool.go` jest wspólny i nie ma build tagu, więc OK.
Ale `pipeline_noop.go` nie eksportuje `BlockCallback` — jeśli ktoś spróbuje użyć `SetOnBlock`
w noop buildzie, dostanie błąd kompilacji. Warto dodać stub:

```go
// W pipeline_noop.go:
type BlockCallback func(srcIP, dstIP uint32, proto uint8)

func (p *Pipeline) SetOnBlock(fn BlockCallback) {}
```

---

## Podsumowanie

| Priorytet | Liczba | Opis |
|---|---|---|
| 🟠 Poważne | 2 | `newDB` pod lockiem, `LookupBlockSource` O(n) |
| 🟡 Umiarkowane | 4 | Stop() kolejność, logger duplikacja, isAdmin(), CacheTTL JSON |
| 🟢 Drobne | 4 | APIKey plaintext, isIPString IPv6, brak limitu rozmiaru, noop BlockCallback |

Aplikacja jest w dobrym stanie — żaden z tych problemów nie blokuje działania.
Najważniejszy do naprawy to #1 (`newDB` pod lockiem) bo przy dużych listach może
powodować zauważalne zawieszenie GUI podczas aktualizacji.
