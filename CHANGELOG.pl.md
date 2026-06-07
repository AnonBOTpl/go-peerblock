# Rejestr zmian

Wszystkie istotne zmiany w projekcie są dokumentowane w tym pliku.

> 🇬🇧 [English version](CHANGELOG.md)

## [0.2.0] — 2026-06-07

### Naprawiono

#### 🔴 Krytyczne — SrcIP blokował wszystko
- `filter/pipeline.go`: `shouldBlock` sprawdzało **zarówno SrcIP jak i DstIP** w bazie. Lokalne IP użytkownika (172.16.3.206) znajdowało się w zakresie `172.16.0.0/12` z firehol-level1, powodując **blokowanie każdego pakietu wychodzącego**. Fix: sprawdzany jest tylko DstIP (źródłowy adres to lokalne IP użytkownika, nigdy złośliwy cel).

#### 🟠 Race condition i zatrucie cache
- `filter/pipeline.go`: usunięto zbędną re-weryfikację cachowanych `blocked=true` w bazie przy każdym trafieniu — zabijała sens cache'owania. Zastąpiona **wersjonowaniem cache** (patrz Zmieniono).
- `app.go`: usunięto podwójne `cache.Clear()` w `onReload` — niepotrzebne po wersjonowaniu.

#### Inne naprawy
- `app.go`: `LastSync` poprawnie synchronizowany z updatera do configu po każdej aktualizacji (GUI pokazywało nieaktualne daty).
- `frontend/App.tsx`: przyciski Aktualizuj w headerze i SourcesView używają teraz wspólnego stanu `updating` — brak desynchronizacji.
- `filter/pipeline_noop.go`: dodano brakujące sygnatury metod.
- `main.go`: usunięto `init()` z `runtime.LockOSThread()` i globalny `appCtx` — Wails i systray same zarządzają wątkami.

### Zmieniono

#### Wersjonowanie cache (unieważnianie O(1))
- `core/cache.go`: `Clear()` inkrementuje teraz licznik wersji (`atomic.Uint64`) zamiast przebudowywać mapę (O(n)). Wpisy ze starszą wersją są ignorowane przez `Get()`. `Set()` zapisuje bieżącą wersję z każdym wpisem.
- Eliminuje race condition gdzie worker mógł zacachować decyzję ze starej bazy po `Clear()` ale przed `Store()`.

#### Minimalizacja do zasobnika
- `main.go`: systray startuje teraz w goroutine **przed** `wails.Run()`, utrzymując proces przy życiu gdy okno jest ukryte.
- `app.go`: dodano `MinimizeToTray()` → `runtime.WindowHide()`.
- `systray/tray.go`: "Zamknij" wywołuje teraz `runtime.Quit(ctx)` przed `systray.Quit()` dla czystego zamknięcia.
- `frontend/App.tsx`: przycisk ⬇ w headerze chowa okno do zasobnika. Przywróć przez ikonkę w tray'u → "Pokaż okno".

#### Autostart z systemem
- `app.go`: dodano `applyAutostart()` — zapisuje/usowa wpis `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\go-peerblock` przez `golang.org/x/sys/windows/registry`.
- Wywoływane przy starcie i przy każdym `SaveConfig()`.
- `frontend/App.tsx`: nowa sekcja "System" w Ustawieniach z przełącznikiem "Uruchamiaj z systemem Windows".

### Dodano

#### Panel ustawień (GUI)
- `frontend/App.tsx`: nowa zakładka **⚙️ Ustawienia** z polami do edycji:
  - Allowlista (textarea, jeden wpis na linię, komentarze `#` usuwane)
  - Liczba workerów (0 = auto/NumCPU)
  - Rozmiar cache (liczba wpisów)
  - Cache TTL (w minutach, konwersja do/z nanosekund dla Go `time.Duration`)
  - Interwał aktualizacji (w godzinach)
  - Poziom logowania (dropdown: DEBUG/INFO/WARN/ERROR)
  - Przełącznik "Uruchamiaj z systemem Windows"
  - Przycisk "Przywróć domyślną allowlistę" (z potwierdzeniem)

#### Wskaźnik użycia cache
- `frontend/App.tsx`: nowa karta **Cache** na dashboardzie pokazująca entries/max (np. "128 / 65 536"), w stonowanym kolorze slate.
- `app.go`: dodano `GetCacheInfo()` zwracającą liczbę wpisów w cache i maksymalną pojemność.

#### Multicast w domyślnej allowliście
- `config/config.go`: dodano `"224.0.0.0/4"` (multicast: SSDP, mDNS, BitTorrent LPD) do domyślnej allowlisty.

### Benchmarki

- Cache Clear: **O(1)** zamiast O(n) — zerowy koszt alokacji
- Cache Get/Set: bez zmian (~89ns / ~242ns)
- Wyszukiwanie binarne: bez zmian (~186ns na 500k zakresów)

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

### Dodano
- GUI Źródła: lista źródeł blokad z przełącznikami włącz/wyłącz
- Parser CIDR: usuwanie komentarzy inline (po `;` lub `#`) dla formatu Spamhaus DROP (`1.2.3.0/24 ; SBL123`)
- Logger updatera: komunikaty postępu per-źródło widoczne w panelu logów GUI
- Konfigurowalny interwał aktualizacji z config.json
- Fetcher: nagłówek User-Agent i automatyczna dekompresja gzip

### Zmieniono
- Domyślne źródła: zastąpiono martwy iblocklist-level1 działającym firehol-level2
- API Updatera: `NewUpdater` przyjmuje `LogFunc` callback i konfigurowalny `interval`

### Naprawiono
- Nieskończona pętla WinDivert: dodano sprawdzanie flagi Impostor — reinjektowane pakiety pomijają pipeline (przerywa pętlę przechwytywania, przywraca internet)
- Wyciek uchwytu WinDivert: `ToggleProtection` i `SetProtectionEnabled` używają `Close()` zamiast `Stop()`, poprawnie zamykając uchwyt
- Wyciek gorutyn pipeline: worker/sendLoop używają `select` z `<-p.done` do czystego zamykania
- `startProtection()`: zamyka istniejący pipeline przed utworzeniem nowego (zapobiega duplikacji uchwytów WinDivert)
- `isAdmin()`: usunięto błędne `os.IsPermission(err)` które fałszywie raportowało nie-adminów jako adminów
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
