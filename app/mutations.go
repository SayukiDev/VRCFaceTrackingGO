package app

import "VRCFaceTracking/internal/mixer"

func (a *App) GetMutatorEnabled() bool { return a.svc.MixerEnabled() }

func (a *App) SetMutatorEnabled(v bool) { a.svc.SetMixerEnabled(v) }

func (a *App) GetMutations() []mixer.StageState { return a.svc.MixerStages() }

func (a *App) SetMutationActive(name string, active bool) error {
	return a.svc.SetStageActive(name, active)
}

// SetMutationValue updates a checkbox or slider.
func (a *App) SetMutationValue(mutationName, componentName string, value any) error {
	return a.svc.SetStageValue(mutationName, componentName, value)
}

// SetMutationRange updates a range component.
func (a *App) SetMutationRange(mutationName, componentName string, floor, ceil float32) error {
	return a.svc.SetStageValue(mutationName, componentName, mixer.Range{Floor: floor, Ceil: ceil})
}

func (a *App) InvokeMutationButton(mutationName, componentName string) error {
	return a.svc.InvokeStageButton(mutationName, componentName)
}
