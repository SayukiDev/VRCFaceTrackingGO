package oscquery

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"VRCFaceTrackingGo/internal/params"
)

// avatarConfigFile is the JSON VRChat writes to OSC\<user>\Avatars\<id>.json.
type avatarConfigFile struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Parameters []struct {
		Name  string `json:"name"`
		Input *struct {
			Address string `json:"address"`
			Type    string `json:"type"`
		} `json:"input"`
	} `json:"parameters"`
}

func (f *avatarConfigFile) toAvatarInfo() AvatarInfo {
	info := AvatarInfo{Name: f.Name, ID: f.ID}
	for _, p := range f.Parameters {
		if p.Input == nil {
			continue
		}
		info.Parameters = append(info.Parameters, params.Definition{
			Address: p.Input.Address,
			Name:    p.Name,
			Type:    params.ValueTypeFromConfig(p.Input.Type),
		})
	}
	return info
}

var errAvatarNotFound = errors.New("avatar config not found")

// FileParser resolves avatars from the legacy OSC config files.
type FileParser struct {
	logger *zap.Logger
	oscDir string
}

func NewFileParser(logger *zap.Logger, oscDir string) *FileParser {
	return &FileParser{logger: logger.Named("AvatarConfigParser"), oscDir: oscDir}
}

// Parse looks for the config file whose id matches avatarID.
func (p *FileParser) Parse(avatarID string) (AvatarInfo, error) {
	if avatarID == "" {
		return AvatarInfo{}, errAvatarNotFound
	}
	userDirs, err := os.ReadDir(p.oscDir)
	if err != nil {
		return AvatarInfo{}, fmt.Errorf("read OSC directory: %w", err)
	}
	for _, user := range userDirs {
		if !user.IsDir() {
			continue
		}
		avatarDir := filepath.Join(p.oscDir, user.Name(), "Avatars")
		files, err := filepath.Glob(filepath.Join(avatarDir, "*.json"))
		if err != nil {
			continue
		}
		for _, file := range files {
			if !strings.Contains(file, avatarID) {
				continue
			}
			cfg, err := p.readConfig(file)
			if err != nil {
				return AvatarInfo{}, err
			}
			if cfg.ID != avatarID {
				p.logger.Warn("Avatar ID mismatch", zap.String("avatarId", avatarID), zap.String("fileId", cfg.ID))
				continue
			}
			p.logger.Info("Parsing config file for avatar", zap.String("avatar", cfg.Name))
			return cfg.toAvatarInfo(), nil
		}
	}
	return AvatarInfo{}, errAvatarNotFound
}

func (p *FileParser) readConfig(file string) (*avatarConfigFile, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	// Remove UTF-8 BOM.
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var cfg avatarConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		if err := os.RemoveAll(file); err != nil {
			p.logger.Error("Failed to remove malformed JSON file", zap.String("file", file), zap.Error(err))
		}
		return nil, err
	}
	return &cfg, nil
}
