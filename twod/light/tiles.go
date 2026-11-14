package light

import (
	"sync"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/lit"
	"github.com/lantervale/ember/twod"
)

// The tile properties and shape class light reads, and the collision
// property it reads too: a solid cell with no shapes blocks light whole.
const (
	propShadow = "shadow"
	propGroup  = "occluder_group"
	propHeight = "height"
	propSolid  = "solid"
	propTall   = "tall"
	shapeBody  = "body"
	shadowNone = "none"
	shadowCast = "cast"
)

// TileProperties declares the tile properties light reads, for the loader
// to check and Tiled to show: shadow, occluder_group and height. A shape
// of class "shadow" on a tile blocks light only.
func TileProperties() twod.TiledOption {
	return twod.TileProperties(
		twod.Property{
			Name: propShadow, Kind: twod.KindEnum, Default: shadowCast, Values: []string{shadowCast, shadowNone},
			Doc: "none: blocks bodies, not light",
		},
		twod.Property{Name: propGroup, Kind: twod.KindInt, Default: 0, Min: 0, Max: 2, Doc: "occluder group, for BlockedBy"},
		twod.Property{
			Name: propHeight, Kind: twod.KindFloat, Default: 0.0, Min: 0, Max: 4096,
			Doc: "how tall it stands, in its pixels, for shadow length; " +
				"0: its picture's height when tall, its cell's when solid",
		},
	)
}

// TileShadow says whether tiles block light, true by default for those
// with collision or shadow shapes; false keeps them stopping bodies only:
// a fence, a low wall. For a tileset made in code.
func TileShadow(on bool, ids ...int) twod.TilesetOption {
	v := shadowCast
	if !on {
		v = shadowNone
	}
	return eachTile(propShadow, v, ids)
}

// TileOccluderGroup puts tiles' shadows in occluder group n (0, 1 or 2),
// for lights BlockedBy only some groups. For a tileset made in code.
func TileOccluderGroup(n int, ids ...int) twod.TilesetOption {
	return eachTile(propGroup, min(max(n, 0), 2), ids)
}

// TileHeight sets how tall tiles stand, in their pixels, for the length of
// their shadows from a light with an Elevation. For a tileset made in code.
func TileHeight(pixels float32, ids ...int) twod.TilesetOption {
	return eachTile(propHeight, float64(max(pixels, 0)), ids)
}

// TileShadowShapes gives tile id shapes that block light only, rectangles
// in its pixels from its top-left corner: a tree's trunk and canopy shadow
// it, though only its trunk stops bodies. For a tileset made in code.
func TileShadowShapes(id int, shapes ...geom.Rect) twod.TilesetOption {
	return twod.TileShapes(id, propShadow, shapes...)
}

func eachTile(name string, v any, ids []int) twod.TilesetOption {
	return func(t *twod.Tileset) {
		for _, id := range ids {
			twod.TileValue(id, name, v)(t)
		}
	}
}

// caster is what a tile blocks light with: its whole cell, or rects in its
// pixels, in an occluder group, standing height pixels tall.
type caster struct {
	whole  bool
	rects  []geom.Rect
	group  int
	height float32
}

// table is light's reading of a tileset, by tile id, and how far its
// tiles block light past cells of each size asked.
type table struct {
	tiles    []caster
	mu       sync.Mutex
	overhang map[geom.Vec2]overhang
}

// overhangAt is how far the tiles block light past cells of size cell.
func (tb *table) overhangAt(ts *twod.Tileset, cell geom.Vec2) overhang {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	if o, ok := tb.overhang[cell]; ok {
		return o
	}
	if tb.overhang == nil {
		tb.overhang = map[geom.Vec2]overhang{}
	}
	o := overhangOf(ts, tb, cell)
	tb.overhang[cell] = o
	return o
}

// tableKey keys light's table among a tileset's cached ones.
type tableKey struct{}

