package plugin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

// The test binary doubles as a module program when childEnv is set.
// childPanicEnv makes its first Update panic, childHangEnv makes
// Initialize never return. With hostEnv set it acts as a host instead: it
// starts a hanging module, prints its PID and waits to be killed.
const (
	childEnv      = "VRCFT_PLUGIN_TEST_CHILD"
	childPanicEnv = "VRCFT_PLUGIN_TEST_PANIC"
	childHangEnv  = "VRCFT_PLUGIN_TEST_HANG"
	hostEnv       = "VRCFT_PLUGIN_TEST_HOST"
)

func TestMain(m *testing.M) {
	if os.Getenv(childEnv) != "" {
		Serve(&testModule{})
		return
	}
	if os.Getenv(hostEnv) != "" {
		runHangingHost()
		return
	}
	os.Exit(m.Run())
}

func runHangingHost() {
	exe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	os.Setenv(childEnv, "1")
	os.Setenv(childHangEnv, "1")
	c, err := Start(exe, "", zap.NewNop())
	if err != nil {
		panic(err)
	}
	go func() { _, _ = c.Initialize(context.Background(), adapter.Support{Eye: true}) }()
	fmt.Println(c.cmd.Process.Pid)
	// Sleep rather than select{}: the runtime would call that a deadlock
	// and end the process.
	for {
		time.Sleep(time.Hour)
	}
}

type testModule struct{ panicked bool }

func (*testModule) ID() string { return "test" }
func (*testModule) Metadata() adapter.Metadata {
	return adapter.Metadata{Name: "Test", Icons: [][]byte{{1, 2, 3}}}
}
func (*testModule) Supported() adapter.Support { return adapter.Support{Eye: true, Expression: true} }

func (*testModule) Initialize(_ context.Context, logger *zap.Logger, avail adapter.Support) (adapter.Support, error) {
	if os.Getenv(childHangEnv) != "" {
		for { // stuck: never sees stdin close
			time.Sleep(time.Hour)
		}
	}
	logger.Named("Test").Warn("Initializing", zap.Bool("eye", avail.Eye))
	if !avail.Eye && !avail.Expression {
		return adapter.Support{}, errors.New("nothing to do")
	}
	return adapter.Support{Expression: avail.Expression}, nil
}

func (m *testModule) Update(_ context.Context, d *tracking.TrackingData) error {
	if os.Getenv(childPanicEnv) != "" && !m.panicked {
		m.panicked = true
		panic("boom")
	}
	d.Shapes[tracking.JawOpen].Weight += 0.25
	return nil
}

func (*testModule) Teardown() error { return nil }

func startChild(t *testing.T, logger *zap.Logger) *Client {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(childEnv, "1")
	c, err := Start(exe, "", logger)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestClientServerRoundTrip(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	c := startChild(t, zap.New(core))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := c.Initialize(ctx, adapter.Support{}); err == nil || err.Error() != "nothing to do" {
		t.Fatalf("module error not forwarded: %v", err)
	}
	granted, err := c.Initialize(ctx, adapter.Support{Eye: true, Expression: true})
	if err != nil || granted != (adapter.Support{Expression: true}) {
		t.Fatalf("granted = %+v, err = %v", granted, err)
	}

	// The module keeps its data between calls.
	for _, want := range []float32{0.25, 0.5} {
		d, err := c.Update(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got := d.Shapes[tracking.JawOpen].Weight; got != want {
			t.Fatalf("JawOpen = %v, want %v", got, want)
		}
		if d.Eye.Left.Openness != 1 {
			t.Fatalf("eye openness = %v", d.Eye.Left.Openness)
		}
	}

	if err := c.Teardown(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Update(ctx); !errors.Is(err, ErrExited) {
		t.Fatalf("update after close: %v", err)
	}

	found := logs.FilterMessage("Initializing").FilterLoggerName("Test").All()
	if len(found) == 0 || found[0].Level != zap.WarnLevel || found[0].ContextMap()["eye"] != false {
		t.Fatalf("module log not forwarded: %+v", logs.All())
	}
}

func TestReadInfo(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(childEnv, "1")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	desc, err := ReadInfo(ctx, exe)
	if err != nil {
		t.Fatal(err)
	}
	if desc.ID != "test" || desc.Metadata.Name != "Test" || len(desc.Metadata.Icons) != 1 || desc.Metadata.Icons[0][2] != 3 || !desc.Supported.Eye {
		t.Fatalf("description = %+v", desc)
	}
}

func TestRefusesToRunWithoutHostFlag(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(childEnv, "1")
	cmd := exec.Command(exe)
	cmd.Stdin = strings.NewReader("") // would end Serve at once if it ran
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 2 {
		t.Fatalf("err = %v, output = %s", err, out)
	}
	if !strings.Contains(string(out), "is a VRCFaceTracking module") {
		t.Fatalf("output = %s", out)
	}
}

func TestPanicIsReportedAndSurvived(t *testing.T) {
	t.Setenv(childPanicEnv, "1")
	core, logs := observer.New(zap.DebugLevel)
	c := startChild(t, zap.New(core))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := c.Update(ctx); err == nil || !strings.Contains(err.Error(), "module panicked in Update: boom") {
		t.Fatalf("first update: %v", err)
	}
	d, err := c.Update(ctx)
	if err != nil {
		t.Fatalf("process did not survive the panic: %v", err)
	}
	if d.Shapes[tracking.JawOpen].Weight != 0.25 {
		t.Fatalf("JawOpen = %v", d.Shapes[tracking.JawOpen].Weight)
	}
	// The log arrives on stderr independently of the response.
	deadline := time.Now().Add(5 * time.Second)
	for logs.FilterMessage("Module panicked").Len() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("panic not logged")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// A module stuck in a call never notices stdin closing; the platform
// mechanism (job object, Pdeathsig, parent watch) must still end it when
// the host dies.
func TestStuckModuleDiesWithHost(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(hostEnv, "1")
	host := exec.Command(exe)
	out, err := host.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := host.Start(); err != nil {
		t.Fatal(err)
	}
	var pid int32
	if _, err := fmt.Fscan(bufio.NewReader(out), &pid); err != nil {
		_ = host.Process.Kill()
		t.Fatal(err)
	}
	// Let the module get stuck in Initialize.
	time.Sleep(500 * time.Millisecond)
	if alive, _ := process.PidExists(pid); !alive {
		t.Fatal("module did not start")
	}

	if err := host.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = host.Wait()
	deadline := time.Now().Add(5 * time.Second)
	for {
		alive, err := process.PidExists(pid)
		if err != nil {
			t.Fatal(err)
		}
		if !alive {
			return
		}
		if time.Now().After(deadline) {
			_ = killPID(pid)
			t.Fatal("stuck module outlived its host")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func killPID(pid int32) error {
	p, err := os.FindProcess(int(pid))
	if err != nil {
		return err
	}
	return p.Kill()
}
