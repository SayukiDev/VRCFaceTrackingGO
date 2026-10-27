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

// MainWindow is the name of the application window.
const MainWindow = "main"

// App is the service bound to the frontend. Every exported method except
// the Wails lifecycle hooks is callable from JavaScript.
type App struct {
	ctx context.Context
	app *application.App
	svc *service.Service
}

// New creates the binding object. The service is created in ServiceStartup.
func New() *App { return &App{} }

// ServiceStartup is the Wails service startup hook. It only runs in the
// first instance. Returning an error makes application.Run fail.
func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.ctx = ctx
	a.app = application.Get()
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

// SingleInstance makes a second launch bring a's window to the front
// instead of starting another instance. It is a function rather than a
// method so Wails does not bind it to the frontend.
func SingleInstance(a *App, uniqueID string) *application.SingleInstanceOptions {
	return &application.SingleInstanceOptions{
		UniqueID:               uniqueID,
		OnSecondInstanceLaunch: a.onSecondInstanceLaunch,
	}
}

// onSecondInstanceLaunch runs on its own goroutine; window methods are safe
// to call from there.
func (a *App) onSecondInstanceLaunch(application.SecondInstanceData) {
	if a.app == nil {
		return
	}
	w, ok := a.app.Window.GetByName(MainWindow)
	if !ok {
		return
	}
	w.Restore()
	w.Show()
	w.Focus()
}

func (a *App) GetMetadata() metadata.Metadata { return metadata.Info }

func (a *App) OpenURL(url string) error { return a.app.Browser.OpenURL(url) }
