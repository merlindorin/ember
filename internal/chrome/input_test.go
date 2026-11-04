//go:build chrome

package chrome_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/merlindorin/ember/internal/chrome"
)

type probe struct {
	Down   []int      `json:"down"`
	Mouse  [2]float32 `json:"mouse"`
	Left   bool       `json:"left"`
	Right  bool       `json:"right"`
	Clicks int        `json:"clicks"`
	Wheel  float32    `json:"wheel"`
	Text   string     `json:"text"`
	Frame  int        `json:"frame"`
}

// TestThePagesInputReachesTheWorld runs testdata/probe in a page through
// platform/web, sends it Chrome's own key, mouse and wheel events, and
// reads what the World's input saw (decision 0054's W2).
func TestThePagesInputReachesTheWorld(t *testing.T) {
	dir := t.TempDir()
	buildWasm(t, "./internal/chrome/testdata/probe", filepath.Join(dir, "main.wasm"))
	webPage(t, dir)
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()
	b, err := chrome.Launch(640, 400, 1)
	if err != nil {
		t.Skipf("no Chrome to drive: %v", err)
	}
	defer b.Close()
	if err = b.Navigate(srv.URL + "/index.html"); err != nil {
		t.Fatal(err)
	}
	if _, err = b.Wait(`window.emberProbe || ""`, time.Minute); err != nil {
		t.Fatal(err)
	}
	read := func() probe {
		var s string
		if evalErr := b.Eval(`window.emberProbe`, &s); evalErr != nil {
			t.Fatal(evalErr)
		}
		var p probe
		if jsonErr := json.Unmarshal([]byte(s), &p); jsonErr != nil {
			t.Fatal(jsonErr)
		}
		return p
	}
	// until waits a few frames for the World to see what was sent.
	until := func(what string, ok func(probe) bool) probe {
		t.Helper()
		var p probe
		for range 50 {
			if p = read(); ok(p) {
				return p
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("%s: the World saw %+v", what, p)
		return p
	}
	call := func(method string, params map[string]any) {
		t.Helper()
		if _, callErr := b.Call(method, params); callErr != nil {
			t.Fatal(callErr)
		}
	}
	key := func(kind, code, name string, vk int, text string) {
		t.Helper()
		params := map[string]any{"type": kind, "code": code, "key": name, "windowsVirtualKeyCode": vk}
		if text != "" {
			params["text"] = text
		}
		call("Input.dispatchKeyEvent", params)
	}
	mouse := func(kind, button string) {
		t.Helper()
		call("Input.dispatchMouseEvent", map[string]any{"type": kind, "x": 120, "y": 70, "button": button, "clickCount": 1})
	}
	const keyW = 87
	key("keyDown", "KeyW", "w", keyW, "w")
	until("W down, typed as w", func(p probe) bool {
		return slices.Contains(p.Down, keyW) && strings.Contains(p.Text, "w")
	})
	key("keyUp", "KeyW", "w", keyW, "")
	until("W up", func(p probe) bool { return !slices.Contains(p.Down, keyW) })
	key("rawKeyDown", "ArrowLeft", "ArrowLeft", 37, "")
	until("the left arrow by its place", func(p probe) bool { return slices.Contains(p.Down, 263) })
	key("keyUp", "ArrowLeft", "ArrowLeft", 37, "")

	call("Input.dispatchMouseEvent", map[string]any{"type": "mouseMoved", "x": 120, "y": 70})
	until("the pointer in window points", func(p probe) bool { return p.Mouse == [2]float32{120, 70} })
	mouse("mousePressed", "left")
	until("the left button down", func(p probe) bool { return p.Left })
	mouse("mouseReleased", "left")
	until("and up", func(p probe) bool { return !p.Left })
	mouse("mousePressed", "right")
	until("the right button, no page menu", func(p probe) bool { return p.Right })
	mouse("mouseReleased", "right")

	clicks := read().Clicks
	for range 3 {
		mouse("mousePressed", "left")
		mouse("mouseReleased", "left")
	}
	until("three quick clicks, each pressed", func(p probe) bool { return p.Clicks == clicks+3 })

	call("Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": 120, "y": 70, "deltaX": 0, "deltaY": -100})
	p := until("one notch up", func(p probe) bool { return p.Wheel > 0 })
	if p.Wheel < 0.9 || p.Wheel > 1.1 {
		t.Fatalf("a 100-pixel notch is one line, as GLFW reports a wheel: %v", p.Wheel)
	}
}
