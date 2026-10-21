// Package app is the Wails binding layer: it forwards frontend calls to the
// service and turns service callbacks into Wails events.
package app

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/SayukiDev/VRCFaceTrackingGO/internal/logging"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/metadata"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/service"
)

// App is the object bound to the frontend. Every exported method is
// callable from JavaScript.
type App struct {
	ctx context.Context
	svc *service.Service
}

// New creates the binding object. The service is created in Startup.
func New() *App { return &App{} }

// Startup is the Wails OnStartup hook. It only runs in the first instance.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	if err := a.start(ctx); err != nil {
		_, _ = runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   metadata.Info.Name,
			Message: err.Error(),
		})
		runtime.Quit(ctx)
	}
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
	if err := a.svc.Start(ctx, &emitter{ctx: ctx}); err != nil {
		return fmt.Errorf("service failed to start: %w", err)
	}
	return nil
}

// Shutdown is the Wails OnShutdown hook.
func (a *App) Shutdown(context.Context) {
	if a.svc != nil {
		a.svc.Stop()
	}
}

// SingleInstanceLock makes a second launch bring a's window to the front
// instead of starting another instance. It is a function rather than a
// method so Wails does not bind it to the frontend.
func SingleInstanceLock(a *App, uniqueID string) *options.SingleInstanceLock {
	return &options.SingleInstanceLock{
		UniqueId:               uniqueID,
		OnSecondInstanceLaunch: a.onSecondInstanceLaunch,
	}
}

func (a *App) onSecondInstanceLaunch(options.SecondInstanceData) {
	if a.ctx == nil {
		return
	}
	runtime.WindowUnminimise(a.ctx)
	runtime.WindowShow(a.ctx)
}

func (a *App) GetMetadata() metadata.Metadata { return metadata.Info }

func (a *App) OpenURL(url string) { runtime.BrowserOpenURL(a.ctx, url) }
