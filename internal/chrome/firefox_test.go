//go:build chrome

package chrome_test

import (
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lantervale/ember/internal/chrome"
)

// TestExamplesRunInFirefox runs a sprite, a light and a ui example, with
// its command line, in Firefox. Headless Firefox offers no WebGPU adapter
// on macOS, so it opens a window, on a fresh profile, and runs only when
// EMBER_FIREFOX_WINDOW is set.
func TestExamplesRunInFirefox(t *testing.T) {
	if os.Getenv("EMBER_FIREFOX_WINDOW") == "" {
		t.Skip("EMBER_FIREFOX_WINDOW unset")
	}
	b, err := chrome.LaunchFirefox(800, 600, 1)
	if err != nil {
		t.Skipf("no Firefox to drive: %v", err)
	}
	defer b.Close()
	for _, run := range []struct{ pkg, args string }{
		{"./examples/19-sprites", ""},
		{"./examples/36-point-lights", ""},
		{"./examples/92-ui-kit", "&arg=-element&arg=file-dialog"},
	} {
		t.Run(run.pkg, func(t *testing.T) {
			srv := servePackage(t, run.pkg)
			if err = b.Navigate(srv.URL + "/index.html?wasm=main.wasm.gz" + run.args); err != nil {
				t.Fatal(err)
			}
			time.Sleep(5 * time.Second)
			var state string
			stopped := `window.emberError || (document.getElementById("nogpu").hidden ? "" : "no WebGPU")`
			if err = b.Eval(stopped, &state); err != nil {
				t.Fatal(err)
			}
			if state != "" {
				t.Fatalf("stopped: %s", state)
			}
		})
	}
}

// TestTheNoGPUPageNamesAnInsecureOrigin loads the page from this machine's
// network address over http, which browsers don't trust with WebGPU: the
// page says so rather than blame the browser.
func TestTheNoGPUPageNamesAnInsecureOrigin(t *testing.T) {
	ip := lanAddress()
	if ip == "" {
		t.Skip("no network address besides loopback")
	}
	l, err := net.Listen("tcp", ip+":0")
	if err != nil {
		t.Skip(err)
	}
	srv := &http.Server{Handler: http.FileServer(http.Dir("../../platform/web"))}
	go func() { _ = srv.Serve(l) }()
	defer srv.Close()
	b, err := chrome.Launch(640, 400, 1)
	if err != nil {
		t.Skipf("no Chrome to drive: %v", err)
	}
	defer b.Close()
	origin := "http://" + l.Addr().String()
	if err = b.Navigate(origin + "/index.html"); err != nil {
		t.Fatal(err)
	}
	named := `document.getElementById("nogpu-insecure").hidden ? "" : ` +
		`document.getElementById("nogpu-origin").textContent`
	got, err := b.Wait(named, 10*time.Second)
	if err != nil || got != origin {
		t.Fatalf("the page names %q as not secure: %q %v", origin, got, err)
	}
}

func lanAddress() string {
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		n, ok := a.(*net.IPNet)
		if ok && !n.IP.IsLoopback() && n.IP.To4() != nil && !strings.HasPrefix(n.IP.String(), "169.254.") {
			return n.IP.String()
		}
	}
	return ""
}
