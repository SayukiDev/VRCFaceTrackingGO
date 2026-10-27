package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/SayukiDev/VRCFaceTrackingGO/app"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/metadata"
)

// singleInstanceID identifies this application to Wails' single instance
// lock. It must never change.
const singleInstanceID = "a9309d2e-6bc4-46f3-9592-f0cb6979885b"

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/config.yml
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
	// A second instance hands over to the running one inside application.New
	// and exits before the services start, so everything that touches the
	// data or log directory happens in ui.ServiceStartup.
	ui := app.New()

	wails := application.New(application.Options{
		Name:        metadata.Info.Name,
		Description: metadata.Info.Info.Description,
		Services:    []application.Service{application.NewService(ui)},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		SingleInstance: app.SingleInstance(ui, singleInstanceID),
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	wails.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             app.MainWindow,
		Title:            metadata.Info.Name,
		Width:            1200,
		Height:           860,
		MinWidth:         800,
		MinHeight:        500,
		Frameless:        true,
		BackgroundColour: application.NewRGBA(24, 24, 28, 255),
	})

	return wails.Run()
}
