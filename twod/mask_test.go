package twod_test

import (
	"image"
	stdcolor "image/color"
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// maskTest is a 4×1 tile of one opaque colour at x 4–7 of an 8×1 sheet, and
// a 4×1 mask at x 2–5 of its own sheet: opaque, clear, half, opaque.
func maskTest() (*twod.Texture, geom.Rect, *twod.Texture, geom.Rect) {
	tile := twod.NewTexture(picture(8, 1, func(img *image.NRGBA) {
		for x := range 8 {
			img.SetNRGBA(x, 0, stdcolor.NRGBA{R: uint8(x * 30), G: 0x80, B: 0x40, A: 0xff})
		}
	}))
	mask := twod.NewTexture(picture(6, 1, func(img *image.NRGBA) {
		for x, a := range []uint8{0, 0, 0xff, 0, 0x80, 0xff} {
			img.SetNRGBA(x, 0, stdcolor.NRGBA{A: a})
		}
	}))
	return tile, geom.Rect{X: 4, W: 4, H: 1}, mask, geom.Rect{X: 2, W: 4, H: 1}
}

func TestAMaskKeepsWhereOpaqueCutsWhereClearAndFadesBetween(t *testing.T) {
	tile, region, mask, maskRegion := maskTest()
	cut, err := twod.Masked(tile, region, mask, maskRegion)
	if err != nil {
		t.Fatal(err)
	}
	if cut.Size() != geom.V2(4, 1) {
		t.Fatalf("a masked texture is its region's size: %v", cut.Size())
	}
	src := func(x int) stdcolor.NRGBA { return nrgbaAt(tile.Image, 4+x, 0) }
	for x, wantA := range []uint8{0xff, 0, 0x80, 0xff} {
		got, want := nrgbaAt(cut.Image, x, 0), src(x)
		want.A = wantA
		if wantA == 0 {
			if got.A != 0 {
				t.Errorf("pixel %d is cut where the mask is clear: %v", x, got)
			}
			continue
		}
		if got != want {
			t.Errorf("pixel %d is %v, want %v: its colour as drawn, its alpha the mask's", x, got, want)
		}
	}
}

func TestAnInvertedMaskKeepsWhereItIsClear(t *testing.T) {
	tile, region, mask, maskRegion := maskTest()
	cut, err := twod.Masked(tile, region, mask, maskRegion, twod.Inverted())
	if err != nil {
		t.Fatal(err)
	}
	for x, wantA := range []uint8{0, 0xff, 0x7f, 0} {
		if got := nrgbaAt(cut.Image, x, 0).A; got != wantA {
			t.Errorf("pixel %d's alpha is %#x, want %#x: the plug of the hole", x, got, wantA)
		}
	}
}

func TestAMaskOfAnotherSizeOrOutsideItsPictureIsRefused(t *testing.T) {
	tile, region, mask, maskRegion := maskTest()
	for _, c := range []struct {
		what               string
		region, maskRegion geom.Rect
	}{
		{"a mask smaller than the region", region, geom.Rect{X: 2, W: 3, H: 1}},
		{"a region outside its texture", geom.Rect{X: 6, W: 4, H: 1}, maskRegion},
		{"a mask region outside its mask", region, geom.Rect{X: 4, W: 4, H: 1}},
		{"a region off whole pixels", geom.Rect{X: 4.5, W: 3, H: 1}, geom.Rect{X: 2, W: 3, H: 1}},
	} {
		if _, err := twod.Masked(tile, c.region, mask, c.maskRegion); err == nil {
			t.Errorf("%s is refused", c.what)
		}
	}
}

func TestTwoEqualMasksGiveOneTexture(t *testing.T) {
	tile, region, mask, maskRegion := maskTest()
	first, err := twod.Masked(tile, region, mask, maskRegion)
	if err != nil {
		t.Fatal(err)
	}
	again, err := twod.Masked(tile, region, mask, maskRegion)
	if err != nil {
		t.Fatal(err)
	}
	inverted, err := twod.Masked(tile, region, mask, maskRegion, twod.Inverted())
	if err != nil {
		t.Fatal(err)
	}
	elsewhere, err := twod.Masked(tile, geom.Rect{X: 3, W: 4, H: 1}, mask, maskRegion)
	if err != nil {
		t.Fatal(err)
	}
	if first != again {
		t.Fatal("the same cut asked twice is one texture")
	}
	if inverted == first || elsewhere == first {
		t.Fatal("an inverted mask, or another region, is another texture")
	}
}

func TestARecolouredTileIsMaskedAfter(t *testing.T) {
	tile := twod.NewTexture(picture(2, 1, func(img *image.NRGBA) {
		img.SetNRGBA(0, 0, blue)
		img.SetNRGBA(1, 0, blue)
	}))
	mask := twod.NewTexture(picture(2, 1, func(img *image.NRGBA) { img.SetNRGBA(0, 0, stdcolor.NRGBA{A: 0xff}) }))
	dark, err := twod.Recolour(tile, darkWater())
	if err != nil {
		t.Fatal(err)
	}
	whole := geom.Rect{W: 2, H: 1}
	cut, err := twod.Masked(dark, whole, mask, whole)
	if err != nil {
		t.Fatal(err)
	}
	if nrgbaAt(cut.Image, 0, 0) != darkBlue || nrgbaAt(cut.Image, 1, 0).A != 0 {
		t.Fatalf("recolour, then mask: %v %v", nrgbaAt(cut.Image, 0, 0), nrgbaAt(cut.Image, 1, 0))
	}
}

func TestOnlyAMasksOpacityCountsAndARegionMayTouchItsEdge(t *testing.T) {
	tile, region, _, _ := maskTest()
	if region.X+region.W != tile.Size().X {
		t.Fatal("the tile's region ends at its sheet's right edge")
	}
	red := twod.NewTexture(picture(4, 1, func(img *image.NRGBA) {
		for x := range 4 {
			img.SetNRGBA(x, 0, stdcolor.NRGBA{R: 0xff, A: 0xff})
		}
	}))
	cut, err := twod.Masked(tile, region, red, geom.Rect{W: 4, H: 1})
	if err != nil {
		t.Fatal(err)
	}
	for x := range 4 {
		if got, want := nrgbaAt(cut.Image, x, 0), nrgbaAt(tile.Image, 4+x, 0); got != want {
			t.Errorf("pixel %d is %v: an opaque red mask keeps the tile as drawn, %v", x, got, want)
		}
	}
}
