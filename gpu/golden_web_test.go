//go:build chrome && gpu

package gpu //nolint:testpackage // white-box: drives the renderer's unexported passes

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/merlindorin/ember/golden"
	"github.com/merlindorin/ember/internal/chrome"
)

// TestGoldenInTheBrowser renders the 2D golden scenes in a headless Chrome
// through WebGPU (js/wasm, platform/web) and compares each with the same
// reference picture the native renderer is held to, within the same
// tolerance: decision 0054's W1. EMBER_GOLDEN_WEB_OUT names a folder to
// keep the pictures in.
//
//	go test -tags 'chrome gpu' ./gpu -run GoldenInTheBrowser
func TestGoldenInTheBrowser(t *testing.T) {
	dir := t.TempDir()
	copyFile(t, filepath.Join(goroot(t), "lib", "wasm", "wasm_exec.js"), filepath.Join(dir, "wasm_exec.js"))
	copyFile(t, filepath.Join("..", "platform", "web", "index.html"), filepath.Join(dir, "index.html"))
	copyFile(t, filepath.Join("..", "platform", "web", "fs.js"), filepath.Join(dir, "fs.js"))
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()
	b, err := chrome.Launch(1400, 1000, 1)
	if err != nil {
		t.Skipf("no Chrome to drive: %v", err)
	}
	defer b.Close()
	for _, s := range goldenScenes {
		if s.Edges > 0 {
			continue // the 3D scenes
		}
		t.Run(s.Name, func(t *testing.T) {
			wasm := s.Name + ".wasm"
			build := exec.Command("go", "build", "-o", filepath.Join(dir, wasm), "./"+s.Pkg)
			build.Dir = ".."
			build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=js", "GOARCH=wasm")
			if out, buildErr := build.CombinedOutput(); buildErr != nil {
				t.Skipf("doesn't build for the web yet: %s", firstLine(out))
			}
			q := url.Values{"wasm": {wasm}, "EMBER_CAPTURE": {"1"}, "EMBER_SCALE": {"2"},
				"EMBER_CAPTURE_FRAME": {fmt.Sprint(s.Frame)}}
			for _, kv := range s.Env {
				k, v, _ := strings.Cut(kv, "=")
				q.Set(k, v)
			}
			if navErr := b.Navigate(srv.URL + "/index.html?" + q.Encode()); navErr != nil {
				t.Fatal(navErr)
			}
			got, waitErr := b.Wait(
				`window.emberError ? "error: " + window.emberError : (window.emberCapture || "")`,
				2*time.Minute,
			)
			if waitErr != nil {
				t.Fatal(waitErr)
			}
			if rest, ok := strings.CutPrefix(got, "error: "); ok {
				t.Fatalf("the page failed: %s", rest)
			}
			raw, decodeErr := base64.StdEncoding.DecodeString(got)
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			img, decodeErr := png.Decode(bytes.NewReader(raw))
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			half := golden.ToPoints(t, img)
			if out := os.Getenv("EMBER_GOLDEN_WEB_OUT"); out != "" {
				golden.Save(t, filepath.Join(out, s.Name+".png"), half)
			}
			ref := filepath.Join("testdata", "golden", s.Name+".png")
			d, marks := golden.Compare(t, golden.Load(t, ref), half)
			allowed := 0
			if share := golden.Share(t); share > 0 {
				allowed = int(share * float64(d.Total))
			}
			t.Logf(
				"%s in Chrome: %d of %d pixels over %d levels, at most %d off",
				s.Name,
				d.Over,
				d.Total,
				golden.Levels,
				d.Most,
			)
			golden.Report(t, fmt.Sprintf("| %s (web) | %d of %d | %d | %d |", s.Name, d.Over, d.Total, d.Most, allowed))
			if d.Over > allowed {
				shot := filepath.Join(os.TempDir(), "golden-web-"+s.Name+".png")
				diff := filepath.Join(os.TempDir(), "golden-web-"+s.Name+"-diff.png")
				golden.Save(t, shot, half)
				golden.Save(t, diff, marks)
				t.Errorf(
					"%s in Chrome: %d of %d pixels differ by more than %d levels (at most %d, %d allowed); picture %s, where %s",
					s.Name,
					d.Over,
					d.Total,
					golden.Levels,
					d.Most,
					allowed,
					shot,
					diff,
				)
			}
		})
	}
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()
	data, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(to, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// goroot is the GOROOT of the go command the test builds with.
func goroot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func firstLine(b []byte) string {
	s, _, _ := strings.Cut(strings.TrimSpace(string(b)), "\n")
	return s
}
