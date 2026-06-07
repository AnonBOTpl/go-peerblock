package main

import (
	"bytes"
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
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed frontend/src/assets/tray.ico
var appIconData []byte

func main() {
	// Check admin rights and WinDivert driver
	if err := checkAdminAndDriver(); err != nil {
		log.Fatalf("Błąd uruchomienia: %v\n"+
			"Uruchom aplikację jako Administrator.", err)
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
		log.Fatalf("Błąd uruchomienia Wails: %v", err)
	}

	// Wails exited (window closed). Signal systray to quit and wait.
	systray.QuitTray()
	<-systrayDone
}

// checkAdminAndDriver verifies admin privileges and WinDivert driver.
func checkAdminAndDriver() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("aplikacja wymaga systemu Windows")
	}

	if !isAdmin() {
		return fmt.Errorf("aplikacja wymaga uprawnień administratora")
	}

	if !isDriverLoaded("WinDivert") {
		if err := installDriver(); err != nil {
			return fmt.Errorf("nie można zainstalować sterownika WinDivert: %w", err)
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

// installDriver installs and starts the WinDivert kernel driver.
func installDriver() error {
	batPath := filepath.Join(execDir(), "build", "installer", "install-driver.bat")
	cmd := exec.Command("cmd", "/C", batPath)
	cmd.Dir = execDir()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("install-driver failed: %w\nOutput: %s", err, out)
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
