package manager

import "VRCFaceTracking/module/adapter"

// Config is one entry of the user's module list: which module, whether it
// is enabled and which tracking kinds it may claim. The slice order is the
// priority order.
type Config struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
	// Eye / Expression restrict what the module is asked for. A module set
	// to Expression only leaves eye tracking for the next module in line.
	Eye        bool `json:"eye"`
	Expression bool `json:"expression"`
}

// Registered describes a registered module for the Modules page: its
// static metadata, the user's setting and whether it is currently loaded.
type Registered struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Icons              []string `json:"icons"`
	SupportsEye        bool     `json:"supportsEye"`
	SupportsExpression bool     `json:"supportsExpression"`
	Enabled            bool     `json:"enabled"`
	UseEye             bool     `json:"useEye"`
	UseExpression      bool     `json:"useExpression"`
	Loaded             bool     `json:"loaded"`
	Active             bool     `json:"active"`
	UsingEye           bool     `json:"usingEye"`
	UsingExpression    bool     `json:"usingExpression"`
	EyeActive          bool     `json:"eyeActive"`
	ExpressionActive   bool     `json:"expressionActive"`
}

func defaultConfig(mod adapter.Module) Config {
	sup := mod.Supported()
	return Config{
		ID:         mod.ID(),
		Enabled:    true,
		Eye:        sup.Eye,
		Expression: sup.Expression,
	}
}

// Config returns the current module list in priority order.
func (m *Manager) Config() []Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Config(nil), m.config...)
}

// SetConfig replaces the module list. Unknown IDs are dropped; registered
// modules missing from cfg are appended with their defaults so nothing
// disappears. It does not re-initialise anything; call Initialize afterwards.
func (m *Manager) SetConfig(cfg []Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = m.normalize(cfg)
}

func (m *Manager) normalize(cfg []Config) []Config {
	seen := make(map[string]bool, len(m.modules))
	out := make([]Config, 0, len(m.modules))
	for _, c := range cfg {
		mod, ok := m.modules[c.ID]
		if !ok || seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		// Never claim a kind the module cannot provide.
		sup := mod.Supported()
		c.Eye = c.Eye && sup.Eye
		c.Expression = c.Expression && sup.Expression
		out = append(out, c)
	}
	for _, c := range m.config {
		if !seen[c.ID] {
			seen[c.ID] = true
			out = append(out, defaultConfig(m.modules[c.ID]))
		}
	}
	return out
}

// Catalog returns every registered module in priority order with its
// setting and runtime state.
func (m *Manager) Catalog() []Registered {
	m.mu.Lock()
	defer m.mu.Unlock()

	loaded := make(map[string]*entry, len(m.entries))
	for _, e := range m.entries {
		loaded[e.info.ID] = e
	}

	out := make([]Registered, 0, len(m.config))
	for _, c := range m.config {
		mod := m.modules[c.ID]
		meta := mod.Metadata()
		sup := mod.Supported()
		r := Registered{
			ID:                 c.ID,
			Name:               meta.Name,
			Icons:              encodeIcons(meta.Icons),
			SupportsEye:        sup.Eye,
			SupportsExpression: sup.Expression,
			Enabled:            c.Enabled,
			UseEye:             c.Eye,
			UseExpression:      c.Expression,
		}
		if e, ok := loaded[c.ID]; ok {
			r.Loaded = true
			r.Active = e.info.Active
			r.UsingEye = e.info.UsingEye
			r.UsingExpression = e.info.UsingExpression
			r.EyeActive = e.info.EyeActive
			r.ExpressionActive = e.info.ExpressionActive
		}
		out = append(out, r)
	}
	return out
}
