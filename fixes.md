# go-peerblock — Plan rozwoju

> **Stan:** Stabilny. Wszystkie krytyczne błędy naprawione. Poniżej pełna lista wykonanych fixów, audyt oraz pomysły na przyszłość.

---

## ✅ Wykonane fixy (20/20)

### 🔴 Krytyczne (5)

- [x] **#1** Race condition w stats — `atomic.Value` z `Load+modify+Store`
- [x] **#2** `isDriverLoaded` zawsze zwracało `false`
- [x] **#3** `installDriver` ignorowało błędy
- [x] **#17** Updater nie widzi źródeł dodanych przez GUI — `SaveConfig` nie przekazywał nowych źródeł do updatera
- [x] **#19** SrcIP w DB blokował WSZYSTKO — `shouldBlock` sprawdzało SrcIP w bazie. Użytkownik ma `172.16.3.206`, firehol-level1 ma `172.16.0.0/12` → każdy pakiet blokowany

### 🟠 Poważne (5)

- [x] **#4** `updateAll` trzymało lock podczas I/O sieciowego
- [x] **#5** Allowlist niesortowany — binary search działał losowo
- [x] **#6** Ignorowany błąd `logger.Close()` w shutdown
- [x] **#7** Magic number dla CacheTTL
- [x] **#18** Cache zatruty po zmianie bazy — double-clear

### 🟡 Umiarkowane (5)

- [x] **#8** Pipeline nie widzi nowej bazy po aktualizacji
- [x] **#9** Blokujący `Recv` bez timeoutu
- [x] **#10** Format detektor — false positive dla P2P vs CIDR
- [x] **#11** `RingBuffer.Len()` było O(n)
- [x] **#12** Niezgodna sygnatura `Open` w noop (`int32` vs `interface{}`)
- [x] **#20** Race condition w shouldBlock — cache "blocked" po Clear

### 🟢 Drobne (4)

- [x] **#14** Magic int (`3`) zamiast `int(core.FormatCIDR)` w sources.go
- [x] **#15** `done` channel w allowlist nigdy nie zamykany
- [x] **#21** `LastSync` nie ustawiany po aktualizacji — GUI pokazywało "1.01.1"
- [x] **#22** Przyciski Aktualizuj z niezależnymi stanami `updating`

### ❌ Nie dotyczy / OK

- [x] **#13** `go.mod` deklaruje `go 1.26.4` — OK, istnieje (wydany 2 czerwca 2026)
- [x] **#16** `go.sum` w .gitignore — nie ma go w .gitignore ✅

---

## 📋 Audyt — co jeszcze można zrobić

### ✅ Zrobione w poprzednich sesjach

- [x] **A3** Minimalizacja do tray — przycisk ⬇ w headerze, systray w goroutine, okno żyje gdy ukryte
- [x] **A4** Panel ustawień GUI — zakładka Settings z edytorem allowlisty, workerów, cache, TTL, interwału, log level
- [x] **A5** Wydzielenie komponentów React — Dashboard, LogView, SourcesView, AddSourceDialog — każdy osobny plik
- [x] **A8** Ikona w systray — `systray.SetIcon(iconData)` z własną ikoną użytkownika
- [x] **A14** `appCtx` global usunięty — brak globalnych zmiennych w main.go
- [x] **I1** Powiadomienia Windows — toast notification gdy lista się zaktualizuje (z opcją wyłączenia w ustawieniach)
- [x] **I2** Podgląd źródła blokady — kliknięcie na zablokowany IP w zakładce Wykresy pokazuje z której listy pochodzi
- [x] **RingBuffer** Testy + benchmarki z `-race` — 10 testów, 2 benchmarki, wszystkie bez race condition

### 🔴 Do zrobienia — krytyczne

- [ ] **A1** Instalator NSIS — `build/installer/installer.nsis` nie istnieje. Pełny instalator z WebView2 bootstrapem i obsługą drivera

### 🟠 Warte dodania

- [x] **A6** Pakiety na sekundę — wyświetlanie przepustowości na pasku statusu
- [ ] **A7** Eventy Wails zamiast pollingu — `runtime.EventsEmit` dla logów i statystyk w czasie rzeczywistym
- [ ] **A9** Testy integracyjne — `go test -race ./...` są testy core, brak pipeline/updater/app
- [ ] **A10** Rotacja logów — `LogMaxSizeMB` istnieje, logger nie rotuje plików
- [x] **A11** Wykres blokad — line chart (Chart.js), blokowane vs przepuszczone pakiety/s, przełącznik 5m/10m/30m
- [x] **I3** Statystyki per lista — ile zakresów pochodzi z FireHOL, ile ze Spamhaus itd., widoczne w zakładce Źródła

### 🟡 Drobne poprawki

- [x] **A12** Podwójny MergeRanges — `updateAll` woła `MergeRanges`, potem `NewDatabase` woła go drugi raz — zbędne
- [x] **A13** README zaktualizowany — dodane nowe features (notyfikacje, podgląd źródła, statystyki per lista, wykres blokad)
- [ ] **I4** Harmonogram aktualizacji — opcja "aktualizuj o konkretnej godzinie" (np. 3:00)
- [ ] **I5** Auto-backup config — kopia `config.json` przed każdą aktualizacją list

### 🟢 Koncepcyjne / przyszłe

- [ ] **A15** Tryb "tylko test-blocklist" — przycisk "Test" który wyłącza wszystko i włącza test-blocklist
- [ ] **A16** Eksport logów do pliku CSV/TXT — przycisk "Eksportuj logi" w LogView
- [ ] **A17** Statystyki historyczne z wykresami — liczniki nie resetują się przy restarcie, zapis do pliku, wykres "zablokowanych pakietów w ostatnich 24h/7 dniach"
- [ ] **A18** Ciemny/jasny motyw — przełącznik motywu
- [ ] **I6** Tryb nauki — zamiast blokować, przez X minut tylko loguj co by było zablokowane
- [ ] **I7** Własne reguły użytkownika — plik `custom.txt` gdzie użytkownik wpisuje własne zakresy CIDR

---

## 🔮 Integracja z go-dnsblock (plan na przyszłość)

**go-dnsblock** to planowana siostrzana aplikacja — lokalny serwer DNS (DNS sinkhole) blokujący reklamy i złośliwe domeny po nazwie domeny zamiast po IP. Działa podobnie do AdGuard Home / Pi-hole ale jako lekka natywna aplikacja Windows.

- Napisana w Go + Wails
- Reużywa dużo kodu z go-peerblock (updater, logger, systray, GUI)
- Plan gotowy, implementacja po ustabilizowaniu go-peerblock

**Wizja:** Docelowo obie aplikacje mogą działać jako jeden pakiet z wspólnym GUI (osobne zakładki: "IP Blocker" i "DNS Blocker").

---

## 📊 Podsumowanie

| Status | Liczba |
|---|---|
| ✅ Fixy wykonane | **20** |
| ✅ Z audytu zrobione | **13** (A3–A6, A8, A11–A14, I1–I3, RingBuffer) |
| ❌ Nie dotyczy | **2** (#13, #16) |
| 🔴 Do zrobienia — krytyczne | **1** (A1) |
| 🟠 Do zrobienia — warte dodania | **3** (A7, A9, A10) |
| 🟡 Do zrobienia — drobne poprawki | **2** (I4–I5) |
| 🟢 Koncepcyjne / przyszłe | **6** (A15–A18, I6–I7) |
| 🔮 go-dnsblock | **1** (integracja future) |
