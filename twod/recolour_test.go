package twod_test

import (
	"bytes"
	"image"
	stdcolor "image/color"
	"image/png"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/twod"
)

// The colours the recolour tests swap, as a pack's palette writes them.
//
//nolint:gochecknoglobals // fixed test colours
var (
	blue     = stdcolor.NRGBA{0x68, 0x6f, 0x99, 0xff}
	darkBlue = stdcolor.NRGBA{0x2c, 0x35, 0x4d, 0xff}
	grey     = stdcolor.NRGBA{0x40, 0x49, 0x73, 0xff}
	black    = stdcolor.NRGBA{0x14, 0x18, 0x2e, 0xff}
)

func rgb8(c stdcolor.NRGBA) color.Color { return color.RGB8(c.R, c.G, c.B) }

func nrgbaAt(img image.Image, x, y int) stdcolor.NRGBA {
	c, _ := stdcolor.NRGBAModel.Convert(img.At(x, y)).(stdcolor.NRGBA)
	return c
}

func picture(w, h int, set func(img *image.NRGBA)) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	set(img)
	return img
}

func darkWater() *twod.Palette {
	return twod.NewPalette(
		twod.Swap(rgb8(blue), rgb8(darkBlue)),
		twod.Swap(rgb8(grey), rgb8(black)),
	)
}

func TestARecolourSwapsExactColoursAndKeepsAlpha(t *testing.T) {
	near := stdcolor.NRGBA{0x68, 0x6f, 0x9a, 0xff}
	other := stdcolor.NRGBA{0xff, 0x00, 0x00, 0xff}
	half := blue
	half.A = 0x80
	src := twod.NewTexture(picture(5, 1, func(img *image.NRGBA) {
		img.SetNRGBA(0, 0, blue)
		img.SetNRGBA(1, 0, half)
		img.SetNRGBA(2, 0, near)
		img.SetNRGBA(3, 0, other)
		img.SetNRGBA(4, 0, grey)
	}), twod.Pixelated())
	got, err := twod.Recolour(src, darkWater())
	if err != nil {
		t.Fatal(err)
	}
	halfDark := darkBlue
	halfDark.A = 0x80
	for x, want := range []stdcolor.NRGBA{darkBlue, halfDark, near, other, black} {
		if c := nrgbaAt(got.Image, x, 0); c != want {
			t.Errorf("pixel %d is %v, want %v", x, c, want)
		}
	}
	if c := nrgbaAt(src.Image, 0, 0); c != blue {
		t.Errorf("the source keeps its colours: %v", c)
	}
	if got.Filter != src.Filter {
		t.Error("the recoloured texture samples as its source does")
	}
}

func TestAPaletteThatMatchesNothingIsRefused(t *testing.T) {
	red := twod.NewTexture(picture(2, 2, func(img *image.NRGBA) { img.SetNRGBA(0, 0, stdcolor.NRGBA{R: 0xff, A: 0xff}) }))
	if _, err := twod.Recolour(red, darkWater()); err == nil {
		t.Fatal("a palette none of whose colours the picture holds is an error")
	}
	var b bytes.Buffer
	if err := png.Encode(&b, red.Image); err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{"red.png": &fstest.MapFile{Data: b.Bytes()}}
	if _, err := twod.LoadTexture(fsys, "red.png", twod.Recoloured(darkWater())); err == nil ||
		!strings.Contains(err.Error(), "red.png") {
		t.Fatalf("loading it is refused, naming the picture: %v", err)
	}
}

func TestTwoEqualAsksGiveOneTexture(t *testing.T) {
	src := twod.NewTexture(picture(2, 1, func(img *image.NRGBA) { img.SetNRGBA(0, 0, blue) }))
	first, err := twod.Recolour(src, darkWater())
	if err != nil {
		t.Fatal(err)
	}
	again, err := twod.Recolour(src, darkWater())
	if err != nil {
		t.Fatal(err)
	}
	if first != again {
		t.Fatal("the same source through an equal palette is one texture")
	}
	other, err := twod.Recolour(src, twod.NewPalette(twod.Swap(rgb8(blue), rgb8(black))))
	if err != nil {
		t.Fatal(err)
	}
	copied := twod.NewTexture(src.Image)
	elsewhere, err := twod.Recolour(copied, darkWater())
	if err != nil {
		t.Fatal(err)
	}
	if other == first || elsewhere == first {
		t.Fatal("another palette, or another texture, gives another texture")
	}
}

