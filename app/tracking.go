package app

import (
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/osc"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/oscquery"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/router"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/sender"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/service"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/settings"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/plugin"
)

func (a *App) GetModules() []router.ModuleInfo { return a.svc.Modules() }

func (a *App) SetModuleActive(id string, active bool) { a.svc.SetModuleActive(id, active) }

func (a *App) SetModuleTracking(id string, eye, expression bool) {
	a.svc.SetModuleTracking(id, eye, expression)
}

func (a *App) GetModuleCatalog() []router.Registered { return a.svc.ModuleCatalog() }

func (a *App) SetModuleEnabled(id string, enabled bool) error {
	return a.svc.SetModuleEnabled(id, enabled)
}

func (a *App) SetModuleOrder(order []string) error { return a.svc.SetModuleOrder(order) }

func (a *App) SetModuleUse(id string, eye, expression bool) error {
	return a.svc.SetModuleUse(id, eye, expression)
}

func (a *App) GetModuleConfigDirty() bool { return a.svc.ModuleConfigDirty() }

// InstallModule asks for a module program and installs it. It returns the
// installed module's name, or "" when cancelled.
func (a *App) InstallModule() (string, error) {
	path, err := a.app.Dialog.OpenFile().
		SetTitle("Install module").
		AddFilter("VRCFaceTracking module (*"+plugin.HostSuffix+")", "*"+plugin.HostSuffix).
		PromptForSingleSelection()
	// Cancelling returns an empty path together with an error.
	if path == "" {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	info, err := a.svc.InstallModule(a.ctx, path)
	if err != nil {
		return "", err
	}
	return info.Metadata.Name, nil
}

func (a *App) UninstallModule(id string) error { return a.svc.UninstallModule(id) }

func (a *App) GetTrackingAssignment() router.Assignment { return a.svc.TrackingAssignment() }

// SetTrackingAssignment picks the eye and face module by ID; "" means
// automatic.
func (a *App) SetTrackingAssignment(eye, expression string) error {
	return a.svc.SetTrackingAssignment(router.Assignment{Eye: eye, Expression: expression})
}

func (a *App) ReloadModules() { a.svc.ReloadModules() }

func (a *App) GetAvatar() oscquery.AvatarInfo { return a.svc.Avatar() }

func (a *App) GetParameterStats() sender.ParameterStats { return a.svc.ParameterStats() }

func (a *App) GetOscStatus() service.OscStatus { return a.svc.OscStatus() }

func (a *App) GetOscTarget() osc.Target { return a.svc.Target() }

func (a *App) SetOscTarget(t osc.Target) error { return a.svc.SetTarget(t) }

// GetOscMode returns "oscquery" or "osc".
func (a *App) GetOscMode() string { return string(a.svc.OSCMode()) }

func (a *App) SetOscMode(mode string) error { return a.svc.SetOSCMode(osc.Mode(mode)) }

func (a *App) GetAllParametersRelevant() bool { return a.svc.AllParametersRelevant() }

func (a *App) SetAllParametersRelevant(v bool) { a.svc.SetAllParametersRelevant(v) }

func (a *App) ForceReInit() { a.svc.Reinitialize() }

func (a *App) GetModuleTiming() settings.Timing { return a.svc.Timing() }

func (a *App) SetModuleTiming(t settings.Timing) error { return a.svc.SetTiming(t) }

// Output modes cross the binding as plain strings; Wails does not generate
// TypeScript types for named string types.
func (a *App) GetOutputMode() string { return string(a.svc.OutputMode()) }

func (a *App) SetOutputMode(m string) error { return a.svc.SetOutputMode(sender.OutputMode(m)) }
