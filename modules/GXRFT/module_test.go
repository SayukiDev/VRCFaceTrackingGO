package main

import (
	"context"
	"encoding/binary"
	"math"
	"net"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

// encodeSection appends one prefix and its parameters in wire format.
func encodeSection(buf []byte, prefix string, params []float32) []byte {
	buf = append(buf, prefix...)
	for _, v := range params {
		buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(v))
	}
	return buf
}

func TestCounts(t *testing.T) {
	if fbFace1Count != 63 || fbFace2Count != 70 || fbTongueOut != 68 {
		t.Fatalf("fb counts = %d/%d, tongue %d", fbFace1Count, fbFace2Count, fbTongueOut)
	}
	if picoCount != 72 || htcEyesCount != 14 || htcLipCount != 37 {
		t.Fatalf("VRCFTPicoModule %d, htc eyes %d, htc lip %d", picoCount, htcEyesCount, htcLipCount)
	}
	for prefix := range consumers {
		if len(prefix) != prefixSize {
			t.Errorf("prefix %q is not %d bytes", prefix, prefixSize)
		}
	}
}

func TestParseMultipleSections(t *testing.T) {
	face := make([]float32, fbFace2Count)
	face[fbJawDrop] = 0.5
	face[fbTongueOut] = 0.25
	face[fbEyesClosedL] = 1

	var pkt []byte
	pkt = encodeSection(pkt, "CombQuat", []float32{0, 0, 0, 1})
	pkt = encodeSection(pkt, "Face2Fb\x00", face)

	sections, unknown := parsePacket(pkt)
	if unknown != "" || len(sections) != 2 {
		t.Fatalf("sections = %d, unknown = %q", len(sections), unknown)
	}
	data := tracking.NewTrackingData()
	apply(sections, data)
	if data.Shapes[tracking.JawOpen].Weight != 0.5 || data.Shapes[tracking.TongueOut].Weight != 0.25 {
		t.Errorf("jaw = %v, tongue = %v", data.Shapes[tracking.JawOpen].Weight, data.Shapes[tracking.TongueOut].Weight)
	}
	if data.Eye.Left.Openness != 0 || data.Eye.Right.Openness != 1 {
		t.Errorf("openness = %v / %v", data.Eye.Left.Openness, data.Eye.Right.Openness)
	}
	if data.Eye.Left.Gaze != (tracking.Vector2{}) {
		t.Errorf("identity gaze = %+v", data.Eye.Left.Gaze)
	}
}

func TestParseStopsAtUnknownPrefix(t *testing.T) {
	pkt := encodeSection(nil, "Bogus\x00\x00\x00", []float32{1})
	pkt = encodeSection(pkt, "CombQuat", []float32{0, 0, 0, 1})
	sections, unknown := parsePacket(pkt)
	if unknown != "Bogus\x00\x00\x00" || len(sections) != 0 {
		t.Fatalf("sections = %d, unknown = %q", len(sections), unknown)
	}
}

func TestParseTruncated(t *testing.T) {
	sections, _ := parsePacket(encodeSection(nil, "EyesQuat", []float32{0, 0, 0, 1, 0, 0}))
	if len(sections) != 1 || len(sections[0].params) != 8 {
		t.Fatalf("sections = %+v", sections)
	}
	// Must not panic on the zero padded quaternion.
	apply(sections, tracking.NewTrackingData())
}

func TestUnifiedExpr(t *testing.T) {
	p := make([]float32, uniExprCount)
	p[tracking.MouthClosed] = 0.3
	tail := p[tracking.ExpressionCount:]
	tail[uniLeftOpenness] = 0.8
	tail[uniRightGazeYaw] = -0.2

	sections, _ := parsePacket(encodeSection(nil, "UniExpr\x00", p))
	data := tracking.NewTrackingData()
	apply(sections, data)
	if data.Shapes[tracking.MouthClosed].Weight != 0.3 || data.Eye.Left.Openness != 0.8 || data.Eye.Right.Gaze.Y != -0.2 {
		t.Fatalf("unexpected data: %+v", data.Eye)
	}
}

func TestInitializeAndUpdateWithStream(t *testing.T) {
	m := New()
	if _, err := m.Initialize(context.Background(), zap.NewNop(), adapter.Support{Eye: true, Expression: true}); err != nil {
		t.Fatal(err)
	}
	defer m.Teardown()

	sender, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port})
	if err != nil {
		t.Fatal(err)
	}
	defer sender.Close()

	face := make([]float32, picoCount)
	face[picoJawOpen] = 0.9
	if _, err := sender.Write(encodeSection(nil, "FacePico", face)); err != nil {
		t.Fatal(err)
	}

	data := tracking.NewTrackingData()
	deadline := time.Now().Add(2 * time.Second)
	for data.Shapes[tracking.JawOpen].Weight != 0.9 {
		if time.Now().After(deadline) {
			t.Fatal("never received jaw value")
		}
		if err := m.Update(context.Background(), data); err != nil {
			t.Fatal(err)
		}
	}
}

// Update must return promptly when ALVR is silent.
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