func TestAnAnimatedSheetIsRecolouredWhole(t *testing.T) {
	const frame = 4
	sheet := twod.NewTexture(picture(frame*4, frame, func(img *image.NRGBA) {
		for f := range 4 {
			img.SetNRGBA(f*frame+f, f%frame, blue)
		}
	}))
	got, err := twod.Recolour(sheet, darkWater())
	if err != nil {
		t.Fatal(err)
	}
	if got.Size() != sheet.Size() {
		t.Fatalf("the sheet keeps its size: %v, %v", got.Size(), sheet.Size())
	}
	for f := range 4 {
		if c := nrgbaAt(got.Image, f*frame+f, f%frame); c != darkBlue {
			t.Errorf("frame %d's pixel is %v, recoloured where it was", f, c)
		}
	}
}

func TestATextureLoadedRecolouredSaysItsPalette(t *testing.T) {
	var b bytes.Buffer
	if err := png.Encode(&b, picture(2, 1, func(img *image.NRGBA) { img.SetNRGBA(0, 0, blue) })); err != nil {
		t.Fatal(err)
	}
	pal := darkWater()
	tex, err := twod.LoadTexture(fstest.MapFS{"water.png": &fstest.MapFile{Data: b.Bytes()}}, "water.png",
		twod.Recoloured(pal))
	if err != nil {
		t.Fatal(err)
	}
	if tex.Recoloured() != pal || nrgbaAt(tex.Image, 0, 0) != darkBlue {
		t.Fatalf("loaded recoloured, through its palette: %v", nrgbaAt(tex.Image, 0, 0))
	}
	if twod.NewTexture(tex.Image).Recoloured() != nil {
		t.Fatal("a texture with no palette says none")
	}
}

// swatches is a palette picture: a row of from colours above a row of to
// colours, each a size×size square.
func swatches(size int, from, to []stdcolor.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, len(from)*size, 2*size))
	for i := range from {
		for y := range size {
			for x := range size {
				img.SetNRGBA(i*size+x, y, from[i])
				img.SetNRGBA(i*size+x, size+y, to[i])
			}
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func TestAPalettePictureIsTwoRowsOfSwatches(t *testing.T) {
	fsys := fstest.MapFS{
		"dark-water.png": &fstest.MapFile{Data: swatches(4, []stdcolor.NRGBA{blue, grey}, []stdcolor.NRGBA{darkBlue, black})},
		"clash.png":      &fstest.MapFile{Data: swatches(4, []stdcolor.NRGBA{blue, blue}, []stdcolor.NRGBA{darkBlue, black})},
		"odd.png":        &fstest.MapFile{Data: swatches(4, []stdcolor.NRGBA{blue}, []stdcolor.NRGBA{darkBlue})[:0]},
	}
	pal, err := twod.LoadPalette(fsys, "dark-water.png")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(pal.Swaps(), darkWater().Swaps()) {
		t.Fatalf("the swatches above become the ones below: %v", pal.Swaps())
	}
	for _, bad := range []string{"clash.png", "odd.png", "missing.png"} {
		if _, err = twod.LoadPalette(fsys, bad); err == nil || !strings.Contains(err.Error(), bad) {
			t.Errorf("%s is refused, naming it: %v", bad, err)
		}
	}
	square := fstest.MapFS{"square.png": &fstest.MapFile{Data: func() []byte {
		var b bytes.Buffer
		_ = png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 9, 4)))
		return b.Bytes()
	}()}}
	if _, err = twod.LoadPalette(square, "square.png"); err == nil {
		t.Error("a picture that isn't two rows of square swatches is refused")
	}
}

