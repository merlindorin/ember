//go:build chrome

// Package chrome is the layout tests' reference: it lays an exported
// Document out in headless Google Chrome and compares every element's
// getBoundingClientRect with its Node.Box. Tests run with -tags chrome;
// EMBER_CHROME names the browser when it isn't in /Applications.
package chrome

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/export"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/fonts"
)

// Tolerance is how far, in points, an edge may be from Chrome's.
const Tolerance = 0.5

const script = `<script>
addEventListener("load", () => document.fonts.ready.then(() => {
	const els = [document.documentElement, document.body,
		...document.body.querySelectorAll("*:not([data-ember-implied])")];
	const out = document.createElement("script");
	out.type = "application/json";
	out.id = "ember-boxes";
	out.textContent = JSON.stringify(els.map(e => {
		const r = e.getBoundingClientRect();
		return [r.x, r.y, r.width, r.height];
	}));
	document.head.appendChild(out);
}));
</script>`

var boxesJSON = regexp.MustCompile(`<script type="application/json" id="ember-boxes">(.*?)</script>`)

func browser() string {
	if p := os.Getenv("EMBER_CHROME"); p != "" {
		return p
	}
	return "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
}

// Screenshot writes Chrome's picture of page in a viewport of size
// points to png, to set beside ember's drawing of the same document.
func Screenshot(page []byte, size geom.Vec2, out string) error {
	shoot := func() ([]byte, error) {
		if _, err := run(page, size, "--screenshot="+out); err != nil {
			return nil, err
		}
		if err := crop(out, int(size.X*deviceScale+0.5), int(size.Y*deviceScale+0.5)); err != nil {
			return nil, err
		}
		return os.ReadFile(out)
	}
	if os.Getenv("CI") == "" {
		_, err := shoot()
		return err
	}
	// A CI runner without a GPU can shoot before every tile is rasterized,
	// a picture cut short: there a shot counts once two in a row agree.
	shot, err := steady(shoot, func(n int) {
		fmt.Fprintf(os.Stderr, "chrome: shot %d of %s differs from the one before, retaken\n", n, out)
	})
	if err != nil {
		return err
	}
	return os.WriteFile(out, shot, 0o600)
}

// steadyTries is how many shots steady takes at most.
const steadyTries = 4

// steady takes shots until two in a row are the same picture, at most
// steadyTries, and returns the last; retaken is told each shot that differed.
func steady(shoot func() ([]byte, error), retaken func(n int)) ([]byte, error) {
	last, err := shoot()
	if err != nil {
		return nil, err
	}
	for n := 2; n <= steadyTries; n++ {
		next, shotErr := shoot()
		if shotErr != nil {
			return nil, shotErr
		}
		if bytes.Equal(next, last) {
			return next, nil
		}
		retaken(n)
		last = next
	}
	return last, nil
}

// ScreenshotScaled is Screenshot on a display of scale device pixels per
// point, such as 2 to set beside ember's capture on a Retina display.
func ScreenshotScaled(page []byte, size geom.Vec2, scale float32, out string) error {
	deviceScale = scale
	defer func() { deviceScale = 1 }()
	return Screenshot(page, size, out)
}

// crop cuts the picture at path to w×h pixels from its top-left, when a
// narrow page ran in a wider window.
func crop(path string, w, h int) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		return err
	}
	if img.Bounds().Dx() <= w && img.Bounds().Dy() <= h {
		return nil
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(out, out.Bounds(), img, img.Bounds().Min, draw.Src)
	f, err = os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, out)
}

// Boxes lays page out in a viewport of size points, and returns the
// border boxes of html, body and body's elements in document order.
func Boxes(page []byte, size geom.Vec2) ([]geom.Rect, error) {
	out, err := run(page, size, dumpDOM)
	if err != nil {
		return nil, err
	}
	m := boxesJSON.FindSubmatch(out)
	if m == nil {
		return nil, fmt.Errorf("chrome: no boxes in the page:\n%s", out)
	}
	var raw [][4]float32
	if err = json.Unmarshal(m[1], &raw); err != nil {
		return nil, err
	}
	rects := make([]geom.Rect, len(raw))
	for i, r := range raw {
		rects[i] = geom.Rect{X: r[0], Y: r[1], W: r[2], H: r[3]}
	}
	return rects, nil
}

