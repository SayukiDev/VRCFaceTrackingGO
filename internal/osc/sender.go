package osc

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"go.uber.org/zap"

	"VRCFaceTracking/internal/params"
)

// maxDatagram is the largest bundle we send in one datagram.
const maxDatagram = 4096

// Sender encodes parameter messages into OSC bundles and sends them to the
// configured endpoint.
type Sender struct {
	logger *zap.Logger

	mu   sync.Mutex
	conn *net.UDPConn

	// OnDispatched is called with the number of messages sent.
	OnDispatched func(count int)
}

func NewSender(logger *zap.Logger) *Sender {
	return &Sender{logger: logger.Named("OscSend")}
}

// SetTarget (re)connects the sender to address:port.
func (s *Sender) SetTarget(address string, port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		return fmt.Errorf("resolve sender endpoint: %w", err)
	}
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return fmt.Errorf("connect sender endpoint %s: %w", addr, err)
	}
	s.conn = conn
	return nil
}

// Connected reports whether a target has been set.
func (s *Sender) Connected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn != nil
}

// Send packs messages into as many bundles as needed to stay under
// maxDatagram bytes and sends them.
func (s *Sender) Send(messages []params.Message) error {
	if len(messages) == 0 {
		return nil
	}
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return nil
	}

	const bundleHeader = 16 // "#bundle\0" + timetag
	bundle := osc.NewBundle(time.Now())
	size := bundleHeader
	sent := 0

	flush := func() error {
		if len(bundle.Messages) == 0 {
			return nil
		}
		data, err := bundle.MarshalBinary()
		if err != nil {
			return fmt.Errorf("marshal bundle: %w", err)
		}
		if _, err := conn.Write(data); err != nil {
			return fmt.Errorf("send bundle: %w", err)
		}
		sent += len(bundle.Messages)
		bundle = osc.NewBundle(time.Now())
		size = bundleHeader
		return nil
	}

	for _, m := range messages {
		msg, err := toOSC(m)
		if err != nil {
			s.logger.Warn("Skipping message", zap.Error(err))
			continue
		}
		encoded, err := msg.MarshalBinary()
		if err != nil {
			s.logger.Warn("Skipping message", zap.String("address", m.Address), zap.Error(err))
			continue
		}
		elementSize := 4 + len(encoded)
		if size+elementSize > maxDatagram {
			if err := flush(); err != nil {
				return err
			}
		}
		if elementSize+bundleHeader > maxDatagram {
			s.logger.Error("OSC message too large to send, skipping", zap.String("address", m.Address))
			continue
		}
		_ = bundle.Append(msg)
		size += elementSize
	}
	if err := flush(); err != nil {
		return err
	}
	if s.OnDispatched != nil && sent > 0 {
		s.OnDispatched(sent)
	}
	return nil
}

func (s *Sender) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return nil
	}
	err := s.conn.Close()
	s.conn = nil
	return err
}
