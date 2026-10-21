package mixer

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"VRCFaceTrackingGo/pkg/tracking"
)

// Config is the persisted state of every built-in stage.
type Config struct {
	Enabled    bool             `json:"enabled"`
	Adjustment AdjustmentConfig `json:"adjustment"`
	Correctors CorrectorsConfig `json:"correctors"`
	Filter     FilterConfig     `json:"filter"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:    true,
		Adjustment: DefaultAdjustmentConfig(),
		Correctors: DefaultCorrectorsConfig(),
		Filter:     DefaultFilterConfig(),
	}
}

// ComponentState is a snapshot of one Component for the UI.
type ComponentState struct {
	Name  string  `json:"name"`
	Kind  Kind    `json:"kind"`
	Min   float32 `json:"min"`
	Max   float32 `json:"max"`
	Value any     `json:"value"`
}

// StageState is a snapshot of one Stage for the UI.
type StageState struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Active      bool             `json:"active"`
	Components  []ComponentState `json:"components"`
}

// Mixer owns the ordered list of stages and applies them to a private copy
// of the incoming data. Every method is safe for concurrent use.
type Mixer struct {
	mu         sync.Mutex
	enabled    bool
	adjustment Adjustment
	correctors Correctors
	filter     Filter
	stages     []Stage
	buffer     *tracking.TrackingData
}

// New creates a mixer with the built-in stages, sorted by priority.
func New(cfg Config) *Mixer {
	m := &Mixer{buffer: tracking.NewTrackingData()}
	m.stages = []Stage{&m.adjustment, &m.correctors, &m.filter}
	sort.SliceStable(m.stages, func(i, j int) bool {
		return m.stages[i].Priority() < m.stages[j].Priority()
	})
	m.SetConfig(cfg)
	return m
}

// Config returns the current state of every stage.
func (m *Mixer) Config() Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return Config{
		Enabled:    m.enabled,
		Adjustment: m.adjustment.AdjustmentConfig,
		Correctors: m.correctors.CorrectorsConfig,
		Filter:     m.filter.FilterConfig,
	}
}

// SetConfig replaces the state of every stage.
func (m *Mixer) SetConfig(cfg Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = cfg.Enabled
	m.adjustment.AdjustmentConfig = cfg.Adjustment
	m.correctors.CorrectorsConfig = cfg.Correctors
	m.filter.FilterConfig = cfg.Filter
}

func (m *Mixer) Enabled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.enabled
}

func (m *Mixer) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
}

// Stages returns a snapshot of the ordered stage list.
func (m *Mixer) Stages() []StageState {
	m.mu.Lock()
	defer m.mu.Unlock()
	states := make([]StageState, 0, len(m.stages))
	for _, s := range m.stages {
		st := StageState{Name: s.Name(), Description: s.Description(), Active: s.Active()}
		for _, c := range s.Components() {
			cs := ComponentState{Name: c.Name, Kind: c.Kind, Min: c.Min, Max: c.Max}
			if c.Get != nil {
				cs.Value = c.Get()
			}
			st.Components = append(st.Components, cs)
		}
		states = append(states, st)
	}
	return states
}

// SetActive toggles one stage.
func (m *Mixer) SetActive(stage string, active bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, err := m.findLocked(stage)
	if err != nil {
		return err
	}
	s.SetActive(active)
	return nil
}

// SetValue updates a checkbox, slider or range component. value must be a
// bool, a number or a Range.
func (m *Mixer) SetValue(stage, component string, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, err := m.componentLocked(stage, component)
	if err != nil {
		return err
	}
	if c.Set == nil {
		return errors.New("component is not editable")
	}
	return c.Set(value)
}

// Invoke runs a button component's action.
func (m *Mixer) Invoke(stage, component string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, err := m.componentLocked(stage, component)
	if err != nil {
		return err
	}
	if c.Invoke == nil {
		return errors.New("component is not a button")
	}
	c.Invoke()
	return nil
}

func (m *Mixer) findLocked(name string) (Stage, error) {
	for _, s := range m.stages {
		if s.Name() == name {
			return s, nil
		}
	}
	return nil, fmt.Errorf("unknown stage %q", name)
}

func (m *Mixer) componentLocked(stage, component string) (Component, error) {
	s, err := m.findLocked(stage)
	if err != nil {
		return Component{}, err
	}
	for _, c := range s.Components() {
		if c.Name == component {
			return c, nil
		}
	}
	return Component{}, fmt.Errorf("unknown component %q on %q", component, stage)
}

// Mix returns the transformed data. When disabled the input is returned
// unchanged. The result is only valid until the next call.
func (m *Mixer) Mix(input *tracking.TrackingData) *tracking.TrackingData {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.enabled {
		return input
	}
	m.buffer.CopyFrom(input)
	for _, s := range m.stages {
		if s.Active() {
			s.Mutate(m.buffer)
		}
	}
	return m.buffer
}
