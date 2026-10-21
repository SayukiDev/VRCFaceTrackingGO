package adapter

import (
	"testing"
	"testing/fstest"
)

func TestLoadMetadata(t *testing.T) {
	fsys := fstest.MapFS{
		"metadata.json": {Data: []byte(`{"name":"Test","icons":["a.png"],"platforms":["windows"]}`)},
		"a.png":         {Data: []byte("png")},
	}
	md, err := LoadMetadata(fsys, "metadata.json")
	if err != nil {
		t.Fatal(err)
	}
	if md.Name != "Test" || len(md.Icons) != 1 || string(md.Icons[0]) != "png" {
		t.Fatalf("metadata = %+v", md)
	}
	if !md.SupportsPlatform("windows") || md.SupportsPlatform("linux") {
		t.Fatalf("platforms = %v", md.Platforms)
	}

	if _, err := LoadMetadata(fstest.MapFS{"metadata.json": {Data: []byte(`{"icons":["missing.png"]}`)}}, "metadata.json"); err == nil {
		t.Fatal("missing icon not reported")
	}
}

func TestSupportsPlatformEmpty(t *testing.T) {
	if !(Metadata{}).SupportsPlatform("darwin") {
		t.Fatal("empty platforms must mean every platform")
	}
}
