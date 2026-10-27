// Package service wires the tracking pipeline together (module manager,
// router, mixer, sender, OSC and OSCQuery) and exposes the operations the UI
// layer needs, independent of any GUI framework.
package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/internal/logging"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/mixer"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/module/manager"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/osc"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/oscquery"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/router"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/sender"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/settings"
)

const statsInterval = time.Second

// Listener receives state changes. Every method may be called from any
// goroutine; implementations must be cheap and non-blocking.
type Listener interface {
	OnLog(line logging.Line)
	OnModulesChanged(modules []router.ModuleInfo)
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
	mixer     *mixer.Mixer
	modules   *manager.Manager
	router    *router.Router
	transport *osc.Sender
	sender    *sender.Sender
	query     *oscquery.Service

	// relevanceMu guards forceRelevant.
	relevanceMu sync.Mutex
	// forceRelevant is the user's (or /vrcft/settings/forceRelevant) choice.
	forceRelevant bool

	messagesIn  atomic.Int64
	messagesOut atomic.Int64

	moduleConfigDirty atomic.Bool
	// loading tracks the background module loads started by loadModules.
	loading sync.WaitGroup

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

	s.mixer = mixer.New(cfg.Mixer)
	s.modules = manager.New(s.root, manager.DefaultDir(s.dataDir))
	s.router = router.New(s.root, s.mixer, s.modules)
	s.router.OnChange = listener.OnModulesChanged
	if cfg.Modules != nil {
		s.router.SetConfig(cfg.Modules)
	}
	s.applyTiming(cfg.Timing)

	s.transport = osc.NewSender(s.root)
	s.transport.OnDispatched = func(n int) { s.messagesOut.Add(int64(n)) }

	s.sender = sender.New(s.root, s.router, s.transport)
	s.sender.OnParametersChanged = listener.OnParametersChanged
	s.sender.SetOutputMode(cfg.Output)

	s.query = oscquery.NewService(s.root)
	s.query.OnAvatarChanged = func(info oscquery.AvatarInfo) {
		s.sender.SetDefinitions(info.Parameters)
		listener.OnAvatarChanged(info)
	}
	s.query.OnForceRelevant = s.SetAllParametersRelevant
	s.query.OnMessageReceived = func() { s.messagesIn.Add(1) }

	s.initialize()
	go s.router.Run(s.ctx, s.sender)
	go s.statsLoop()
	return nil
}

// initialize is the restartable part of Start: network endpoints and
// modules.
func (s *Service) initialize() {
	target := s.Target()
	if err := s.transport.SetTarget(target.Address, target.OutPort); err != nil {
		s.logger.Warn("Failed to bind to sender endpoint", zap.Error(err))
	}
	s.startQuery()
	s.loadModules()
}

// Reinitialize tears down the modules and network endpoints and brings
// them back up. Most modules will not enjoy this.
func (s *Service) Reinitialize() {
	s.logger.Info("Reinitializing VRCFT...")
	s.router.Teardown()
	_ = s.query.Close()
	s.initialize()
}

// Stop persists state and shuts every component down.
func (s *Service) Stop() {
	s.logger.Info("VRCFT Standalone Exiting!")
	if s.cancel != nil {
		s.cancel()
	}
	// A module that finishes starting after the teardown would be left
	// running, so let the cancelled loads return first.
	s.loading.Wait()
	if s.router != nil {
		s.router.Teardown()
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
