//go:build chrome

package chrome_test

import (
	"testing"
	"time"

	"github.com/merlindorin/ember/internal/chrome"
)

// TestAClickOnTheCanvasStartsTheSound runs the sound examples under
// Chrome's autoplay rule (its default, made explicit): their Web Audio
// stays suspended until the player's first click on the canvas, and runs
// after it. The canvas keeps mousedown and mouseup from the page, so oto's
// own unlock on mouseup never sees the click.
func TestAClickOnTheCanvasStartsTheSound(t *testing.T) {
	t.Setenv("EMBER_CHROME_FLAGS", "--autoplay-policy=document-user-activation-required")
	sounds := []string{"./examples/52-play-a-sound", "./examples/53-music-and-positional", "./examples/54-audio-effects"}
	for _, pkg := range sounds {
		t.Run(pkg, func(t *testing.T) {
			srv := servePackage(t, pkg)
			b, err := chrome.Launch(640, 400, 1)
			if err != nil {
				t.Skipf("no Chrome to drive: %v", err)
			}
			defer b.Close()
			// Every AudioContext the page makes is kept in window.emberAudio.
			if _, err = b.Call("Page.enable", map[string]any{}); err != nil {
				t.Fatal(err)
			}
			if _, err = b.Call("Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": `
				window.emberAudio = [];
				const Real = window.AudioContext;
				window.AudioContext = class extends Real {
					constructor(...a) { super(...a); window.emberAudio.push(this); }
				};`}); err != nil {
				t.Fatal(err)
			}
			if err = b.Navigate(srv.URL + "/index.html"); err != nil {
				t.Fatal(err)
			}
			state := `window.emberError ? "error: " + window.emberError : ` +
				`(window.emberAudio.length ? window.emberAudio[0].state : "")`
			if got, waitErr := b.Wait(state, time.Minute); waitErr != nil || got != "suspended" {
				t.Fatalf("before a click, sound waits: %q %v", got, waitErr)
			}
			for _, kind := range []string{"mousePressed", "mouseReleased"} {
				click := map[string]any{"type": kind, "x": 100, "y": 100, "button": "left", "clickCount": 1}
				if _, err = b.Call("Input.dispatchMouseEvent", click); err != nil {
					t.Fatal(err)
				}
			}
			running := `(` + state + `) === "suspended" ? "" : (` + state + `)`
			if got, waitErr := b.Wait(running, 10*time.Second); waitErr != nil || got != "running" {
				t.Fatalf("after one click on the canvas, sound runs: %q %v", got, waitErr)
			}
		})
	}
}
