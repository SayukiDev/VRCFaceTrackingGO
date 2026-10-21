package settings

import (
	"fmt"
	"regexp"
	"time"

	"github.com/SayukiDev/VRCFaceTrackingGO/internal/mixer"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/module/manager"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/osc"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/router"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/sender"
)

// Settings is every persisted application setting.
type Settings struct {
	// Theme is the UI colour scheme: "light", "dark" or "system".
	Theme string `json:"theme"`
	// SeedColor is the "#RRGGBB" source colour the Material 3 colour scheme
	// is generated from.
	SeedColor string `json:"seedColor"`
	// UseSystemAccent makes the Windows accent colour the seed instead of
	// SeedColor.
	UseSystemAccent bool `json:"useSystemAccent"`
	// Locale is the UI language, or "" to follow the system.
	Locale string     `json:"locale"`
	OSC    osc.Target `json:"osc"`
	// OSCMode is how VRChat and the loaded avatar are found.
	OSCMode osc.Mode          `json:"oscMode"`
	Output  sender.OutputMode `json:"output"`
	Timing  Timing            `json:"timing"`
	// Modules is the module priority list. Nil means the router's default
	// (every installed module, in file name order).
	Modules []router.Config `json:"modules"`
	Mixer   mixer.Config    `json:"mixer"`
}

// Default returns the settings used for anything missing from the file.
func Default() Settings {
	return Settings{
		Theme:     "dark",
		SeedColor: DefaultSeedColor,
		OSC:       osc.DefaultTarget(),
		OSCMode:   osc.ModeOSCQuery,
		Output:    sender.OutputParameters,
		Timing:    DefaultTiming(),
		Mixer:     mixer.DefaultConfig(),
	}
}

// DefaultSeedColor is the pink the UI used before the scheme became configurable.
const DefaultSeedColor = "#EC4899"

var seedColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// ValidateSeedColor checks c is a "#RRGGBB" colour.
func ValidateSeedColor(c string) error {
	if !seedColorPattern.MatchString(c) {
		return fmt.Errorf("seed colour must be #RRGGBB, got %q", c)
	}
	return nil
}

const (
	minUpdateIntervalMs = 1
	maxUpdateIntervalMs = 1000
	minInitTimeoutSec   = 1
	maxInitTimeoutSec   = 300
	minSendIntervalMs   = 1
	maxSendIntervalMs   = 1000
)

// Timing holds the user adjustable scheduling values.
type Timing struct {
	// UpdateIntervalMs is the time between two Update calls of one module.
	UpdateIntervalMs int `json:"updateIntervalMs"`
	// InitTimeoutSec is the maximum time one module may spend in Initialize.
	InitTimeoutSec int `json:"initTimeoutSec"`
	// SendIntervalMs is the time between two OSC send passes.
	SendIntervalMs int `json:"sendIntervalMs"`
}

// DefaultTiming mirrors the module manager and router defaults.
func DefaultTiming() Timing {
	return Timing{
		UpdateIntervalMs: int(manager.DefaultUpdateInterval / time.Millisecond),
		InitTimeoutSec:   int(manager.DefaultInitTimeout / time.Second),
		SendIntervalMs:   int(router.DefaultSendInterval / time.Millisecond),
	}
}

// Validate checks the values are within a sane range.
func (t Timing) Validate() error {
	if t.UpdateIntervalMs < minUpdateIntervalMs || t.UpdateIntervalMs > maxUpdateIntervalMs {
		return fmt.Errorf("update interval must be between %d and %d ms", minUpdateIntervalMs, maxUpdateIntervalMs)
	}
	if t.InitTimeoutSec < minInitTimeoutSec || t.InitTimeoutSec > maxInitTimeoutSec {
		return fmt.Errorf("init timeout must be between %d and %d s", minInitTimeoutSec, maxInitTimeoutSec)
	}
	if t.SendIntervalMs < minSendIntervalMs || t.SendIntervalMs > maxSendIntervalMs {
		return fmt.Errorf("send interval must be between %d and %d ms", minSendIntervalMs, maxSendIntervalMs)
	}
	return nil
}

func (t Timing) UpdateInterval() time.Duration {
	return time.Duration(t.UpdateIntervalMs) * time.Millisecond
}

func (t Timing) InitTimeout() time.Duration { return time.Duration(t.InitTimeoutSec) * time.Second }

func (t Timing) SendInterval() time.Duration {
	return time.Duration(t.SendIntervalMs) * time.Millisecond
}

// sanitize replaces every invalid section with its default and returns
// one error per replaced section.
func (s *Settings) sanitize() []error {
	def := Default()
	var errs []error
	if err := s.OSC.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("osc: %w", err))
		s.OSC = def.OSC
	}
	if !s.Output.Valid() {
		errs = append(errs, fmt.Errorf("output: unknown mode %q", s.Output))
		s.Output = def.Output
	}
	if err := s.Timing.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("timing: %w", err))
		s.Timing = def.Timing
	}
	return errs
}
