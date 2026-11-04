package golden //nolint:testpackage // white-box: checks which references Update rewrites

import (
	"image"
	"image/color"
	"path/filepath"
	"testing"
)

func TestUpdateRewritesOnlyWhatFailsOrIsNamed(t *testing.T) {
	for _, c := range []struct {
		run, name      string
		missing, fails bool
		want           bool
	}{
		{"Golden", "09-window", false, false, false},
		{"Golden", "09-window", false, true, true},
		{"Golden", "new-scene", true, false, true},
		{"Golden/look3d", "look3d", false, false, true},
		{"Golden/look3d", "look3d-post", false, false, false},
		{"Golden/look3d.*", "look3d-post", false, false, true},
		{"Golden/(", "look3d", false, false, false},
	} {
		if got := rewrite(c.run, c.name, c.missing, c.fails); got != c.want {
			t.Errorf("-run %q, %s (missing %v, fails %v): rewrite %v, want %v", c.run, c.name, c.missing, c.fails, got, c.want)
		}
	}
}

func TestAPictureIsHalvedToPointsByAveraging(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	for x, v := range []uint8{0, 100, 200, 201} {
		for y := range 2 {
			img.SetNRGBA(x, y, color.NRGBA{v, v, v, 255})
		}
	}
	half := ToPoints(t, img)
	if half.Bounds().Dx() != 2 || half.Bounds().Dy() != 1 {
		t.Fatalf("half the size: %v", half.Bounds())
	}
	if got := half.NRGBAAt(0, 0).R; got != 50 {
		t.Fatalf("each point the mean of its 2×2 pixels: %d", got)
	}
	if got := half.NRGBAAt(1, 0).R; got != 201 {
		t.Fatalf("rounded to nearest: %d", got)
	}
}

func TestAPixelMatchesWithinOneLevelAndIsMarkedPast(t *testing.T) {
	want := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	got := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	for x, d := range []uint8{0, Levels, Levels + 3} {
		want.SetNRGBA(x, 0, color.NRGBA{100, 100, 100, 255})
		got.SetNRGBA(x, 0, color.NRGBA{100, 100 + d, 100, 255})
	}
	d, marks := Compare(t, want, got)
	if d.Over != 1 || d.Most != Levels+3 || d.Total != 3 {
		t.Fatalf("one pixel past the levels, by %d: %+v", Levels+3, d)
	}
	if marks.NRGBAAt(2, 0) != (color.NRGBA{255, 0, 0, 255}) || marks.NRGBAAt(1, 0).R == 255 {
		t.Fatal("the pixel past them is marked red, the others dimmed")
	}
}

func TestASceneLikeItsReferencePassesAndAPixelOffCounts(t *testing.T) {
	ref := filepath.Join(t.TempDir(), "scene.png")
	pic := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	Save(t, ref, pic)
	check(t, Scene{Name: "scene"}, ref, pic) // the same picture passes: it would fail t
	off := image.NewNRGBA(pic.Bounds())
	off.SetNRGBA(0, 0, color.NRGBA{200, 0, 0, 255})
	if d, _ := Compare(t, Load(t, ref), off); d.Over != 1 {
		t.Fatalf("a pixel off is counted: %+v", d)
	}
}
