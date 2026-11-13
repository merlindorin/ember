// Recolours and masks: variants of a picture made once, at load. Above, the
// crate in its own colours, through a palette made in code, and through
// one loaded from a picture of swatches; below, a wall tile as it is, cut
// by a disc mask, and by the same mask inverted.
//
//	go run ./examples/43-recolours-and-masks
package main

import (
	"embed"
	"log"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

//go:embed crate.png stone.png tiles.png disc.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · recolours and masks"),
		platform.Size(640, 360),
		platform.FixedSize(),
	)
	sprites, err := variants()
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range sprites {
		win.Add(s)
	}

	ember.New().
		AddEntities(win).
		MustRun()
}

// variants makes the six pictures and stands them in two rows.
func variants() ([]*twod.Sprite, error) {
	// wall is the brick tile in tiles.png, and disc the whole of disc.png.
	wall := geom.Rect{X: 16, W: 16, H: 16}
	disc := geom.Rect{W: 16, H: 16}

	crate, err := twod.LoadTexture(files, "crate.png", twod.Pixelated())
	if err != nil {
		return nil, err
	}
	moss := twod.NewPalette(
		twod.Swap(color.Hex("#d99a5b"), color.Hex("#a8d47e")),
		twod.Swap(color.Hex("#b5763a"), color.Hex("#6fa64c")),
		twod.Swap(color.Hex("#8a5226"), color.Hex("#447a33")),
		twod.Swap(color.Hex("#5a341c"), color.Hex("#264721")),
	)
	mossy, err := twod.Recolour(crate, moss)
	if err != nil {
		return nil, err
	}
	stone, err := twod.LoadPalette(files, "stone.png")
	if err != nil {
		return nil, err
	}
	stony, err := twod.LoadTexture(files, "crate.png", twod.Pixelated(), twod.Recoloured(stone))
	if err != nil {
		return nil, err
	}

	tiles, err := twod.LoadTexture(files, "tiles.png", twod.Pixelated())
	if err != nil {
		return nil, err
	}
	mask, err := twod.LoadTexture(files, "disc.png")
	if err != nil {
		return nil, err
	}
	cut, err := twod.Masked(tiles, wall, mask, disc)
	if err != nil {
		return nil, err
	}
	holed, err := twod.Masked(tiles, wall, mask, disc, twod.Inverted())
	if err != nil {
		return nil, err
	}

	return []*twod.Sprite{
		twod.NewSprite(crate, twod.At(160, 110), twod.Scale(6, 6)),
		twod.NewSprite(mossy, twod.At(320, 110), twod.Scale(6, 6)),
		twod.NewSprite(stony, twod.At(480, 110), twod.Scale(6, 6)),
		twod.NewSprite(tiles, twod.Region(wall), twod.At(160, 260), twod.Scale(6, 6)),
		twod.NewSprite(cut, twod.At(320, 260), twod.Scale(6, 6)),
		twod.NewSprite(holed, twod.At(480, 260), twod.Scale(6, 6)),
	}, nil
}
