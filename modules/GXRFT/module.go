// Command GXRFT is the tracking module program for gxr-root-ft
// (https://github.com/compdoge/gxr-root-ft), which streams the face and eye
// tracking of a rooted Samsung Galaxy XR over UDP. gxr-root-ft uses the
// packet format of ALVR's "VRCFaceTracking" sink, so this module follows
// the VRCFT-ALVR module v1.3.
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
//go:embed metadata.json gxrft.png
var files embed.FS

var metadata = adapter.MustLoadMetadata(files, "metadata.json")

const (
	// port is where the ALVR server sends tracking data.
	port = 0xA1F7

	readTimeout = 100 * time.Millisecond
	maxDatagram = 4096
)

// Module receives face tracking over UDP.
type Module struct {
	logger *zap.Logger

	mu      sync.Mutex
	conn    *net.UDPConn
	buf     []byte
	unknown map[string]bool
}

var _ adapter.Module = (*Module)(nil)

// New returns an uninitialised GXRFT module.
func New() *Module {
	return &Module{buf: make([]byte, maxDatagram)}
}

func (m *Module) ID() string { return "GXRFT" }

func (m *Module) Metadata() adapter.Metadata { return metadata }

func (m *Module) Supported() adapter.Support {
	return adapter.Support{Eye: true, Expression: true}
}

// Initialize binds the UDP port. ALVR and the host can start in any order,
// so it does not wait for data.
func (m *Module) Initialize(_ context.Context, logger *zap.Logger, avail adapter.Support) (adapter.Support, error) {
	m.logger = logger.Named("GXRFT")

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: port})
	if err != nil {
		return adapter.Support{}, fmt.Errorf("GXRFT: listen on port %d: %w", port, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.conn = conn
	m.unknown = make(map[string]bool)
	m.logger.Info("Listening for tracking data", zap.Int("port", port))
	return avail, nil
}

// Update reads one datagram with a short deadline so a silent ALVR never
// blocks the caller.
func (m *Module) Update(ctx context.Context, data *tracking.TrackingData) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn == nil {
		return errors.New("GXRFT: module not initialised")
	}

	deadline := time.Now().Add(readTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = m.conn.SetReadDeadline(deadline)

	n, _, err := m.conn.ReadFromUDP(m.buf)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() || errors.Is(err, net.ErrClosed) {
			return nil
		}
		return fmt.Errorf("GXRFT: receive: %w", err)
	}

	sections, unknown := parsePacket(m.buf[:n])
	if unknown != "" && !m.unknown[unknown] {
		m.unknown[unknown] = true
		m.logger.Error("Unrecognized prefix", zap.String("prefix", unknown))
	}
	apply(sections, data)
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
	return err
}
