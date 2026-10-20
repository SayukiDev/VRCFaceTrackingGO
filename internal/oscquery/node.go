package oscquery

import (
	"encoding/json"
	"strings"
)

// Access values as defined by the OSCQuery spec.
const (
	AccessNoValue   = 0
	AccessReadOnly  = 1
	AccessWriteOnly = 2
	AccessReadWrite = 3
)

// Node is an OSCQuery tree node.
type Node struct {
	Description string           `json:"DESCRIPTION,omitempty"`
	FullPath    string           `json:"FULL_PATH"`
	Access      int              `json:"ACCESS"`
	Contents    map[string]*Node `json:"CONTENTS,omitempty"`
	OSCType     string           `json:"TYPE,omitempty"`
	Value       []any            `json:"VALUE,omitempty"`
}

func (n *Node) name() string {
	return n.FullPath[strings.LastIndex(n.FullPath, "/")+1:]
}

func (n *Node) parentPath() string {
	idx := strings.LastIndex(n.FullPath, "/")
	if idx < 1 {
		return "/"
	}
	return n.FullPath[:idx]
}

// Root is a Node tree with a path index for quick insertion.
type Root struct {
	*Node
	nodes map[string]*Node
}

func NewRoot() *Root {
	root := &Node{FullPath: "/"}
	return &Root{Node: root, nodes: map[string]*Node{"/": root}}
}

// Add inserts node, creating intermediate containers as needed.
func (r *Root) Add(node *Node) *Node {
	parent, ok := r.nodes[node.parentPath()]
	if !ok {
		parent = r.Add(&Node{FullPath: node.parentPath()})
	}
	if parent.Contents == nil {
		parent.Contents = map[string]*Node{}
	}
	parent.Contents[node.name()] = node
	r.nodes[node.FullPath] = node
	return node
}

func (r *Root) JSON() ([]byte, error) { return json.Marshal(r.Node) }

// HostInfo is the HOST_INFO response.
type HostInfo struct {
	Name         string          `json:"NAME"`
	Extensions   map[string]bool `json:"EXTENSIONS"`
	OSCIP        string          `json:"OSC_IP"`
	OSCPort      int             `json:"OSC_PORT"`
	OSCTransport string          `json:"OSC_TRANSPORT"`
}

func newHostInfo(name, ip string, port int) HostInfo {
	return HostInfo{
		Name: name,
		Extensions: map[string]bool{
			"ACCESS": true, "CLIPMODE": false, "RANGE": true, "TYPE": true, "VALUE": true,
		},
		OSCIP:        ip,
		OSCPort:      port,
		OSCTransport: "UDP",
	}
}
