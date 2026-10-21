package oscquery

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"

	"VRCFaceTrackingGo/internal/params"
)

// QueryParser resolves the avatar through VRChat's OSCQuery HTTP endpoint.
type QueryParser struct {
	logger *zap.Logger
	files  *FileParser
	client *http.Client
}

func NewQueryParser(logger *zap.Logger, files *FileParser) *QueryParser {
	return &QueryParser{
		logger: logger.Named("OscQueryConfigParser"),
		files:  files,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Parse fetches /avatar from endpoint and builds the parameter list. The
// avatar name is looked up in the legacy config file because OSCQuery does
// not expose it.
func (p *QueryParser) Parse(endpoint *net.TCPAddr) (AvatarInfo, error) {
	if endpoint == nil {
		return AvatarInfo{}, errors.New("no OSCQuery endpoint")
	}
	resp, err := p.client.Get("http://" + endpoint.String() + "/avatar")
	if err != nil {
		return AvatarInfo{}, fmt.Errorf("query avatar: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return AvatarInfo{}, fmt.Errorf("query avatar: status %s", resp.Status)
	}

	var root Node
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return AvatarInfo{}, fmt.Errorf("decode avatar node: %w", err)
	}

	info, ok := avatarFromNode(&root)
	if !ok {
		return AvatarInfo{}, errors.New("avatar node has no parameters")
	}

	p.logger.Info("Attempting to resolve avatar config file", zap.String("avatarId", info.ID))
	if fileInfo, err := p.files.Parse(info.ID); err == nil && fileInfo.Name != "" {
		info.Name = fileInfo.Name
		p.logger.Info("Successfully found config containing avatar name", zap.String("name", info.Name))
	} else {
		if err != nil {
			p.logger.Warn("Odd. Parse config for this avatar failed.", zap.String("avatarId", info.ID), zap.Error(err))
		} else {
			p.logger.Warn("Odd. Our attempt to find the legacy osc config json for this avatar failed.", zap.String("avatarId", info.ID))
		}
	}
	return info, nil
}

func avatarFromNode(root *Node) (AvatarInfo, bool) {
	parameters, ok := root.Contents["parameters"]
	if !ok {
		return AvatarInfo{}, false
	}
	info := AvatarInfo{Name: "Unknown", ID: "Unknown"}
	if change, ok := root.Contents["change"]; ok && len(change.Value) > 0 {
		if id, ok := change.Value[0].(string); ok {
			info.ID = id
		}
	}
	info.Parameters = collectDefinitions(parameters.Contents, nil)
	return info, true
}

func collectDefinitions(entries map[string]*Node, out []params.Definition) []params.Definition {
	for _, node := range entries {
		if node.Contents != nil {
			out = collectDefinitions(node.Contents, out)
			continue
		}
		if node.OSCType == "" {
			continue
		}
		out = append(out, params.Definition{
			Address: node.FullPath,
			Name:    node.name(),
			Type:    params.ValueTypeFromOSCTag(node.OSCType[0]),
		})
	}
	return out
}
