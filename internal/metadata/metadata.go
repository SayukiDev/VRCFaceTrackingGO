// Package metadata exposes the application identity (name, author, version,
// About page links). It is populated once at startup from the embedded
// wails.json, so the same values drive the executable resources, the
// installer and the UI.
package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Author is the maintainer of the application.
type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// AppInfo mirrors the "info" block of wails.json.
type AppInfo struct {
	Description    string `json:"description"`
	CompanyName    string `json:"companyName"`
	ProductName    string `json:"productName"`
	ProductVersion string `json:"productVersion"`
	Copyright      string `json:"copyright"`
	Repository     string `json:"repository"`
	PrivacyURL     string `json:"privacyUrl"`
	IssuesURL      string `json:"issuesUrl"`
}

// Metadata is the subset of wails.json the application reads.
type Metadata struct {
	Name   string  `json:"name"`
	Author Author  `json:"author"`
	Info   AppInfo `json:"info"`
}

// Info holds the parsed metadata. It is valid after Init.
var Info Metadata

// Init parses the embedded wails.json. It panics on malformed input because
// the file is part of the binary and cannot be fixed at runtime.
func Init(metadata []byte) {
	Info = Metadata{}
	if err := json.Unmarshal(metadata, &Info); err != nil {
		panic(err)
	}
}

// PersistentDataDir is the per-user settings folder, %APPDATA%\<Name> on
// Windows.
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
