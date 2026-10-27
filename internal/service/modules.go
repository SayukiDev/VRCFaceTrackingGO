package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/internal/module/manager"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/oscquery"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/router"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/sender"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/settings"
)

// Modules returns the loaded modules.
func (s *Service) Modules() []router.ModuleInfo { return s.router.Modules() }

// SetModuleActive pauses or resumes a loaded module.
func (s *Service) SetModuleActive(id string, active bool) { s.router.SetActive(id, active) }

// SetModuleTracking switches a loaded module's eye and expression output
// independently.
func (s *Service) SetModuleTracking(id string, eye, expression bool) {
	s.router.SetTracking(id, eye, expression)
}

// ModuleCatalog returns every installed module with its setting and state.
func (s *Service) ModuleCatalog() []router.Registered { return s.router.Catalog() }

// SetModuleConfig replaces the priority list / enabled flags and persists
// it. The running modules are left alone until ReloadModules is called, so
// the user can make several changes before the (slow) re-initialisation.
func (s *Service) SetModuleConfig(cfg []router.Config) error {
	s.router.SetConfig(cfg)
	cfg = s.router.Config()
	s.store.Update(func(c *settings.Settings) { c.Modules = cfg })
	s.moduleConfigDirty.Store(true)
	s.logger.Info("Module configuration changed; reload modules to apply", zap.Any("modules", cfg))
	return nil
}

// ModuleConfigDirty reports whether the module list changed since the
// modules were last (re)loaded.
func (s *Service) ModuleConfigDirty() bool { return s.moduleConfigDirty.Load() }

// ReloadModules re-initialises the modules with the current list in the
// background. Listener.OnModulesChanged fires when it is done.
func (s *Service) ReloadModules() {
	s.logger.Info("Reloading modules...")
	s.loadModules()
}

// loadModules rescans the installed modules and starts the enabled ones in
// the background, so slow modules never hold up the caller (or the window
// at startup). Listener.OnModulesChanged fires when it is done.
func (s *Service) loadModules() {
	s.moduleConfigDirty.Store(false)
	s.loading.Go(func() {
		s.modules.Scan(s.ctx)
		s.router.Initialize(s.ctx)
	})
}

// TrackingAssignment returns the manual eye / expression module choice.
func (s *Service) TrackingAssignment() router.Assignment { return s.router.Assignment() }

// SetTrackingAssignment picks the modules for eye and expression tracking
// by hand (an empty ID means automatic) and re-initialises the modules in
// the background. The choice is not persisted.
func (s *Service) SetTrackingAssignment(a router.Assignment) error {
	if err := s.router.SetAssignment(a); err != nil {
		return err
	}
	s.logger.Info("Tracking assignment changed", zap.String("eye", a.Eye), zap.String("expression", a.Expression))
	s.ReloadModules()
	return nil
}

// InstallModule installs (or updates) the module program at path. A
// running module with the same ID is stopped first. The new module only
// runs after ReloadModules.
func (s *Service) InstallModule(ctx context.Context, path string) (manager.Info, error) {
	desc, err := s.modules.Inspect(ctx, path)
	if err != nil {
		return manager.Info{}, err
	}
	s.router.Unload(desc.ID)
	info, err := s.modules.Install(ctx, path)
	if err != nil {
		return manager.Info{}, err
	}
	s.modulesInstalledChanged()
	return info, nil
}

// UninstallModule stops (if running) and removes an installed module.
func (s *Service) UninstallModule(id string) error {
	s.router.Unload(id)
	if err := s.modules.Uninstall(id); err != nil {
		return err
	}
	s.modulesInstalledChanged()
	return nil
}

// modulesInstalledChanged refreshes the module list after an install or
// uninstall and asks for a reload.
func (s *Service) modulesInstalledChanged() {
	s.router.Refresh()
	cfg := s.router.Config()
	s.store.Update(func(c *settings.Settings) { c.Modules = cfg })
	s.moduleConfigDirty.Store(true)
	s.listener.OnModulesChanged(s.router.Modules())
}

// updateModuleConfig applies fn to the entry with the given id and saves.
func (s *Service) updateModuleConfig(id string, fn func(*router.Config)) error {
	cfg := s.router.Config()
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
	return s.updateModuleConfig(id, func(c *router.Config) { c.Enabled = enabled })
}

// SetModuleUse chooses which tracking kinds a module may claim.
func (s *Service) SetModuleUse(id string, eye, expression bool) error {
	return s.updateModuleConfig(id, func(c *router.Config) {
		c.Eye = eye
		c.Expression = expression
	})
}

// SetModuleOrder reorders the list; every installed module must be named.
func (s *Service) SetModuleOrder(order []string) error {
	current := s.router.Config()
	byID := make(map[string]router.Config, len(current))
	for _, c := range current {
		byID[c.ID] = c
	}
	cfg := make([]router.Config, 0, len(current))
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
func (s *Service) ParameterStats() sender.ParameterStats { return s.sender.Stats() }
