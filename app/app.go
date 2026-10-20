// Package app is the Wails binding layer: it forwards frontend calls to the
// service and turns service callbacks into Wails events.
package app

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"VRCFaceTracking/internal/metadata"
	"VRCFaceTracking/internal/service"
)

// App is the object bound to the frontend. Every exported method is
// callable from JavaScript.
type App struct {
	ctx context.Context
	svc *service.Service
}

// New wraps a service for Wails.
func New(svc *service.Service) *App {
	return &App{
		svc: svc,
	}
}

// Startup is the Wails OnStartup hook.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	if err := a.svc.Start(ctx, &emitter{ctx: ctx}); err != nil {
		runtime.LogErrorf(ctx, "service failed to start: %v", err)
	}
}

// Shutdown is the Wails OnShutdown hook.
func (a *App) Shutdown(context.Context) { a.svc.Stop() }

func (a *App) GetMetadata() metadata.Metadata { return metadata.Info }

func (a *App) OpenURL(url string) { runtime.BrowserOpenURL(a.ctx, url) }
