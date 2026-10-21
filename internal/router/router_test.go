package router

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"VRCFaceTracking/internal/mixer"
	"VRCFaceTracking/internal/module/adapter"
	"VRCFaceTracking/internal/module/manager"
	"VRCFaceTracking/internal/module/plugin"
	"VRCFaceTracking/pkg/tracking"
)

// fakeModule is an installed module whose start behaviour tests script.
type fakeModule struct {
	id string
	// start decides what is granted; nil grants avail.
	start func(avail adapter.Support) (adapter.Support, error)
	data  func(d *tracking.TrackingData)

	mu      sync.Mutex
	asked   adapter.Support
	running *fakeInstance
}

func (f *fakeModule) askedFor() adapter.Support {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.asked
}

func (f *fakeModule) instance() *fakeInstance {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.running
}

// fakeInstance is a started fakeModule.
type fakeInstance struct {
	granted    adapter.Support
	latest     tracking.TrackingData
	done       chan struct{}
	lastChange time.Time

	mu      sync.Mutex
	polling bool
	crashed bool
	stopped bool
}

func (i *fakeInstance) Granted() adapter.Support      { return i.granted }
func (i *fakeInstance) Latest() tracking.TrackingData { return i.latest }
func (i *fakeInstance) Done() <-chan struct{}         { return i.done }
func (i *fakeInstance) LastChange() time.Time         { return i.lastChange }

func (i *fakeInstance) SetPolling(v bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.polling = v
}

func (i *fakeInstance) Crashed() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.crashed
}

func (i *fakeInstance) Stop() {
	i.mu.Lock()
	defer i.mu.Unlock()
	if !i.stopped {
		i.stopped = true
		if !i.crashed {
			close(i.done)
		}
	}
}

func (i *fakeInstance) crash() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.crashed = true
	close(i.done)
}

func (i *fakeInstance) isPolling() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.polling
}

// fakeHost serves fakeModules in the given order.
type fakeHost []*fakeModule

func (h fakeHost) Installed() []manager.Info {
	out := make([]manager.Info, 0, len(h))
	for _, m := range h {
		out = append(out, manager.Info{Description: plugin.Description{
			ID:        m.id,
			Metadata:  adapter.Metadata{Name: m.id},
			Supported: adapter.Support{Eye: true, Expression: true},
		}})
	}
	return out
}

func (h fakeHost) Start(_ context.Context, id string, avail adapter.Support) (Module, error) {
	for _, m := range h {
		if m.id != id {
			continue
		}
		granted := avail
		if m.start != nil {
			var err error
			if granted, err = m.start(avail); err != nil {
				return nil, err
			}
		}
		inst := &fakeInstance{granted: granted, latest: *tracking.NewTrackingData(), done: make(chan struct{}), lastChange: time.Now()}
		if m.data != nil {
			m.data(&inst.latest)
		}
		m.mu.Lock()
		m.asked, m.running = avail, inst
		m.mu.Unlock()
		return inst, nil
	}
	return nil, errors.New("unknown module")
}

func testRouter(mods ...*fakeModule) *Router {
	return newRouter(zap.NewNop(), mixer.New(mixer.DefaultConfig()), fakeHost(mods))
}

func jawOpen(v float32) func(*tracking.TrackingData) {
	return func(d *tracking.TrackingData) { d.Shapes[tracking.JawOpen].Weight = v }
}

func snapshot(r *Router) *tracking.TrackingData {
	d := tracking.NewTrackingData()
	r.Snapshot(d)
	return d
}

func TestInitializeSkipsFailingModules(t *testing.T) {
	failing := &fakeModule{id: "fail", start: func(adapter.Support) (adapter.Support, error) {
		return adapter.Support{}, errors.New("no device")
	}}
	nothing := &fakeModule{id: "nothing", start: func(adapter.Support) (adapter.Support, error) {
		return adapter.Support{}, nil
	}}
	good := &fakeModule{id: "good", data: jawOpen(1)}

	r := testRouter(failing, nothing, good)
	r.Initialize(context.Background())
	defer r.Teardown()

	if !nothing.instance().stopped {
		t.Fatal("a module that was granted nothing must be stopped")
	}
	mods := r.Modules()
	if len(mods) != 1 || mods[0].ID != "good" {
		t.Fatalf("modules = %+v", mods)
	}
	if got := snapshot(r).Shapes[tracking.JawOpen].Weight; got != 1 {
		t.Fatalf("JawOpen = %v", got)
	}
}

