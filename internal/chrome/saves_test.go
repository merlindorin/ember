//go:build chrome

package chrome_test

import (
	"bytes"
	"compress/gzip"
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

// goroot is the Go root the go command builds with, for its wasm_exec.js.
func goroot(t *testing.T) string {
	t.Helper()
	out, err := exec.CommandContext(context.Background(), "go", "env", "GOROOT").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

// webPage puts platform/web's page and Go's wasm_exec.js in dir.
func webPage(t *testing.T, dir string) {
	t.Helper()
	root := filepath.Join("..", "..")
	for from, to := range map[string]string{
		filepath.Join(goroot(t), "lib", "wasm", "wasm_exec.js"): "wasm_exec.js",
		filepath.Join(root, "platform", "web", "index.html"):    "index.html",
		filepath.Join(root, "platform", "web", "fs.js"):         "fs.js",
	} {
		data, err := os.ReadFile(from)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, to), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// buildWasm builds the js/wasm package pkg, from the module's root, as out.
func buildWasm(t *testing.T, pkg, out string) {
	t.Helper()
	build := exec.CommandContext(context.Background(), "go", "build", "-o", out, pkg)
	build.Dir = filepath.Join("..", "..")
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=js", "GOARCH=wasm")
	if text, err := build.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, text)
	}
}

// servePackage builds the js/wasm package pkg (from the module's root) as
// main.wasm beside platform/web's page, and serves them.
func servePackage(t *testing.T, pkg string) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	buildWasm(t, pkg, filepath.Join(dir, "main.wasm"))
	webPage(t, dir)
	wasm, err := os.ReadFile(filepath.Join(dir, "main.wasm"))
	if err != nil {
		t.Fatal(err)
	}
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	if _, err = zw.Write(wasm); err != nil {
		t.Fatal(err)
	}
	if err = zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "main.wasm.gz"), gz.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	t.Cleanup(srv.Close)
	return srv
}

// TestAGzippedBuildRunsAsServed loads main.wasm.gz, as the guide hosts its
// examples: the page unpacks it as it arrives.
func TestAGzippedBuildRunsAsServed(t *testing.T) {
	srv := servePackage(t, "./internal/chrome/testdata/saves")
	b, err := chrome.Launch(320, 200, 1)
	if err != nil {
		t.Skipf("no Chrome to drive: %v", err)
	}
	defer b.Close()
	if err = b.Navigate(srv.URL + "/index.html?wasm=main.wasm.gz"); err != nil {
		t.Fatal(err)
	}
	saves := `window.emberError ? "error: " + window.emberError : (window.emberSaves || "")`
	if got, waitErr := b.Wait(saves, time.Minute); waitErr != nil || got != "1" {
		t.Fatalf("the gzipped build runs: %q %v", got, waitErr)
	}
}

// TestSavesStayInTheBrowserAcrossVisits runs testdata/saves twice in one
// browser: its os.ReadFile, MkdirAll, WriteFile and Rename go to the
// page's fs.js, and what the first visit wrote, the second reads.
func TestSavesStayInTheBrowserAcrossVisits(t *testing.T) {
	srv := servePackage(t, "./internal/chrome/testdata/saves")
	b, err := chrome.Launch(320, 200, 1)
	if err != nil {
		t.Skipf("no Chrome to drive: %v", err)
	}
	defer b.Close()
	for want := 1; want <= 2; want++ {
		if err = b.Navigate(srv.URL + "/index.html"); err != nil {
			t.Fatal(err)
		}
		got, waitErr := b.Wait(`window.emberSaves || ""`, time.Minute)
		if waitErr != nil {
			t.Fatal(waitErr)
		}
		if got != string(rune('0'+want)) {
			t.Fatalf("visit %d reads %d runs saved: %q", want, want, got)
		}
	}
}

// TestAPanicShowsOverThePage makes testdata/saves panic in its first
// frame: the page shows why, as the desktop's crash log says it.
func TestAPanicShowsOverThePage(t *testing.T) {
	srv := servePackage(t, "./internal/chrome/testdata/saves")
	b, err := chrome.Launch(320, 200, 1)
	if err != nil {
		t.Skipf("no Chrome to drive: %v", err)
	}
	defer b.Close()
	if err = b.Navigate(srv.URL + "/index.html?EMBER_PANIC=1"); err != nil {
		t.Fatal(err)
	}
	if _, err = b.Wait(`window.emberError || ""`, time.Minute); err != nil {
		t.Fatal(err)
	}
	var shown string
	crash := `document.getElementById("crash").hidden ? "" : document.getElementById("crash-text").textContent`
	if err = b.Eval(crash, &shown); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(shown, "panic: the probe was asked to panic") || !strings.Contains(shown, "goroutine") {
		t.Fatalf("the overlay says why, with the stack: %q", shown)
	}
}
