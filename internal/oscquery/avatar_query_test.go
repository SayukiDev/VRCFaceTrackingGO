package oscquery

import (
	"encoding/json"
	"fmt"
	"testing"
)

const avatarJSON = `{
  "FULL_PATH": "/avatar",
  "ACCESS": 0,
  "CONTENTS": {
    "change": {"FULL_PATH": "/avatar/change", "ACCESS": 3, "TYPE": "s", "VALUE": ["avtr_1234"]},
    %s
    "parameters": {
      "FULL_PATH": "/avatar/parameters",
      "ACCESS": 0,
      "CONTENTS": {
        "v2": {
          "FULL_PATH": "/avatar/parameters/v2",
          "ACCESS": 0,
          "CONTENTS": {
            "JawOpen": {"FULL_PATH": "/avatar/parameters/v2/JawOpen", "ACCESS": 3, "TYPE": "f", "VALUE": [0]}
          }
        }
      }
    }
  }
}`

func parseAvatar(t *testing.T, extra string) AvatarInfo {
	t.Helper()
	var root Node
	if err := json.Unmarshal([]byte(fmt.Sprintf(avatarJSON, extra)), &root); err != nil {
		t.Fatal(err)
	}
	info, ok := avatarFromNode(&root)
	if !ok {
		t.Fatal("no parameters")
	}
	return info
}

func TestAvatarNameFromOSCQuery(t *testing.T) {
	info := parseAvatar(t, `"name": {"FULL_PATH": "/avatar/name", "ACCESS": 1, "TYPE": "s", "VALUE": ["さゆき Avatar"]},`)
	if info.ID != "avtr_1234" || info.Name != "さゆき Avatar" {
		t.Fatalf("info = %+v", info)
	}
	if len(info.Parameters) != 1 || info.Parameters[0].Address != "/avatar/parameters/v2/JawOpen" {
		t.Fatalf("parameters = %+v", info.Parameters)
	}
}

func TestAvatarNameMissing(t *testing.T) {
	// Builds before 2026.4.1 have no /avatar/name.
	if info := parseAvatar(t, ""); info.Name != unknownAvatar {
		t.Fatalf("name = %q", info.Name)
	}
}
