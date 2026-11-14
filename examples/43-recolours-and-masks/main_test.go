package main

import (
	"image"
	stdcolor "image/color"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
)

func rgba(t *testing.T, img image.Image, x, y int) stdcolor.NRGBA {
	t.Helper()
	c, ok := stdcolor.NRGBAModel.Convert(img.At(x, y)).(stdcolor.NRGBA)
	if !ok {
		t.Fatalf("pixel %d,%d is not NRGBA", x, y)
	}
	return c
}

func TestTheVariantsRecolourAndCut(t *testing.T) {
	sprites, err := variants()
	if err != nil {
		t.Fatal(err)
	}
	if len(sprites) != 6 {
		t.Fatalf("three crates and three walls: %d", len(sprites))
	}

	crate, mossy, stony := sprites[0].Texture(), sprites[1].Texture(), sprites[2].Texture()
	wood := stdcolor.NRGBA{0xb5, 0x76, 0x3a, 0xff}
	b := crate.Image.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if rgba(t, crate.Image, x, y) != wood {
				continue
			}
			if got := rgba(t, mossy.Image, x, y); got != (stdcolor.NRGBA{0x6f, 0xa6, 0x4c, 0xff}) {
				t.Fatalf("the moss palette turns the wood at %d,%d into %v", x, y, got)
			}
			if got := rgba(t, stony.Image, x, y); got == wood {
				t.Fatalf("the stone palette leaves the wood at %d,%d", x, y)
			}
		}
	}
	if stony.Recoloured() == nil {
		t.Fatal("the stone crate was recoloured as it loaded")
	}

	cut, holed := sprites[4].Texture(), sprites[5].Texture()
	if cut.Size() != geom.V2(16, 16) || holed.Size() != geom.V2(16, 16) {
		t.Fatalf("a masked tile is its region's size: %v, %v", cut.Size(), holed.Size())
	}
	if rgba(t, cut.Image, 8, 8).A != 255 || rgba(t, cut.Image, 0, 0).A != 0 {
		t.Fatal("the disc keeps the middle of the wall and cuts its corners")
	}
	if rgba(t, holed.Image, 8, 8).A != 0 || rgba(t, holed.Image, 0, 0).A != 255 {
		t.Fatal("the inverted disc cuts the middle of the wall and keeps its corners")
	}
}

func TestEqualAsksShareOneTexture(t *testing.T) {
	crate, err := twod.LoadTexture(files, "crate.png", twod.Pixelated())
	if err != nil {
		t.Fatal(err)
	}
	moss := func() *twod.Palette { return twod.NewPalette(twod.Swap(color.Hex("#b5763a"), color.Hex("#6fa64c"))) }
	a, err := twod.Recolour(crate, moss())
	if err != nil {
		t.Fatal(err)
	}
	b, err := twod.Recolour(crate, moss())
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatal("the same crate through an equal palette is the same texture")
	}
	_, err = twod.Recolour(crate, twod.NewPalette(twod.Swap(color.Hex("#123456"), color.Hex("#654321"))))
	if err == nil {
		t.Fatal("a palette that matches no colour of the crate is an error")
	}
}
