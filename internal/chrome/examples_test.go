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

// TestEveryExampleRunsInTheBrowser builds every example and game for
// js/wasm and runs each in the page for a few seconds: none may fail to build, stop
// with an error or panic (decision 0054's W3). The 3D ones run too, as
// gpu/ draws them the same way. It takes a few minutes, so it runs when
// EMBER_WEB_EXAMPLES is set, as the CI web job does.
func TestEveryExampleRunsInTheBrowser(t *testing.T) {
	if os.Getenv("EMBER_WEB_EXAMPLES") == "" {
		t.Skip("EMBER_WEB_EXAMPLES unset")
	}
	root := filepath.Join("..", "..")
	var pkgs []string
	for _, folder := range []string{"examples", "games"} {
		entries, err := os.ReadDir(filepath.Join(root, folder))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if e.IsDir() && e.Name() != "internal" {
				pkgs = append(pkgs, folder+"/"+e.Name())
			}
		}
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
	for _, pkg := range pkgs {
		name := strings.ReplaceAll(strings.TrimPrefix(pkg, "examples/"), "/", "-")
		t.Run(name, func(t *testing.T) {
			out := filepath.Join(dir, name+".wasm")
			build := exec.CommandContext(context.Background(), "go", "build", "-o", out, "./"+pkg)
			build.Dir = root
			build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=js", "GOARCH=wasm")
			if text, buildErr := build.CombinedOutput(); buildErr != nil {
				if strings.Contains(string(text), "no non-test Go files") {
					t.Skip("a test-only example")
				}
				t.Fatalf("doesn't build for the web: %s", text)
			}
			if navErr := b.Navigate(srv.URL + "/index.html?wasm=" + name + ".wasm"); navErr != nil {
				t.Fatal(navErr)
			}
			time.Sleep(4 * time.Second)
			var failed string
			if evalErr := b.Eval(`window.emberError || ""`, &failed); evalErr != nil {
				t.Fatal(evalErr)
			}
			if failed != "" {
				t.Fatalf("stopped: %s", failed)
			}
		})
	}
}
