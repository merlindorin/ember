package light //nolint:testpackage // beside light's other tests

import (
	"os"
	"testing"

	"github.com/merlindorin/ember/collide/tiles"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A layer recoloured and cut by a mask layer blocks bodies and casts
// shadows as its tiles uncut do: collide/tiles and light read the tile, the
// variant is only what is drawn (decision 0069).
func TestARecolouredAndMaskedLayerBlocksAndCastsAsItsTilesUncut(t *testing.T) {
	fsys := os.DirFS("../tiled/variants")
	masks, err := twod.LoadTiledTileset(fsys, "masks.tsj")
	require.NoError(t, err)
	dark := twod.NewPalette(twod.Swap(color.RGB8(0x68, 0x6f, 0x99), color.RGB8(0x2c, 0x35, 0x4d)))
	level, err := twod.LoadTiled(fsys, "variants.tmj", tiles.Properties(), TileProperties(),
		twod.Palettes(map[string]*twod.Palette{"dark": dark, "deep": dark}), twod.Masks(masks))
	require.NoError(t, err)
	floor, ok := level.TileLayer("floor")
	require.True(t, ok)

	grid := make([][]int, floor.Rows())
	for y := range grid {
		grid[y] = make([]int, floor.Columns())
		for x := range grid[y] {
			grid[y][x] = floor.At(x, y)
		}
	}
	uncut := twod.NewTileMap(floor.Tileset(), twod.Cells(grid), twod.CellSize(16, 16))
	area := geom.Rect{X: -16, Y: -16, W: 96, H: 64}

	solids := tiles.Of(floor).AppendSolidIn(nil, area)
	require.NotEmpty(t, solids, "the water is solid by its shape")
	assert.Equal(t, tiles.Of(uncut).AppendSolidIn(nil, area), solids)
	shadows := TileShadowsIn(nil, floor, area)
	require.NotEmpty(t, shadows)
	assert.Equal(t, TileShadowsIn(nil, uncut, area), shadows)
}
