package oscquery

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/internal/params"
)

// QueryParser resolves the avatar through VRChat's OSCQuery HTTP endpoint.
type QueryParser struct {
	logger *zap.Logger
	client *http.Client
}

func NewQueryParser(logger *zap.Logger) *QueryParser {
	return &QueryParser{
		logger: logger.Named("OscQueryConfigParser"),
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// unknownAvatar fills the avatar name and ID until they are known.
const unknownAvatar = "Unknown"

// Parse fetches /avatar from endpoint and builds the parameter list. The
// avatar name comes from /avatar/name, which VRChat exposes since 2026.4.1;
// older builds report it as unknown.
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

	return info, nil
}

func avatarFromNode(root *Node) (AvatarInfo, bool) {
	parameters, ok := root.Contents["parameters"]
	if !ok {
		return AvatarInfo{}, false
	}
	info := AvatarInfo{Name: unknownAvatar, ID: unknownAvatar}
	if id := stringValue(root.Contents["change"]); id != "" {
		info.ID = id
	}
	// The name set in the SDK (VRChat 2026.4.1 and later).
	if name := stringValue(root.Contents["name"]); name != "" {
		info.Name = name
	}
	info.Parameters = collectDefinitions(parameters.Contents, nil)
	return info, true
}

// stringValue returns the first value of n if it is a string.
func stringValue(n *Node) string {
	if n == nil || len(n.Value) == 0 {
		return ""
	}
	s, _ := n.Value[0].(string)
	return s
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
