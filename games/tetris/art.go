package main

import (
	"embed"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
)

//go:embed art/blocks.png
var art embed.FS

// The block the cells are drawn with: Kenney's marble tile at column 0, row
// 1 of an 18-pixel sheet 9 tiles square, tinted per piece.
var blockUV = geom.Rect{X: 0, Y: 1.0 / 9, W: 1.0 / 9, H: 1.0 / 9} //nolint:gochecknoglobals // a constant region

func loadBlocks() *twod.Texture {
	tex, err := twod.LoadTexture(art, "art/blocks.png", twod.Pixelated())
	if err != nil {
		panic(err) // an embedded file that always decodes
	}
	return tex
}