// slime is a sheet as Lantervale's green slime is drawn: its outline opaque
// and its shadow the same colour at alpha 100.
func slime() (*twod.Texture, stdcolor.NRGBA, stdcolor.NRGBA) {
	outline := stdcolor.NRGBA{0x28, 0x35, 0x40, 0xff}
	shadow := outline
	shadow.A = 100
	return twod.NewTexture(picture(2, 1, func(img *image.NRGBA) {
		img.SetNRGBA(0, 0, outline)
		img.SetNRGBA(1, 0, shadow)
	})), outline, shadow
}

func recoloured(t *testing.T, tex *twod.Texture, swaps ...twod.ColorSwap) []stdcolor.NRGBA {
	t.Helper()
	got, err := twod.Recolour(tex, twod.NewPalette(swaps...))
	if err != nil {
		t.Fatal(err)
	}
	b := got.Image.Bounds()
	px := make([]stdcolor.NRGBA, 0, b.Dx())
	for x := range b.Dx() {
		px = append(px, nrgbaAt(got.Image, x, 0))
	}
	return px
}

func withAlpha(c stdcolor.NRGBA, a uint8) stdcolor.NRGBA {
	c.A = a
	return c
}

func TestASwapAtAnAlphaLeavesTheColourAtOthers(t *testing.T) {
	tex, outline, shadow := slime()
	got := recoloured(t, tex, twod.SwapAt(rgb8(outline), 0xff, rgb8(black)))
	if want := []stdcolor.NRGBA{black, shadow}; !slices.Equal(got, want) {
		t.Fatalf("the opaque outline is swapped and the shadow left: %v, want %v", got, want)
	}
	got = recoloured(t, tex, twod.Swap(rgb8(outline), rgb8(black)))
	if want := []stdcolor.NRGBA{black, withAlpha(black, 100)}; !slices.Equal(got, want) {
		t.Fatalf("a swap without an alpha takes every alpha, keeping it: %v, want %v", got, want)
	}
}

func TestASwapAtAnAlphaWinsOverOneAtAny(t *testing.T) {
	tex, outline, _ := slime()
	half := twod.NewTexture(picture(1, 1, func(img *image.NRGBA) { img.SetNRGBA(0, 0, withAlpha(outline, 50)) }))
	for _, swaps := range [][]twod.ColorSwap{
		{twod.Swap(rgb8(outline), rgb8(black)), twod.SwapAt(rgb8(outline), 100, rgb8(blue))},
		{twod.SwapAt(rgb8(outline), 100, rgb8(blue)), twod.Swap(rgb8(outline), rgb8(black))},
	} {
		got, want := recoloured(t, tex, swaps...), []stdcolor.NRGBA{black, withAlpha(blue, 100)}
		if !slices.Equal(got, want) {
			t.Errorf("%v: %v, want %v", swaps, got, want)
		}
		if other := recoloured(t, half, swaps...); other[0] != withAlpha(black, 50) {
			t.Errorf("%v: another alpha takes the swap at any: %v", swaps, other[0])
		}
	}
}

func TestASwapAtAnAlphaIsItsOwnPalette(t *testing.T) {
	tex, outline, _ := slime()
	at := func() *twod.Palette { return twod.NewPalette(twod.SwapAt(rgb8(outline), 0xff, rgb8(black))) }
	first, err := twod.Recolour(tex, at())
	if err != nil {
		t.Fatal(err)
	}
	again, err := twod.Recolour(tex, at())
	if err != nil {
		t.Fatal(err)
	}
	anyAlpha, err := twod.Recolour(tex, twod.NewPalette(twod.Swap(rgb8(outline), rgb8(black))))
	if err != nil {
		t.Fatal(err)
	}
	if first != again || anyAlpha == first {
		t.Fatal("two equal palettes share a texture, a swap at any alpha is another palette")
	}
	twice := twod.NewPalette(twod.SwapAt(rgb8(outline), 100, rgb8(blue)), twod.SwapAt(rgb8(outline), 100, rgb8(black)),
		twod.Swap(rgb8(outline), rgb8(grey)))
	if want := []twod.ColorSwap{
		twod.SwapAt(rgb8(outline), 100, rgb8(black)), twod.Swap(rgb8(outline), rgb8(grey)),
	}; !slices.Equal(twice.Swaps(), want) {
		t.Fatalf("a colour swapped twice at one alpha takes its last To: %v", twice.Swaps())
	}
}

