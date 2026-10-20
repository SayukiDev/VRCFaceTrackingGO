package pico

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"go.uber.org/zap"

	"VRCFaceTracking/module/adapter"
	"VRCFaceTracking/pkg/tracking"
)

func TestPacketSizes(t *testing.T) {
	if got := binary.Size(packetHeader{}); got != headerSize {
		t.Fatalf("header size = %d, want %d", got, headerSize)
	}
	if got := binary.Size(packetBody{}); got != bodySize {
		t.Fatalf("body size = %d, want %d", got, bodySize)
	}
}

func encodeFacePacket(t *testing.T, legacy bool, set func(*packetBody)) []byte {
	t.Helper()
	var buf bytes.Buffer
	if !legacy {
		h := packetHeader{StartCode1: 0xAB, StartCode2: 0xCD, TrackingType: trackingTypeFace}
		if err := binary.Write(&buf, binary.LittleEndian, h); err != nil {
			t.Fatal(err)
		}
	}
	var body packetBody
	set(&body)
	if err := binary.Write(&buf, binary.LittleEndian, body); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestParsePacket(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		data := encodeFacePacket(t, legacy, func(b *packetBody) { b.BlendShapeWeight[jawOpen] = 0.75 })
		w, err := parsePacket(data, legacy)
		if err != nil {
			t.Fatalf("legacy=%v: %v", legacy, err)
		}
		if w[jawOpen] != 0.75 {
			t.Fatalf("legacy=%v: jawOpen = %v", legacy, w[jawOpen])
		}
	}
	if _, err := parsePacket([]byte{1, 2, 3}, false); err == nil {
		t.Fatal("short packet must fail")
	}
	nonFace := encodeFacePacket(t, false, func(*packetBody) {})
	nonFace[2] = 1 // trackingType
	if _, err := parsePacket(nonFace, false); err != errNotFaceData {
		t.Fatalf("non-face packet: got %v", err)
	}
}

func TestMapping(t *testing.T) {
	var w weights
	w[eyeBlinkL] = 0.25
	w[jawOpen] = 0.5
	w[cheekPuff] = 0.4
	data := tracking.NewTrackingData()
	updateEye(data, &w)
	(&mapper{}).updateExpression(data, &w)
	if data.Eye.Left.Openness != 0.75 {
		t.Errorf("openness = %v", data.Eye.Left.Openness)
	}
	if data.Shapes[tracking.JawOpen].Weight != 0.5 {
		t.Errorf("jaw = %v", data.Shapes[tracking.JawOpen].Weight)
	}
	if data.Shapes[tracking.CheekPuffLeft].Weight != 0.4 || data.Shapes[tracking.CheekPuffRight].Weight != 0.4 {
		t.Errorf("cheek puff not symmetric: %v / %v", data.Shapes[tracking.CheekPuffLeft].Weight, data.Shapes[tracking.CheekPuffRight].Weight)
	}
}

// Initialize must give up when nothing is streaming instead of hanging.
func TestInitializeTimesOutWithoutData(t *testing.T) {
	m := New(t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := m.Initialize(ctx, zap.NewNop(), adapter.Support{Eye: true, Expression: true})
	if err == nil {
		t.Fatal("expected an error without data")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("Initialize took %s, should honour ctx", time.Since(start))
	}
}

func TestInitializeAndUpdateWithStream(t *testing.T) {
	m := New(t.TempDir())
	packet := encodeFacePacket(t, false, func(b *packetBody) { b.BlendShapeWeight[jawOpen] = 0.9 })

	sender, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: portCurrent})
	if err != nil {
		t.Fatal(err)
	}
	defer sender.Close()

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				_, _ = sender.Write(packet)
			}
		}
	}()

	ctx := context.Background()
	granted, err := m.Initialize(ctx, zap.NewNop(), adapter.Support{Eye: true, Expression: true})
	if err != nil {
		t.Fatal(err)
	}
	if !granted.Eye || !granted.Expression {
		t.Fatalf("granted = %+v", granted)
	}
	defer m.Teardown()

	data := tracking.NewTrackingData()
	deadline := time.Now().Add(2 * time.Second)
	for data.Shapes[tracking.JawOpen].Weight != 0.9 {
		if time.Now().After(deadline) {
			t.Fatal("never received jaw value")
		}
		if err := m.Update(ctx, data); err != nil {
			t.Fatal(err)
		}
	}
	if m.Metadata().Name != "PICO / Full FaceTracking" {
		t.Fatalf("name = %q", m.Metadata().Name)
	}
}

// Update must return promptly when the stream goes silent.
func TestUpdateReturnsOnSilence(t *testing.T) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	m := &Module{conn: conn, logger: zap.NewNop(), buf: make([]byte, maxDatagram)}
	defer m.Teardown()

	start := time.Now()
	if err := m.Update(context.Background(), tracking.NewTrackingData()); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("Update blocked for %s", elapsed)
	}
}
