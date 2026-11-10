package main

import (
	"context"
	"errors"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

// fakeXR is a scripted runtime.
type fakeXR struct {
	mu sync.Mutex

	// connectErrs are returned by the first connects, one each.
	connectErrs []error
	feat        features
	// events are handed out one per pollEvent.
	events []sessionEvent
	frame  frame
	// readErr, when set, fails every read.
	readErr error

	connects    int
	disconnects int
	begins      int
	ends        int
	reads       int
}

func (f *fakeXR) connect() (features, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.connects++
	if len(f.connectErrs) > 0 {
		err := f.connectErrs[0]
		f.connectErrs = f.connectErrs[1:]
		return features{}, err
	}
	return f.feat, nil
}

func (f *fakeXR) pollEvent() (sessionEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.events) == 0 {
		return eventNone, nil
	}
	ev := f.events[0]
	f.events = f.events[1:]
	return ev, nil
}

func (f *fakeXR) begin() error { f.mu.Lock(); f.begins++; f.mu.Unlock(); return nil }
func (f *fakeXR) end() error   { f.mu.Lock(); f.ends++; f.mu.Unlock(); return nil }

func (f *fakeXR) read(out *frame) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if f.readErr != nil {
		return f.readErr
	}
	*out = f.frame
	return nil
}

func (f *fakeXR) disconnect() { f.mu.Lock(); f.disconnects++; f.mu.Unlock() }

func (f *fakeXR) push(events ...sessionEvent) {
	f.mu.Lock()
	f.events = append(f.events, events...)
	f.mu.Unlock()
}

func (f *fakeXR) get(v *int) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return *v
}

// start runs a module on top of xr with short timings.
func start(t *testing.T, xr *fakeXR) *Module {
	t.Helper()
	m := New()
	m.open = func() (xrAPI, error) { return xr, nil }
	m.pollInterval = time.Millisecond
	m.retryDelay = 5 * time.Millisecond
	got, err := m.Initialize(context.Background(), zap.NewNop(), adapter.Support{Eye: true, Expression: true})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Eye || !got.Expression {
		t.Fatalf("support = %+v", got)
	}
	t.Cleanup(func() { m.Teardown() })
	return m
}

// eventually polls cond until it holds.
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

func update(t *testing.T, m *Module) *tracking.TrackingData {
	t.Helper()
	data := tracking.NewTrackingData()
	if err := m.Update(context.Background(), data); err != nil {
		t.Fatal(err)
	}
	return data
}

func fb2Frame() frame {
	var f frame
	f.fb2[fbJawDrop] = 0.5
	f.fb2[fbTongueOut] = 0.25
	f.fb2[fbEyesClosedL] = 1
	f.fb2Valid = true
	return f
}

func TestRetriesUntilConnected(t *testing.T) {
	noHeadset := errors.New("no headset")
	xr := &fakeXR{
		connectErrs: []error{noHeadset, noHeadset},
		feat:        features{fb2: true},
		events:      []sessionEvent{eventOther, eventReady},
		frame:       fb2Frame(),
	}
	m := start(t, xr)

	eventually(t, "tracking data", func() bool {
		return update(t, m).Shapes[tracking.JawOpen].Weight == 0.5
	})
	data := update(t, m)
	if data.Shapes[tracking.TongueOut].Weight != 0.25 {
		t.Errorf("tongue = %v", data.Shapes[tracking.TongueOut].Weight)
	}
	if data.Eye.Left.Openness != 0 || data.Eye.Right.Openness != 1 {
		t.Errorf("openness = %v / %v", data.Eye.Left.Openness, data.Eye.Right.Openness)
	}
	if n := xr.get(&xr.connects); n != 3 {
		t.Errorf("connects = %d, want 3", n)
	}
	if n := xr.get(&xr.begins); n != 1 {
		t.Errorf("begins = %d, want 1", n)
	}
}

func TestNoReadBeforeReady(t *testing.T) {
	xr := &fakeXR{feat: features{fb2: true}, frame: fb2Frame()}
	m := start(t, xr)

	eventually(t, "connect", func() bool { return xr.get(&xr.connects) == 1 })
	time.Sleep(20 * time.Millisecond)
	if n := xr.get(&xr.reads); n != 0 {
		t.Fatalf("reads before the session began = %d", n)
	}

	xr.push(eventReady)
	eventually(t, "reads", func() bool { return xr.get(&xr.reads) > 0 })

	// Stopping ends the session and pauses reading.
	xr.push(eventStopping)
	eventually(t, "end", func() bool { return xr.get(&xr.ends) == 1 })
	before := xr.get(&xr.reads)
	time.Sleep(20 * time.Millisecond)
	if after := xr.get(&xr.reads); after != before {
		t.Errorf("reads went from %d to %d while stopped", before, after)
	}
	_ = m
}

