//go:build chrome

package chrome_test

import (
	"bytes"
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/lantervale/ember/internal/chrome"
)

// page is the probe running in Chrome, driven by its own input events.
type page struct {
	t *testing.T
	b *chrome.Browser
}

func openProbe(t *testing.T, query string, scale float64) page {
	t.Helper()
	dir := t.TempDir()
	buildWasm(t, "./internal/chrome/testdata/probe", filepath.Join(dir, "main.wasm"))
	webPage(t, dir)
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	t.Cleanup(srv.Close)
	b, err := chrome.Launch(640, 400, scale)
	if err != nil {
		t.Skipf("no Chrome to drive: %v", err)
	}
	t.Cleanup(b.Close)
	if err = b.Navigate(srv.URL + "/index.html" + query); err != nil {
		t.Fatal(err)
	}
	if _, err = b.Wait(`window.emberProbe || ""`, time.Minute); err != nil {
		t.Fatal(err)
	}
	return page{t, b}
}

func (p page) read() probe {
	p.t.Helper()
	var s string
	if err := p.b.Eval(`window.emberProbe`, &s); err != nil {
		p.t.Fatal(err)
	}
	var r probe
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		p.t.Fatal(err)
	}
	return r
}

// until waits a few frames for the World to see what ok wants.
func (p page) until(what string, ok func(probe) bool) {
	p.t.Helper()
	var r probe
	for range 100 {
		if r = p.read(); ok(r) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	p.t.Fatalf("%s: the World saw %+v", what, r)
}

func (p page) call(method string, params map[string]any) {
	p.t.Helper()
	if _, err := p.b.Call(method, params); err != nil {
		p.t.Fatal(err)
	}
}

func (p page) press(code, name string, vk int) {
	p.t.Helper()
	p.key("rawKeyDown", code, name, vk)
	p.key("keyUp", code, name, vk)
}

func (p page) key(kind, code, name string, vk int) {
	p.t.Helper()
	p.call("Input.dispatchKeyEvent", map[string]any{"type": kind, "code": code, "key": name, "windowsVirtualKeyCode": vk})
}

func (p page) viewport() [2]float32 {
	p.t.Helper()
	var v [2]float32
	if err := p.b.Eval(`[innerWidth, innerHeight]`, &v); err != nil {
		p.t.Fatal(err)
	}
	return v
}

// controls checks the pointer lands where it is in window points and a
// key still reaches the World.
func (p page) controls(at [2]float32) {
	p.t.Helper()
	p.call("Input.dispatchMouseEvent", map[string]any{"type": "mouseMoved", "x": at[0], "y": at[1]})
	p.until("the pointer where it is", func(r probe) bool { return r.Mouse == at })
	const keyW = 87
	p.key("keyDown", "KeyW", "w", keyW)
	p.until("W down", func(r probe) bool { return slices.Contains(r.Down, keyW) })
	p.key("keyUp", "KeyW", "w", keyW)
	p.until("W up", func(r probe) bool { return !slices.Contains(r.Down, keyW) })
}

func TestFullScreenFillsThePageAndKeepsTheControls(t *testing.T) {
	p := openProbe(t, "", 1)
	p.until("the window's own size first", func(r probe) bool { return r.Size == [2]float32{320, 200} })

	p.press("KeyF", "f", 70)
	var full bool
	for range 100 {
		if err := p.b.Eval(`document.fullscreenElement !== null`, &full); err == nil && full {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !full {
		t.Skip("this Chrome grants no full screen")
	}
	view := p.viewport()
	p.until("full screen, the window is the page's size", func(r probe) bool { return r.Full && r.Size == view })
	p.controls([2]float32{view[0] - 40, view[1] - 30})

	p.press("KeyF", "f", 70)
	p.until("back, the window's own size", func(r probe) bool { return !r.Full && r.Size == [2]float32{320, 200} })
	p.controls([2]float32{120, 70})
}

func TestAPageAlreadyFullScreenGivesTheGameItsSize(t *testing.T) {
	p := openProbe(t, "", 1)
	if _, err := p.b.Call("Runtime.evaluate", map[string]any{
		"expression":   `document.documentElement.requestFullscreen()`,
		"userGesture":  true,
		"awaitPromise": true,
	}); err != nil {
		t.Skipf("this Chrome grants no full screen: %v", err)
	}
	view := p.viewport()
	p.until("the page's full screen is the window's", func(r probe) bool { return r.Full && r.Size == view })
	p.controls([2]float32{view[0] - 40, view[1] - 30})
}

func TestAResizableWindowFillsThePageAndAFixedOneKeepsItsSize(t *testing.T) {
	fixed := openProbe(t, "", 1)
	fixed.until("a fixed window, its own size", func(r probe) bool { return r.Size == [2]float32{320, 200} })
	fixed.controls([2]float32{120, 70})

	for _, scale := range []float64{1, 2} {
		fluid := openProbe(t, "?EMBER_PROBE_FLUID=1", scale)
		view := fluid.viewport()
		fluid.until("a resizable window, the page's size", func(r probe) bool { return r.Size == view })
		fluid.controls([2]float32{view[0] - 40, view[1] - 30})
		var backing [2]float64
		size := `[document.getElementById("ember").width, document.getElementById("ember").height]`
		if err := fluid.b.Eval(size, &backing); err != nil {
			t.Fatal(err)
		}
		if want := [2]float64{float64(view[0]) * scale, float64(view[1]) * scale}; backing != want {
			t.Fatalf("at a pixel ratio of %v, the canvas draws %v pixels a point: %v, want %v", scale, scale, backing, want)
		}
	}

	fluid := openProbe(t, "?EMBER_PROBE_FLUID=1", 1)
	fluid.call("Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 800, "height": 500, "deviceScaleFactor": 1, "mobile": false,
	})
	fluid.until("the page made larger, the window follows", func(r probe) bool { return r.Size == [2]float32{800, 500} })
	fluid.controls([2]float32{780, 480})
	shot, err := fluid.b.Screenshot()
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(shot))
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	r, g, bl, _ := img.At(b.Max.X-2, b.Max.Y-2).RGBA()
	if r>>8 > 0x50 || g>>8 < 0x55 || g>>8 > 0x77 || bl>>8 < 0xbb {
		t.Fatalf("the far corner is the game's background, not the page's: %02x%02x%02x", r>>8, g>>8, bl>>8)
	}
}

func TestAGameThatQuitsEndsItsProgram(t *testing.T) {
	p := openProbe(t, "?EMBER_PROBE_QUIT=1", 1)
	catch := `window.pageErrors = []; addEventListener("error", e => pageErrors.push(e.message)); true`
	if err := p.b.Eval(catch, new(bool)); err != nil {
		t.Fatal(err)
	}
	if _, err := p.b.Wait(`window.emberEnded ? "ended" : ""`, 20*time.Second); err != nil {
		t.Fatalf("Quit after 30 frames ends the program, go.run's promise resolving: %v", err)
	}
	var shown bool
	ended := `!document.getElementById("ended").hidden && document.getElementById("ember").hidden`
	if err := p.b.Eval(ended, &shown); err != nil || !shown {
		t.Fatalf("the page says the game has ended: %v %v", shown, err)
	}
	p.key("keyDown", "KeyW", "w", 87)
	p.key("keyUp", "KeyW", "w", 87)
	p.call("Input.dispatchMouseEvent", map[string]any{"type": "mouseMoved", "x": 100, "y": 50})
	if err := p.b.Eval(`dispatchEvent(new ClipboardEvent("paste")); true`, new(bool)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	var errs []string
	if err := p.b.Eval(`pageErrors`, &errs); err != nil {
		t.Fatal(err)
	}
	if len(errs) > 0 {
		t.Fatalf("no handler is left calling into the ended program: %v", errs)
	}
}
