// Tiledvariants renders the Recolours and masks page's Tiled picture: on
// the left the fixture's tiles and masks as Tiled shows them, on the right
// its map as the engine draws it, the floor recoloured by its layer and
// cut by its mask layer, the stone tinted by its tile, and two tile
// objects, one recoloured, one masked and tinted.
//
//	EMBER_HIDDEN=1 EMBER_CAPTURE=variants.png go run ./examples/internal/tiledvariants -dir twod/tiled/variants
package main

import (
	"flag"
	"io/fs"
	"log"
	"os"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

// scale is how many points a fixture pixel takes.
const scale = 6

func main() {
	dir := flag.String("dir", "twod/tiled/variants", "the variants fixture's folder")
	flag.Parse()
	win, err := scene(os.DirFS(*dir))
	if err != nil {
		log.Fatal(err)
	}
	ember.New().
		AddEntities(win).
		MustRun()
}

// palettes are the ones the fixture names: dark turns the water and its
// deep corner darker, deep only the water black.
func palettes() map[string]*twod.Palette {
	water, deep := color.Hex("#686f99"), color.Hex("#404973")
	return map[string]*twod.Palette{
		"dark": twod.NewPalette(twod.Swap(water, color.Hex("#2c354d")), twod.Swap(deep, color.Hex("#14182e"))),
		"deep": twod.NewPalette(twod.Swap(water, color.Hex("#101010"))),
	}
}

// scene is the window: the fixture's pictures, then its map.
func scene(fsys fs.FS) (*platform.Window, error) {
	win := platform.NewWindow(
		platform.Title("ember · recolours and masks in Tiled"),
		platform.Size(672, 336),
		platform.FixedSize(),
		platform.Background(color.Hex("#5a6070")),
	)
	masks, err := twod.LoadTiledTileset(fsys, "masks.tsj")
	if err != nil {
		return nil, err
	}
	level, err := twod.LoadTiled(fsys, "variants.tmj",
		twod.Palettes(palettes()), twod.Masks(masks), twod.PixelScale(scale))
	if err != nil {
		return nil, err
	}
	for i, name := range []string{"tiles.png", "masks.png"} {
		tex, texErr := twod.LoadTexture(fsys, name, twod.Pixelated())
		if texErr != nil {
			return nil, texErr
		}
		win.Add(twod.NewSprite(tex, twod.Pivot(0, 0), twod.At(-240, float32(i)*192), twod.Scale(scale, scale)))
	}
	for _, m := range level.TileLayers() {
		win.Add(m)
	}
	for _, o := range level.Objects("props") {
		win.Add(level.Sprite(o))
	}
	return win.Add(twod.NewCamera(twod.At(72, 144))), nil
}
