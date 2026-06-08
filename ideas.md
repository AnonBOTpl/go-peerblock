# go-peerblock — Pomysły na przyszłość

> Pomysły do zrealizowania gdy aplikacja zdobędzie zainteresowanie użytkowników.
> Obecny stan: stabilna v0.4.1, wszystkie krytyczne błędy naprawione.

---

## 🔵 Średni priorytet

### Code signing (podpis kodu)
- Zakup certyfikatu code signing (np. Sectigo, DigiCert)
- Podpisanie `go-peerblock.exe` i instalatora NSIS
- Eliminuje ostrzeżenie SmartScreen przy instalacji
- Windows Defender przestaje flagować WinDivert

### Auto-update
- Aplikacja sprawdza nową wersję na GitHub Releases przy starcie
- Pobiera i uruchamia nowy instalator
- Wymaga API GitHub Releases

---

## 🟢 Niski priorytet

### Blokowanie per aplikacja (jak PeerBlock)
- Wyświetlanie nazwy procesu który łączy się z blokowanym IP
- Możliwość blokowania/przepuszczania per proces
- Wykorzystanie Windows Event Tracing lub Windows Filtering Platform

### Import / eksport konfiguracji
- Backup ustawień (źródła, allowlista, własne reguły) do pliku JSON
- Przywracanie z pliku
- Przydatne przy reinstalacji systemu

### IPv6 support
- Rozszerzenie WinDivert o filter IPv6
- Parsowanie i blokowanie zakresów IPv6
- Znacznie większa baza IP

### CI/CD na GitHub Actions
- Automatyczny build przy pushu tagu v*
- Budowanie instalatora NSIS
- Upload do GitHub Releases
- Uruchamianie testów przy każdym PR

### Więcej testów
- Testy dla `installDriver()`, `isDriverInstalled()`, `findSysPath()`
- Testy dla nowych funkcji w `main.go`
- Testy integracyjne pipeline + WinDivert

### Dwujęzyczny opis release
- Release notes na GitHub w PL i EN
- Spójność z dwujęzycznym README

---

*Ostatnia aktualizacja: 2026-06-08*
