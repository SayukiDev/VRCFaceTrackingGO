package main

import (
	"context"
	"math"
	"net"
	"testing"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

func message(addr string, args ...any) *osc.Message {
	m := osc.NewMessage(addr)
	for _, a := range args {
		m.Append(a)
	}
	return m
}

// bundle builds a Steam Link like packet: a bundle of messages with one
// nested bundle.
func bundle(t *testing.T) []byte {
	t.Helper()
	inner := osc.NewBundle(time.Now())
	if err := inner.Append(message("/sl/xrfb/facew/NoseWrinklerL", float32(0.25))); err != nil {
		t.Fatal(err)
	}
	b := osc.NewBundle(time.Now())
	for _, m := range []*osc.Message{
		message(addrGazePoint, float32(0), float32(0), float32(-1)),
		message(addrEyesClosedL, float32(0.5)),
		message("/sl/xrfb/facew/LidTightenerL", float32(1)),
		message("/sl/xrfb/facew/JawDrop", float32(0.75)),
		message("/sl/xrfb/facew/BrowLowererR", float32(0.4)),
		message("/sl/xrfb/facew/UpperLipRaiserL", float32(0.6)),
		message("/sl/xrfb/facew/Unknown", float32(1)),
	} {
		if err := b.Append(m); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.Append(inner); err != nil {
		t.Fatal(err)
	}
	data, err := b.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func near(a, b float32) bool { return math.Abs(float64(a-b)) < 1e-5 }

func TestApplyAndUpdate(t *testing.T) {
	packet, err := osc.ParsePacket(string(bundle(t)))
	if err != nil {
		t.Fatal(err)
	}
	var s state
	s.apply(packet)

	data := tracking.NewTrackingData()
	updateEye(&s, data)
	updateFace(&s, data)

	if data.Eye.Left.Gaze != (tracking.Vector2{}) {
		t.Errorf("looking straight ahead, gaze = %+v", data.Eye.Left.Gaze)
	}
	// Closed 0.5 with a full lid tightener: 1 - (0.5 + 0.5*1) = 0.
	if data.Eye.Left.Openness != 0 || data.Eye.Right.Openness != 1 {
		t.Errorf("openness = %v / %v", data.Eye.Left.Openness, data.Eye.Right.Openness)
	}
	if data.Shapes[tracking.JawOpen].Weight != 0.75 {
		t.Errorf("jaw = %v", data.Shapes[tracking.JawOpen].Weight)
	}
	if data.Shapes[tracking.BrowPinchRight].Weight != 0.4 || data.Shapes[tracking.BrowLowererRight].Weight != 0.4 {
		t.Error("BrowLowererR must drive both brow shapes")
	}
	// The nose wrinkler (from the nested bundle) is taken out of the upper
	// lip raiser.
	if up := data.Shapes[tracking.MouthUpperUpLeft].Weight; !near(up, 0.35) {
		t.Errorf("MouthUpperUpLeft = %v", up)
	}
	if deepen := data.Shapes[tracking.MouthUpperDeepenLeft].Weight; !near(deepen, 0.1) {
		t.Errorf("MouthUpperDeepenLeft = %v", deepen)
	}
}

func TestGazeAngles(t *testing.T) {
	s := state{gazePoint: [3]float32{1, 0, -1}}
	data := tracking.NewTrackingData()
	updateEye(&s, data)
	if !near(data.Eye.Right.Gaze.X, math.Pi/4) || data.Eye.Right.Gaze.Y != 0 {
		t.Errorf("gaze = %+v", data.Eye.Right.Gaze)
	}
}

func TestReceive(t *testing.T) {
	m := New()
	if _, err := m.Initialize(context.Background(), zap.NewNop(), adapter.Support{Expression: true}); err != nil {
		t.Fatal(err)
	}
	defer m.Teardown()

	sender, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port})
	if err != nil {
		t.Fatal(err)
	}
	defer sender.Close()
	if _, err := sender.Write(bundle(t)); err != nil {
		t.Fatal(err)
	}

	data := tracking.NewTrackingData()
	deadline := time.Now().Add(2 * time.Second)
	for data.Shapes[tracking.JawOpen].Weight != 0.75 {
		if time.Now().After(deadline) {
			t.Fatal("never received the jaw value")
		}
		time.Sleep(10 * time.Millisecond)
		if err := m.Update(context.Background(), data); err != nil {
			t.Fatal(err)
		}
	}
	if data.Eye.Left.Openness != 1 {
		t.Error("eye data written although only expressions were granted")
	}
}
