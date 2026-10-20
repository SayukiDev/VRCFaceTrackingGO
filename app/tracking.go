package app

import (
	"VRCFaceTracking/internal/engine"
	"VRCFaceTracking/internal/osc"
	"VRCFaceTracking/internal/oscquery"
	"VRCFaceTracking/internal/service"
	"VRCFaceTracking/module/manager"
)

func (a *App) GetModules() []manager.ModuleInfo { return a.svc.Modules() }

func (a *App) SetModuleActive(id string, active bool) { a.svc.SetModuleActive(id, active) }

func (a *App) SetModuleTracking(id string, eye, expression bool) {
	a.svc.SetModuleTracking(id, eye, expression)
}

func (a *App) GetModuleCatalog() []manager.Registered { return a.svc.ModuleCatalog() }

func (a *App) SetModuleEnabled(id string, enabled bool) error {
	return a.svc.SetModuleEnabled(id, enabled)
}

func (a *App) SetModuleOrder(order []string) error { return a.svc.SetModuleOrder(order) }

func (a *App) SetModuleUse(id string, eye, expression bool) error {
	return a.svc.SetModuleUse(id, eye, expression)
}

func (a *App) GetModuleConfigDirty() bool { return a.svc.ModuleConfigDirty() }

func (a *App) ReloadModules() { a.svc.ReloadModules() }

func (a *App) GetAvatar() oscquery.AvatarInfo { return a.svc.Avatar() }

func (a *App) GetParameterStats() engine.ParameterStats { return a.svc.ParameterStats() }

func (a *App) GetOscStatus() service.OscStatus { return a.svc.OscStatus() }

func (a *App) GetOscTarget() osc.Target { return a.svc.Target() }

func (a *App) SetOscTarget(t osc.Target) error { return a.svc.SetTarget(t) }

func (a *App) GetAllParametersRelevant() bool { return a.svc.AllParametersRelevant() }

func (a *App) SetAllParametersRelevant(v bool) { a.svc.SetAllParametersRelevant(v) }

func (a *App) ForceReInit() { a.svc.Reinitialize() }

func (a *App) GetModuleTiming() service.ModuleTiming { return a.svc.ModuleTiming() }

func (a *App) SetModuleTiming(t service.ModuleTiming) error { return a.svc.SetModuleTiming(t) }
