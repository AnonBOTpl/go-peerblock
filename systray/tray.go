package systray

import (
	"context"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the interface the systray needs from the main App.
type App interface {
	GetCtx() context.Context
	IsProtectionEnabled() bool
	ToggleProtection()
}

// RunTray starts the system tray icon loop.
func RunTray(app App) {
	systray.Run(func() {
		setupMenu(app)
	}, nil)
}

func setupMenu(app App) {
	// We need icon data. For now, use a placeholder.
	// In production: systray.SetIcon(iconData)
	systray.SetTitle("go-peerblock")
	systray.SetTooltip("go-peerblock - IP Blocker")

	mShow := systray.AddMenuItem("Pokaż okno", "Open the main window")
	mToggle := systray.AddMenuItem("Wyłącz ochronę", "Toggle protection")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Zamknij", "Quit the application")

	updateToggleLabel(mToggle, app.IsProtectionEnabled())

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				if ctx := app.GetCtx(); ctx != nil {
					runtime.WindowShow(ctx)
				}
			case <-mToggle.ClickedCh:
				app.ToggleProtection()
				updateToggleLabel(mToggle, app.IsProtectionEnabled())
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func updateToggleLabel(item *systray.MenuItem, enabled bool) {
	if enabled {
		item.SetTitle("Wyłącz ochronę")
	} else {
		item.SetTitle("Włącz ochronę")
	}
}
