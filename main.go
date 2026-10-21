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
	"VRCFaceTracking/internal/metadata"
)

// singleInstanceID identifies this application to Wails' single instance
// lock. It must never change.
const singleInstanceID = "a9309d2e-6bc4-46f3-9592-f0cb6979885b"

//go:embed all:frontend/dist
var assets embed.FS

//go:embed wails.json
var metadataBytes []byte

func init() {
	metadata.Init(metadataBytes)
	metadataBytes = nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run() error {
	// A second instance hands over to the running one inside wails.Run and
	// exits before OnStartup, so everything that touches the data or log
	// directory happens in ui.Startup.
	ui := app.New()

	return wails.Run(&options.App{
		Title:     metadata.Info.Name,
		Width:     1200,
		Height:    860,
		MinWidth:  800,
		MinHeight: 500,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:   &options.RGBA{R: 24, G: 24, B: 28, A: 1},
		SingleInstanceLock: app.SingleInstanceLock(ui, singleInstanceID),
		OnStartup:          ui.Startup,
		OnShutdown:         ui.Shutdown,
		Bind:               []any{ui},
		Frameless:          true,
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})
}
