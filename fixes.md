# go-peerblock — Stan projektu (czerwiec 2026)

Wszystkie krytyczne błędy naprawione. Poniżej pełna lista wykonanych fixów oraz plan dalszego rozwoju.

---

## ✅ Wykonane fixy (20/20)

### 🔴 Krytyczne (5)

| # | Problem | Fix | Plik |
|---|---|---|---|
| 1 | Race condition w stats — `atomic.Value` z `Load+modify+Store` |
| 2 | `isDriverLoaded` zawsze zwracało `false` |
| 3 | `installDriver` ignorowało błędy |
| 17 | **Updater nie widzi źródeł dodanych przez GUI** — `SaveConfig` nie przekazywał nowych źródeł do updatera |
| 19 | **🔴 SrcIP w DB blokował WSZYSTKO** — `shouldBlock` sprawdzało SrcIP w bazie. Użytkownik ma `172.16.3.206`, firehol-level1 ma `172.16.0.0/12` → każdy pakiet blokowany |

### 🟠 Poważne (5)

| # | Problem | Fix | Plik |
|---|---|---|---|
| 4 | `updateAll` trzymało lock podczas I/O sieciowego |
| 5 | Allowlist niesortowany — binary search działał losowo |
| 6 | Ignorowany błąd `logger.Close()` w shutdown |
| 7 | Magic number dla CacheTTL |
| 18 | Cache zatruty po zmianie bazy — double-clear |

### 🟡 Umiarkowane (5)

| # | Problem | Fix | Plik |
|---|---|---|---|
| 8 | Pipeline nie widzi nowej bazy po aktualizacji |
| 9 | Blokujący `Recv` bez timeoutu |
| 10 | Format detektor — false positive dla P2P vs CIDR |
| 11 | `RingBuffer.Len()` było O(n) |
| 12 | Niezgodna sygnatura `Open` w noop (`int32` vs `interface{}`) |
| 20 | Race condition w shouldBlock — cache "blocked" po Clear |

### 🟢 Drobne (4)

| # | Problem | Fix | Plik |
|---|---|---|---|
| 14 | Magic int (`3`) zamiast `int(core.FormatCIDR)` w sources.go |
| 15 | `done` channel w allowlist nigdy nie zamykany |
| 21 | `LastSync` nie ustawiany po aktualizacji — GUI pokazywało "1.01.1" |
| 22 | Przyciski Aktualizuj z niezależnymi stanami `updating` |

### ❌ Nie dotyczy (1)

| # | Problem | Werdykt |
|---|---|---|
| 13 | `go.mod` deklaruje `go 1.26.4` | **OK** — Go 1.26.4 istnieje (wydany 2 czerwca 2026) ✅ |

### ✅ Od początku OK (1)

| # | Problem | Werdykt |
|---|---|---|
| 16 | `go.sum` w .gitignore | Nie ma go w .gitignore ✅ |

---

## 📋 Audyt — co jeszcze można zrobić

### 🔴 Krytyczne — brakujące funkcje

| LP | Co | Opis | Plik |
|---|---|---|---|
| A1 | **Instalator NSIS** | `build/installer/installer.nsis` nie istnieje. Plan opisuje pełny instalator z WebView2 bootstrapem i obsługą drivera | nowy plik |
| A2 | **Autostart z systemem** | `Config.StartWithSystem` istnieje, ale brak implementacji (rejestr Windows) | app.go |
| A3 | **Minimalizacja do tray** | Zamknięcie okna (X) zamyka aplikację. Powinno chować do systray | main.go / app.go |

### 🟠 Poważne — warte dodania

| LP | Co | Opis | Plik |
|---|---|---|---|
| A4 | **Panel ustawień GUI** | UI do edycji: allowlisty, liczby workerów, rozmiaru cache, interwału aktualizacji | frontend/App.tsx |
| A5 | **Wydzielenie komponentów React** | Dashboard, LogView, SourcesView, AddSourceDialog — każdy osobny plik | frontend/src/ |
| A6 | **Pakiety na sekundę** | Wyświetlanie przepustowości na pasku statusu | frontend/App.tsx |
| A7 | **Eventy Wails zamiast pollingu** | `runtime.EventsEmit` dla logów i statystyk w czasie rzeczywistym | app.go / App.tsx |
| A8 | **Ikona w systray** | `systray.SetIcon(iconData)` — zakomentowane, brak pliku .ico | systray/tray.go |
| A9 | **Testy integracyjne** | `go test -race ./...` — są testy core, brak pipeline/updater/app | filter/, updater/ |
| A10 | **Rotacja logów** | `LogMaxSizeMB` istnieje, logger nie rotuje plików | logger/logger.go |
| A11 | **Wykres blokad** | Sparkline ostatnich 60 minut blokad (plan fazy 10) | frontend/App.tsx |

### 🟡 Drobne poprawki

| LP | Co | Opis | Plik |
|---|---|---|---|
| A12 | **Podwójny MergeRanges** | `updateAll` woła `MergeRanges`, potem `NewDatabase` wozi go drugi raz — zbędne | updater/updater.go |
| A13 | **README nieaktualne** | Wspomina o winutil, iblocklist, starych źródłach | README.md |
| A14 | **`appCtx` global** | Zmienna globalna w main.go — kod smell | main.go |

### 🟢 Koncepcyjne

| LP | Co | Opis |
|---|---|---|
| A15 | **Tryb "tylko test-blocklist" jednym kliknięciem** | Przycisk "Test" który wyłącza wszystko i włącza test-blocklist |
| A16 | **Eksport logów do pliku CSV/TXT** | Przycisk "Eksportuj logi" w LogView |
| A17 | **Statystyki dzienne/tygodniowe** | Podsumowanie blokad w czasie |
| A18 | **Ciemny/jasny motyw** | Przełącznik motywu |

---

## Podsumowanie

| Status | Liczba |
|---|---|
| ✅ Fixy wykonane | **20** |
| ❌ Nie dotyczy | 1 (#13) |
| 🔴 Brakujące funkcje | **3** (A1–A3) |
| 🟠 Warte dodania | **8** (A4–A11) |
| 🟡 Drobne poprawki | **3** (A12–A14) |
| 🟢 Koncepcyjne | **4** (A15–A18) |
