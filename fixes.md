# go-peerblock — Plan rozwoju

> **Stan:** Stabilny. Wszystkie krytyczne błędy naprawione. Poniżej pełna lista wykonanych fixów.

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

## ⏳ Do zrobienia na później

- [ ] **A1** Instalator NSIS — `build/installer/installer.nsis` nie istnieje. Pełny instalator z WebView2 bootstrapem i obsługą drivera
