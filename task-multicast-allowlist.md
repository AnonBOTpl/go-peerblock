# Task: Dodaj multicast do domyślnej allowlisty + przycisk reset allowlisty

## Problem

Domyślna allowlista w `config/config.go` nie zawiera zakresu multicast (`224.0.0.0/4`).
Skutek: go-peerblock blokuje lokalny ruch multicast (SSDP, BitTorrent LPD, mDNS) gdy
listy FireHOL są aktywne, bo FireHOL traktuje prywatne zakresy jako podejrzane w ruchu publicznym.

Objaw widoczny w logach przy uruchomionym uTorrencie:
```
BLOCK 172.16.3.206 → 239.255.255.250 [UDP]   // SSDP / UPnP discovery
BLOCK 172.16.3.206 → 239.192.152.143 [UDP]   // BitTorrent Local Peer Discovery
```

## Zakres zmian

### 1. `config/config.go` — dodaj multicast do Defaults()

Znajdź funkcję `Defaults()` i dodaj `"224.0.0.0/4"` do slice'a `Allowlist`:

```go
Allowlist: []string{
    "8.8.8.8",
    "8.8.4.4",
    "1.1.1.1",
    "192.168.0.0/16",
    "10.0.0.0/8",
    "172.16.0.0/12",
    "224.0.0.0/4",    // multicast — SSDP, mDNS, BitTorrent LPD, inne
},
```

`224.0.0.0/4` obejmuje cały zakres multicast od `224.0.0.0` do `239.255.255.255` — jeden wpis wystarczy.

---

### 2. `app.go` — dodaj metodę `ResetAllowlist()` eksportowaną do frontendu

Dodaj nową metodę w `app.go`:

```go
// ResetAllowlist resets the allowlist to the default values and saves config.
func (a *App) ResetAllowlist() error {
    defaults := config.Defaults()
    a.cfg.Allowlist = defaults.Allowlist
    a.allowlist = core.NewAllowlist(a.cfg.Allowlist)
    return a.configP.Save(a.cfg)
}
```

---

### 3. Frontend — dodaj przycisk "Przywróć domyślną allowlistę"

W widoku Ustawień (Settings), w sekcji allowlisty, dodaj przycisk który wywołuje `ResetAllowlist()`.

Przycisk powinien:
- Być oznaczony np. "Przywróć domyślne" lub "Reset allowlisty"
- Po kliknięciu pokazać prosty confirm dialog (żeby użytkownik nie skasował swoich wpisów przez przypadek)
- Po potwierdzeniu wywołać `window.go.main.App.ResetAllowlist()`
- Odświeżyć wyświetlaną listę allowlisty

Przykładowy kod React (dostosuj do istniejącego stylu komponentu Settings):

```tsx
const handleResetAllowlist = async () => {
    if (!confirm("Czy na pewno chcesz przywrócić domyślną allowlistę? Twoje własne wpisy zostaną usunięte.")) {
        return;
    }
    try {
        await ResetAllowlist();
        // odśwież config
        const cfg = await GetConfig();
        setConfig(cfg);
    } catch (e) {
        console.error("Reset allowlisty nie powiódł się:", e);
    }
};

// W JSX:
<button onClick={handleResetAllowlist}>
    Przywróć domyślną allowlistę
</button>
```

---

## Uwaga dla istniejących instalacji

Zmiana `Defaults()` działa tylko dla nowych instalacji (brak `config.json`).
Użytkownicy z istniejącym `config.json` muszą użyć przycisku "Przywróć domyślną allowlistę"
lub ręcznie dodać `224.0.0.0/4` w Ustawieniach.

## Weryfikacja

Po wprowadzeniu zmian:
1. Usuń `%APPDATA%\go-peerblock\config.json` żeby wymusić fresh start
2. Uruchom aplikację
3. Sprawdź w Ustawieniach że allowlista zawiera `224.0.0.0/4`
4. Uruchom uTorrent — logi nie powinny pokazywać BLOCK dla `239.x.x.x`
