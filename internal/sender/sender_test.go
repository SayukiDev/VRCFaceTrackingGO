package sender

import (
	"net"
	"testing"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/internal/osc"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/params"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

type fakeStatus struct{ eye, expr adapter.State }

func (f *fakeStatus) EyeStatus() adapter.State        { return f.eye }
func (f *fakeStatus) ExpressionStatus() adapter.State { return f.expr }

func newTestSender(t *testing.T, status *fakeStatus) (*Sender, *int) {
	t.Helper()
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })

	transport := osc.NewSender(zap.NewNop())
	if err := transport.SetTarget("127.0.0.1", conn.LocalAddr().(*net.UDPAddr).Port); err != nil {
		t.Fatal(err)
	}
	var dispatched int
	transport.OnDispatched = func(n int) { dispatched += n }
	return New(zap.NewNop(), status, transport), &dispatched
}

// sendCount sends d and returns how many messages went out.
func sendCount(s *Sender, dispatched *int, d *tracking.TrackingData) int {
	*dispatched = 0
	s.Send(d)
	return *dispatched
}

func TestSendSkipsUnchangedParameters(t *testing.T) {
	s, dispatched := newTestSender(t, &fakeStatus{eye: adapter.Active, expr: adapter.Active})
	s.SetDefinitions([]params.Definition{{Address: "/avatar/parameters/v2/JawOpen", Name: "v2/JawOpen", Type: params.TypeFloat}})

	d := tracking.NewTrackingData()
	d.Shapes[tracking.JawOpen].Weight = 0.3
	if n := sendCount(s, dispatched, d); n == 0 {
		t.Fatal("first send must emit values")
	}
	if n := sendCount(s, dispatched, d); n != 0 {
		t.Fatalf("unchanged data re-sent %d messages", n)
	}
	d.Shapes[tracking.JawOpen].Weight = 0.4
	if n := sendCount(s, dispatched, d); n != 1 {
		t.Fatalf("changed JawOpen sent %d messages, want 1", n)
	}

	// A new avatar gets everything again.
	s.SetDefinitions([]params.Definition{{Address: "/avatar/parameters/v2/JawOpen", Name: "v2/JawOpen", Type: params.TypeFloat}})
	if n := sendCount(s, dispatched, d); n == 0 {
		t.Fatal("reset must resend values")
	}
}

func TestSendForceRelevantSharedAddress(t *testing.T) {
	s, dispatched := newTestSender(t, &fakeStatus{eye: adapter.Active, expr: adapter.Active})
	s.SetAllRelevant(true)
	d := tracking.NewTrackingData()
	sendCount(s, dispatched, d)
	// The bool and float variants share an address; they must not keep
	// overwriting each other's last value.
	if n := sendCount(s, dispatched, d); n != 0 {
		t.Fatalf("unchanged data re-sent %d messages", n)
	}
}

func TestSendSkipsUnchangedNative(t *testing.T) {
	status := &fakeStatus{eye: adapter.Active, expr: adapter.Active}
	s, dispatched := newTestSender(t, status)
	s.SetOutputMode(OutputNative)

	d := tracking.NewTrackingData()
	if n := sendCount(s, dispatched, d); n != 3 {
		t.Fatalf("first native send emitted %d messages, want 3", n)
	}
	if n := sendCount(s, dispatched, d); n != 0 {
		t.Fatalf("unchanged native data re-sent %d messages", n)
	}
	d.Shapes[tracking.JawOpen].Weight = 1
	if n := sendCount(s, dispatched, d); n != 1 {
		t.Fatalf("face change sent %d messages, want 1", n)
	}

	// Eye tracking dropping out and coming back resends the eye endpoints.
	status.eye = adapter.Idle
	sendCount(s, dispatched, d)
	status.eye = adapter.Active
	if n := sendCount(s, dispatched, d); n != 2 {
		t.Fatalf("eye recovery sent %d messages, want 2", n)
	}
}
