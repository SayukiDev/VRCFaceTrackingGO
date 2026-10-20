package osc

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/hypebeast/go-osc/osc"
	"go.uber.org/zap"
)

// Receiver listens for OSC messages from VRChat on a loopback port.
type Receiver struct {
	logger  *zap.Logger
	handler func(*osc.Message)

	mu   sync.Mutex
	conn *net.UDPConn
	done chan struct{}
}

func NewReceiver(logger *zap.Logger, handler func(*osc.Message)) *Receiver {
	return &Receiver{logger: logger.Named("OscRecv"), handler: handler}
}

// Listen binds to address:port (port 0 picks a free port) and starts the
// receive loop. It returns the port actually bound.
func (r *Receiver) Listen(address string, port int) (int, error) {
	ip := net.ParseIP(address)
	if ip == nil || !ip.IsLoopback() {
		return 0, fmt.Errorf("osc: cannot bind to non-loopback address %q", address)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.closeLocked()

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: ip, Port: port})
	if err != nil {
		return 0, fmt.Errorf("bind receiver endpoint %s:%d: %w", address, port, err)
	}
	r.conn = conn
	r.done = make(chan struct{})
	go r.loop(conn, r.done)

	actual := conn.LocalAddr().(*net.UDPAddr).Port
	r.logger.Info("Listening for OSC", zap.String("endpoint", conn.LocalAddr().String()))
	return actual, nil
}

// Port returns the bound port, or 0 when not listening.
func (r *Receiver) Port() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.conn == nil {
		return 0
	}
	return r.conn.LocalAddr().(*net.UDPAddr).Port
}

func (r *Receiver) loop(conn *net.UDPConn, done chan struct{}) {
	defer close(done)
	buf := make([]byte, maxDatagram)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			r.logger.Error("Error in OSC receive loop", zap.Error(err))
			continue
		}
		packet, err := osc.ParsePacket(string(buf[:n]))
		if err != nil {
			r.logger.Debug("Ignoring malformed OSC packet", zap.Error(err))
			continue
		}
		r.dispatch(packet)
	}
}

func (r *Receiver) dispatch(packet osc.Packet) {
	switch p := packet.(type) {
	case *osc.Message:
		r.handler(p)
	case *osc.Bundle:
		for _, m := range p.Messages {
			r.handler(m)
		}
		for _, b := range p.Bundles {
			r.dispatch(b)
		}
	}
}

func (r *Receiver) closeLocked() {
	if r.conn == nil {
		return
	}
	r.conn.Close()
	<-r.done
	r.conn = nil
}

func (r *Receiver) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closeLocked()
	return nil
}
