package main

import (
	"image/color"
	"math"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// sight is how many cells round him the hero sees.
const sight = 3

// Fog is the fog of war: a tiled picture of one texel per 32-point cell,
// dark where the hero hasn't been, clear where he has.
type Fog struct {
	pic  *twod.TiledPicture
	seen map[[2]int]bool
}

func newFog() *Fog {
	f := &Fog{seen: map[[2]int]bool{}}
	f.pic = twod.NewTiledPicture(f.paint, twod.TexelSize(32, 32), twod.Pixelated())
	return f
}

// paint fills a tile's area, a cell a texel: transparent where seen, the
// night's dark blue elsewhere.
func (f *Fog) paint(tile twod.PictureTile) {
	for y := tile.Area.Min.Y; y < tile.Area.Max.Y; y++ {
		for x := tile.Area.Min.X; x < tile.Area.Max.X; x++ {
			c := color.RGBA{8, 10, 20, 255}
			if f.seen[cellOf(tile.Cell(x, y))] {
				c = color.RGBA{}
			}
			tile.Image.SetRGBA(x, y, c)
		}
	}
}

func cellOf(x, y int) [2]int { return [2]int{x, y} }

// Reveal clears the fog round the hero as he walks: each cell he sees for
// the first time is marked seen, and only its texel is painted again.
type Reveal struct {
	fog  *Fog
	hero *twod.Sprite
}

func (r *Reveal) Update(world.Frame) {
	at := r.hero.Position()
	cx, cy := int(math.Floor(float64(at.X/32))), int(math.Floor(float64(at.Y/32)))
	for y := cy - sight; y <= cy+sight; y++ {
		for x := cx - sight; x <= cx+sight; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy > sight*sight || r.fog.seen[cellOf(x, y)] {
				continue
			}
			r.fog.seen[cellOf(x, y)] = true
			r.fog.pic.MarkDirty(geom.Rect{X: float32(x * 32), Y: float32(y * 32), W: 32, H: 32})
		}
	}
}
