package service

import (
	"fmt"

	"go.uber.org/zap"

	"VRCFaceTracking/internal/engine"
	"VRCFaceTracking/internal/oscquery"
	"VRCFaceTracking/module/manager"
)

const modulesKey = "Modules"

// Modules returns the loaded modules.
func (s *Service) Modules() []manager.ModuleInfo { return s.manager.Modules() }

// SetModuleActive pauses or resumes a loaded module.
func (s *Service) SetModuleActive(id string, active bool) { s.manager.SetActive(id, active) }

// SetModuleTracking switches a loaded module's eye and expression output
// independently.
func (s *Service) SetModuleTracking(id string, eye, expression bool) {
	s.manager.SetTracking(id, eye, expression)
}

// ModuleCatalog returns every registered module with its setting and state.
func (s *Service) ModuleCatalog() []manager.Registered { return s.manager.Catalog() }

// savedModuleConfig is Config with optional kind flags so lists saved by
// older versions (which had no per-kind setting) still mean "everything".
type savedModuleConfig struct {
	ID         string `json:"id"`
	Enabled    bool   `json:"enabled"`
	Eye        *bool  `json:"eye"`
	Expression *bool  `json:"expression"`
}

// loadModuleConfig applies the persisted module list to the manager.
func (s *Service) loadModuleConfig() {
	var saved []savedModuleConfig
	found, err := s.store.Get(modulesKey, &saved)
	if err != nil {
		s.logger.Warn("Ignoring unreadable module list", zap.Error(err))
		return
	}
	if !found {
		return
	}
	cfg := make([]manager.Config, 0, len(saved))
	for _, c := range saved {
		cfg = append(cfg, manager.Config{
			ID:         c.ID,
			Enabled:    c.Enabled,
			Eye:        c.Eye == nil || *c.Eye,
			Expression: c.Expression == nil || *c.Expression,
		})
	}
	s.manager.SetConfig(cfg)
}

// SetModuleConfig replaces the priority list / enabled flags and persists
// it. The running modules are left alone until ReloadModules is called, so
// the user can make several changes before the (slow) re-initialisation.
func (s *Service) SetModuleConfig(cfg []manager.Config) error {
	s.manager.SetConfig(cfg)
	if err := s.store.Set(modulesKey, s.manager.Config()); err != nil {
		return err
	}
	s.moduleConfigDirty.Store(true)
	s.logger.Info("Module configuration changed; reload modules to apply", zap.Any("modules", s.manager.Config()))
	return nil
}

// ModuleConfigDirty reports whether the module list changed since the
// modules were last (re)loaded.
func (s *Service) ModuleConfigDirty() bool { return s.moduleConfigDirty.Load() }

// ReloadModules re-initialises the modules with the current list in the
// background. Listener.OnModulesChanged fires when it is done.
func (s *Service) ReloadModules() {
	s.moduleConfigDirty.Store(false)
	s.logger.Info("Reloading modules...")
	go s.manager.Initialize(s.ctx, s.root)
}

// updateModuleConfig applies fn to the entry with the given id and saves.
func (s *Service) updateModuleConfig(id string, fn func(*manager.Config)) error {
	cfg := s.manager.Config()
	for i := range cfg {
		if cfg[i].ID == id {
			fn(&cfg[i])
			return s.SetModuleConfig(cfg)
		}
	}
	return fmt.Errorf("unknown module %q", id)
}

// SetModuleEnabled toggles one module in the list.
func (s *Service) SetModuleEnabled(id string, enabled bool) error {
	return s.updateModuleConfig(id, func(c *manager.Config) { c.Enabled = enabled })
}

// SetModuleUse chooses which tracking kinds a module may claim.
func (s *Service) SetModuleUse(id string, eye, expression bool) error {
	return s.updateModuleConfig(id, func(c *manager.Config) {
		c.Eye = eye
		c.Expression = expression
	})
}

// SetModuleOrder reorders the list; every registered module must be named.
func (s *Service) SetModuleOrder(order []string) error {
	current := s.manager.Config()
	byID := make(map[string]manager.Config, len(current))
	for _, c := range current {
		byID[c.ID] = c
	}
	cfg := make([]manager.Config, 0, len(current))
	for _, id := range order {
		c, ok := byID[id]
		if !ok {
			return fmt.Errorf("unknown module %q", id)
		}
		cfg = append(cfg, c)
	}
	return s.SetModuleConfig(cfg)
}

// Avatar returns the last resolved avatar.
func (s *Service) Avatar() oscquery.AvatarInfo { return s.query.Avatar() }

// ParameterStats summarises the relevant parameter set.
func (s *Service) ParameterStats() engine.ParameterStats { return s.engine.Stats() }
