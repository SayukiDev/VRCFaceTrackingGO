package router

import "github.com/SayukiDev/VRCFaceTrackingGO/internal/manager"

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

// Registered describes an installed module for the Modules page: its
// static metadata, the user's setting and whether it is currently loaded.
type Registered struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Author            string   `json:"author"`
	Version           string   `json:"version"`
	PageURL           string   `json:"pageUrl"`
	UsageInstructions string   `json:"usageInstructions"`
	Icons             []string `json:"icons"`
	// Driver is the ID of the driver containing the module, if any.
	Driver             string `json:"driver"`
	SupportsEye        bool   `json:"supportsEye"`
	SupportsExpression bool   `json:"supportsExpression"`
	Enabled            bool   `json:"enabled"`
	UseEye             bool   `json:"useEye"`
	UseExpression      bool   `json:"useExpression"`
	Loaded             bool   `json:"loaded"`
	Active             bool   `json:"active"`
	UsingEye           bool   `json:"usingEye"`
	UsingExpression    bool   `json:"usingExpression"`
	EyeActive          bool   `json:"eyeActive"`
	ExpressionActive   bool   `json:"expressionActive"`
}

func defaultConfig(desc manager.Info) Config {
	return Config{
		ID:         desc.ID,
		Enabled:    true,
		Eye:        desc.Supported.Eye,
		Expression: desc.Supported.Expression,
	}
}

func (r *Router) infoLocked(id string) (manager.Info, bool) {
	for _, info := range r.registry {
		if info.ID == id {
			return info, true
		}
	}
	return manager.Info{}, false
}

// Config returns the current module list in priority order.
func (r *Router) Config() []Config {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Config(nil), r.config...)
}

// SetConfig replaces the module list. Unknown IDs are dropped; installed
// modules missing from cfg are appended with their defaults so nothing
// disappears. Before the first Refresh cfg is kept as is. It does not
// re-initialise anything; call Initialize afterwards.
func (r *Router) SetConfig(cfg []Config) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = r.normalizeLocked(cfg)
}

func (r *Router) normalizeLocked(cfg []Config) []Config {
	if r.registry == nil {
		return append([]Config(nil), cfg...)
	}
	seen := make(map[string]bool, len(r.registry))
	out := make([]Config, 0, len(r.registry))
	for _, c := range cfg {
		desc, ok := r.infoLocked(c.ID)
		if !ok || seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		// Never claim a kind the module cannot provide.
		c.Eye = c.Eye && desc.Supported.Eye
		c.Expression = c.Expression && desc.Supported.Expression
		out = append(out, c)
	}
	for _, desc := range r.registry {
		if !seen[desc.ID] {
			out = append(out, defaultConfig(desc))
		}
	}
	return out
}

// Catalog returns every installed module in priority order with its
// setting and runtime state.
func (r *Router) Catalog() []Registered {
	r.mu.Lock()
	defer r.mu.Unlock()

	loaded := make(map[string]*entry, len(r.entries))
	for _, e := range r.entries {
		loaded[e.info.ID] = e
	}

	out := make([]Registered, 0, len(r.config))
	for _, c := range r.config {
		desc, ok := r.infoLocked(c.ID)
		if !ok {
			continue
		}
		reg := Registered{
			ID:                 c.ID,
			Name:               desc.Metadata.Name,
			Description:        desc.Metadata.Description,
			Author:             desc.Metadata.Author,
			Version:            desc.Metadata.Version,
			PageURL:            desc.Metadata.PageURL,
			UsageInstructions:  desc.Metadata.UsageInstructions,
			Icons:              EncodeIcons(desc.Metadata.Icons),
			Driver:             desc.Driver,
			SupportsEye:        desc.Supported.Eye,
			SupportsExpression: desc.Supported.Expression,
			Enabled:            c.Enabled,
			UseEye:             c.Eye,
			UseExpression:      c.Expression,
		}
		if e, ok := loaded[c.ID]; ok {
			reg.Loaded = true
			reg.Active = e.info.Active
			reg.UsingEye = e.info.UsingEye
			reg.UsingExpression = e.info.UsingExpression
			reg.EyeActive = e.info.EyeActive
			reg.ExpressionActive = e.info.ExpressionActive
		}
		out = append(out, reg)
	}
	return out
}