// run runs one Chrome at a time across test processes: two headless
// Chromes started together on macOS can fail at teardown. A failed run is
// tried once more.
func run(page []byte, size geom.Vec2, action string) ([]byte, error) {
	unlock, err := lockChrome()
	if err != nil {
		return nil, err
	}
	defer unlock()
	out, err := runOnce(page, size, action)
	if err != nil {
		out, err = runOnce(page, size, action)
	}
	return out, err
}

// lockPath is the file every run of Chrome locks, across test processes.
func lockPath() string { return filepath.Join(os.TempDir(), "ember-chrome.lock") }

// waitNote tells that a run waits for another's Chrome; waitAfter is how
// long it waits before saying so, and again between notes.
//
//nolint:gochecknoglobals // the lock test swaps them to watch the notes quickly
var (
	waitNote  = func(s string) { fmt.Fprintln(os.Stderr, s) }
	waitAfter = 5 * time.Second
)

// lockChrome takes the Chrome lock and writes who holds it into the file:
// full suites in several sessions queue on it for an hour, and a run
// waiting behind them says whom it waits for instead of seeming to hang.
func lockChrome() (func(), error) {
	f, err := os.OpenFile(lockPath(), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err = lockFile(f, false); err != nil {
		done := make(chan struct{})
		go func() {
			start := time.Now()
			for {
				select {
				case <-done:
					return
				case <-time.After(waitAfter):
					holder, _ := os.ReadFile(lockPath())
					waitNote(fmt.Sprintf("chrome: waiting for the Chrome lock %s for %s, held by %s",
						lockPath(), time.Since(start).Round(time.Second), strings.TrimSpace(string(holder))))
				}
			}
		}()
		err = lockFile(f, true)
		close(done)
		if err != nil {
			f.Close()
			return nil, err
		}
	}
	// The holder's note only tells those waiting whom for: a failed write
	// leaves the lock held all the same.
	_ = f.Truncate(0)
	note := fmt.Sprintf(
		"pid %d (%s) since %s",
		os.Getpid(),
		filepath.Base(os.Args[0]),
		time.Now().Format(time.TimeOnly),
	)
	_, _ = f.WriteAt([]byte(note), 0)
	return func() {
		_ = f.Truncate(0)
		// Closing the file releases the lock too, should unlocking fail.
		_ = unlockFile(f)
		f.Close()
	}, nil
}

// dumpDOM is the action that prints the laid-out page.
const dumpDOM = "--dump-dom"

// minWidth is the narrowest window headless Chrome keeps.
const minWidth = 500

// fit makes page lay out in a viewport of size as ui's does: the root
// never shows a scrollbar, and a page narrower than Chrome's narrowest
// window keeps its width on the root. It returns the window to open.
func fit(page []byte, size geom.Vec2, shot bool) ([]byte, geom.Vec2) {
	rule := "html { scrollbar-width: none; }"
	window := size
	if size.X < minWidth {
		rule += fmt.Sprintf(" html { width: %gpx; }", size.X)
		if shot {
			// Chrome's window is wider than the page, so the initial
			// containing block is too: for a picture, html becomes the
			// containing block of absolute boxes at the page's size, so right
			// and bottom resolve as ui's do. Box dumps leave html's own box
			// as ui has it.
			rule += fmt.Sprintf(" html { height: %gpx; position: relative; }", size.Y)
		}
		window.X = minWidth
	}
	return bytes.Replace(page, []byte("</head>"), []byte("<style>"+rule+"</style>\n</head>"), 1), window
}

// margin is how much bigger than its viewport Chrome's window is when it
// dumps the DOM, for the browser's own parts, measured once; a screenshot
// run's viewport is its window.
//
//nolint:gochecknoglobals // measured once per test process
var (
	margin     geom.Vec2
	marginOnce sync.Once
)

var viewportJSON = regexp.MustCompile(`<script type="application/json" id="ember-viewport">(.*?)</script>`)

// measureMargin opens an empty page and reads the viewport Chrome gave
// it, to size later windows so their viewports are as asked.
func measureMargin() {
	page := []byte(
		`<!DOCTYPE html><html><head><script>addEventListener("load", () => {
	const o = document.createElement("script");
	o.type = "application/json";
	o.id = "ember-viewport";
	o.textContent = JSON.stringify([innerWidth, innerHeight]);
	document.head.appendChild(o);
});</script></head><body></body></html>`,
	)
	want := geom.V2(800, 600)
	out, err := runRaw(page, want, dumpDOM)
	if err != nil {
		return
	}
	m := viewportJSON.FindSubmatch(out)
	if m == nil {
		return
	}
	var got [2]float32
	if json.Unmarshal(m[1], &got) == nil {
		margin = geom.V2(want.X-got[0], want.Y-got[1])
	}
}

func runOnce(page []byte, size geom.Vec2, action string) ([]byte, error) {
	page, size = fit(page, size, action != dumpDOM)
	if action == dumpDOM {
		marginOnce.Do(measureMargin)
		size = size.Add(margin)
	}
	return runRaw(page, size, action)
}

// runRaw runs Chrome once on page with a window of size.
func runRaw(page []byte, size geom.Vec2, action string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "ember-chrome-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "page.html")
	if err = os.WriteFile(file, page, 0o600); err != nil { //nolint:gosec // a file in our own temporary folder
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	// --run-all-compositor-stages-before-draw: headless shoots when the
	// virtual time runs out, before its tiles are rasterized on a loaded
	// machine, which left pages half drawn.
	cmd := exec.CommandContext( //nolint:gosec // the Chrome found on this machine, run on our own page
		ctx,
		browser(),
		"--headless=new",
		"--disable-gpu",
		"--run-all-compositor-stages-before-draw",
		fmt.Sprintf("--force-device-scale-factor=%g", deviceScale),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-extensions",
		"--user-data-dir="+filepath.Join(dir, "profile"),
		fmt.Sprintf("--window-size=%d,%d", int(size.X), int(size.Y)),
		"--virtual-time-budget=10000",
		action,
		"file://"+file,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if bytes.Contains(stderr.Bytes(), []byte("Teardown watchdog expired")) && done(action, out) {
			return out, nil
		}
		return nil, fmt.Errorf("chrome: %w: %s", err, stderr.String())
	}
	return out, nil
}

// done reports whether Chrome finished action before it exited: a whole
// dumped page, or a picture that decodes. On a loaded Mac, Chrome's
// teardown watchdog kills it after its work is written, which is no
// failure of the page.
func done(action string, out []byte) bool {
	if action == dumpDOM {
		return bytes.Contains(out, []byte("</html>"))
	}
	path, ok := strings.CutPrefix(action, "--screenshot=")
	if !ok {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = png.DecodeConfig(f)
	return err == nil
}

// deviceScale is the device pixels per point Chrome lays out at; 1
// unless MatchScaled sets it.
var deviceScale float32 = 1 //nolint:gochecknoglobals // set for the length of one MatchScaled

// MatchScaled is Match on a display of scale device pixels per point,
// where both round font metrics to device pixels.
func MatchScaled(t testing.TB, d *ui.Document, size geom.Vec2, scale float32) Report {
	t.Helper()
	deviceScale = scale
	fonts.Of(d).SetScale(scale)
	defer func() { deviceScale = 1; fonts.Of(d).SetScale(1) }()
	return Match(t, d, size)
}

// Report is what Match compared.
type Report struct {
	Boxes    int
	MaxDelta float32
}

// Match lays d out in a viewport of size points both in ui and in
// Chrome, and fails t for every element whose box differs by more than
// Tolerance on an edge.
func Match(t testing.TB, d *ui.Document, size geom.Vec2) Report {
	t.Helper()
	var page bytes.Buffer
	if err := export.Write(&page, d, export.Head(script)); err != nil {
		t.Fatal(err)
	}
	want, err := Boxes(page.Bytes(), size)
	if err != nil {
		t.Fatal(err)
	}
	d.Layout(size)
	var got []*dom.Node
	d.HTML().Walk(func(n *dom.Node) bool {
		if !n.IsText() {
			got = append(got, n)
		}
		return true
	})
	if len(got) != len(want) {
		t.Fatalf("ember has %d elements, Chrome %d", len(got), len(want))
	}
	r := Report{Boxes: len(got)}
	for i, n := range got {
		g, w := n.Box(), want[i]
		delta := max(abs(g.X-w.X), abs(g.Y-w.Y), abs(g.X+g.W-w.X-w.W), abs(g.Y+g.H-w.Y-w.H))
		r.MaxDelta = max(r.MaxDelta, delta)
		if delta > Tolerance {
			t.Errorf("%s: ember %v, Chrome %v", path(n), g, w)
		}
	}
	return r
}

func abs(v float32) float32 { return float32(math.Abs(float64(v))) }

func path(n *dom.Node) string {
	s := n.Tag().String()
	if n.ID() != "" {
		s += "#" + n.ID()
	}
	if p := n.Parent(); p != nil {
		return fmt.Sprintf("%s > %s:%d", path(p), s, n.Index())
	}
	return s
}

var stylesJSON = regexp.MustCompile(`<script type="application/json" id="ember-styles">(.*?)</script>`)

// Styles lays d out in Chrome in a viewport of size points, and returns
// getComputedStyle's value of each of props for html, body and body's
// elements, in document order.
func Styles(d *ui.Document, size geom.Vec2, props []string) ([]map[string]string, error) {
	return StylesAfter(d, size, props, "")
}

// StylesAfter is Styles with script run in the page first, such as a
// class change that starts transitions, paused at a time.
func StylesAfter(d *ui.Document, size geom.Vec2, props []string, script string) ([]map[string]string, error) {
	names, err := json.Marshal(props)
	if err != nil {
		return nil, err
	}
	head := `<script>
addEventListener("load", () => document.fonts.ready.then(() => {
	const els = [document.documentElement, document.body,
		...document.body.querySelectorAll("*:not([data-ember-implied])")];
	` + script + `
	const out = document.createElement("script");
	out.type = "application/json";
	out.id = "ember-styles";
	out.textContent = JSON.stringify(els.map(e => {
		const s = getComputedStyle(e);
		return Object.fromEntries(` + string(names) + `.map(p => [p, s.getPropertyValue(p)]));
	}));
	document.head.appendChild(out);
}));
</script>`
	var page bytes.Buffer
	if err = export.Write(&page, d, export.Head(head)); err != nil {
		return nil, err
	}
	out, err := run(page.Bytes(), size, dumpDOM)
	if err != nil {
		return nil, err
	}
	m := stylesJSON.FindSubmatch(out)
	if m == nil {
		return nil, fmt.Errorf("chrome: no styles in the page:\n%s", out)
	}
	var styles []map[string]string
	if err = json.Unmarshal(m[1], &styles); err != nil {
		return nil, err
	}
	return styles, nil
}

var evalJSON = regexp.MustCompile(`<script type="application/json" id="ember-eval">(.*?)</script>`)

// Eval lays d out in Chrome in a viewport of size points, evaluates the
// JavaScript expression expr once the fonts are ready, and returns its
// value as JSON.
func Eval(d *ui.Document, size geom.Vec2, expr string) ([]byte, error) {
	head := `<script>
addEventListener("load", () => document.fonts.ready.then(() => {
	const out = document.createElement("script");
	out.type = "application/json";
	out.id = "ember-eval";
	out.textContent = JSON.stringify((() => { ` + expr + ` })());
	document.head.appendChild(out);
}));
</script>`
	var page bytes.Buffer
	if err := export.Write(&page, d, export.Head(head)); err != nil {
		return nil, err
	}
	out, err := run(page.Bytes(), size, dumpDOM)
	if err != nil {
		return nil, err
	}
	m := evalJSON.FindSubmatch(out)
	if m == nil {
		return nil, fmt.Errorf("chrome: no result in the page:\n%s", out)
	}
	return m[1], nil
}