func TestEyeAndExpressionCanComeFromDifferentModules(t *testing.T) {
	a := &fakeModule{id: "a", data: jawOpen(0.7)}
	b := &fakeModule{id: "b", data: func(d *tracking.TrackingData) {
		d.Eye.Left.Openness = 0.3
		d.Shapes[tracking.JawOpen].Weight = 0.1 // not granted, must be ignored
	}}
	r := testRouter(a, b)
	// "a" first but restricted to expression; "b" gets eye.
	r.SetConfig([]Config{
		{ID: "a", Enabled: true, Eye: false, Expression: true},
		{ID: "b", Enabled: true, Eye: true, Expression: true},
	})
	r.Initialize(context.Background())
	defer r.Teardown()

	if got := a.askedFor(); got != (adapter.Support{Expression: true}) {
		t.Fatalf("a was asked for %+v", got)
	}
	if got := b.askedFor(); got != (adapter.Support{Eye: true}) {
		t.Fatalf("b was asked for %+v", got)
	}
	if r.EyeStatus() != adapter.Active || r.ExpressionStatus() != adapter.Active {
		t.Fatalf("status eye=%v expr=%v", r.EyeStatus(), r.ExpressionStatus())
	}
	d := snapshot(r)
	if d.Shapes[tracking.JawOpen].Weight != 0.7 || d.Eye.Left.Openness != 0.3 {
		t.Fatalf("merged jaw=%v openness=%v", d.Shapes[tracking.JawOpen].Weight, d.Eye.Left.Openness)
	}
}

func TestSetConfigOrderAndEnabled(t *testing.T) {
	r := testRouter(&fakeModule{id: "a"}, &fakeModule{id: "b"}, &fakeModule{id: "c"})
	both := func(id string, enabled bool) Config {
		return Config{ID: id, Enabled: enabled, Eye: true, Expression: true}
	}
	// Reverse the order, disable "a", forget "c" and mention a stranger.
	r.SetConfig([]Config{both("b", true), both("stranger", true), both("a", false)})
	r.Initialize(context.Background())
	defer r.Teardown()

	got := r.Config()
	want := []Config{both("b", true), both("a", false), both("c", true)}
	if len(got) != len(want) {
		t.Fatalf("config = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("config[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}

	// "b" is first and claims everything; "a" is disabled; "c" has nothing left.
	mods := r.Modules()
	if len(mods) != 1 || mods[0].ID != "b" {
		t.Fatalf("loaded = %+v", mods)
	}
	cat := r.Catalog()
	if len(cat) != 3 || !cat[0].Loaded || cat[1].Enabled || cat[2].Loaded {
		t.Fatalf("catalog = %+v", cat)
	}
}

func TestSetTrackingMasksOnlyTheSwitchedOffKind(t *testing.T) {
	mod := &fakeModule{id: "both", data: func(d *tracking.TrackingData) {
		d.Eye.Left.Openness = 0.2
		d.Shapes[tracking.JawOpen].Weight = 0.8
	}}
	r := testRouter(mod)
	r.Initialize(context.Background())
	defer r.Teardown()

	r.SetTracking("both", false, true)
	d := snapshot(r)
	if d.Eye.Left.Openness != 1 || d.Shapes[tracking.JawOpen].Weight != 0.8 {
		t.Fatalf("openness=%v jaw=%v", d.Eye.Left.Openness, d.Shapes[tracking.JawOpen].Weight)
	}
	if r.EyeStatus() != adapter.Idle || r.ExpressionStatus() != adapter.Active {
		t.Fatalf("status eye=%v expr=%v", r.EyeStatus(), r.ExpressionStatus())
	}

	r.SetTracking("both", false, false)
	if got := snapshot(r).Shapes[tracking.JawOpen].Weight; got != 0 {
		t.Fatalf("jaw = %v", got)
	}
	if r.Modules()[0].Active || mod.instance().isPolling() {
		t.Fatal("module should be inactive and not polled when both kinds are off")
	}

	r.SetTracking("both", true, true)
	if !mod.instance().isPolling() || snapshot(r).Eye.Left.Openness != 0.2 {
		t.Fatal("tracking did not resume")
	}
}

func TestCrashedModuleIsDropped(t *testing.T) {
	mod := &fakeModule{id: "crash", data: jawOpen(1)}
	r := testRouter(mod)
	r.Initialize(context.Background())
	defer r.Teardown()

	mod.instance().crash()
	deadline := time.Now().Add(time.Second)
	for len(r.Modules()) != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("crashed module still listed: %+v", r.Modules())
		}
		time.Sleep(time.Millisecond)
	}
	if got := snapshot(r).Shapes[tracking.JawOpen].Weight; got != 0 {
		t.Fatalf("jaw = %v", got)
	}
	if r.ExpressionStatus() != adapter.Uninitialized {
		t.Fatalf("expression status = %v", r.ExpressionStatus())
	}
}

