package adapter

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

// metadataFile is the JSON form of Metadata kept next to a module's source.
// Icons are file names in the same file system instead of PNG data.
type metadataFile struct {
	Metadata
	Icons []string `json:"icons"`
}

// LoadMetadata reads the JSON metadata file name from fsys, typically an
// embed.FS holding the file and the icons it names.
func LoadMetadata(fsys fs.FS, name string) (Metadata, error) {
	raw, err := fs.ReadFile(fsys, name)
	if err != nil {
		return Metadata{}, err
	}
	var f metadataFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return Metadata{}, fmt.Errorf("decode %s: %w", name, err)
	}
	md := f.Metadata
	for _, icon := range f.Icons {
		png, err := fs.ReadFile(fsys, icon)
		if err != nil {
			return Metadata{}, fmt.Errorf("icon of %s: %w", name, err)
		}
		md.Icons = append(md.Icons, png)
	}
	return md, nil
}

// MustLoadMetadata is LoadMetadata for files compiled into the module. It
// panics on error because such a file cannot be fixed at runtime.
func MustLoadMetadata(fsys fs.FS, name string) Metadata {
	md, err := LoadMetadata(fsys, name)
	if err != nil {
		panic(err)
	}
	return md
}
