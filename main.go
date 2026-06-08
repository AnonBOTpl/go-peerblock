package main

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"go-peerblock/systray"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed frontend/src/assets/tray.ico
var appIconData []byte

func main() {
	// Check admin rights and WinDivert driver
	if err := checkAdminAndDriver(); err != nil {
		log.Fatalf("Startup error: %v\n"+
			"Run the application as Administrator.", err)
	}

	// Create application instance
	app := NewApp()

	// Channel to signal that systray has fully exited
	systrayDone := make(chan struct{})

	// Pass the icon data to the systray package before starting it.
	systray.SetAppIcon(appIconData)

	// Start system tray in background goroutine.
	// This keeps the app alive even when the main window is hidden.
	go func() {
		systray.RunTray(app)
		close(systrayDone)
	}()

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "GO PeerBlock - IP Filter",
		Width:     1024,
		Height:    768,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 59, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			// Jeśli app już zamyka (QuitApp), przepuść bez dialogu
			if app.isQuitting() {
				return false
			}
			// Jeśli użytkownik zaznaczył "Nie pytaj więcej", minimalizuj od razu
			if app.cfg.MinimizeToTrayOnClose {
				wailsRuntime.WindowHide(ctx)
				return true
			}
			// Wyślij event do frontendu — pokażemy custom modal z wyborem
			wailsRuntime.EventsEmit(ctx, "close-request")
			return true // prevent close — frontend zdecyduje przez QuitApp lub MinimizeToTray
		},
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})

	if err != nil {
		log.Fatalf("Wails startup error: %v", err)
	}

	// Wails exited (window closed). Signal systray to quit and wait.
	systray.QuitTray()
	<-systrayDone
}

// checkAdminAndDriver verifies admin privileges and WinDivert driver.
func checkAdminAndDriver() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("application requires Windows")
	}

	if !isAdmin() {
		return fmt.Errorf("application requires administrator privileges")
	}

	if !isDriverLoaded("WinDivert") {
		if err := installDriver(); err != nil {
			return fmt.Errorf("cannot install WinDivert driver: %w", err)
		}
	}

	return nil
}

// isAdmin checks if the process is running with administrator privileges.
func isAdmin() bool {
	// On Windows, this checks if the process has elevated token.
	// Simplified check: try to access a protected path.
	// os.IsPermission(err) = ACCESS_DENIED = NOT admin, so we only check err == nil.
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

// isDriverLoaded checks if a Windows service/driver is loaded.
func isDriverLoaded(name string) bool {
	out, err := exec.Command("sc", "query", name).Output()
	if err != nil {
		return false
	}
	return bytes.Contains(out, []byte("RUNNING"))
}

// isDriverInstalled checks if a Windows service/driver entry exists (regardless of state).
func isDriverInstalled(name string) bool {
	out, err := exec.Command("sc", "query", name).Output()
	if err != nil {
		return false
	}
	return len(out) > 0 && !bytes.Contains(out, []byte("FAILED")) &&
		bytes.Contains(out, []byte(name))
}

// findSysPath searches for WinDivert64.sys in execDir() and the current directory.
func findSysPath() string {
	// During runtime (installed app): same dir as the executable
	sysPath := filepath.Join(execDir(), "WinDivert64.sys")
	if _, err := os.Stat(sysPath); err == nil {
		return sysPath
	}
	// During development (wails build, go run): current working directory
	if wd, err := os.Getwd(); err == nil && wd != execDir() {
		sysPath = filepath.Join(wd, "WinDivert64.sys")
		if _, err := os.Stat(sysPath); err == nil {
			return sysPath
		}
	}
	// Fallback: return execDir() path anyway (caller will handle errors)
	return filepath.Join(execDir(), "WinDivert64.sys")
}

// removeDriverService stops and deletes an existing (possibly broken) WinDivert service entry.
func removeDriverService() {
	exec.Command("sc", "stop", "WinDivert").Run()
	// Small sleep to let Windows process the stop
	_ = exec.Command("cmd", "/C", "timeout", "/t", "1", "/nobreak").Run()
	exec.Command("sc", "delete", "WinDivert").Run()
}

// installDriver installs and starts the WinDivert kernel driver.
func installDriver() error {
	sysPath := findSysPath()

	// Try to start existing service first
	if isDriverInstalled("WinDivert") {
		if _, err := exec.Command("sc", "start", "WinDivert").CombinedOutput(); err != nil {
			// Existing service has a broken binPath (e.g. file in Temp was deleted)
			// Remove it and recreate below
			removeDriverService()
		} else {
			return nil
		}
	}

	// Register the driver
	out, err := exec.Command("sc", "create", "WinDivert",
		"type=", "kernel",
		"start=", "demand",
		"binPath=", sysPath,
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("sc create WinDivert failed: %w\nOutput: %s", err, out)
	}

	// Start the driver
	out, err = exec.Command("sc", "start", "WinDivert").CombinedOutput()
	if err != nil {
		return fmt.Errorf("sc start WinDivert failed: %w\nOutput: %s", err, out)
	}

	return nil
}

// execDir returns the directory of the current executable.
func execDir() string {
	if exe, err := os.Executable(); err == nil {
		if d := filepath.Dir(exe); d != "" {
			return d
		}
	}
	return "."
}
