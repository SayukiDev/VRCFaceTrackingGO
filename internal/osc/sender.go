package osc

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"go.uber.org/zap"

	"VRCFaceTrackingGo/internal/params"
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

// bundleHeader is "#bundle\0" followed by an 8 byte timetag.
const bundleHeader = 16

// Send packs messages into as many bundles as needed to stay under
// maxDatagram bytes and sends them. Each message is encoded once with
// go-osc and framed directly, rather than encoding it again through
// osc.Bundle.
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

	buf := make([]byte, 0, maxDatagram)
	newBundle := func() {
		buf = append(buf[:0], "#bundle\x00"...)
		buf = binary.BigEndian.AppendUint64(buf, osc.NewTimetag(time.Now()).TimeTag())
	}
	newBundle()
	inBundle, sent := 0, 0
	flush := func() error {
		if inBundle == 0 {
			return nil
		}
		if _, err := conn.Write(buf); err != nil {
			return fmt.Errorf("send bundle: %w", err)
		}
		sent += inBundle
		inBundle = 0
		newBundle()
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
		if bundleHeader+elementSize > maxDatagram {
			s.logger.Error("OSC message too large to send, skipping", zap.String("address", m.Address))
			continue
		}
		if len(buf)+elementSize > maxDatagram {
			if err := flush(); err != nil {
				return err
			}
		}
		buf = binary.BigEndian.AppendUint32(buf, uint32(len(encoded)))
		buf = append(buf, encoded...)
		inBundle++
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