func TestASwapAtAnAlphaThePictureLacksMatchesNothing(t *testing.T) {
	opaque := twod.NewTexture(picture(1, 1, func(img *image.NRGBA) { img.SetNRGBA(0, 0, blue) }))
	if _, err := twod.Recolour(opaque, twod.NewPalette(twod.SwapAt(rgb8(blue), 100, rgb8(black)))); err == nil {
		t.Fatal("a palette whose colour the picture holds at no alpha it names is refused")
	}
}

func TestASwapAtAlphaZeroMatchesNothing(t *testing.T) {
	tex, outline, _ := slime()
	atZero := twod.SwapAt(rgb8(outline), 0, rgb8(blue))
	if _, err := twod.Recolour(tex, twod.NewPalette(atZero)); err == nil {
		t.Fatal("a palette of one swap at alpha 0 is refused")
	}
	for _, swaps := range [][]twod.ColorSwap{
		{twod.Swap(rgb8(outline), rgb8(black)), atZero},
		{atZero, twod.Swap(rgb8(outline), rgb8(black))},
	} {
		got, want := recoloured(t, tex, swaps...), []stdcolor.NRGBA{black, withAlpha(black, 100)}
		if !slices.Equal(got, want) {
			t.Errorf("%v: the swap at any alpha still applies: %v, want %v", swaps, got, want)
		}
	}
}

func TestAHalfClearSwatchNamesItsAlpha(t *testing.T) {
	shadow := withAlpha(blue, 100)
	fsys := fstest.MapFS{
		"slime.png": &fstest.MapFile{Data: swatches(4,
			[]stdcolor.NRGBA{blue, shadow}, []stdcolor.NRGBA{darkBlue, withAlpha(black, 100)})},
		"unlike.png": &fstest.MapFile{Data: swatches(2, []stdcolor.NRGBA{shadow}, []stdcolor.NRGBA{black})},
		"clear.png": &fstest.MapFile{
			Data: swatches(4, []stdcolor.NRGBA{withAlpha(blue, 0)}, []stdcolor.NRGBA{withAlpha(black, 0)}),
		},
	}
	pal, err := twod.LoadPalette(fsys, "slime.png")
	if err != nil {
		t.Fatal(err)
	}
	if want := []twod.ColorSwap{
		twod.Swap(rgb8(blue), rgb8(darkBlue)), twod.SwapAt(rgb8(blue), 100, rgb8(black)),
	}; !slices.Equal(pal.Swaps(), want) {
		t.Fatalf("an opaque swatch swaps at any alpha, a half-clear one at its own: %v", pal.Swaps())
	}
	for _, bad := range []string{"unlike.png", "clear.png"} {
		if _, err = twod.LoadPalette(fsys, bad); err == nil || !strings.Contains(err.Error(), bad) {
			t.Errorf("%s is refused, naming it: %v", bad, err)
		}
	}
}

func BenchmarkRecolourAWaterSheet(b *testing.B) {
	// lantervale's water sheet is 384×48, a few dozen colours, about half
	// of them swapped.
	sheet := picture(384, 48, func(img *image.NRGBA) {
		for y := range 48 {
			for x := range 384 {
				switch (x + y) % 4 {
				case 0:
					img.SetNRGBA(x, y, blue)
				case 1:
					img.SetNRGBA(x, y, grey)
				default:
					img.SetNRGBA(x, y, stdcolor.NRGBA{uint8(x), uint8(y), 0x40, 0xff})
				}
			}
		}
	})
	pal := darkWater()
	for b.Loop() {
		if _, err := twod.Recolour(twod.NewTexture(sheet), pal); err != nil {
			b.Fatal(err)
		}
	}
}
