package light

import (
	"math"
	"slices"

	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
)

// bakeTile is a static light tile's side, in the world's points.
const bakeTile = 256

// bake is a camera's static lights, in its world, and which version of
// each tile the GPU must have drawn: a tile's version changes when a light
// reaching it comes, goes or changes, or a shape near it does.
type bake struct {
	lights map[*Light]draw.Light
	spaces []*collide.Space
	// maps are the tile maps blocking its lights, and cacheOf their
	// shadows, placed.
	maps     []tileSource
	cacheOf  func(tileSource) *mapCache
	versions map[any]uint64
	mapKeys  map[tileSource]mapKey
	// mapped is whether the bake has seen its maps once: a map coming or
	// going after that changes every tile.
	mapped bool
	tiles  map[[2]int]uint64
	base   uint64
	next   uint64
}

func newBake() *bake {
	return &bake{
		lights:   map[*Light]draw.Light{},
		versions: map[any]uint64{},
		mapKeys:  map[tileSource]mapKey{},
		tiles:    map[[2]int]uint64{},
		base:     1,
		next:     2,
	}
}

// worldLight is dl, in its window's points, in the world view maps to
// those points.
func worldLight(dl draw.Light, view geom.Transform2D) draw.Light {
	inv := view.Inverse()
	k := 1 / max(view.X.Len(), 1e-6)
	dir := inv.X.Scale(dl.Dir.X).Add(inv.Y.Scale(dl.Dir.Y))
	if n := dir.Len(); n > 0 {
		dir = dir.Scale(1 / n)
	}
	dl.At, dl.Dir = inv.Xform(dl.At), dir
	dl.Radius, dl.Elevation, dl.ShadowSize = dl.Radius*k, dl.Elevation*k, dl.ShadowSize*k
	return dl
}

func reachOf(l draw.Light) geom.Rect {
	return geom.Rect{X: l.At.X - l.Radius, Y: l.At.Y - l.Radius, W: 2 * l.Radius, H: 2 * l.Radius}
}

// same reports whether a and b, point or cone lights, light alike.
func same(a, b draw.Light) bool {
	return a.Kind == b.Kind && a.At == b.At && a.Radius == b.Radius && a.Falloff == b.Falloff && a.Color == b.Color &&
		a.XRay == b.XRay && a.Groups == b.Groups && a.Dir == b.Dir && a.Spread == b.Spread && a.Elevation == b.Elevation &&
		a.ShadowSize == b.ShadowSize && a.ShadowFade == b.ShadowFade && a.FadeFrom == b.FadeFrom
}

// dirty gives the tiles r covers a new version.
func (b *bake) dirty(r geom.Rect) {
	x0, y0, x1, y1 := tileSpan(r)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			b.tiles[[2]int{x, y}] = b.next
		}
	}
	b.next++
}

func (b *bake) version(k [2]int) uint64 {
	if v, ok := b.tiles[k]; ok {
		return v
	}
	return b.base
}

// tileSpan is the tiles r covers, first and last on each axis.
func tileSpan(r geom.Rect) (int, int, int, int) {
	f := func(v float32) int { return int(math.Floor(float64(v / bakeTile))) }
	return f(r.X), f(r.Y), f(r.X + r.W), f(r.Y + r.H)
}

