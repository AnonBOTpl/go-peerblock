package i18n

import "fmt"

// T returns a translated string based on the language code ("pl" or "en").
// Falls back to English if the key is missing for the requested language.
// Supports fmt-style format arguments.
func T(lang, key string, args ...interface{}) string {
	var msg string
	switch lang {
	case "pl":
		msg = pl[key]
		if msg == "" {
			msg = en[key]
		}
	default:
		msg = en[key]
	}
	if msg == "" {
		return key
	}
	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

var en = map[string]string{
	// app.go
	"app.started":                    "go-peerblock started",
	"app.config.load.error":         "Cannot load configuration: %s",
	"app.logger.create.error":       "Cannot create logger: %s",
	"app.config.backup.error":       "Cannot create config backup: %v",
	"app.db.reloaded":               "IP database reloaded: %d ranges",
	"app.protection.disabled":       "Protection disabled",
	"app.protection.enabled":        "Protection enabled (%d workers)",
	"app.windivert.open.error":      "Cannot open WinDivert: %v",
	"app.windivert.opened":          "WinDivert opened: %s",
	"app.autostart.open.error":      "Cannot open autostart registry key: %v",
	"app.autostart.path.error":      "Cannot get executable path for autostart: %v",
	"app.autostart.set.error":       "Cannot set autostart in registry: %v",
	"app.autostart.enabled":         "Autostart enabled: %s",
	"app.autostart.delete.error":    "Cannot remove autostart from registry: %v",
	"app.autostart.disabled":        "Autostart disabled",
	"app.shutdown.logger.error":     "Logger close error: %s",

	// updater/updater.go
	"updater.scheduled":             "Scheduled IP list update...",
	"updater.manual":                "Manual update triggered...",
	"updater.stopped":               "Updater stopped",
	"updater.fetch.error":           "Cannot fetch %s: %v",
	"updater.parse.error":           "Parse error %s: %v",
	"updater.loaded":                "Loaded %d ranges from %s",
	"updater.db.reloaded":           "IP database reloaded: %d ranges (post-merge)",

	// systray/tray.go
	"systray.tooltip":               "GO PeerBlock - IP Filter",
	"systray.show":                  "Show window",
	"systray.disable":               "Disable protection",
	"systray.enable":                "Enable protection",
	"systray.quit":                  "Quit",

	// main.go
	"main.startup.error":            "Startup error: %v\nRun the application as Administrator.",
	"main.wails.error":              "Wails startup error: %v",
	"main.windows.required":         "Application requires Windows",
	"main.admin.required":           "Application requires administrator privileges",
	"main.driver.install.error":     "Cannot install WinDivert driver: %w",
}

var pl = map[string]string{
	// app.go
	"app.started":                    "go-peerblock uruchomiony",
	"app.config.load.error":         "Nie można załadować konfiguracji: %s",
	"app.logger.create.error":       "Nie można utworzyć loggera: %s",
	"app.config.backup.error":       "Nie można utworzyć kopii zapasowej configu: %v",
	"app.db.reloaded":               "Baza IP przeładowana: %d zakresów",
	"app.protection.disabled":       "Ochrona wyłączona",
	"app.protection.enabled":        "Ochrona włączona (%d workerów)",
	"app.windivert.open.error":      "Nie można otworzyć WinDivert: %v",
	"app.windivert.opened":          "WinDivert otwarty: %s",
	"app.autostart.open.error":      "Nie można otworzyć klucza rejestru autostart: %v",
	"app.autostart.path.error":      "Nie można pobrać ścieżki exe dla autostart: %v",
	"app.autostart.set.error":       "Nie można ustawić autostart w rejestrze: %v",
	"app.autostart.enabled":         "Autostart włączony: %s",
	"app.autostart.delete.error":    "Nie można usunąć autostart z rejestru: %v",
	"app.autostart.disabled":        "Autostart wyłączony",
	"app.shutdown.logger.error":     "Logger close error: %s",

	// updater/updater.go
	"updater.scheduled":             "Zaplanowana aktualizacja list IP...",
	"updater.manual":                "Ręczne wyzwolenie aktualizacji...",
	"updater.stopped":               "Aktualizator zatrzymany",
	"updater.fetch.error":           "Nie można pobrać %s: %v",
	"updater.parse.error":           "Błąd parsowania %s: %v",
	"updater.loaded":                "Załadowano %d zakresów z %s",
	"updater.db.reloaded":           "Baza IP przeładowana: %d zakresów (po merge'u)",

	// systray/tray.go
	"systray.tooltip":               "GO PeerBlock - IP Filter",
	"systray.show":                  "Pokaż okno",
	"systray.disable":               "Wyłącz ochronę",
	"systray.enable":                "Włącz ochronę",
	"systray.quit":                  "Zamknij",

	// main.go
	"main.startup.error":            "Błąd uruchomienia: %v\nUruchom aplikację jako Administrator.",
	"main.wails.error":              "Błąd uruchomienia Wails: %v",
	"main.windows.required":         "Aplikacja wymaga systemu Windows",
	"main.admin.required":           "Aplikacja wymaga uprawnień administratora",
	"main.driver.install.error":     "Nie można zainstalować sterownika WinDivert: %w",
}
