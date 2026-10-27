// Package app is the Wails binding layer: it forwards frontend calls to the
// service and turns service callbacks into Wails events.
package app

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/SayukiDev/VRCFaceTrackingGO/internal/logging"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/metadata"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/service"
)

// mainWindow is the name of the application window.
const mainWindow = "main"

// creatingMainWindow is only accessed on the main thread. WebView2 pumps
// window messages while the main window is being created, so a second tray
// click re-enters showMainWindow before newMainWindow returns.
var creatingMainWindow bool

// App is the service bound to the frontend. Every exported method except
// the Wails lifecycle hooks is callable from JavaScript.
type App struct {
	ctx context.Context
	app *application.App
	svc *service.Service
}

// New creates the binding object and registers it as a service of wails.
// The service is created in ServiceStartup.
func New(wails *application.App) *App {
	a := &App{app: wails}
	wails.RegisterService(application.NewService(a))
	return a
}

// Run opens the main window and the system tray of a's application and
// blocks until the application quits. Closing the main window only sends the
// application to the tray. It is a function rather than a method so Wails
// does not bind it to the frontend.
func Run(a *App, trayIcon []byte) error {
	newMainWindow(a.app)
	newTray(a.app, trayIcon)
	return a.app.Run()
}

func newMainWindow(wails *application.App) *application.WebviewWindow {
	return wails.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             mainWindow,
		Title:            metadata.Info.Name,
		Width:            1200,
		Height:           860,
		MinWidth:         800,
		MinHeight:        500,
		Frameless:        true,
		BackgroundColour: application.NewRGBA(24, 24, 28, 255),
	})
}

// ServiceStartup is the Wails service startup hook. It only runs in the
// first instance. Returning an error makes application.Run fail.
func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.ctx = ctx
	if err := a.start(ctx); err != nil {
		a.app.Dialog.Error().SetTitle(metadata.Info.Name).SetMessage(err.Error()).Show()
		return err
	}
	return nil
}

func (a *App) start(ctx context.Context) error {
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
	a.svc = service.New(logger, logBuf, dataDir)
	if err := a.svc.Start(ctx, &emitter{app: a.app}); err != nil {
		return fmt.Errorf("service failed to start: %w", err)
	}
	return nil
}

// ServiceShutdown is the Wails service shutdown hook.
func (a *App) ServiceShutdown() error {
	if a.svc != nil {
		a.svc.Stop()
	}
	return nil
}

// SingleInstance makes a second launch bring the main window to the front
// instead of starting another instance.
func SingleInstance(uniqueID string) *application.SingleInstanceOptions {
	return &application.SingleInstanceOptions{
		UniqueID:               uniqueID,
		OnSecondInstanceLaunch: onSecondInstanceLaunch,
	}
}

// onSecondInstanceLaunch runs on its own goroutine; window methods are safe
// to call from there.
func onSecondInstanceLaunch(application.SecondInstanceData) {
	showMainWindow(application.Get())
}

// showMainWindow brings the main window to the front, recreating it when it
// has been closed to the tray. It runs on the main thread so that a tray
// click and a second instance launch cannot both recreate the window.
func showMainWindow(wails *application.App) {
	application.InvokeAsync(func() {
		if creatingMainWindow {
			return
		}
		w, ok := wails.Window.GetByName(mainWindow)
		if !ok {
			creatingMainWindow = true
			defer func() { creatingMainWindow = false }()
			newMainWindow(wails)
			return
		}
		w.Restore()
		w.Show()
		w.Focus()
	})
}

func (a *App) GetMetadata() metadata.Metadata { return metadata.Info }

func (a *App) OpenURL(url string) error { return a.app.Browser.OpenURL(url) }

// CloseAllWindows closes every window, which sends the application to the
// tray.
func (a *App) CloseAllWindows() {
	for _, w := range a.app.Window.GetAll() {
		w.Close()
	}
}
