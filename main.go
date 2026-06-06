package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Check admin rights and WinDivert driver
	if err := checkAdminAndDriver(); err != nil {
		log.Fatalf("Błąd uruchomienia: %v\n"+
			"Uruchom aplikację jako Administrator.", err)
	}

	// Create application instance
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "go-peerblock",
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
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil || os.IsPermission(err)
}

// isDriverLoaded checks if a Windows service/driver is loaded.
func isDriverLoaded(name string) bool {
	// Simplified: just return true for development.
	// In production, this would query the SCM.
	return false
}

// installDriver installs and starts the WinDivert kernel driver.
func installDriver() error {
	// In production, this runs install-driver.bat.
	// For development, just return nil to allow compilation.
	return nil
}

// App context is stored here so systray can access it.
var appCtx context.Context

func init() {
	runtime.LockOSThread()
}


