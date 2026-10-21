// Command SteamLink is the tracking module program for Steam Link on Meta
// Quest headsets. Steam Link forwards eye and face tracking as OSC to UDP
// port 9015 when "Enable OSC" and the eye / face tracking sharing options
// are turned on in its advanced settings. It follows LinkFT
// (https://github.com/ykeara/LinkFT).
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"math"
	"net"
	"sync"

	"github.com/hypebeast/go-osc/osc"
	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

// files holds metadata.json and the icons it names.
//
//go:embed metadata.json steamlink.png
var files embed.FS

var metadata = adapter.MustLoadMetadata(files, "metadata.json")

const (
	// port is the "OSC Output Port" to set in Steam Link.
	port        = 9015
	maxDatagram = 8192
)

// state is the latest tracking values received from Steam Link.
type state struct {
	gazePoint  [3]float32 // point in space the user looks at
	eyesClosed [2]float32 // left, right
	shapes     [tracking.ExpressionCount]float32
}

// Module receives Steam Link's OSC output.
type Module struct {
	logger  *zap.Logger
	granted adapter.Support

	conn *net.UDPConn
	done chan struct{}

	mu     sync.Mutex
	latest state
}

var _ adapter.Module = (*Module)(nil)

// New returns an uninitialised Steam Link module.
func New() *Module { return &Module{} }

func (m *Module) ID() string { return "SteamLink" }

func (m *Module) Metadata() adapter.Metadata { return metadata }

func (m *Module) Supported() adapter.Support {
	return adapter.Support{Eye: true, Expression: true}
}

// Initialize binds the OSC port and starts receiving in the background.
// Steam Link and the host can start in any order, so it does not wait for
// data.
func (m *Module) Initialize(_ context.Context, logger *zap.Logger, avail adapter.Support) (adapter.Support, error) {
	m.logger = logger.Named("SteamLink")
	m.granted = avail
	if !avail.Eye && !avail.Expression {
		m.logger.Info("Neither eye nor expression tracking is needed, staying idle")
		return avail, nil
	}

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port})
	if err != nil {
		return adapter.Support{}, fmt.Errorf("SteamLink: listen on port %d: %w", port, err)
	}
	m.conn = conn
	m.done = make(chan struct{})
	go m.receive()
	m.logger.Info("Listening for Steam Link OSC", zap.Int("port", port), zap.Bool("eye", avail.Eye), zap.Bool("expression", avail.Expression))
	return avail, nil
}

// receive stores every OSC packet until the connection is closed.
func (m *Module) receive() {
	defer close(m.done)
	buf := make([]byte, maxDatagram)
	for {
		n, err := m.conn.Read(buf)
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			m.logger.Warn("Receive failed", zap.Error(err))
			continue
		}
		packet, err := osc.ParsePacket(string(buf[:n]))
		if err != nil {
			m.logger.Debug("Ignoring malformed OSC packet", zap.Error(err))
			continue
		}
		m.mu.Lock()
		m.latest.apply(packet)
		m.mu.Unlock()
	}
}

// apply stores the values of every message in packet, recursing into
// bundles.
func (s *state) apply(packet osc.Packet) {
	switch p := packet.(type) {
	case *osc.Bundle:
		for _, msg := range p.Messages {
			s.apply(msg)
		}
		for _, b := range p.Bundles {
			s.apply(b)
		}
	case *osc.Message:
		s.applyMessage(p)
	}
}

func (s *state) applyMessage(msg *osc.Message) {
	if len(msg.Arguments) == 0 {
		return
	}
	switch msg.Address {
	case addrGazePoint:
		for i := range s.gazePoint {
			if i < len(msg.Arguments) {
				s.gazePoint[i] = toFloat(msg.Arguments[i])
			}
		}
	case addrEyesClosedL:
		s.eyesClosed[0] = toFloat(msg.Arguments[0])
	case addrEyesClosedR:
		s.eyesClosed[1] = toFloat(msg.Arguments[0])
	default:
		v := toFloat(msg.Arguments[0])
		for _, u := range faceMapping[msg.Address] {
			s.shapes[u] = v
		}
	}
}

func toFloat(arg any) float32 {
	switch v := arg.(type) {
	case float32:
		return v
	case float64:
		return float32(v)
	case int32:
		return float32(v)
	}
	return 0
}

// Update writes the latest received values into data.
func (m *Module) Update(_ context.Context, data *tracking.TrackingData) error {
	m.mu.Lock()
	s := m.latest
	m.mu.Unlock()

	if m.granted.Eye {
		updateEye(&s, data)
	}
	if m.granted.Expression {
		updateFace(&s, data)
	}
	return nil
}

func updateEye(s *state, data *tracking.TrackingData) {
	// Both eyes share the gaze point. The angles are radians, as in LinkFT.
	gaze := tracking.Vector2{
		X: nanToZero(float32(math.Atan2(float64(s.gazePoint[0]), float64(-s.gazePoint[2])))),
		Y: nanToZero(float32(math.Atan2(float64(s.gazePoint[1]), float64(-s.gazePoint[2])))),
	}
	eye := &data.Eye
	eye.Left.Gaze, eye.Right.Gaze = gaze, gaze

	// Steam Link has no pupil data, but leaving it unset causes issues.
	eye.Left.PupilDiameterMM, eye.Right.PupilDiameterMM = 5, 5
	eye.MaxDilation, eye.MinDilation = 10, 0

	eye.Left.Openness = openness(s.eyesClosed[0], s.shapes[tracking.EyeSquintLeft])
	eye.Right.Openness = openness(s.eyesClosed[1], s.shapes[tracking.EyeSquintRight])
}

func openness(closed, tightener float32) float32 {
	return 1 - min(max(closed+closed*tightener, 0), 1)
}

func nanToZero(v float32) float32 {
	if v != v {
		return 0
	}
	return v
}

func updateFace(s *state, data *tracking.TrackingData) {
	for _, shapes := range faceMapping {
		for _, u := range shapes {
			data.Shapes[u].Weight = s.shapes[u]
		}
	}

	w := func(u tracking.UnifiedExpression) float32 { return data.Shapes[u].Weight }
	set := func(u tracking.UnifiedExpression, v float32) { data.Shapes[u].Weight = v }

	// The upper lip raiser includes the nose wrinkler; take it out. The
	// deepen shapes deliberately use the already corrected values, as LinkFT
	// does.
	set(tracking.MouthUpperUpLeft, max(0, w(tracking.MouthUpperUpLeft)-w(tracking.NoseSneerLeft)))
	set(tracking.MouthUpperUpRight, max(0, w(tracking.MouthUpperUpRight)-w(tracking.NoseSneerRight)))
	set(tracking.MouthUpperDeepenLeft, max(0, w(tracking.MouthUpperUpLeft)-w(tracking.NoseSneerLeft)))
	set(tracking.MouthUpperDeepenRight, max(0, w(tracking.MouthUpperUpRight)-w(tracking.NoseSneerRight)))

	// Limit lip suck while the mouth moves sideways.
	set(tracking.LipSuckUpperLeft, min(1-pow6root(w(tracking.MouthUpperLeft)), w(tracking.LipSuckUpperLeft)))
	set(tracking.LipSuckUpperRight, min(1-pow6root(w(tracking.MouthUpperRight)), w(tracking.LipSuckUpperRight)))
}

func pow6root(v float32) float32 { return float32(math.Pow(float64(v), 1.0/6)) }

func (m *Module) Teardown() error {
	if m.conn == nil {
		return nil
	}
	err := m.conn.Close()
	<-m.done
	m.conn = nil
	return err
}
