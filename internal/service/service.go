// Package service wires the tracking pipeline together (modules, mutator,
// parameter engine, OSC and OSCQuery) and exposes the operations the UI
// layer needs, independent of any GUI framework.
package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"VRCFaceTracking/internal/engine"
	"VRCFaceTracking/internal/logging"
	"VRCFaceTracking/internal/mutation"
	"VRCFaceTracking/internal/osc"
	"VRCFaceTracking/internal/oscquery"
	"VRCFaceTracking/internal/settings"
	"VRCFaceTracking/internal/vrchat"
	"VRCFaceTracking/module/manager"
	"VRCFaceTracking/pkg/tracking"
)

const (
	oscTargetKey  = "OSCTarget"
	statsInterval = time.Second
)

// Listener receives state changes. Every method may be called from any
// goroutine; implementations must be cheap and non-blocking.
type Listener interface {
	OnLog(line logging.Line)
	OnModulesChanged(modules []manager.ModuleInfo)
	OnAvatarChanged(avatar oscquery.AvatarInfo)
	OnParametersChanged(stats engine.ParameterStats)
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

	store   *settings.Store
	manager *manager.Manager
	mutator *mutation.Mutator
	sender  *osc.Sender
	engine  *engine.Engine
	query   *oscquery.Service

	targetMu    sync.RWMutex
	target      osc.Target
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

	target := settings.GetOr(store, oscTargetKey, osc.DefaultTarget())
	if err := target.Validate(); err != nil {
		s.logger.Warn("Invalid OSC target in settings, reverting to default", zap.Error(err))
		target = osc.DefaultTarget()
	}
	s.setTarget(target)

	s.manager = manager.New(s.root)
	s.manager.OnChange = listener.OnModulesChanged
	s.loadModuleConfig()

	s.mutator = mutation.NewMutator(s.root, store)
	s.sender = osc.NewSender(s.root)
	s.sender.OnDispatched = func(n int) { s.messagesOut.Add(int64(n)) }

	s.engine = engine.New(s.root, s.manager, s.mutator, s.sender)
	s.engine.OnParametersChanged = listener.OnParametersChanged
	s.loadTiming()

	s.query = oscquery.NewService(s.root, vrchat.OSCDirectory())
	s.query.OnAvatarChanged = func(info oscquery.AvatarInfo) {
		s.engine.SetDefinitions(info.Parameters)
		listener.OnAvatarChanged(info)
	}
	s.query.OnForceRelevant = s.engine.SetAllRelevant
	s.query.OnMessageReceived = func() { s.messagesIn.Add(1) }

	s.initialize()
	go s.engine.Run(s.ctx)
	go s.statsLoop()
	return nil
}

// initialize is the restartable part of Start: VRChat checks, network
// endpoints, mutations and modules.
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
	if err := s.sender.SetTarget(target.Address, target.OutPort); err != nil {
		s.logger.Warn("Failed to bind to sender endpoint", zap.Error(err))
	}
	if err := s.query.Start(s.ctx, target); err != nil {
		s.logger.Error("OSCQuery service failed to start", zap.Error(err))
	}

	s.mutator.Load(tracking.NewTrackingData())
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
	if s.mutator != nil {
		if err := s.mutator.Save(); err != nil {
			s.logger.Error("Failed to save mutations", zap.Error(err))
		}
	}
	if s.manager != nil {
		s.manager.Teardown()
	}
	if s.query != nil {
		_ = s.query.Close()
	}
	if s.sender != nil {
		_ = s.sender.Close()
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
