package manager

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/plugin"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

// The test binary doubles as a module program when childIDEnv is set; the
// value is the module ID. With childDriverEnv also set it is a driver
// program instead, containing the comma separated module IDs. childFailEnv ("crash" or "hang") makes Update
// fail; with childMarkerEnv set it fails only until that file exists.
const (
	childIDEnv     = "VRCFT_MANAGER_TEST_ID"
	childDriverEnv = "VRCFT_MANAGER_TEST_DRIVER"
	childFailEnv   = "VRCFT_MANAGER_TEST_FAIL"
	childMarkerEnv = "VRCFT_MANAGER_TEST_MARKER"
)

func TestMain(m *testing.M) {
	if id := os.Getenv(childIDEnv); id != "" {
		if mods := os.Getenv(childDriverEnv); mods != "" {
			var list []adapter.Module
			for _, m := range strings.Split(mods, ",") {
				list = append(list, &testModule{id: m})
			}
			plugin.ServeDriver(plugin.Driver{ID: id, Metadata: adapter.Metadata{Name: "Driver " + id}}, list...)
			return
		}
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
	d.Eye.Left.Openness = 0.25
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
	t.Setenv(childDriverEnv, "")
	return path
}

// driver is program for a driver program containing mods.
func driver(t *testing.T, dir, name, id string, mods ...string) string {
	t.Helper()
	path := program(t, dir, name, id)
	t.Setenv(childDriverEnv, strings.Join(mods, ","))
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
	program(t, dir, "custom-name"+plugin.HostSuffix, "alpha")
	got := m.Scan(ctx)
	if len(got) != 1 || got[0].ID != "alpha" || got[0].Metadata.Name != "Module alpha" {
		t.Fatalf("scan = %+v", got)
	}

	// Installing the same ID again replaces the old file name.
	src := program(t, t.TempDir(), "download"+plugin.HostSuffix, "alpha")
	if _, err := m.Install(ctx, src); err != nil {
		t.Fatal(err)
	}
	if info, _ := m.Info("alpha"); info.Path != filepath.Join(dir, "alpha"+plugin.HostSuffix) {
		t.Fatalf("installed at %s", info.Path)
	}
	if _, err := os.Stat(filepath.Join(dir, "custom-name"+plugin.HostSuffix)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old program not removed: %v", err)
	}

	src = program(t, t.TempDir(), "beta"+plugin.HostSuffix, "beta")
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

func TestOtherPlatformIgnored(t *testing.T) {
	dir := t.TempDir()
	m := New(zap.NewNop(), dir)
	ctx := context.Background()

	other := plugin.PlatformSuffix("plan9", "mips")
	program(t, dir, "alpha"+other, "alpha")
	program(t, dir, "beta"+plugin.Suffix, "beta")
	if got := ids(m.Scan(ctx)); len(got) != 0 {
		t.Fatalf("scanned %v", got)
	}
	src := program(t, t.TempDir(), "gamma"+other, "gamma")
	if _, err := m.Install(ctx, src); err == nil {
		t.Fatal("program for another platform installed")
	}
}

func TestInstallRejectsUnsafeID(t *testing.T) {
	m := New(zap.NewNop(), t.TempDir())
	src := program(t, t.TempDir(), "evil"+plugin.HostSuffix, `..\evil`)
	if _, err := m.Install(context.Background(), src); err == nil {
		t.Fatal("unsafe ID accepted")
	}
}

func TestStartRunsModuleAndBlocksUninstall(t *testing.T) {
	dir := t.TempDir()
	m := New(zap.NewNop(), dir)
	program(t, dir, "alpha"+plugin.HostSuffix, "alpha")
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

func TestDriverModules(t *testing.T) {
	dir := t.TempDir()
	m := New(zap.NewNop(), dir)
	ctx := context.Background()
	program(t, dir, "solo"+plugin.HostSuffix, "solo")
	m.Scan(ctx)
	// Last, so the started processes are the driver.
	driver(t, dir, "drv"+plugin.HostSuffix, "drv", "eye", "face")
	m.Scan(ctx)

	if got := ids(m.Installed()); strings.Join(got, ",") != "eye,face,solo" {
		t.Fatalf("installed = %v", got)
	}
	if info, _ := m.Info("face"); info.Driver != "drv" {
		t.Fatalf("face belongs to %q", info.Driver)
	}
	if d := m.Drivers(); len(d) != 1 || d[0].ID != "drv" || len(d[0].Modules) != 2 {
		t.Fatalf("drivers = %+v", d)
	}
	if _, err := m.Start(ctx, "drv", adapter.Support{Eye: true}); err == nil {
		t.Fatal("a driver itself was started")
	}
	if err := m.Uninstall("eye"); err == nil {
		t.Fatal("a module inside a driver was uninstalled on its own")
	}

	inst, err := m.Start(ctx, "face", adapter.Support{Expression: true})
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
	// Another module of the same driver runs in its own process.
	other, err := m.Start(ctx, "eye", adapter.Support{Eye: true})
	if err != nil {
		t.Fatal(err)
	}
	other.Stop()
	if err := m.Uninstall("drv"); !errors.Is(err, ErrRunning) {
		t.Fatalf("uninstall while running: %v", err)
	}

	inst.Stop()

	// Another program may not take a module ID of the driver.
	src := program(t, t.TempDir(), "face"+plugin.HostSuffix, "face")
	if _, err := m.Install(ctx, src); err == nil {
		t.Fatal("duplicate module ID installed")
	}
	if err := m.Uninstall("drv"); err != nil {
		t.Fatal(err)
	}
	if got := ids(m.Installed()); len(got) != 1 || got[0] != "solo" {
		t.Fatalf("after uninstall = %v", got)
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
	program(t, dir, "alpha"+plugin.HostSuffix, "alpha")
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
