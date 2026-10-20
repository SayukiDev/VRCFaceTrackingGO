package app

import "VRCFaceTracking/internal/mutation"

// MutationView is the frontend representation of a mutation.
type MutationView struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Active      bool            `json:"active"`
	Components  []ComponentView `json:"components"`
}

// ComponentView is the frontend representation of a mutation component.
type ComponentView struct {
	Name  string  `json:"name"`
	Kind  string  `json:"kind"`
	Min   float32 `json:"min"`
	Max   float32 `json:"max"`
	Value any     `json:"value"`
}

func (a *App) GetMutatorEnabled() bool { return a.svc.MutatorEnabled() }

func (a *App) SetMutatorEnabled(v bool) { a.svc.SetMutatorEnabled(v) }

func (a *App) GetMutations() []MutationView {
	muts := a.svc.Mutations()
	views := make([]MutationView, 0, len(muts))
	for _, m := range muts {
		v := MutationView{Name: m.Name(), Description: m.Description(), Active: m.Active()}
		for _, c := range m.Components() {
			cv := ComponentView{Name: c.Name, Kind: string(c.Kind), Min: c.Min, Max: c.Max}
			if c.Get != nil {
				cv.Value = c.Get()
			}
			v.Components = append(v.Components, cv)
		}
		views = append(views, v)
	}
	return views
}

func (a *App) SetMutationActive(name string, active bool) error {
	return a.svc.SetMutationActive(name, active)
}

// SetMutationValue updates a checkbox or slider.
func (a *App) SetMutationValue(mutationName, componentName string, value any) error {
	return a.svc.SetMutationValue(mutationName, componentName, value)
}

// SetMutationRange updates a range component.
func (a *App) SetMutationRange(mutationName, componentName string, floor, ceil float32) error {
	return a.svc.SetMutationValue(mutationName, componentName, mutation.Range{Floor: floor, Ceil: ceil})
}

func (a *App) InvokeMutationButton(mutationName, componentName string) error {
	return a.svc.InvokeMutationButton(mutationName, componentName)
}
