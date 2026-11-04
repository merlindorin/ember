//go:build chrome

package chrome_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/merlindorin/ember/internal/chrome"
)

// TestEveryExampleRunsInTheBrowser builds every example for js/wasm and
// runs each in the page for a few seconds: none may fail to build, stop
// with an error or panic (decision 0054's W3). The 3D ones run too, as
// gpu/ draws them the same way. It takes a few minutes, so it runs when
// EMBER_WEB_EXAMPLES is set, as the CI web job does.
func TestEveryExampleRunsInTheBrowser(t *testing.T) {
	if os.Getenv("EMBER_WEB_EXAMPLES") == "" {
		t.Skip("EMBER_WEB_EXAMPLES unset")
	}
	root := filepath.Join("..", "..")
	entries, err := os.ReadDir(filepath.Join(root, "examples"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	webPage(t, dir)
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()
	b, err := chrome.Launch(1280, 800, 1)
	if err != nil {
		t.Skipf("no Chrome to drive: %v", err)
	}
	defer b.Close()
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || name == "internal" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			out := filepath.Join(dir, name+".wasm")
			build := exec.CommandContext(context.Background(), "go", "build", "-o", out, "./examples/"+name)
			build.Dir = root
			build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=js", "GOARCH=wasm")
			if text, buildErr := build.CombinedOutput(); buildErr != nil {
				if strings.Contains(string(text), "no non-test Go files") {
					t.Skip("a test-only example")
				}
				t.Fatalf("doesn't build for the web: %s", text)
			}
			if err = b.Navigate(srv.URL + "/index.html?wasm=" + name + ".wasm"); err != nil {
				t.Fatal(err)
			}
			time.Sleep(4 * time.Second)
			var failed string
			if err = b.Eval(`window.emberError || ""`, &failed); err != nil {
				t.Fatal(err)
			}
			if failed != "" {
				t.Fatalf("stopped: %s", failed)
			}
		})
	}
}
