package service

import (
	"VRCFaceTracking/internal/mixer"
	"VRCFaceTracking/internal/settings"
)

// MixerEnabled reports whether the mixer stages are applied at all.
func (s *Service) MixerEnabled() bool { return s.mixer.Enabled() }

// SetMixerEnabled toggles the whole mixer and persists it.
func (s *Service) SetMixerEnabled(v bool) {
	s.mixer.SetEnabled(v)
	s.saveMixer()
}

// MixerStages returns the ordered stage list.
func (s *Service) MixerStages() []mixer.StageState { return s.mixer.Stages() }

// SetStageActive toggles one stage and persists it.
func (s *Service) SetStageActive(name string, active bool) error {
	if err := s.mixer.SetActive(name, active); err != nil {
		return err
	}
	s.saveMixer()
	return nil
}

// SetStageValue updates a checkbox, slider or range component and persists
// it. value must be a bool, a number or a mixer.Range.
func (s *Service) SetStageValue(stage, component string, value any) error {
	if err := s.mixer.SetValue(stage, component, value); err != nil {
		return err
	}
	s.saveMixer()
	return nil
}

// InvokeStageButton runs a button component's action and persists the
// result.
func (s *Service) InvokeStageButton(stage, component string) error {
	if err := s.mixer.Invoke(stage, component); err != nil {
		return err
	}
	s.saveMixer()
	return nil
}

func (s *Service) saveMixer() {
	cfg := s.mixer.Config()
	s.store.Update(func(c *settings.Settings) { c.Mixer = cfg })
}