// tableOf is light's reading of ts, made again when ts changed.
func tableOf(ts *twod.Tileset) *table {
	tb, _ := ts.Cached(tableKey{}, func() any {
		tb := &table{tiles: make([]caster, ts.MaxID()+1)}
		for id := range tb.tiles {
			tb.tiles[id] = casterOf(ts, id)
		}
		return tb
	}).(*table)
	return tb
}

// at is tile id's caster; a tile past the table, of which the tileset
// says nothing, blocks no light.
func (tb *table) at(id int) caster {
	if id >= 0 && id < len(tb.tiles) {
		return tb.tiles[id]
	}
	return caster{}
}

// casterOf is what tile id of ts blocks light with: its plain shapes, or
// its whole cell when solid with no shapes of any kind, and its shadow
// shapes, unless its shadow is none.
func casterOf(ts *twod.Tileset, id int) caster {
	if id == 0 {
		return caster{}
	}
	if v, ok := ts.Value(id, propShadow); ok && v.String() == shadowNone {
		return caster{}
	}
	var c caster
	bodies := false
	for _, s := range ts.Shapes(id) {
		switch s.Class {
		case "", propShadow:
			c.rects = append(c.rects, s.Rect)
		case shapeBody:
			bodies = true
		}
	}
	solid := boolOf(ts, id, propSolid)
	c.whole = solid && !bodies && len(shapesOf(ts, id, "")) == 0
	if !c.whole && len(c.rects) == 0 {
		return caster{}
	}
	if g, ok := ts.Value(id, propGroup); ok {
		c.group = min(max(g.Int(), 0), 2)
	}
	c.height = heightOf(ts, id, solid)
	return c
}

// heightOf is how tall tile id stands, in its pixels: its height property,
// else its picture's height when tall, its cell's when solid, else 0.
func heightOf(ts *twod.Tileset, id int, solid bool) float32 {
	if h, ok := ts.Value(id, propHeight); ok && h.Float() > 0 {
		return float32(h.Float())
	}
	switch {
	case boolOf(ts, id, propTall):
		_, r := ts.Picture(id)
		return r.H
	case solid:
		_, h := ts.TilePixels()
		return float32(h)
	}
	return 0
}

func boolOf(ts *twod.Tileset, id int, name string) bool {
	v, ok := ts.Value(id, name)
	return ok && v.Bool()
}

func shapesOf(ts *twod.Tileset, id int, class string) []geom.Rect {
	var out []geom.Rect
	for _, s := range ts.Shapes(id) {
		if s.Class == class {
			out = append(out, s.Rect)
		}
	}
	return out
}

// shadow is a rectangle blocking light, in a map's space, in an occluder
// group, standing height points tall, 0 for all the way.
type shadow struct {
	rect   geom.Rect
	group  int
	height float32
}

// place appends what tile id of ts blocks light with, standing on the cell
// at origin, of size cell, flipped by f.
func place(dst []shadow, ts *twod.Tileset, tb *table, id int, origin, cell geom.Vec2, f twod.TileFlip) []shadow {
	c := tb.at(id)
	if !c.whole && len(c.rects) == 0 {
		return dst
	}
	w, _ := ts.TilePixels()
	h := c.height * cell.X / float32(w)
	if c.whole {
		dst = append(dst, shadow{geom.Rect{X: origin.X, Y: origin.Y, W: cell.X, H: cell.Y}, c.group, h})
	}
	var scratch [8]geom.Rect
	for _, r := range ts.Place(scratch[:0], id, c.rects, origin, cell, f) {
		dst = append(dst, shadow{r, c.group, h})
	}
	return dst
}

//nolint:gochecknoinits // gives twod the shapes its lit tiles take their light round
func init() {
	lit.TileBlocks = func(tileset any, id int, origin, cell geom.Vec2, flip uint8, dst []geom.Rect) []geom.Rect {
		ts, ok := tileset.(*twod.Tileset)
		if !ok {
			return dst
		}
		var scratch [8]shadow
		for _, s := range place(scratch[:0], ts, tableOf(ts), id, origin, cell, twod.TileFlip(flip)) {
			dst = append(dst, s.rect)
		}
		return dst
	}
}
