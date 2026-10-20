package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"VRCFaceTracking/app"
	"VRCFaceTracking/internal/logging"
	"VRCFaceTracking/internal/metadata"
	"VRCFaceTracking/internal/service"
	"VRCFaceTracking/pkg/processlock"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed wails.json
var metadataBytes []byte

func init() {
	metadata.Init(metadataBytes)
	metadataBytes = nil
}

func main() {
	// Process lock
	ep, _ := os.Executable()
	if processlock.IsLocked(ep, os.Getpid()) {
		fmt.Fprintf(os.Stderr, "Error: %s is already running.", metadata.Info.Name)
		os.Exit(1)
	}

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run() error {
	dataDir, err := metadata.Info.PersistentDataDir()
	if err != nil {
		return err
	}
	if err := service.ApplyResetIfRequested(dataDir); err != nil {
		return err
	}

	logger, logBuf, err := logging.New(metadata.Info.LogDir(), metadata.Info.IsDev())
	if err != nil {
		return err
	}
	defer logger.Sync()
	svc := service.New(logger, logBuf, dataDir)
	ui := app.New(svc)

	return wails.Run(&options.App{
		Title:     metadata.Info.Name,
		Width:     1200,
		Height:    860,
		MinWidth:  800,
		MinHeight: 500,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 24, G: 24, B: 28, A: 1},
		OnStartup:        ui.Startup,
		OnShutdown:       ui.Shutdown,
		Bind:             []any{ui},
		Frameless:        true,
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})
}
