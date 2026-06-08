package systray

import (
	"context"
	"time"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go-peerblock/i18n"
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
	GetLanguage() string
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
	systray.SetTooltip(i18n.T(app.GetLanguage(), "systray.tooltip"))

	// Set the tray icon from pre-converted ICO bytes
	if len(appIconData) > 0 {
		systray.SetIcon(appIconData)
	} else {
		systray.SetTitle("GO PeerBlock")
	}

	lang := app.GetLanguage()

	mShow := systray.AddMenuItem(i18n.T(lang, "systray.show"), "Open the main window")
	mToggle := systray.AddMenuItem(i18n.T(lang, "systray.disable"), "Toggle protection")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem(i18n.T(lang, "systray.quit"), "Quit the application")

	updateToggleLabel(mToggle, app.IsProtectionEnabled(), i18n.T(lang, "systray.disable"), i18n.T(lang, "systray.enable"))

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				if ctx := app.GetCtx(); ctx != nil {
					runtime.WindowShow(ctx)
				}
			case <-mToggle.ClickedCh:
				app.ToggleProtection()
				curLang := app.GetLanguage()
				updateToggleLabel(mToggle, app.IsProtectionEnabled(), i18n.T(curLang, "systray.disable"), i18n.T(curLang, "systray.enable"))
			case <-mQuit.ClickedCh:
				if ctx := app.GetCtx(); ctx != nil {
					runtime.Quit(ctx)
				}
				systray.Quit()
			}
		}
	}()
}

func updateToggleLabel(item *systray.MenuItem, enabled bool, disableStr, enableStr string) {
	if enabled {
		item.SetTitle(disableStr)
	} else {
		item.SetTitle(enableStr)
	}
}