func TestUnloadStopsOneModule(t *testing.T) {
	a := &fakeModule{id: "a", start: func(adapter.Support) (adapter.Support, error) {
		return adapter.Support{Expression: true}, nil
	}}
	b := &fakeModule{id: "b"}
	r := testRouter(a, b)
	r.Initialize(context.Background())
	defer r.Teardown()

	r.Unload("a")
	if !a.instance().stopped || b.instance().stopped {
		t.Fatal("wrong module stopped")
	}
	if mods := r.Modules(); len(mods) != 1 || mods[0].ID != "b" {
		t.Fatalf("modules = %+v", mods)
	}
	if r.ExpressionStatus() != adapter.Uninitialized || r.EyeStatus() != adapter.Active {
		t.Fatalf("status eye=%v expr=%v", r.EyeStatus(), r.ExpressionStatus())
	}
}

func TestAssignmentOverridesPriority(t *testing.T) {
	a, b, c := &fakeModule{id: "a"}, &fakeModule{id: "b"}, &fakeModule{id: "c"}
	r := testRouter(a, b, c)
	r.Refresh()
	r.SetConfig([]Config{
		{ID: "a", Enabled: true, Eye: true, Expression: true},
		{ID: "b", Enabled: true, Eye: true, Expression: true},
		{ID: "c", Enabled: false, Eye: true, Expression: true},
	})
	// Eye from the lower "b", face from the disabled "c".
	if err := r.SetAssignment(Assignment{Eye: "b", Expression: "c"}); err != nil {
		t.Fatal(err)
	}
	r.Initialize(context.Background())

	if a.instance() != nil {
		t.Fatal("a has nothing left and must not start")
	}
	if got := b.askedFor(); got != (adapter.Support{Eye: true}) {
		t.Fatalf("b was asked for %+v", got)
	}
	if got := c.askedFor(); got != (adapter.Support{Expression: true}) {
		t.Fatalf("c was asked for %+v", got)
	}

	// Back to automatic: "a" takes everything again.
	if err := r.SetAssignment(Assignment{}); err != nil {
		t.Fatal(err)
	}
	r.Initialize(context.Background())
	defer r.Teardown()
	if mods := r.Modules(); len(mods) != 1 || mods[0].ID != "a" {
		t.Fatalf("modules = %+v", mods)
	}

	if err := r.SetAssignment(Assignment{Eye: "stranger"}); err == nil {
		t.Fatal("unknown module accepted")
	}
}

func TestStaleModuleIsReported(t *testing.T) {
	mod := &fakeModule{id: "a"}
	r := testRouter(mod)
	notified := 0
	r.OnChange = func([]ModuleInfo) { notified++ }
	r.Initialize(context.Background())
	defer r.Teardown()
	start := mod.instance().lastChange

	notified = 0
	r.checkStale(start.Add(staleAfter / 2))
	if r.Modules()[0].Stale || notified != 0 {
		t.Fatal("fresh module reported as stale")
	}
	r.checkStale(start.Add(2 * staleAfter))
	if !r.Modules()[0].Stale || notified != 1 {
		t.Fatalf("stale module not reported (notified %d)", notified)
	}
	r.checkStale(start.Add(3 * staleAfter))
	if notified != 1 {
		t.Fatal("unchanged staleness notified again")
	}

	// A module switched off is not stale.
	r.SetTracking("a", false, false)
	if r.Modules()[0].Stale {
		t.Fatal("inactive module reported as stale")
	}
}
