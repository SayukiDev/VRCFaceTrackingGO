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

	maxDatagram = 4096
)

// Module receives face tracking over UDP.
type Module struct {
	logger *zap.Logger

	conn *net.UDPConn
	done chan struct{}

	mu     sync.Mutex
	latest tracking.TrackingData
}

var _ adapter.Module = (*Module)(nil)

// New returns an uninitialised GXRFT module.
func New() *Module { return &Module{} }

func (m *Module) ID() string { return "GXRFT" }

func (m *Module) Metadata() adapter.Metadata { return metadata }

func (m *Module) Supported() adapter.Support {
	return adapter.Support{Eye: true, Expression: true}
}

// Initialize binds the UDP port and starts receiving in the background.
// ALVR and the host can start in any order, so it does not wait for data.
func (m *Module) Initialize(_ context.Context, logger *zap.Logger, avail adapter.Support) (adapter.Support, error) {
	m.logger = logger.Named("GXRFT")

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: port})
	if err != nil {
		return adapter.Support{}, fmt.Errorf("GXRFT: listen on port %d: %w", port, err)
	}
	m.start(conn)
	m.logger.Info("Listening for tracking data", zap.Int("port", port))
	return avail, nil
}

func (m *Module) start(conn *net.UDPConn) {
	m.conn = conn
	m.latest = *tracking.NewTrackingData()
	m.done = make(chan struct{})
	go m.receive()
}

// receive applies every datagram to the latest values until the connection
// is closed, so no backlog builds up however fast it is sent.
func (m *Module) receive() {
	defer close(m.done)
	buf := make([]byte, maxDatagram)
	unknown := make(map[string]bool)
	for {
		n, err := m.conn.Read(buf)
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			m.logger.Warn("Receive failed", zap.Error(err))
			continue
		}

		sections, prefix := parsePacket(buf[:n])
		if prefix != "" && !unknown[prefix] {
			unknown[prefix] = true
			m.logger.Error("Unrecognized prefix", zap.String("prefix", prefix))
		}
		m.mu.Lock()
		apply(sections, &m.latest)
		m.mu.Unlock()
	}
}

// Update writes the latest received values into data without waiting.
func (m *Module) Update(_ context.Context, data *tracking.TrackingData) error {
	if m.conn == nil {
		return errors.New("GXRFT: module not initialised")
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
	return err
}
