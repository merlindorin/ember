package main

import (
	"strings"

	"github.com/merlindorin/ember/twod"
)

// newMeadow is grass under the whole window, the first tile of
// 26-tilemap's meadow, in cells of 32 points.
func newMeadow(tiles *twod.Texture) *twod.TileMap {
	grass := strings.Repeat(strings.Repeat(".", 20)+"\n", 12)
	return twod.NewTileMap(twod.NewTileset(tiles, twod.Cell(16, 16)),
		twod.ASCII(grass, map[rune]int{'.': 1}),
		twod.CellSize(32, 32),
	)
}