// update takes this frame's static lights, in the world, spaces and tile
// maps, giving a new version to the tiles of every light that came, went
// or changed, and to all of them when a shape source changed.
func (b *bake) update(
	lights map[*Light]draw.Light, spaces []*collide.Space, maps []tileSource, cacheOf func(tileSource) *mapCache,
) {
	b.spaces, b.maps, b.cacheOf = spaces, maps, cacheOf
	for l, s := range lights {
		old, ok := b.lights[l]
		switch {
		case !ok:
			b.dirty(reachOf(s))
		case !same(old, s):
			b.dirty(reachOf(old))
			b.dirty(reachOf(s))
		}
		b.lights[l] = s
	}
	for l, old := range b.lights {
		if _, ok := lights[l]; !ok {
			b.dirty(reachOf(old))
			delete(b.lights, l)
		}
	}
	changed := false
	for _, m := range maps {
		k := keyOf(m)
		if old, known := b.mapKeys[m]; !known || old != k {
			changed = changed || known || b.mapped
			b.mapKeys[m] = k
		}
	}
	if len(b.mapKeys) > len(maps) {
		for m := range b.mapKeys {
			if !slices.Contains(maps, m) {
				delete(b.mapKeys, m)
				changed = true
			}
		}
	}
	b.mapped = true
	for _, sp := range spaces {
		for _, src := range sp.SolidSources() {
			v, ok := src.(interface{ Version() uint64 })
			if !ok || fromMap(src) {
				continue
			}
			if old, known := b.versions[src]; !known || old != v.Version() {
				changed = changed || known
				b.versions[src] = v.Version()
			}
		}
	}
	if changed {
		b.base, b.next = b.next, b.next+1
		clear(b.tiles)
	}
}

// static is the draw data of the tiles in area, a rectangle of the world,
// that a static light reaches, view mapping the world to the window.
func (b *bake) static(key any, area geom.Rect, view geom.Transform2D) *draw.StaticLights {
	st := &draw.StaticLights{Key: key, TileSize: bakeTile, ToWindow: view, Count: len(b.lights)}
	x0, y0, x1, y1 := tileSpan(area)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			tile := geom.Rect{X: float32(x) * bakeTile, Y: float32(y) * bakeTile, W: bakeTile, H: bakeTile}
			reached := false
			for _, l := range b.lights {
				if overlaps(reachOf(l), tile) {
					reached = true
					break
				}
			}
			if !reached {
				continue
			}
			k := [2]int{x, y}
			st.Tiles = append(
				st.Tiles,
				draw.StaticTile{
					X:       x,
					Y:       y,
					Version: b.version(k),
					Bake:    func() draw.TileBake { return b.tileBake(tile) },
				},
			)
		}
	}
	return st
}

// tileBake is what to draw tile, a rectangle of the world, with: the
// static lights reaching it and the shapes round it, in a frame grown by
// their reach.
func (b *bake) tileBake(tile geom.Rect) draw.TileBake {
	var lights []draw.Light
	reach := float32(0)
	for _, l := range b.lights {
		if overlaps(reachOf(l), tile) {
			lights = append(lights, l)
			reach = max(reach, l.Radius)
		}
	}
	reach = float32(math.Ceil(float64(reach)))
	origin := geom.V2(tile.X-reach, tile.Y-reach)
	size := geom.V2(tile.W+2*reach, tile.H+2*reach)
	li := &draw.Lighting{Ambient: color.Black, Occluders: &draw.List{}}
	li.Occluders.SetBlend(draw.Occlude)
	for _, l := range lights {
		l.At = l.At.Sub(origin)
		li.Lights = append(li.Lights, l)
	}
	frame := geom.Rect{
		X: origin.X - draw.LightMargin,
		Y: origin.Y - draw.LightMargin,
		W: size.X + 2*draw.LightMargin,
		H: size.Y + 2*draw.LightMargin,
	}
	toFrame := geom.Transform2D{X: geom.V2(1, 0), Y: geom.V2(0, 1), Origin: origin.Scale(-1)}
	r := draw.LightRect(size)
	for _, m := range b.maps {
		for _, sh := range b.cacheOf(m).shadowsIn(nil, m, frame) {
			quad(li.Occluders, nil, toFrame, sh.rect, r, 1<<sh.group, sh.height)
		}
	}
	for _, sp := range b.spaces {
		for _, src := range sp.SolidSources() {
			if fromMap(src) {
				continue
			}
			for _, solid := range src.SolidIn(frame) {
				quad(li.Occluders, nil, toFrame, solid, r, 1, 0)
			}
		}
	}
	return draw.TileBake{Size: size, Inner: geom.Rect{X: reach, Y: reach, W: tile.W, H: tile.H}, Lighting: li}
}

func overlaps(a, b geom.Rect) bool {
	return a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y
}
