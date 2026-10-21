package osc

import (
	"net"
	"testing"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/internal/params"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

func TestSenderBundlesAndSplits(t *testing.T) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	port := conn.LocalAddr().(*net.UDPAddr).Port

	s := NewSender(zap.NewNop())
	if err := s.SetTarget("127.0.0.1", port); err != nil {
		t.Fatal(err)
	}
	var dispatched int
	s.OnDispatched = func(n int) { dispatched += n }

	var msgs []params.Message
	for i := 0; i < 300; i++ {
		msgs = append(msgs, params.Message{Address: "/avatar/parameters/v2/SomeLongParameterName", Value: float32(i)})
	}
	msgs = append(msgs, params.Message{Address: "/tracking/eye/LeftRightPitchYaw", Value: tracking.Vector4{X: 1, Y: 2, Z: 3, W: 4}})
	if err := s.Send(msgs); err != nil {
		t.Fatal(err)
	}
	if dispatched != len(msgs) {
		t.Fatalf("dispatched %d, want %d", dispatched, len(msgs))
	}

	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 65536)
	received := 0
	datagrams := 0
	for received < len(msgs) {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			t.Fatalf("received %d of %d messages: %v", received, len(msgs), err)
		}
		if n > maxDatagram {
			t.Fatalf("datagram of %d bytes exceeds limit", n)
		}
		datagrams++
		pkt, err := osc.ParsePacket(string(buf[:n]))
		if err != nil {
			t.Fatal(err)
		}
		b := pkt.(*osc.Bundle)
		received += len(b.Messages)
		for _, m := range b.Messages {
			if m.Address == "/tracking/eye/LeftRightPitchYaw" && len(m.Arguments) != 4 {
				t.Fatalf("vector4 expanded into %d arguments", len(m.Arguments))
			}
		}
	}
	if datagrams < 2 {
		t.Fatalf("expected the batch to be split, got %d datagrams", datagrams)
	}
}

func TestSenderBlob(t *testing.T) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	s := NewSender(zap.NewNop())
	if err := s.SetTarget("127.0.0.1", conn.LocalAddr().(*net.UDPAddr).Port); err != nil {
		t.Fatal(err)
	}
	payload := make([]byte, 102)
	payload[36] = 255
	if err := s.Send([]params.Message{{Address: "/tracking/face/v1", Value: payload}}); err != nil {
		t.Fatal(err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 65536)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		t.Fatal(err)
	}
	pkt, err := osc.ParsePacket(string(buf[:n]))
	if err != nil {
		t.Fatal(err)
	}
	msg := pkt.(*osc.Bundle).Messages[0]
	if tags, _ := msg.TypeTags(); tags != ",b" {
		t.Fatalf("type tags %q, want \",b\"", tags)
	}
	got := msg.Arguments[0].([]byte)
	if len(got) != 102 || got[36] != 255 {
		t.Fatalf("blob len %d, byte 36 = %d", len(got), got[36])
	}
}
