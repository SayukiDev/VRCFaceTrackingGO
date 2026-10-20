package settings

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	type target struct {
		Port int `json:"port"`
	}
	if err := s.Set("k", target{Port: 9000}); err != nil {
		t.Fatal(err)
	}
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dir, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	if got := GetOr(reopened, "k", target{}); got.Port != 9000 {
		t.Fatalf("got %+v", got)
	}
	if got := GetOr(reopened, "missing", target{Port: 1}); got.Port != 1 {
		t.Fatalf("default not applied: %+v", got)
	}
}

func TestStoreRestoresBackupWhenCorrupt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, fileName+".bak"), []byte(`{"k":"1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(`{not json`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	if got := GetOr(s, "k", ""); got != "1" {
		t.Fatalf("backup not restored, got %q", got)
	}
}
