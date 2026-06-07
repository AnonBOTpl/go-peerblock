package systray

import (
	"context"
	"time"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// appIconData stores the application icon (ICO bytes) passed from main.
var appIconData []byte

// SetAppIcon sets the application icon data (ICO format bytes) for the systray.
// Must be called before RunTray.
func SetAppIcon(data []byte) {
	appIconData = data
}

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
	}, func() {
		// Wait for Windows Shell to process the icon removal (Shell_NotifyIcon NIM_DELETE).
		// Without this delay, ghost icons accumulate when the process exits too quickly.
		time.Sleep(200 * time.Millisecond)
	})
}

// QuitTray signals the system tray to exit.
func QuitTray() {
	systray.Quit()
}

func setupMenu(app App) {
	systray.SetTooltip("GO PeerBlock - IP Filter")

	// Set the tray icon from pre-converted ICO bytes
	if len(appIconData) > 0 {
		systray.SetIcon(appIconData)
	} else {
		systray.SetTitle("GO PeerBlock")
	}

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
				if ctx := app.GetCtx(); ctx != nil {
					runtime.Quit(ctx)
				}
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
