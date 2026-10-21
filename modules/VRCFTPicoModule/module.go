// Command VRCFTPicoModule is the tracking module program for PICO 4 Pro / Enterprise
// headsets streaming face data over UDP through PICO Connect or the
// Streaming Assistant.
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"

	"VRCFaceTrackingGo/internal/module/adapter"
	"VRCFaceTrackingGo/pkg/tracking"
)

// files holds metadata.json and the icons it names.
//
//go:embed metadata.json pico.png
var files embed.FS

var metadata = adapter.MustLoadMetadata(files, "metadata.json")

const (
	portCurrent = 29765
	portLegacy  = 29763

	// discoveryTimeout bounds how long Initialize waits for the first
	// packet; the original module waited forever and froze the host.
	discoveryTimeout = 5 * time.Second
	readTimeout      = 100 * time.Millisecond
	// timeoutWarnEvery is the number of consecutive read timeouts before
	// a warning is logged (600 * 100ms = 60s).
	timeoutWarnEvery = 600

	maxDatagram = 4096
)

// Module receives PICO face tracking over UDP.
type Module struct {
	logger *zap.Logger

	mu       sync.Mutex
	conn     *net.UDPConn
	legacy   bool
	granted  adapter.Support
	timeouts int
	mapper   mapper
	buf      []byte
}

var _ adapter.Module = (*Module)(nil)

// New creates a module.
func New() *Module {
	return &Module{buf: make([]byte, maxDatagram)}
}

func (m *Module) ID() string { return "VRCFTPicoModule" }

func (m *Module) Metadata() adapter.Metadata {
	m.mu.Lock()
	defer m.mu.Unlock()
	md := metadata
	md.Name = m.displayName()
	return md
}

// displayName appends the granted tracking and protocol to the base name.
func (m *Module) displayName() string {
	name := metadata.Name
	switch {
	case m.granted.Eye && m.granted.Expression:
		name += " / Full FaceTracking"
	case m.granted.Eye:
		name += " / Eye Tracking"
	case m.granted.Expression:
		name += " / Expression Tracking"
	}
	if m.legacy {
		name += " [Legacy Protocol]"
	}
	return name
}

func (m *Module) Supported() adapter.Support {
	return adapter.Support{Eye: true, Expression: true}
}

func (m *Module) Initialize(ctx context.Context, logger *zap.Logger, avail adapter.Support) (adapter.Support, error) {
	m.logger = logger.Named("PICO")
	m.logger.Info("Starting Initialization...")

	conn, legacy, err := m.discover(ctx)
	if err != nil {
		return adapter.Support{}, err
	}
	want := adapter.Support{Eye: true, Expression: true}
	m.mu.Lock()
	m.conn, m.legacy, m.granted = conn, legacy, want
	m.timeouts = 0
	m.mapper = mapper{}
	m.mu.Unlock()

	m.logger.Info("Using port", zap.Int("port", conn.LocalAddr().(*net.UDPAddr).Port), zap.Bool("legacy", legacy))
	return want, nil
}

type discovered struct {
	conn   *net.UDPConn
	legacy bool
}

// discover listens on both protocol ports and returns the first one that
// receives data. It gives up after discoveryTimeout or when ctx ends.
func (m *Module) discover(ctx context.Context) (*net.UDPConn, bool, error) {
	ports := []struct {
		port   int
		legacy bool
	}{{portCurrent, false}, {portLegacy, true}}
	m.logger.Debug("Initializing UDP clients", zap.Ints("ports", []int{portCurrent, portLegacy}))

	ctx, cancel := context.WithTimeout(ctx, discoveryTimeout)
	defer cancel()

	results := make(chan discovered, len(ports))
	var conns []*net.UDPConn
	for _, p := range ports {
		conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: p.port})
		if err != nil {
			m.logger.Warn("Could not listen on port", zap.Int("port", p.port), zap.Error(err))
			continue
		}
		conns = append(conns, conn)
		go func(conn *net.UDPConn, legacy bool) {
			buf := make([]byte, maxDatagram)
			// The deadline makes the goroutine exit once ctx is done.
			_ = conn.SetReadDeadline(deadlineOf(ctx))
			if _, _, err := conn.ReadFromUDP(buf); err == nil {
				results <- discovered{conn: conn, legacy: legacy}
			}
		}(conn, p.legacy)
	}
	if len(conns) == 0 {
		return nil, false, errors.New("VRCFTPicoModule: could not listen on any port")
	}

	var winner *discovered
	select {
	case d := <-results:
		winner = &d
	case <-ctx.Done():
	}

	for _, c := range conns {
		if winner != nil && c == winner.conn {
			continue
		}
		c.Close()
	}
	if winner == nil {
		return nil, false, fmt.Errorf("VRCFTPicoModule: no data received on ports %d/%d within %s (is PICO Connect streaming face data?)", portCurrent, portLegacy, discoveryTimeout)
	}
	_ = winner.conn.SetReadDeadline(time.Time{})
	return winner.conn, winner.legacy, nil
}

func deadlineOf(ctx context.Context) time.Time {
	if d, ok := ctx.Deadline(); ok {
		return d
	}
	return time.Now().Add(discoveryTimeout)
}

// Update reads one datagram with a short deadline so a silent headset never
// blocks the caller.
func (m *Module) Update(ctx context.Context, data *tracking.TrackingData) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn == nil {
		return errors.New("VRCFTPicoModule: module not initialised")
	}

	deadline := time.Now().Add(readTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = m.conn.SetReadDeadline(deadline)

	n, _, err := m.conn.ReadFromUDP(m.buf)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			m.timeouts++
			if m.timeouts >= timeoutWarnEvery {
				m.logger.Warn("Receive data timed out")
				m.timeouts = 0
			}
			return nil
		}
		if errors.Is(err, net.ErrClosed) {
			return nil
		}
		return fmt.Errorf("VRCFTPicoModule: receive: %w", err)
	}
	m.timeouts = 0

	w, err := parsePacket(m.buf[:n], m.legacy)
	if err != nil {
		if errors.Is(err, errNotFaceData) {
			return nil
		}
		return err
	}
	if m.granted.Eye {
		updateEye(data, w)
	}
	if m.granted.Expression {
		m.mapper.updateExpression(data, w)
	}
	return nil
}

func (m *Module) Teardown() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn == nil {
		return nil
	}
	err := m.conn.Close()
	m.conn = nil
	m.granted = adapter.Support{}
	return err
}
