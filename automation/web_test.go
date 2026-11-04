package automation_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestABrowserBuildLeavesTheServerOut guards decision 0054's download: a
// js/wasm build records widgets and actions but can't serve, so it must
// not link net/http, Connect or protobuf, which tripled a page's wasm.
func TestABrowserBuildLeavesTheServerOut(t *testing.T) {
	if testing.Short() {
		t.Skip("lists a js build's packages")
	}
	cmd := exec.Command("go", "list", "-deps", "github.com/merlindorin/ember/automation")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=js", "GOARCH=wasm")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range strings.Fields(string(out)) {
		if p == "net/http" || strings.HasPrefix(p, "connectrpc.com/") || strings.HasPrefix(p, "google.golang.org/protobuf") {
			t.Fatalf("a browser build links %s", p)
		}
	}
}
