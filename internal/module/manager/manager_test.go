package manager

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/plugin"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

// The test binary doubles as a module program when childIDEnv is set; the
// value is the module ID. childFailEnv ("crash" or "hang") makes Update
// fail; with childMarkerEnv set it fails only until that file exists.
const (
	childIDEnv     = "VRCFT_MANAGER_TEST_ID"
	childFailEnv   = "VRCFT_MANAGER_TEST_FAIL"
	childMarkerEnv = "VRCFT_MANAGER_TEST_MARKER"
)

func TestMain(m *testing.M) {
	if id := os.Getenv(childIDEnv); id != "" {
		plugin.Serve(&testModule{id: id})
		return
	}
	os.Exit(m.Run())
}

type testModule struct{ id string }

func (t *testModule) ID() string                 { return t.id }
func (t *testModule) Metadata() adapter.Metadata { return adapter.Metadata{Name: "Module " + t.id} }
func (t *testModule) Supported() adapter.Support { return adapter.Support{Eye: true, Expression: true} }
func (t *testModule) Initialize(_ context.Context, _ *zap.Logger, avail adapter.Support) (adapter.Support, error) {
	return avail, nil
}
func (t *testModule) Update(_ context.Context, d *tracking.TrackingData) error {
	if fail := os.Getenv(childFailEnv); fail != "" && firstFailure() {
		if fail == "hang" {
			select {}
		}
		os.Exit(3)
	}
	d.Shapes[tracking.JawOpen].Weight = 0.5
	return nil
}
func (t *testModule) Teardown() error { return nil }

// program copies the test binary to dir/name as a module program with the
// given ID.
func program(t *testing.T, dir, name, id string) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o755); err != nil {
		t.Fatal(err)
	}
	// Copies report the ID from the environment, so a program has to be
	// inspected before the next one is created.
	t.Setenv(childIDEnv, id)
	return path
}

func ids(infos []Info) []string {
	out := make([]string, 0, len(infos))
	for _, i := range infos {
		out = append(out, i.ID)
	}
	return out
}

func TestScanInstallUninstall(t *testing.T) {
	dir := t.TempDir()
	m := New(zap.NewNop(), dir)
	ctx := context.Background()

	if got := m.Scan(ctx); len(got) != 0 {
		t.Fatalf("empty dir scanned as %v", ids(got))
	}

	// Not a module program: ignored.
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	program(t, dir, "custom-name"+plugin.Suffix, "alpha")
	got := m.Scan(ctx)
	if len(got) != 1 || got[0].ID != "alpha" || got[0].Metadata.Name != "Module alpha" {
		t.Fatalf("scan = %+v", got)
	}

	// Installing the same ID again replaces the old file name.
	src := program(t, t.TempDir(), "download"+plugin.Suffix, "alpha")
	info, err := m.Install(ctx, src)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "alpha"+plugin.Suffix); info.Path != want {
		t.Fatalf("installed at %s, want %s", info.Path, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "custom-name"+plugin.Suffix)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old program not removed: %v", err)
	}

	src = program(t, t.TempDir(), "beta"+plugin.Suffix, "beta")
	if _, err := m.Install(ctx, src); err != nil {
		t.Fatal(err)
	}
	if got := ids(m.Installed()); len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("installed = %v", got)
	}

	if err := m.Uninstall("alpha"); err != nil {
		t.Fatal(err)
	}
	if got := ids(m.Scan(ctx)); len(got) != 1 || got[0] != "beta" {
		t.Fatalf("after uninstall = %v", got)
	}
}

func TestInstallRejectsUnsafeID(t *testing.T) {
	m := New(zap.NewNop(), t.TempDir())
	src := program(t, t.TempDir(), "evil"+plugin.Suffix, `..\evil`)
	if _, err := m.Install(context.Background(), src); err == nil {
		t.Fatal("unsafe ID accepted")
	}
}

func TestStartRunsModuleAndBlocksUninstall(t *testing.T) {
	dir := t.TempDir()
	m := New(zap.NewNop(), dir)
	program(t, dir, "alpha"+plugin.Suffix, "alpha")
	m.Scan(context.Background())

	inst, err := m.Start(context.Background(), "alpha", adapter.Support{Expression: true})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Granted() != (adapter.Support{Expression: true}) {
		t.Fatalf("granted = %+v", inst.Granted())
	}
	deadline := time.Now().Add(5 * time.Second)
	for inst.Latest().Shapes[tracking.JawOpen].Weight != 0.5 {
		if time.Now().After(deadline) {
			t.Fatal("never received data")
		}
		time.Sleep(DefaultUpdateInterval)
	}

	if err := m.Uninstall("alpha"); !errors.Is(err, ErrRunning) {
		t.Fatalf("uninstall while running: %v", err)
	}
	if _, err := m.Start(context.Background(), "alpha", adapter.Support{Eye: true}); !errors.Is(err, ErrRunning) {
		t.Fatalf("second start: %v", err)
	}

	inst.Stop()
	if inst.Crashed() {
		t.Fatal("a stopped module is not a crash")
	}
	if err := m.Uninstall("alpha"); err != nil {
		t.Fatal(err)
	}
}

// firstFailure reports whether the child should fail now: always without
// a marker file, otherwise only until it created the marker.
func firstFailure() bool {
	marker := os.Getenv(childMarkerEnv)
	if marker == "" {
		return true
	}
	if _, err := os.Stat(marker); err == nil {
		return false
	}
	_ = os.WriteFile(marker, nil, 0o644)
	return true
}

// startFailing starts a module that fails as fail says.
func startFailing(t *testing.T, fail string, once bool) (*Manager, *Instance) {
	t.Helper()
	dir := t.TempDir()
	m := New(zap.NewNop(), dir)
	m.restartDelay = 10 * time.Millisecond
	m.updateTimeout = 500 * time.Millisecond
	program(t, dir, "alpha"+plugin.Suffix, "alpha")
	m.Scan(context.Background())

	t.Setenv(childFailEnv, fail)
	if once {
		t.Setenv(childMarkerEnv, filepath.Join(t.TempDir(), "failed"))
	}
	inst, err := m.Start(context.Background(), "alpha", adapter.Support{Eye: true, Expression: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(inst.Stop)
	return m, inst
}

// waitForData waits until the module delivers data again.
func waitForData(t *testing.T, inst *Instance) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for inst.Latest().Shapes[tracking.JawOpen].Weight != 0.5 {
		if time.Now().After(deadline) {
			t.Fatal("module never delivered data")
		}
		select {
		case <-inst.Done():
			t.Fatalf("instance stopped (crashed=%v)", inst.Crashed())
		case <-time.After(DefaultUpdateInterval):
		}
	}
}

func TestCrashedModuleIsRestarted(t *testing.T) {
	_, inst := startFailing(t, "crash", true)
	waitForData(t, inst)
	if inst.Crashed() {
		t.Fatal("restarted module reported as crashed")
	}
}

func TestHungModuleIsRestarted(t *testing.T) {
	_, inst := startFailing(t, "hang", true)
	waitForData(t, inst)
}

func TestModuleThatKeepsCrashingIsGivenUp(t *testing.T) {
	m, inst := startFailing(t, "crash", false)
	select {
	case <-inst.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("module was never given up")
	}
	if !inst.Crashed() {
		t.Fatal("Crashed() = false")
	}
	// A given up module no longer counts as running.
	if err := m.Uninstall("alpha"); err != nil {
		t.Fatal(err)
	}
}
