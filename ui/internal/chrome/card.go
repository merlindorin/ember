//go:build chrome

package chrome

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Card compares a component's page as Pixels does, fails t past
// threshold, and writes its review card, ember | Chrome | the differing
// pixels under its name and mean difference, to EMBER_UI_CARDS/<NN-name>.png
// for a page named kit-<NN-name>.
func Card(t testing.TB, pkg string, w, h int, threshold float64) Diff {
	t.Helper()
	d := Pixels(t, pkg, w, h)
	check(t, "card", filepath.Base(pkg), d, threshold)
	dir := os.Getenv("EMBER_UI_CARDS")
	if dir == "" {
		return d
	}
	name := strings.TrimPrefix(filepath.Base(pkg), "kit-")
	card := cardOf(d, name, threshold)
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // the folder the gate names in EMBER_UI_CARDS
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name+".png")) //nolint:gosec // the folder the gate names in EMBER_UI_CARDS
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = png.Encode(f, card); err != nil {
		t.Fatal(err)
	}
	t.Logf("card %s", f.Name())
	return d
}

func cardOf(d Diff, name string, threshold float64) *image.NRGBA {
	r := d.Report.Bounds()
	third := r.Dx() / 3
	head := 140
	out := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()+head))
	draw.Draw(out, out.Bounds(), image.NewUniform(color.NRGBA{24, 22, 20, 255}), image.Point{}, draw.Src)
	draw.Draw(out, image.Rect(0, head, r.Dx(), head+r.Dy()), d.Report, r.Min, draw.Src)
	title := face(gobold.TTF, 36)
	label := face(goregular.TTF, 26)
	text := func(f font.Face, x, y int, s string, c color.Color) {
		(&font.Drawer{Dst: out, Src: image.NewUniform(c), Face: f, Dot: fixed.P(x, y)}).DrawString(s)
	}
	light, muted := color.NRGBA{247, 239, 221, 255}, color.NRGBA{184, 177, 166, 255}
	verdict := color.NRGBA{120, 200, 120, 255}
	if d.Mean > threshold {
		verdict = color.NRGBA{230, 90, 80, 255}
	}
	text(title, 24, 52, name, light)
	line := fmt.Sprintf(
		"mean difference %.4f (threshold %.4f) · %d of %d pixels differ",
		d.Mean,
		threshold,
		d.Pixels,
		d.Total,
	)
	if threshold >= 1 {
		// A placeholder threshold: the card isn't final.
		verdict = color.NRGBA{230, 90, 80, 255}
		line = fmt.Sprintf("mean difference %.4f · THRESHOLD NOT SET, not a final card", d.Mean)
	}
	text(label, 24, 92, line, verdict)
	for i, s := range []string{"ember", "Chrome", "differs"} {
		text(label, i*third+24, 128, s, muted)
	}
	return out
}

func face(ttf []byte, size float64) font.Face {
	f, err := opentype.Parse(ttf)
	if err != nil {
		panic(err)
	}
	fc, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72})
	if err != nil {
		panic(err)
	}
	return fc
}
