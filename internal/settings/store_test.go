package settings

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"VRCFaceTracking/internal/sender"
)

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	s.Update(func(c *Settings) {
		c.OSC.OutPort = 9100
		c.Mixer.Filter.MinCutoff = 1.5
	})
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}

	got := mustOpen(t, dir).Get()
	if got.OSC.OutPort != 9100 || got.Mixer.Filter.MinCutoff != 1.5 {
		t.Fatalf("settings not persisted: %+v", got)
	}
	if got.Theme != "dark" {
		t.Fatalf("default not applied: %q", got.Theme)
	}
}

func TestStoreFillsMissingFieldsWithDefaults(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, fileName), `{"locale":"ja"}`)

	got := mustOpen(t, dir).Get()
	if got.Locale != "ja" {
		t.Fatalf("locale = %q", got.Locale)
	}
	if want := Default(); got.OSC != want.OSC || got.Timing != want.Timing || got.Output != want.Output {
		t.Fatalf("defaults not applied: %+v", got)
	}
}

func TestStoreRevertsInvalidSections(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, fileName), `{"output":"bogus","osc":{"inPort":0},"timing":{"sendIntervalMs":5000}}`)

	got := mustOpen(t, dir).Get()
	if want := Default(); got.OSC != want.OSC || got.Timing != want.Timing || got.Output != sender.OutputParameters {
		t.Fatalf("invalid sections kept: %+v", got)
	}
}

func TestStoreRestoresBackupWhenCorrupt(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, fileName+".bak"), `{"locale":"ja"}`)
	writeFile(t, filepath.Join(dir, fileName), `{not json`)

	if got := mustOpen(t, dir).Get(); got.Locale != "ja" {
		t.Fatalf("backup not restored, got %q", got.Locale)
	}
}

func mustOpen(t *testing.T, dir string) *Store {
	t.Helper()
	s, err := Open(dir, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
