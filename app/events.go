package app

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"VRCFaceTracking/internal/logging"
	"VRCFaceTracking/internal/oscquery"
	"VRCFaceTracking/internal/router"
	"VRCFaceTracking/internal/sender"
	"VRCFaceTracking/internal/service"
)

// Event names emitted to the frontend; keep in sync with
// frontend/src/composables/useBackend.ts.
const (
	eventLog        = "log:line"
	eventModules    = "modules:changed"
	eventAvatar     = "avatar:changed"
	eventOscStats   = "osc:stats"
	eventParameters = "parameters:changed"
)

// emitter implements service.Listener by emitting Wails events.
type emitter struct {
	ctx context.Context
}

func (e *emitter) OnLog(line logging.Line) { runtime.EventsEmit(e.ctx, eventLog, line) }

func (e *emitter) OnModulesChanged(mods []router.ModuleInfo) {
	runtime.EventsEmit(e.ctx, eventModules, mods)
}

func (e *emitter) OnAvatarChanged(info oscquery.AvatarInfo) {
	runtime.EventsEmit(e.ctx, eventAvatar, info)
}

func (e *emitter) OnParametersChanged(stats sender.ParameterStats) {
	runtime.EventsEmit(e.ctx, eventParameters, stats)
}

func (e *emitter) OnOscStats(status service.OscStatus) {
	runtime.EventsEmit(e.ctx, eventOscStats, status)
}
