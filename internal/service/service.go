// Package service wires the tracking pipeline together (modules, mixer,
// sender, router, OSC and OSCQuery) and exposes the operations the UI
// layer needs, independent of any GUI framework.
package service

import (
	"context"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"VRCFaceTracking/internal/logging"
	"VRCFaceTracking/internal/mixer"
	"VRCFaceTracking/internal/osc"
	"VRCFaceTracking/internal/oscquery"
	"VRCFaceTracking/internal/router"
	"VRCFaceTracking/internal/sender"
	"VRCFaceTracking/internal/settings"
	"VRCFaceTracking/internal/vrchat"
	"VRCFaceTracking/module/manager"
)

const statsInterval = time.Second

// Listener receives state changes. Every method may be called from any
// goroutine; implementations must be cheap and non-blocking.
type Listener interface {
	OnLog(line logging.Line)
	OnModulesChanged(modules []manager.ModuleInfo)
	OnAvatarChanged(avatar oscquery.AvatarInfo)
	OnParametersChanged(stats sender.ParameterStats)
	OnOscStats(status OscStatus)
}

// Service owns every long-lived component of the application.
type Service struct {
	root    *zap.Logger
	logger  *zap.Logger
	logBuf  *logging.Buffer
	dataDir string

	ctx    context.Context
	cancel context.CancelFunc

	store     *settings.Store
	manager   *manager.Manager
	mixer     *mixer.Mixer
	transport *osc.Sender
	sender    *sender.Sender
	router    *router.Router
	query     *oscquery.Service

	messagesIn  atomic.Int64
	messagesOut atomic.Int64

	moduleConfigDirty atomic.Bool

	listener Listener
}

// New creates the service. Nothing runs until Start is called.
func New(logger *zap.Logger, logBuf *logging.Buffer, dataDir string) *Service {
	return &Service{
		root:    logger,
		logger:  logger.Named("Service"),
		logBuf:  logBuf,
		dataDir: dataDir,
	}
}

// Start loads settings, builds the pipeline and starts every background
// loop. The listener is notified of changes from this point on.
func (s *Service) Start(ctx context.Context, listener Listener) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.listener = listener
	s.logBuf.SetListener(listener.OnLog)

	store, err := settings.Open(s.dataDir, s.root)
	if err != nil {
		return err
	}
	s.store = store
	cfg := store.Get()

	s.manager = manager.New(s.root)
	s.manager.OnChange = listener.OnModulesChanged
	if cfg.Modules != nil {
		s.manager.SetConfig(cfg.Modules)
	}

	s.mixer = mixer.New(cfg.Mixer)
	s.transport = osc.NewSender(s.root)
	s.transport.OnDispatched = func(n int) { s.messagesOut.Add(int64(n)) }

	s.sender = sender.New(s.root, s.manager, s.transport)
	s.sender.OnParametersChanged = listener.OnParametersChanged
	s.sender.SetOutputMode(cfg.Output)

	s.router = router.New(s.manager, s.mixer, s.sender)
	s.applyTiming(cfg.Timing)

	s.query = oscquery.NewService(s.root, vrchat.OSCDirectory())
	s.query.OnAvatarChanged = func(info oscquery.AvatarInfo) {
		s.sender.SetDefinitions(info.Parameters)
		listener.OnAvatarChanged(info)
	}
	s.query.OnForceRelevant = s.sender.SetAllRelevant
	s.query.OnMessageReceived = func() { s.messagesIn.Add(1) }

	s.initialize()
	go s.router.Run(s.ctx)
	go s.statsLoop()
	return nil
}

// initialize is the restartable part of Start: VRChat checks, network
// endpoints and modules.
func (s *Service) initialize() {
	if changed, err := vrchat.ForceEnableOSC(); err != nil {
		s.logger.Warn("Could not verify VRChat OSC setting", zap.Error(err))
	} else if changed {
		s.logger.Warn("VRCFT detected OSC was disabled and automatically enabled it.")
		if vrchat.IsRunning() {
			s.logger.Error("However, VRChat was running while this change was made.\nIf parameters do not update, please restart VRChat or manually enable OSC yourself in your avatar's expressions menu.")
		}
	}

	target := s.Target()
	if err := s.transport.SetTarget(target.Address, target.OutPort); err != nil {
		s.logger.Warn("Failed to bind to sender endpoint", zap.Error(err))
	}
	if err := s.query.Start(s.ctx, target); err != nil {
		s.logger.Error("OSCQuery service failed to start", zap.Error(err))
	}

	s.moduleConfigDirty.Store(false)
	s.manager.Initialize(s.ctx, s.root)
}

// Reinitialize tears down the modules and network endpoints and brings
// them back up. Most modules will not enjoy this.
func (s *Service) Reinitialize() {
	s.logger.Info("Reinitializing VRCFT...")
	s.manager.Teardown()
	_ = s.query.Close()
	s.initialize()
}

// Stop persists state and shuts every component down.
func (s *Service) Stop() {
	s.logger.Info("VRCFT Standalone Exiting!")
	if s.cancel != nil {
		s.cancel()
	}
	if s.manager != nil {
		s.manager.Teardown()
	}
	if s.query != nil {
		_ = s.query.Close()
	}
	if s.transport != nil {
		_ = s.transport.Close()
	}
	if s.store != nil {
		_ = s.store.Flush()
	}
	s.logger.Debug("Teardown complete. Awaiting exit...")
	_ = s.root.Sync()
}

func (s *Service) statsLoop() {
	ticker := time.NewTicker(statsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.listener.OnOscStats(s.OscStatus())
		}
	}
}

// Store exposes the settings store for UI-only preferences (e.g. theme).
func (s *Service) Store() *settings.Store { return s.store }