func TestReconnectsAfterSessionEnds(t *testing.T) {
	xr := &fakeXR{feat: features{fb2: true}, events: []sessionEvent{eventReady}, frame: fb2Frame()}
	m := start(t, xr)
	eventually(t, "tracking data", func() bool {
		return update(t, m).Shapes[tracking.JawOpen].Weight == 0.5
	})

	xr.push(eventEnded)
	eventually(t, "disconnect", func() bool { return xr.get(&xr.disconnects) == 1 })
	// A lost session must not leave the last expression frozen.
	eventually(t, "reset", func() bool {
		return update(t, m).Shapes[tracking.JawOpen].Weight == 0
	})

	eventually(t, "reconnect", func() bool { return xr.get(&xr.connects) == 2 })
	xr.push(eventReady)
	eventually(t, "tracking data again", func() bool {
		return update(t, m).Shapes[tracking.JawOpen].Weight == 0.5
	})
}

func TestReconnectsAfterReadFailures(t *testing.T) {
	xr := &fakeXR{feat: features{fb2: true}, events: []sessionEvent{eventReady}, readErr: errors.New("boom")}
	start(t, xr)
	eventually(t, "reconnect", func() bool { return xr.get(&xr.connects) >= 2 })
	if n := xr.get(&xr.reads); n < maxReadFailures {
		t.Errorf("gave up after %d reads, want %d", n, maxReadFailures)
	}
}

func TestLostSessionReconnectsAtOnce(t *testing.T) {
	xr := &fakeXR{
		feat:    features{fb2: true},
		events:  []sessionEvent{eventReady},
		readErr: &xrErrorForTest{},
	}
	start(t, xr)
	eventually(t, "disconnect", func() bool { return xr.get(&xr.disconnects) >= 1 })
	if n := xr.get(&xr.reads); n > 2 {
		t.Errorf("reads before giving up = %d", n)
	}
}

// xrErrorForTest stands in for an OpenXR error that reports a lost session.
type xrErrorForTest struct{}

func (*xrErrorForTest) Error() string        { return "lost" }
func (*xrErrorForTest) Is(target error) bool { return target == errLost }

func TestNoTrackingFeaturesRetries(t *testing.T) {
	xr := &fakeXR{}
	start(t, xr)
	eventually(t, "retry", func() bool { return xr.get(&xr.connects) >= 2 })
	if n := xr.get(&xr.begins); n != 0 {
		t.Errorf("begins = %d", n)
	}
}

func TestTeardown(t *testing.T) {
	xr := &fakeXR{connectErrs: []error{errors.New("no headset")}}
	m := start(t, xr)
	// Teardown must not wait out the retry delay.
	m.retryDelay = time.Hour
	eventually(t, "connect", func() bool { return xr.get(&xr.connects) >= 1 })

	done := make(chan error, 1)
	go func() { done <- m.Teardown() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Teardown did not return")
	}
	if err := m.Teardown(); err != nil {
		t.Errorf("second Teardown: %v", err)
	}
	if err := m.Update(context.Background(), tracking.NewTrackingData()); err == nil {
		t.Error("Update after Teardown succeeded")
	}
}

// blockingXR never returns from connect, like WiVRn while no headset is
// connected.
type blockingXR struct {
	fakeXR
	entered chan struct{}
}

func (b *blockingXR) connect() (features, error) {
	close(b.entered)
	select {}
}

func TestTeardownWhileConnectBlocks(t *testing.T) {
	xr := &blockingXR{entered: make(chan struct{})}
	m := New()
	m.open = func() (xrAPI, error) { return xr, nil }
	if _, err := m.Initialize(context.Background(), zap.NewNop(), adapter.Support{Eye: true}); err != nil {
		t.Fatal(err)
	}
	<-xr.entered

	// Update keeps answering with neutral data while connecting.
	if data := update(t, m); data.Eye.Left.Openness != 1 {
		t.Errorf("openness = %v", data.Eye.Left.Openness)
	}
	start := time.Now()
	if err := m.Teardown(); err == nil {
		t.Error("Teardown reported success while the runtime was blocked")
	}
	if d := time.Since(start); d > teardownTimeout+time.Second {
		t.Errorf("Teardown took %v", d)
	}
}

