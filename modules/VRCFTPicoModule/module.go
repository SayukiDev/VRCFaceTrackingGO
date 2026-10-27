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

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
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
	// silenceWarnAfter is how long the stream may be silent before a
	// warning is logged.
	silenceWarnAfter = 60 * time.Second

	maxDatagram = 4096
)

// Module receives PICO face tracking over UDP.
type Module struct {
	logger *zap.Logger

	conn *net.UDPConn
	done chan struct{}
	// mapper is owned by the receive goroutine.
	mapper mapper

	mu      sync.Mutex
	legacy  bool
	granted adapter.Support
	latest  tracking.TrackingData
}

var _ adapter.Module = (*Module)(nil)

// New creates a module.
func New() *Module { return &Module{} }

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
	m.start(conn, legacy, want)
	m.logger.Info("Using port", zap.Int("port", conn.LocalAddr().(*net.UDPAddr).Port), zap.Bool("legacy", legacy))
	return want, nil
}

func (m *Module) start(conn *net.UDPConn, legacy bool, granted adapter.Support) {
	m.mu.Lock()
	m.legacy, m.granted = legacy, granted
	m.latest = *tracking.NewTrackingData()
	m.mu.Unlock()
	m.conn = conn
	m.mapper = mapper{}
	m.done = make(chan struct{})
	go m.receive(legacy, granted)
}

// receive applies every packet to the latest values until the connection
// is closed, so no backlog builds up however fast the headset sends.
func (m *Module) receive(legacy bool, granted adapter.Support) {
	defer close(m.done)
	buf := make([]byte, maxDatagram)
	for {
		_ = m.conn.SetReadDeadline(time.Now().Add(silenceWarnAfter))
		n, err := m.conn.Read(buf)
		if err != nil {
			var netErr net.Error
			switch {
			case errors.Is(err, net.ErrClosed):
				return
			case errors.As(err, &netErr) && netErr.Timeout():
				m.logger.Warn("Receive data timed out")
			default:
				m.logger.Warn("Receive failed", zap.Error(err))
			}
			continue
		}

		w, err := parsePacket(buf[:n], legacy)
		if errors.Is(err, errNotFaceData) {
			continue
		}
		if err != nil {
			m.logger.Debug("Ignoring malformed packet", zap.Error(err))
			continue
		}
		m.mu.Lock()
		if granted.Eye {
			updateEye(&m.latest, w)
		}
		if granted.Expression {
			m.mapper.updateExpression(&m.latest, w)
		}
		m.mu.Unlock()
	}
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

// Update writes the latest received values into data without waiting.
func (m *Module) Update(_ context.Context, data *tracking.TrackingData) error {
	if m.conn == nil {
		return errors.New("VRCFTPicoModule: module not initialised")
	}
	m.mu.Lock()
	data.CopyFrom(&m.latest)
	m.mu.Unlock()
	return nil
}

func (m *Module) Teardown() error {
	if m.conn == nil {
		return nil
	}
	err := m.conn.Close()
	<-m.done
	m.conn = nil
	m.mu.Lock()
	m.granted = adapter.Support{}
	m.mu.Unlock()
	return err
}
