//go:build chrome

package chrome

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/merlindorin/ember/geom"
)

// Diff is how two pictures of the same page differ.
type Diff struct {
	// Mean is the mean difference per channel, 0 to 1, over every pixel.
	Mean float64
	// Pixels is how many pixels differ by more than 8 levels on a channel.
	Pixels, Total int
	// Report is ember | Chrome | the differing pixels in red.
	Report *image.NRGBA
}

// Compare compares two pictures of the same size, a ember's and b
// Chrome's.
func Compare(a, b image.Image) Diff {
	r := a.Bounds()
	w, h := r.Dx(), r.Dy()
	d := Diff{Total: w * h, Report: image.NewNRGBA(image.Rect(0, 0, 3*w, h))}
	draw.Draw(d.Report, image.Rect(0, 0, w, h), a, r.Min, draw.Src)
	draw.Draw(d.Report, image.Rect(w, 0, 2*w, h), b, b.Bounds().Min, draw.Src)
	draw.Draw(d.Report, image.Rect(2*w, 0, 3*w, h), image.White, image.Point{}, draw.Src)
	var sum float64
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ca := nrgba(a.At(r.Min.X+x, r.Min.Y+y))
			cb := nrgba(b.At(b.Bounds().Min.X+x, b.Bounds().Min.Y+y))
			most := 0
			for _, p := range [][2]uint8{{ca.R, cb.R}, {ca.G, cb.G}, {ca.B, cb.B}} {
				diff := int(p[0]) - int(p[1])
				if diff < 0 {
					diff = -diff
				}
				sum += float64(diff)
				most = max(most, diff)
			}
			if most > 8 {
				d.Pixels++
				d.Report.Set(2*w+x, y, color.NRGBA{R: 220, A: 255})
			}
		}
	}
	d.Mean = sum / float64(3*w*h) / 255
	return d
}

// pageEnv is environ for a page's run: hidden, and at 2 pixels a point
// unless EMBER_SCALE says otherwise, since the thresholds are measured at
// 2x and a 1x display (a CI runner) antialiases text and edges coarser.
func pageEnv(environ []string) []string {
	env := append(slices.Clone(environ), "EMBER_HIDDEN=1")
	if !slices.ContainsFunc(environ, func(e string) bool { return strings.HasPrefix(e, "EMBER_SCALE=") }) {
		env = append(env, "EMBER_SCALE=2")
	}
	return env
}

// nrgba is c in non-premultiplied 8-bit colour.
func nrgba(c color.Color) color.NRGBA {
	n, ok := color.NRGBAModel.Convert(c).(color.NRGBA)
	if !ok {
		panic(fmt.Sprintf("chrome: the NRGBA model gave %T", n))
	}
	return n
}

// Pixels runs the page program pkg (see ui/internal/pages) twice, once
// captured by ember and once exported for Chrome at the capture's scale,
// and compares the pictures. The report is saved as <name>.png in
// EMBER_UI_REPORTS when it's set, else in a test folder; the captures stay in
// the test's own folder, so tests running side by side never share them.
func Pixels(t testing.TB, pkg string, w, h int) Diff {
	t.Helper()
	dir, reports := t.TempDir(), t.TempDir()
	if out := os.Getenv("EMBER_UI_REPORTS"); out != "" {
		reports = out
	}
	shot, page := filepath.Join(dir, "ember.png"), filepath.Join(dir, "page.html")
	run := func(env ...string) {
		cmd := exec.CommandContext(t.Context(), "go", "run", pkg)
		cmd.Env = append(pageEnv(os.Environ()), env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: %v: %s", pkg, err, out)
		}
	}
	run("EMBER_CAPTURE="+shot, "EMBER_CAPTURE_FRAME=10")
	run("EMBER_UI_EXPORT="+page, "EMBER_UI_SNAPSHOTS="+shot)
	a := load(t, shot)
	html, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	scale := float32(a.Bounds().Dx()) / float32(w)
	cpng := filepath.Join(dir, "chrome.png")
	if err = ScreenshotScaled(html, geom.V2(float32(w), float32(h)), scale, cpng); err != nil {
		t.Fatal(err)
	}
	b := load(t, cpng)
	if b.Bounds().Size() != a.Bounds().Size() {
		t.Fatalf("ember's capture is %v, Chrome's %v", a.Bounds().Size(), b.Bounds().Size())
	}
	d := Compare(a, b)
	report := filepath.Join(reports, filepath.Base(pkg)+".png")
	// The report is for reading the difference; failing to write it
	// leaves the comparison as it is.
	f, err := os.Create(report) //nolint:gosec // the folder the gate names in EMBER_UI_REPORTS
	if err == nil {
		if err = png.Encode(f, d.Report); err != nil {
			t.Logf("%s: %v", report, err)
		}
		f.Close()
	}
	t.Logf(
		"%s: mean difference %.4f, %d of %d pixels differ; report %s",
		filepath.Base(pkg),
		d.Mean,
		d.Pixels,
		d.Total,
		report,
	)
	return d
}

func load(t testing.TB, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(fmt.Errorf("%s: %w", path, err))
	}
	return img
}
