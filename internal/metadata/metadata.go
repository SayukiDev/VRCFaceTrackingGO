// Package metadata exposes the application identity (name, author, version,
// About page links). It is populated once at startup from the embedded
// build/config.yml, so the same values drive the executable resources, the
// installer and the UI.
package metadata

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Author is the maintainer of the application.
type Author struct {
	Name  string `json:"name" yaml:"name"`
	Email string `json:"email" yaml:"email"`
}

// AppInfo mirrors the "info" block of build/config.yml.
type AppInfo struct {
	Description    string `json:"description" yaml:"description"`
	CompanyName    string `json:"companyName" yaml:"companyName"`
	ProductName    string `json:"productName" yaml:"productName"`
	ProductVersion string `json:"productVersion" yaml:"version"`
	Copyright      string `json:"copyright" yaml:"copyright"`
	Repository     string `json:"repository" yaml:"repository"`
	NoticesURL     string `json:"noticesUrl" yaml:"noticesUrl"`
	IssuesURL      string `json:"issuesUrl" yaml:"issuesUrl"`
}

// Metadata is the subset of build/config.yml the application reads.
type Metadata struct {
	Name   string  `json:"name" yaml:"name"`
	Author Author  `json:"author" yaml:"author"`
	Info   AppInfo `json:"info" yaml:"info"`
}

// Info holds the parsed metadata. It is valid after Init.
var Info Metadata

// Init parses the embedded build/config.yml. It panics on malformed input because
// the file is part of the binary and cannot be fixed at runtime.
func Init(metadata []byte) {
	Info = Metadata{}
	if err := yaml.Unmarshal(metadata, &Info); err != nil {
		panic(err)
	}
}

// PersistentDataDir is the per-user settings folder: %APPDATA%\<Name> on
// Windows, ~/.config/<Name> on Linux and ~/Library/Application
// Support/<Name> on macOS.
func (m Metadata) PersistentDataDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, m.Name), nil
}

// LogDir is Documents\<Name>, where users can find latest.log.
func (m Metadata) LogDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return m.Name
	}
	return filepath.Join(home, "Documents", m.Name)
}

// IsDev reports whether debug logging was requested via VRCFT_DEBUG.
func (m Metadata) IsDev() bool { return os.Getenv("VRCFT_DEBUG") != "" }
