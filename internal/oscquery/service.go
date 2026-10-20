package oscquery

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"sync"

	goosc "github.com/hypebeast/go-osc/osc"
	"go.uber.org/zap"

	"VRCFaceTracking/internal/osc"
)

// Service ties discovery, the HTTP server, the OSC receiver and the avatar
// parsers together. It owns the "which avatar is loaded" state.
type Service struct {
	logger    *zap.Logger
	discovery *Discovery
	server    *Server
	receiver  *osc.Receiver
	files     *FileParser
	query     *QueryParser

	mu           sync.Mutex
	target       osc.Target
	avatar       AvatarInfo
	lastAvatarID string
	listenPort   int

	// OnAvatarChanged fires when a new avatar definition was resolved.
	OnAvatarChanged func(AvatarInfo)
	// OnForceRelevant fires for the /vrcft/settings/forceRelevant endpoint.
	OnForceRelevant func(bool)
	// OnMessageReceived fires for every incoming OSC message.
	OnMessageReceived func()
}

func NewService(logger *zap.Logger, oscDir string) *Service {
	files := NewFileParser(logger, oscDir)
	s := &Service{
		logger:    logger.Named("OscQueryService"),
		discovery: NewDiscovery(logger),
		server:    NewServer(logger),
		files:     files,
		query:     NewQueryParser(logger, files),
		avatar:    NullAvatar("Loading...", "Loading..."),
	}
	s.receiver = osc.NewReceiver(logger, s.handleMessage)
	s.discovery.OnClientDiscovered = s.clientDiscovered
	s.server.OnHostInfoQueried = func() { s.resolve("") }
	return s
}

// Start binds the legacy receive port and begins mDNS discovery.
func (s *Service) Start(ctx context.Context, target osc.Target) error {
	s.logger.Debug("OSC Service Initializing")
	s.mu.Lock()
	s.target = target
	s.mu.Unlock()

	if port, err := s.receiver.Listen(target.Address, target.InPort); err != nil {
		s.logger.Warn("Could not bind to recv endpoint", zap.Error(err))
	} else {
		s.setListenPort(port)
	}
	if err := s.discovery.Start(ctx); err != nil {
		return err
	}
	return nil
}

// SetTarget updates the endpoint. The legacy receive port is only rebound
// while OSCQuery has not taken over port negotiation.
func (s *Service) SetTarget(target osc.Target) {
	s.mu.Lock()
	s.target = target
	s.mu.Unlock()
	if s.discovery.Endpoint() != nil {
		return
	}
	if port, err := s.receiver.Listen(target.Address, target.InPort); err != nil {
		s.logger.Warn("Could not bind to recv endpoint", zap.Error(err))
	} else {
		s.setListenPort(port)
	}
}

func (s *Service) setListenPort(port int) {
	s.mu.Lock()
	s.listenPort = port
	s.mu.Unlock()
}

// ListenPort is the port we currently receive OSC on.
func (s *Service) ListenPort() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listenPort
}

// Avatar returns the last resolved avatar.
func (s *Service) Avatar() AvatarInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.avatar
}

// OSCQueryActive reports whether a VRChat client was discovered.
func (s *Service) OSCQueryActive() bool { return s.discovery.Endpoint() != nil }

func randomSuffix(n int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}

func (s *Service) clientDiscovered(endpoint *net.TCPAddr) {
	s.logger.Info("OSCQuery detected. Setting port negotiation to autopilot.", zap.String("endpoint", endpoint.String()))

	s.mu.Lock()
	target := s.target
	s.mu.Unlock()

	port, err := s.receiver.Listen(target.Address, 0)
	if err != nil {
		s.logger.Error("Very strange. We were unable to bind to a random port.", zap.Error(err))
		port = target.InPort
	}
	s.setListenPort(port)

	instance := "VRCFT-" + randomSuffix(6)
	s.server.SetAppName(instance)
	httpPort, err := s.server.Bind(target.Address, port)
	if err != nil {
		s.logger.Error("Failed to start OSCQuery HTTP server", zap.Error(err))
		return
	}
	if err := s.discovery.Advertise(instance, httpPort, port); err != nil {
		s.logger.Error("Failed to advertise OSCQuery services", zap.Error(err))
	}
	s.resolve("")
}

// resolve re-reads the avatar definition, through OSCQuery when available
// and the legacy config file otherwise.
func (s *Service) resolve(avatarID string) {
	var info AvatarInfo
	var err error
	if ep := s.discovery.Endpoint(); ep != nil {
		info, err = s.query.Parse(ep)
	} else {
		if avatarID == "" {
			s.mu.Lock()
			avatarID = s.lastAvatarID
			s.mu.Unlock()
		}
		info, err = s.files.Parse(avatarID)
	}
	if err != nil {
		s.logger.Debug("Avatar not resolved", zap.Error(err))
		return
	}

	s.mu.Lock()
	s.avatar = info
	s.lastAvatarID = info.ID
	fn := s.OnAvatarChanged
	s.mu.Unlock()
	if fn != nil {
		fn(info)
	}
}

func (s *Service) handleMessage(msg *goosc.Message) {
	if fn := s.OnMessageReceived; fn != nil {
		fn()
	}
	switch msg.Address {
	case "/avatar/change":
		if len(msg.Arguments) > 0 {
			if id, ok := msg.Arguments[0].(string); ok {
				s.mu.Lock()
				s.lastAvatarID = id
				s.mu.Unlock()
				go s.resolve(id)
			}
		}
	case "/vrcft/settings/forceRelevant":
		if len(msg.Arguments) > 0 {
			if v, ok := msg.Arguments[0].(bool); ok && s.OnForceRelevant != nil {
				s.OnForceRelevant(v)
			}
		}
	}
}

func (s *Service) Close() error {
	var errs []error
	for _, c := range []interface{ Close() error }{s.discovery, s.server, s.receiver} {
		if err := c.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("oscquery close: %v", errs)
	}
	return nil
}
