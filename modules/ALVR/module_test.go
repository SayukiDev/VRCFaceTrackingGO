package main

import (
	"context"
	"encoding/binary"
	"math"
	"net"
	"testing"
	"time"

	"go.uber.org/zap"

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

// startOnFreePort receives on a free port, so the tests do not clash with a
// running app that holds the real one. It returns a sender aimed at it.
func startOnFreePort(t *testing.T) (*Module, *net.UDPConn) {
	t.Helper()
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	m := &Module{logger: zap.NewNop()}
	m.start(conn)
	t.Cleanup(func() { m.Teardown() })

	sender, err := net.DialUDP("udp4", nil, conn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sender.Close() })
	return m, sender
}

// waitForJaw calls Update until JawOpen is want.
func waitForJaw(t *testing.T, m *Module, want float32) {
	t.Helper()
	data := tracking.NewTrackingData()
	deadline := time.Now().Add(2 * time.Second)
	for data.Shapes[tracking.JawOpen].Weight != want {
		if time.Now().After(deadline) {
			t.Fatalf("JawOpen = %v, want %v", data.Shapes[tracking.JawOpen].Weight, want)
		}
		time.Sleep(time.Millisecond)
		if err := m.Update(context.Background(), data); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUpdateWithStream(t *testing.T) {
	m, sender := startOnFreePort(t)
	face := make([]float32, picoCount)
	face[picoJawOpen] = 0.9
	if _, err := sender.Write(encodeSection(nil, "FacePico", face)); err != nil {
		t.Fatal(err)
	}
	waitForJaw(t, m, 0.9)
}

// A burst sent faster than Update is called must not queue up: Update
// returns the last packet at once.
func TestUpdateReturnsNewestPacket(t *testing.T) {
	m, sender := startOnFreePort(t)
	face := make([]float32, picoCount)
	const n = 100
	for i := 1; i <= n; i++ {
		face[picoJawOpen] = float32(i) / n
		if _, err := sender.Write(encodeSection(nil, "FacePico", face)); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(200 * time.Millisecond) // let the receiver drain the burst
	data := tracking.NewTrackingData()
	if err := m.Update(context.Background(), data); err != nil {
		t.Fatal(err)
	}
	if got := data.Shapes[tracking.JawOpen].Weight; got != 1 {
		t.Fatalf("JawOpen = %v, want the last packet's 1", got)
	}
}

// Update must not wait for data while the sender is silent.
func TestUpdateDoesNotWait(t *testing.T) {
	m, _ := startOnFreePort(t)
	start := time.Now()
	if err := m.Update(context.Background(), tracking.NewTrackingData()); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("Update blocked for %s", elapsed)
	}
}

func TestUpdateBeforeInitialize(t *testing.T) {
	if err := New().Update(context.Background(), tracking.NewTrackingData()); err == nil {
		t.Fatal("expected an error before Initialize")
	}
}
