// Package mutation post-processes raw tracking data (filtering,
// range adjustment) before it is turned into output parameters.
package mutation

import (
	"fmt"

	"VRCFaceTracking/pkg/tracking"
)

// Priority orders mutations: preprocessors run first, postprocessors last.
type Priority int

const (
	Preprocessor Priority = iota
	None
	Postprocessor
)

// Kind is the UI control used to edit a Component.
type Kind string

const (
	KindCheckbox Kind = "checkbox"
	KindSlider   Kind = "slider"
	KindRange    Kind = "range"
	KindButton   Kind = "button"
)

// Range is a (floor, ceil) pair used by ParameterAdjustment.
type Range struct {
	Floor float32 `json:"floor"`
	Ceil  float32 `json:"ceil"`
}

// Component is one user editable property (or action) of a mutation.
type Component struct {
	Name string
	Kind Kind
	Min  float32
	Max  float32
	// SaveImmediately requests the mutation be persisted after every change.
	SaveImmediately bool

	// Get returns the current value: bool, float32 or Range. Nil for buttons.
	Get func() any
	// Set stores a new value of the same type Get returns. Nil for buttons.
	Set func(v any) error
	// Invoke runs the action. Only set for buttons.
	Invoke func()
}

// Mutation transforms tracking data in place.
type Mutation interface {
	Name() string
	Description() string
	Priority() Priority
	// Persisted reports whether the mutation's state is saved to settings.
	Persisted() bool
	Active() bool
	SetActive(bool)
	Initialize(data *tracking.TrackingData)
	Mutate(data *tracking.TrackingData)
	Components() []Component
}

func checkbox(name string, saveImmediately bool, field *bool) Component {
	return Component{
		Name: name, Kind: KindCheckbox, SaveImmediately: saveImmediately,
		Get: func() any { return *field },
		Set: func(v any) error {
			b, ok := v.(bool)
			if !ok {
				return fmt.Errorf("%s: expected bool, got %T", name, v)
			}
			*field = b
			return nil
		},
	}
}

func slider(name string, saveImmediately bool, minValue, maxValue float32, field *float32) Component {
	return Component{
		Name: name, Kind: KindSlider, Min: minValue, Max: maxValue, SaveImmediately: saveImmediately,
		Get: func() any { return *field },
		Set: func(v any) error {
			f, ok := toFloat32(v)
			if !ok {
				return fmt.Errorf("%s: expected number, got %T", name, v)
			}
			*field = f
			return nil
		},
	}
}

func rangeSlider(name string, minValue, maxValue float32, field *Range) Component {
	return Component{
		Name: name, Kind: KindRange, Min: minValue, Max: maxValue,
		Get: func() any { return *field },
		Set: func(v any) error {
			r, ok := v.(Range)
			if !ok {
				return fmt.Errorf("%s: expected range, got %T", name, v)
			}
			*field = r
			return nil
		},
	}
}

func button(name string, invoke func()) Component {
	return Component{Name: name, Kind: KindButton, Invoke: invoke}
}

func toFloat32(v any) (float32, bool) {
	switch n := v.(type) {
	case float32:
		return n, true
	case float64:
		return float32(n), true
	case int:
		return float32(n), true
	case int64:
		return float32(n), true
	}
	return 0, false
}

// activeState is embedded by mutations to provide Active/SetActive.
type activeState struct {
	IsActive bool `json:"isActive"`
}

func (a *activeState) Active() bool     { return a.IsActive }
func (a *activeState) SetActive(b bool) { a.IsActive = b }
