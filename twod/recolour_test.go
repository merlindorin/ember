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

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/twod"
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