func TestApplyAndroid(t *testing.T) {
	var f frame
	f.android[fbJawDrop] = 0.5
	f.android[fbEyesClosedR] = 1
	f.android[androidTongueOut] = 0.1
	f.android[androidTongueLeft] = 0.2
	f.android[androidTongueRight] = 0.3
	f.android[androidTongueUp] = 0.4
	f.android[androidTongueDown] = 0.6
	f.androidValid = true

	data := tracking.NewTrackingData()
	f.apply(data)
	want := map[tracking.UnifiedExpression]float32{
		tracking.JawOpen:     0.5,
		tracking.TongueOut:   0.1,
		tracking.TongueLeft:  0.2,
		tracking.TongueRight: 0.3,
		tracking.TongueUp:    0.4,
		tracking.TongueDown:  0.6,
	}
	for shape, w := range want {
		if got := data.Shapes[shape].Weight; got != w {
			t.Errorf("shape %d = %v, want %v", shape, got, w)
		}
	}
	if data.Eye.Right.Openness != 0 || data.Eye.Left.Openness != 1 {
		t.Errorf("openness = %v / %v", data.Eye.Left.Openness, data.Eye.Right.Openness)
	}
}

func TestApplyHTCAndGaze(t *testing.T) {
	var f frame
	f.htcEye[htcLeftBlink] = 1
	f.htcEye[htcRightOut] = 1
	f.htcEyeValid = true
	f.htcLip[htcJawOpen] = 0.75
	f.htcLipValid = true

	data := tracking.NewTrackingData()
	f.apply(data)
	if data.Eye.Left.Openness != 0 || data.Shapes[tracking.JawOpen].Weight != 0.75 {
		t.Errorf("openness = %v, jaw = %v", data.Eye.Left.Openness, data.Shapes[tracking.JawOpen].Weight)
	}
	if data.Eye.Right.Gaze.X != 1 {
		t.Errorf("htc gaze = %+v", data.Eye.Right.Gaze)
	}

	// Looking 30 degrees to the right: a rotation of -30 degrees about +Y.
	// The gaze pose replaces the gaze derived from the HTC weights.
	half := -30 * math.Pi / 180 / 2
	f.gaze = [4]float32{0, float32(math.Sin(half)), 0, float32(math.Cos(half))}
	f.gazeValid = true
	f.apply(data)
	want := float32(30 * math.Pi / 180)
	for _, eye := range []tracking.SingleEyeData{data.Eye.Left, data.Eye.Right} {
		if math.Abs(float64(eye.Gaze.X-want)) > 1e-5 || math.Abs(float64(eye.Gaze.Y)) > 1e-5 {
			t.Errorf("gaze = %+v, want X %v", eye.Gaze, want)
		}
		if eye.PupilDiameterMM != 5 {
			t.Errorf("pupil = %v", eye.PupilDiameterMM)
		}
	}
}

func TestInvalidPartsAreIgnored(t *testing.T) {
	var f frame
	f.fb2[fbJawDrop] = 1
	f.gaze = [4]float32{0, 1, 0, 0}
	data := tracking.NewTrackingData()
	f.apply(data)
	if data.Shapes[tracking.JawOpen].Weight != 0 || data.Eye.Left.Gaze != (tracking.Vector2{}) {
		t.Errorf("invalid frame was applied: jaw %v, gaze %+v", data.Shapes[tracking.JawOpen].Weight, data.Eye.Left.Gaze)
	}
}

// TestLive talks to the real runtime. Run it with WIVRN_LIVE=1 while the
// WiVRn server is up; with a headset connected it also prints tracking data.
func TestLive(t *testing.T) {
	if os.Getenv("WIVRN_LIVE") == "" {
		t.Skip("set WIVRN_LIVE=1 to talk to the running WiVRn server")
	}
	xr, err := openXR()
	if err != nil {
		t.Fatal(err)
	}
	defer xr.disconnect()

	feat, err := xr.connect()
	if err != nil {
		t.Skipf("connect: %v", err)
	}
	t.Logf("features: %+v", feat)
	if !feat.any() {
		t.Skip("the headset offers no face or eye tracking")
	}

	var f frame
	running := false
	valid := 0
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for {
			ev, err := xr.pollEvent()
			if err != nil {
				t.Fatal(err)
			}
			if ev == eventNone {
				break
			}
			if ev == eventReady {
				if err := xr.begin(); err != nil {
					t.Fatal(err)
				}
				running = true
			}
		}
		if running {
			if err := xr.read(&f); err != nil {
				t.Fatal(err)
			}
			if f.fb2Valid || f.androidValid || f.htcEyeValid || f.htcLipValid || f.gazeValid {
				valid++
			}
		}
		time.Sleep(pollInterval)
	}
	if !running {
		t.Fatal("the session never became ready")
	}
	data := tracking.NewTrackingData()
	f.apply(data)
	t.Logf("valid frames: %d; last: fb2=%v android=%v htcEye=%v htcLip=%v gaze=%v %v",
		valid, f.fb2Valid, f.androidValid, f.htcEyeValid, f.htcLipValid, f.gazeValid, f.gaze)
	t.Logf("jaw=%v openness=%v/%v gaze=%+v", data.Shapes[tracking.JawOpen].Weight,
		data.Eye.Left.Openness, data.Eye.Right.Openness, data.Eye.Left.Gaze)
}
