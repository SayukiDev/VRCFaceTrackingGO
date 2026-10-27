package app

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/SayukiDev/VRCFaceTrackingGO/internal/logging"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/oscquery"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/router"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/sender"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/service"
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
	app *application.App
}

func (e *emitter) OnLog(line logging.Line) { e.app.Event.Emit(eventLog, line) }

func (e *emitter) OnModulesChanged(mods []router.ModuleInfo) {
	e.app.Event.Emit(eventModules, mods)
}

func (e *emitter) OnAvatarChanged(info oscquery.AvatarInfo) {
	e.app.Event.Emit(eventAvatar, info)
}

func (e *emitter) OnParametersChanged(stats sender.ParameterStats) {
	e.app.Event.Emit(eventParameters, stats)
}

func (e *emitter) OnOscStats(status service.OscStatus) {
	e.app.Event.Emit(eventOscStats, status)
}
