// Package tiles is what a tile map is to collision and paths (design
// tiled-maps.md §6): it declares the tile property solid and the shape
// class body, and reads them, so twod draws a map without knowing what
// stops a body and collide and nav stay free of twod.
//
//	level, err := twod.LoadTiled(fsys, name, tiles.Properties())
//	walls, _ := level.TileLayer("walls")
//	solid := tiles.Of(walls) // a collide.Solid and a nav grid
//	space.AddSolid(solid)
//	paths := nav.New(solid)
//
// A tile is solid when its solid property is true or it has shapes of
// class "" or body; it stops bodies with those shapes, placed on its cell,
// else with its whole cell. A hidden map still blocks.
package tiles

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
)

// PropSolid is the tile property that makes a tile stop bodies with its
// whole cell, and block paths.
const PropSolid = "solid"

// Body is the class of a tile shape that stops bodies and casts no shadow;
// a shape of no class does both.
const Body = "body"

// Properties declares collide/tiles' tile properties for the Tiled loader:
// solid, a bool, false by default.
func Properties() twod.TiledOption {
	return twod.TileProperties(twod.Property{
		Name:    PropSolid,
		Kind:    twod.KindBool,
		Default: false,
		Doc:     "blocks bodies, as its collision shapes or its whole cell; true when it has shapes",
	})
}

// Solid makes tiles ids solid, stopping bodies with their whole cell, for
// a tileset made in code.
func Solid(ids ...int) twod.TilesetOption {
	return func(t *twod.Tileset) {
		for _, id := range ids {
			twod.TileValue(id, PropSolid, true)(t)
		}
	}
}

// Collision gives tile id shapes that stop bodies and cast shadows,
// rectangles in its pixels from its top-left corner, for a tileset made in
// code; a tile with shapes is solid.
func Collision(id int, shapes ...geom.Rect) twod.TilesetOption {
	return twod.TileShapes(id, "", shapes...)
}

// BodyShapes gives tile id shapes that stop bodies only, for a tileset made
// in code.
func BodyShapes(id int, shapes ...geom.Rect) twod.TilesetOption {
	return twod.TileShapes(id, Body, shapes...)
}

// Stops reports whether tile id of ts stops bodies: its solid property is
// true, or it has shapes of class "" or Body. What a Layer reads, for code
// that asks of a tileset without a map.
func Stops(ts *twod.Tileset, id int) bool { return id != 0 && read(ts, id).solid }

// Rects are the shapes tile id of ts stops bodies with, those of class ""
// then Body, in its pixels from its top-left corner; none for a tile that
// stops bodies with its whole cell, or not at all.
func Rects(ts *twod.Tileset, id int) []geom.Rect { return read(ts, id).rects }
