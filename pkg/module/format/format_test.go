package format

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/plugin"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

// The test binary doubles as a module program when childIDEnv is set; the
// value is the ID it reports.
const childIDEnv = "VRCFT_FORMAT_TEST_ID"

func TestMain(m *testing.M) {
	if id := os.Getenv(childIDEnv); id != "" {
		plugin.Serve(testModule(id))
		return
	}
	os.Exit(m.Run())
}

type testModule string

func (t testModule) ID() string                 { return string(t) }
func (t testModule) Metadata() adapter.Metadata { return adapter.Metadata{Name: string(t)} }
func (t testModule) Supported() adapter.Support { return adapter.Support{Eye: true} }
func (t testModule) Initialize(context.Context, *zap.Logger, adapter.Support) (adapter.Support, error) {
	return adapter.Support{}, nil
}
func (t testModule) Update(context.Context, *tracking.TrackingData) error { return nil }
func (t testModule) Teardown() error                                      { return nil }

// selfProgram copies the test binary to dir/name; it reports id.
func selfProgram(t *testing.T, dir, name, id string) string {
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
	t.Setenv(childIDEnv, id)
	return path
}

func TestValidate(t *testing.T) {
	ctx := context.Background()
	good := selfProgram(t, t.TempDir(), "good"+plugin.HostSuffix, "Good")
	desc, err := Validate(ctx, good)
	if err != nil || desc.ID != "Good" || !desc.Supported.Eye {
		t.Fatalf("validate = %+v, %v", desc, err)
	}

	bad := selfProgram(t, t.TempDir(), "bad"+plugin.HostSuffix, `..\evil`)
	if _, err := Validate(ctx, bad); err == nil {
		t.Fatal("program with an unsafe ID accepted")
	}

	notProgram := writeFile(t, "text"+plugin.HostSuffix, "not a program")
	if _, err := Validate(ctx, notProgram); err == nil {
		t.Fatal("non-program accepted")
	}
}

func TestValidateStagedArchive(t *testing.T) {
	program := selfProgram(t, t.TempDir(), "p", "Archived")
	data, err := os.ReadFile(program)
	if err != nil {
		t.Fatal(err)
	}
	src := archive(t, t.TempDir(), "x"+plugin.HostFolderSuffix, map[string]string{plugin.FolderEntry: string(data)})
	s, err := Stage(t.TempDir(), src)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if desc, err := s.Validate(context.Background()); err != nil || desc.ID != "Archived" {
		t.Fatalf("validate = %+v, %v", desc, err)
	}
}

// archive writes a zip with the given files (name → content) to dir/name.
func archive(t *testing.T, dir, name string, files map[string]string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for n, content := range files {
		fw, err := w.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestStageAndInstallArchive(t *testing.T) {
	dir := t.TempDir()
	src := archive(t, t.TempDir(), "x"+plugin.HostFolderSuffix, map[string]string{
		plugin.FolderEntry: "program",
		"lib/data.txt":     "data",
	})

	s, err := Stage(dir, src)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if !s.Folder || filepath.Base(s.Program) != plugin.FolderEntry {
		t.Fatalf("staged %+v", s.Location)
	}
	loc, err := s.Install(dir, "mod")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "mod"+plugin.HostFolderSuffix); loc.Root != want || !loc.Folder {
		t.Fatalf("installed %+v", loc)
	}
	if b, err := os.ReadFile(filepath.Join(loc.Root, "lib", "data.txt")); err != nil || string(b) != "data" {
		t.Fatalf("data.txt = %q, %v", b, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	// Only the module folder is left, and Locate recognises it.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("left behind: %v", entries)
	}
	if got, ok := Locate(dir, entries[0]); !ok || got != loc {
		t.Fatalf("located %+v, %v", got, ok)
	}
}

func TestStageProgramFile(t *testing.T) {
	src := filepath.Join(t.TempDir(), "x"+plugin.HostSuffix)
	if err := os.WriteFile(src, []byte("program"), 0o755); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	s, err := Stage(dir, src)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.Folder || s.Program != src {
		t.Fatalf("staged %+v", s.Location)
	}
	loc, err := s.Install(dir, "mod")
	if err != nil {
		t.Fatal(err)
	}
	if loc.Root != filepath.Join(dir, "mod"+plugin.HostSuffix) || loc.Program != loc.Root {
		t.Fatalf("installed %+v", loc)
	}
}

func TestStageRejects(t *testing.T) {
	dir := t.TempDir()
	for name, src := range map[string]string{
		"other platform": filepath.Join(t.TempDir(), "x"+plugin.PlatformFolderSuffix("plan9", "mips")),
		"no entry":       archive(t, t.TempDir(), "x"+plugin.HostFolderSuffix, map[string]string{"readme.txt": ""}),
		"not a zip":      writeFile(t, "x"+plugin.HostFolderSuffix, "not a zip"),
	} {
		if _, err := Stage(dir, src); err == nil {
			t.Errorf("%s staged", name)
		}
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("left behind: %v", entries)
	}
}

func TestExtractStaysInside(t *testing.T) {
	root := t.TempDir()
	dst := filepath.Join(root, "dst")
	if err := os.Mkdir(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	src := archive(t, t.TempDir(), "x.zip", map[string]string{
		plugin.FolderEntry: "program",
		"../evil.txt":      "evil",
	})
	_ = Extract(src, dst)
	if _, err := os.Stat(filepath.Join(root, "evil.txt")); err == nil {
		t.Fatal("archive wrote outside its folder")
	}
}

func TestValidateIDs(t *testing.T) {
	ok := plugin.Description{ID: "Drv", Modules: []plugin.Description{{ID: "a.b-c_1"}}}
	if err := ValidateIDs(ok); err != nil {
		t.Fatal(err)
	}
	for _, d := range []plugin.Description{
		{ID: `..\evil`},
		{ID: "Drv", Modules: []plugin.Description{{ID: "a/b"}}},
		{ID: ".hidden"},
	} {
		if err := ValidateIDs(d); err == nil {
			t.Errorf("%+v accepted", d)
		}
	}
}

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
