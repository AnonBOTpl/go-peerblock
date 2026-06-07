# Task: Napraw mechanizm inwalidacji cache w DecisionCache

## Problem

W `filter/pipeline.go` w metodzie `shouldBlock()` jest następująca logika:

```go
if blocked, ok := p.cache.Get(ip); ok {
    if blocked {
        // Re-verify with current DB — guards against stale cache entries
        db := p.db.Load()
        if db == nil || !db.Contains(ip) {
            p.cache.Set(ip, false)
            return false
        }
        return true
    }
    return false
}
```

**Skutek:** Dla każdego pakietu którego IP jest w cache jako `blocked=true`, i tak wykonywany jest binary search w DB. Cache dla zablokowanych IP de facto nie działa — każdy taki pakiet trafia do bazy. Przy dużym ruchu do zablokowanych IP to niepotrzebny narzut.

**Przyczyna powstania tego kodu:** Race condition przy reloadzie bazy:
1. Worker wczytuje stary DB i sprawdza IP → `blocked=true`
2. `onReload` w `app.go` robi: `cache.Clear()` → `db.Store(newDB)` → `cache.Clear()`
3. Worker cachuje `blocked=true` z już nieaktualnego DB (między pierwszym a drugim Clear)
4. Nowy DB tego IP nie blokuje, ale cache mówi że tak → fałszywy blok

## Rozwiązanie: wersjonowanie cache

Zamiast re-weryfikacji w DB przy każdym hit, dodaj pole `version` do cache. `Clear()` zamiast czyścić mapę (O(n)) tylko inkrementuje wersję (O(1)). Wpisy z inną wersją są traktowane jako nieaktualne i pomijane w `Get()`.

---

## Zmiany do wprowadzenia

### 1. `core/cache.go` — dodaj wersjonowanie

Zmień `cachedDecision` — dodaj pole `version`:

```go
type cachedDecision struct {
    blocked bool
    ts      time.Time
    version uint64
}
```

Zmień `DecisionCache` — dodaj pole `version`:

```go
type DecisionCache struct {
    mu      sync.RWMutex
    entries map[uint32]cachedDecision
    lru     []uint32
    maxSize int
    pos     int
    ttl     time.Duration
    version atomic.Uint64
}
```

Pamiętaj o dodaniu `"sync/atomic"` do importów jeśli jeszcze nie ma (w Go 1.19+ `atomic.Uint64` jest w `sync/atomic`).

Zmień `Get()` — sprawdzaj wersję:

```go
func (c *DecisionCache) Get(ip uint32) (blocked bool, ok bool) {
    currentVersion := c.version.Load()
    c.mu.RLock()
    defer c.mu.RUnlock()
    if d, found := c.entries[ip]; found {
        if d.version == currentVersion && time.Since(d.ts) < c.ttl {
            return d.blocked, true
        }
    }
    return false, false
}
```

Zmień `Set()` — zapisuj bieżącą wersję:

```go
func (c *DecisionCache) Set(ip uint32, blocked bool) {
    currentVersion := c.version.Load()
    c.mu.Lock()
    defer c.mu.Unlock()
    if old := c.lru[c.pos]; old != 0 {
        delete(c.entries, old)
    }
    c.entries[ip] = cachedDecision{
        blocked: blocked,
        ts:      time.Now(),
        version: currentVersion,
    }
    c.lru[c.pos] = ip
    c.pos = (c.pos + 1) % c.maxSize
}
```

Zmień `Clear()` — tylko inkrementuj wersję, nie czyść mapy:

```go
func (c *DecisionCache) Clear() {
    c.version.Add(1)
    // Nie czyścimy mapy — stare wpisy będą ignorowane przez Get() bo mają starą wersję.
    // Mapa zostanie naturalnie nadpisana przez nowe wpisy przez mechanizm LRU.
}
```

---

### 2. `filter/pipeline.go` — uprość `shouldBlock()`

Usuń całą logikę re-weryfikacji z cache hitu. Nowy `shouldBlock()`:

```go
func (p *Pipeline) shouldBlock(pkt Packet) bool {
    if p.allowlist.Contains(pkt.DstIP) {
        return false
    }

    ip := pkt.DstIP

    if blocked, ok := p.cache.Get(ip); ok {
        return blocked
    }

    db := p.db.Load()
    if db == nil {
        return false
    }
    blocked := db.Contains(ip)
    p.cache.Set(ip, blocked)
    return blocked
}
```

---

### 3. `app.go` — uprość `onReload` callback

Stary kod robił `cache.Clear()` dwukrotnie jako obejście race condition. Po tej zmianie wystarczy jedno wywołanie przed `db.Store()`. Znajdź callback `onReload` i zmień:

```go
func(newDB *core.IPDatabase) {
    a.cache.Clear()         // inwaliduje wszystkie stare wpisy przez wersję
    a.db.Store(newDB)       // atomowa podmiana bazy
    // Sync LastSync ...
    a.logger.Info("Baza IP przeładowana: %d zakresów", len(newDB.Ranges()))
},
```

Usuń drugie wywołanie `a.cache.Clear()` które było po `a.db.Store()`.

---

## Dlaczego to jest bezpieczne (brak race condition)

Sekwencja po zmianie:

1. `cache.Clear()` → wersja inkrementowana do np. `v=2`
2. `db.Store(newDB)` → nowa baza aktywna
3. Worker wywołuje `cache.Get(ip)` → wpis ma `version=1`, bieżąca `version=2` → miss → idzie do DB
4. DB zwraca poprawną decyzję z nowej bazy → `cache.Set(ip, ..., version=2)`

Nie ma okna w którym stary wpis z `blocked=true` mógłby przepuścić fałszywy blok po reloadzie.

---

## Testy do zaktualizowania

W `core/cache_test.go` metoda `TestCache_Clear` sprawdza czy mapa jest pusta po `Clear()`. Po tej zmianie mapa nie jest czyszczona — zaktualizuj test:

```go
func TestCache_Clear(t *testing.T) {
    c := NewDecisionCache(100, 5*time.Minute)

    c.Set(1, true)
    c.Set(2, false)
    c.Clear()

    // Po Clear() wpisy są niewidoczne (inna wersja), ale mapa nie jest czyszczona
    if _, ok := c.Get(1); ok {
        t.Error("expected no entry visible after Clear()")
    }
    if _, ok := c.Get(2); ok {
        t.Error("expected no entry visible after Clear()")
    }

    // Len() może być > 0 bo mapa nie jest czyszczona — to jest poprawne
}
```

Dodaj nowy test sprawdzający że wpisy po Clear() są niewidoczne ale nowe wpisy działają:

```go
func TestCache_ClearVersioning(t *testing.T) {
    c := NewDecisionCache(100, 5*time.Minute)

    c.Set(1, true)
    c.Clear()

    // Stary wpis niewidoczny
    if _, ok := c.Get(1); ok {
        t.Error("expected stale entry to be invisible after Clear()")
    }

    // Nowy wpis po Clear() działa normalnie
    c.Set(1, false)
    blocked, ok := c.Get(1)
    if !ok {
        t.Error("expected new entry to be visible after Clear()")
    }
    if blocked {
        t.Error("expected blocked=false for new entry")
    }
}
```

Uruchom po zmianach:

```bash
go test ./core/... -v -count=1
go test ./core/... -race
```
