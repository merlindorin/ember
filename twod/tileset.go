package twod

import (
	"encoding/json"
	"fmt"
	"image"
	"maps"
	"slices"
	"sync"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/node"
)

//nolint:gochecknoinits // registers the package's resource types on import
func init() {
	node.RegisterResource("twod.Tileset", func() node.Resource { return &Tileset{} })
}

// Tileset is a texture atlas cut into tiles, with what each tile is: solid
// for collision, tall to sort with characters, animated, or carrying
// properties. Tiles are numbered from 1, left to right then top to bottom;
// 0 is an empty cell. Maps share it. A tileset can also hold tiles of
// their own, each a part of any texture at any size, as a Tiled
// collection does or as MergeTilesets puts several tilesets together.
type Tileset struct {
	texture        *Texture
	tileW, tileH   int
	spaceX, spaceY int
	margin         int
	solid, tall    map[int]bool
	// shapes are the tiles' collision shapes, in their pixels; roomShapes
	// the same placed in a cell, by cell size, for Room.
	shapes map[int][]geom.Rect
	// shadows are tiles' shapes that block light only; noShadow are tiles
	// that block no light; occluderGroup are tiles' occluder groups.
	shadows map[int][]geom.Rect
	// bodies are tiles' shapes that stop bodies only, casting nothing.
	bodies        map[int][]geom.Rect
	noShadow      map[int]bool
	occluderGroup map[int]int
	// faces caches, by tile, cell size and flip, the box a tile that
	// blocks light takes its light round.
	faces map[faceKey]geom.Rect
	// faceList holds the faces of tiles unflipped on cells of faceCell,
	// by id: what a map's every cell asks each frame.
	faceCell geom.Vec2
	faceList []faceEntry
	// height and emissive are the tiles' height and emissive properties,
	// in their pixels; load the properties declared when they were read.
	height, emissive map[int]float32
	load             *tiledLoad
	roomShapes       map[roomKey]shaped
	spills           map[geom.Vec2]spill
	roomMu           sync.Mutex
	props            map[int]TiledProperties
	anims            map[int]tileAnimation
	// tiles are the tiles with a picture of their own rather than a cell of
	// the grid.
	tiles map[int]tileImage
	// file is the atlas's image file and files the tiles' own, as read
	// from Tiled, kept whether or not the pictures were loaded; source is
	// the tileset's own .tsj file, when a map names one.
	file   string
	files  map[int]string
	source string
	// bounds caches each tile's opaque bounds, found the first time asked;
	// names each tile's name property, made the first time asked.
	bounds map[int]geom.Rect
	names  map[string]int
	// largest is the size of its largest tile, in pixels, found the first
	// time asked.
	largest *geom.Vec2
}

// tileImage is a tile's own picture: rect, in pixels, of texture.
type tileImage struct {
	texture *Texture
	rect    geom.Rect
}

type tileAnimation struct {
	fps    float32
	frames []int
}

// TilesetOption sets up a Tileset.
type TilesetOption func(*Tileset)

// NewTileset cuts tex into tiles of cell.
func NewTileset(tex *Texture, cell Grid, opts ...TilesetOption) *Tileset {
	t := &Tileset{
		texture:       tex,
		tileW:         max(cell.w, 1),
		tileH:         max(cell.h, 1),
		solid:         map[int]bool{},
		tall:          map[int]bool{},
		props:         map[int]TiledProperties{},
		anims:         map[int]tileAnimation{},
		tiles:         map[int]tileImage{},
		shapes:        map[int][]geom.Rect{},
		shadows:       map[int][]geom.Rect{},
		bodies:        map[int][]geom.Rect{},
		noShadow:      map[int]bool{},
		occluderGroup: map[int]int{},
		height:        map[int]float32{},
		emissive:      map[int]float32{},
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Solid marks tiles that stop moving bodies.
func Solid(ids ...int) TilesetOption {
	return func(t *Tileset) {
		for _, id := range ids {
			t.solid[id] = true
		}
	}
}

// Collision gives tile id its collision shapes, rectangles in its pixels
// from its top-left corner, such as a thin wall's bar: it stops moving
// bodies there only, and still blocks its whole cell for paths.
func Collision(id int, shapes ...geom.Rect) TilesetOption {
	return func(t *Tileset) { t.shapes[id] = append(t.shapes[id], shapes...) }
}

// ShadowShapes gives tile id shapes that block light only, rectangles in
// its pixels as Collision's: a tree's trunk and canopy shadow it, though
// only its trunk stops bodies. A tile with only these blocks light, not
// movement.
func ShadowShapes(id int, shapes ...geom.Rect) TilesetOption {
	return func(t *Tileset) { t.shadows[id] = append(t.shadows[id], shapes...) }
}

// BodyShapes gives tile id shapes that stop bodies only, rectangles in its
// pixels as Collision's, casting no shadow: a bar's reach past its art, a
// low rail. Its Collision shapes, or its whole cell when Solid, still cast;
// a tile with only these casts nothing.
func BodyShapes(id int, shapes ...geom.Rect) TilesetOption {
	return func(t *Tileset) { t.bodies[id] = append(t.bodies[id], shapes...) }
}

// CastShadow says whether tiles block light, true by default for those
// with collision or shadow shapes; false keeps them stopping bodies only:
// a fence, a low wall.
func CastShadow(on bool, ids ...int) TilesetOption {
	return func(t *Tileset) {
		for _, id := range ids {
			if on {
				delete(t.noShadow, id)
			} else {
				t.noShadow[id] = true
			}
		}
	}
}

// OccluderGroup puts tiles' shadows in occluder group n (0, 1 or 2), for
// lights BlockedBy only some groups; 0 by default.
func OccluderGroup(n int, ids ...int) TilesetOption {
	return func(t *Tileset) {
		for _, id := range ids {
			t.occluderGroup[id] = min(max(n, 0), 2)
		}
	}
}

// Height sets how tall tiles stand, in their pixels, for the length of
// their shadows from a light with an Elevation.
func Height(points float32, ids ...int) TilesetOption {
	return func(t *Tileset) {
		for _, id := range ids {
			t.height[id] = max(points, 0)
		}
	}
}

// Emissive sets how far tiles glow, in their pixels.
func Emissive(reach float32, ids ...int) TilesetOption {
	return func(t *Tileset) {
		for _, id := range ids {
			t.emissive[id] = max(reach, 0)
		}
	}
}

// Tall marks tiles that rise above the ground, such as trees: a map draws
// them as sprites standing on their cell, to sort with characters.
func Tall(ids ...int) TilesetOption {
	return func(t *Tileset) {
		for _, id := range ids {
			t.tall[id] = true
		}
	}
}

// Spacing is the gap between tiles in the atlas, in pixels.
func Spacing(x, y int) TilesetOption { return func(t *Tileset) { t.spaceX, t.spaceY = x, y } }

// Margin is the border around the atlas's tiles, in pixels.
func Margin(px int) TilesetOption { return func(t *Tileset) { t.margin = px } }

// TileAnimation shows tile id as frames, tile ids too, played at fps:
// water, torches. Every cell of id plays in step.
func TileAnimation(id int, fps float32, frames ...int) TilesetOption {
	return func(t *Tileset) { t.anims[id] = tileAnimation{fps: fps, frames: frames} }
}

func (t *Tileset) Texture() *Texture { return t.texture }

// Bounds is the part of tile id that isn't transparent, in its pixels from
// its top-left corner, the first frame of an animated one: what a crate or
// a column really covers of its cell. A tile whose texture has no pixels to
// read is its whole picture; an empty one is a zero rectangle.
func (t *Tileset) Bounds(id int) geom.Rect {
	if b, ok := t.bounds[id]; ok {
		return b
	}
	tex, r := t.image(id, 0)
	w, h := int(r.W), int(r.H)
	b := geom.Rect{W: r.W, H: r.H}
	if tex != nil && tex.Image != nil {
		img := tex.Image
		x0, y0 := int(r.X), int(r.Y)
		lo, hi := image.Pt(w, h), image.Pt(-1, -1)
		for y := range h {
			for x := range w {
				if _, _, _, a := img.At(x0+x, y0+y).RGBA(); a > 0 {
					lo, hi = image.Pt(min(lo.X, x), min(lo.Y, y)), image.Pt(max(hi.X, x), max(hi.Y, y))
				}
			}
		}
		b = geom.Rect{}
		if hi.X >= 0 {
			b = geom.Rect{X: float32(lo.X), Y: float32(lo.Y), W: float32(hi.X - lo.X + 1), H: float32(hi.Y - lo.Y + 1)}
		}
	}
	if t.bounds == nil {
		t.bounds = map[int]geom.Rect{}
	}
	t.bounds[id] = b
	return b
}

// picture is what tile id shows on the cell at origin, of size cell, in
// the map's space, trimmed to its Bounds: the tile at its own size, at
// the map's scale, standing on the cell's bottom-left as Tiled draws it.
// ok is false when nothing shows.
func (t *Tileset) picture(id int, origin, cell geom.Vec2) (geom.Rect, bool) {
	b := t.Bounds(id)
	if b.W <= 0 || b.H <= 0 {
		return geom.Rect{}, false
	}
	at, k := t.standing(id, origin, cell), t.scale(cell)
	return geom.Rect{X: at.X + b.X*k, Y: at.Y + b.Y*k, W: b.W * k, H: b.H * k}, true
}

// scale is how many map points one tileset pixel takes in a cell of size
// cell: the grid's tile fits the cell's width, and every tile is drawn at
// that one scale, never resized or cut.
func (t *Tileset) scale(cell geom.Vec2) float32 { return cell.X / float32(t.tileW) }

// standing is the top-left corner of tile id standing on the cell at
// origin, of size cell: its bottom-left on the cell's, so a tile taller
// or wider than the grid rises above it or spreads to its right.
func (t *Tileset) standing(id int, origin, cell geom.Vec2) geom.Vec2 {
	_, r := t.image(id, 0)
	return geom.V2(origin.X, origin.Y+cell.Y-r.H*t.scale(cell))
}

// overhang widens area, in a map's space with cells of size cell, by
// how far its tiles spread past their cells: to the left and down, as a
// tile on a cell outside it can rise or spread into it.
func (t *Tileset) overhang(area geom.Rect, cell geom.Vec2) geom.Rect {
	if t.largest == nil {
		l := geom.V2(float32(t.tileW), float32(t.tileH))
		for _, own := range t.tiles {
			l = geom.V2(max(l.X, own.rect.W), max(l.Y, own.rect.H))
		}
		t.largest = &l
	}
	k := t.scale(cell)
	right, up := max(t.largest.X*k-cell.X, 0), max(t.largest.Y*k-cell.Y, 0)
	sp := t.shapeSpill(cell)
	right = max(right, sp.right)
	return geom.Rect{
		X: area.X - right,
		Y: area.Y - sp.below,
		W: area.W + right + sp.left,
		H: area.H + max(up, sp.above) + sp.below,
	}
}

// spill is how far tiles' shapes reach past their cell on each side.
type spill struct{ left, above, right, below float32 }

// shapeSpill is the farthest any tile's shapes reach past its cell, by
// side, for a cell of size cell, found once.
func (t *Tileset) shapeSpill(cell geom.Vec2) spill {
	t.roomMu.Lock()
	defer t.roomMu.Unlock()
	if s, ok := t.spills[cell]; ok {
		return s
	}
	var s spill
	ids := map[int]bool{}
	for id := range t.shapes {
		ids[id] = true
	}
	for id := range t.bodies {
		ids[id] = true
	}
	for id := range ids {
		for _, r := range t.solids(nil, id, geom.Vec2{}, cell, 0) {
			s.left = max(s.left, -r.X)
			s.above = max(s.above, -r.Y)
			s.right = max(s.right, r.X+r.W-cell.X)
			s.below = max(s.below, r.Y+r.H-cell.Y)
		}
	}
	for id, shapes := range t.shadows {
		for _, r := range t.place(nil, shapes, id, geom.Vec2{}, cell, 0) {
			s.left = max(s.left, -r.X)
			s.above = max(s.above, -r.Y)
			s.right = max(s.right, r.X+r.W-cell.X)
			s.below = max(s.below, r.Y+r.H-cell.Y)
		}
	}
	if t.spills == nil {
		t.spills = map[geom.Vec2]spill{}
	}
	t.spills[cell] = s
	return s
}

// draw draws tile id on the cell at origin, of size cell, at time,
// flipped by f, tinted, standing as picture says.
func (t *Tileset) draw(c *Canvas, id int, origin, cell geom.Vec2, time float32, f TileFlip, tint color.Color) {
	tex, r := t.image(id, time)
	if tex == nil || tex.Size() == (geom.Vec2{}) {
		return
	}
	at, k := t.standing(id, origin, cell), t.scale(cell)
	dst, uv := geom.Rect{X: at.X, Y: at.Y, W: r.W * k, H: r.H * k}, tex.uv(r)
	if f != 0 {
		c.list.ImageCorners(dst, &tex.Texture, flipped(uv, f), tint)
		return
	}
	c.Image(dst, tex, uv, tint)
}

// sprite is tile id standing on the cell at origin, of size cell, as a
// sprite of its own, flipped by f: a Tall tile, to sort with characters.
// Its picture is drawn as standing says, from the cell's bottom-left; it
// is placed, and so sorted, at the foot of its art, the bottom of its
// Bounds, above the middle of the cell's bottom: a column whose art ends
// above its cell's bottom sorts behind whoever stands in front of its
// base.
func (t *Tileset) sprite(id int, origin, cell geom.Vec2, f TileFlip) *Sprite {
	tex, r := t.image(id, 0)
	k := t.scale(cell)
	pivot, foot := float32(0.5), r.H
	if r.W > 0 {
		pivot = cell.X / 2 / (r.W * k)
	}
	if b := t.Bounds(id); b.H > 0 && f&TileFlipY == 0 {
		foot = b.Y + b.H
	}
	up := float32(1)
	if r.H > 0 {
		up = foot / r.H
	}
	return NewSprite(
		tex,
		Region(r),
		Pivot(pivot, up),
		At(origin.X+cell.X/2, origin.Y+cell.Y-(r.H-foot)*k),
		Scale(k, k),
		Flip(f&TileFlipX != 0, f&TileFlipY != 0),
	)
}

// TilePixels is the grid's tile width and height in the atlas, in pixels:
// the size a map fits to its cells.
func (t *Tileset) TilePixels() (int, int) { return t.tileW, t.tileH }

// Picture is tile id's texture and its rectangle in it, in pixels, its
// first frame if it is animated: what a map draws in its cell.
func (t *Tileset) Picture(id int) (*Texture, geom.Rect) { return t.image(id, 0) }

// TileSize is tile id's width and height in pixels: the grid's, or its
// own picture's.
func (t *Tileset) TileSize(id int) (int, int) {
	_, r := t.image(id, 0)
	return int(r.W), int(r.H)
}

// Solid is whether tile id stops moving bodies, whole or by its
// Collision or body shapes.
func (t *Tileset) Solid(id int) bool {
	return t.solid[id] || len(t.shapes[id]) > 0 || len(t.bodies[id]) > 0
}

// Collision is tile id's collision shapes, in its pixels from its top-left
// corner; none for a tile solid whole or not at all.
func (t *Tileset) Collision(id int) []geom.Rect { return t.shapes[id] }

// ShadowShapes are tile id's shapes that block light only.
func (t *Tileset) ShadowShapes(id int) []geom.Rect { return t.shadows[id] }

// BodyShapes are tile id's shapes that stop bodies only.
func (t *Tileset) BodyShapes(id int) []geom.Rect { return t.bodies[id] }

// CastShadow reports whether tile id blocks light: by its collision
// shapes, its solid cell (when it has no body shapes) or its shadow
// shapes, unless CastShadow(false). Body shapes never do.
func (t *Tileset) CastShadow(id int) bool {
	cell := t.solid[id] && len(t.shapes[id]) == 0 && len(t.bodies[id]) == 0
	return !t.noShadow[id] && (len(t.shapes[id]) > 0 || cell || len(t.shadows[id]) > 0)
}

// OccluderGroup is tile id's occluder group.
func (t *Tileset) OccluderGroup(id int) int { return t.occluderGroup[id] }

// ShadowRect is a rectangle that blocks light, in a map's space, and its
// occluder group.
type ShadowRect struct {
	Rect  geom.Rect
	Group int
	// Height is how tall it stands, in the map's points, 0 for a height
	// not known: it then blocks all the way.
	Height float32
}

// ShadowCaster is a solid whose shadows aren't simply its solid
// rectangles: a tile map, whose tiles may block light only, or bodies only.
// The light package reads it.
type ShadowCaster interface {
	ShadowsIn(area geom.Rect) []ShadowRect
}

// shadowsOf are the shadows tile id casts standing on the cell at origin,
// of size cell, flipped by f: its collision shapes, or its whole cell when
// solid, and its shadow shapes, unless it casts none.
func (t *Tileset) shadowsOf(out []ShadowRect, id int, origin, cell geom.Vec2, f TileFlip) []ShadowRect {
	if t.noShadow[id] {
		return out
	}
	g, h := t.occluderGroup[id], t.Height(id)*t.scale(cell)
	for _, r := range t.casters(nil, id, origin, cell, f) {
		out = append(out, ShadowRect{r, g, h})
	}
	for _, r := range t.place(nil, t.shadows[id], id, origin, cell, f) {
		out = append(out, ShadowRect{r, g, h})
	}
	return out
}

// solids are the rects tile id stops bodies with, standing on the cell at
// origin, of size cell, flipped by f: its shapes and body shapes, or the
// whole cell.
func (t *Tileset) solids(out []geom.Rect, id int, origin, cell geom.Vec2, f TileFlip) []geom.Rect {
	if len(t.shapes[id]) == 0 && len(t.bodies[id]) == 0 {
		if !t.solid[id] {
			return out
		}
		return append(out, geom.Rect{X: origin.X, Y: origin.Y, W: cell.X, H: cell.Y})
	}
	out = t.place(out, t.shapes[id], id, origin, cell, f)
	return t.place(out, t.bodies[id], id, origin, cell, f)
}

// casters are the rects of tile id's collision that block light: its
// shapes, or its whole cell when solid with no shapes of any kind.
func (t *Tileset) casters(out []geom.Rect, id int, origin, cell geom.Vec2, f TileFlip) []geom.Rect {
	if len(t.shapes[id]) == 0 {
		if !t.solid[id] || len(t.bodies[id]) > 0 {
			return out
		}
		return append(out, geom.Rect{X: origin.X, Y: origin.Y, W: cell.X, H: cell.Y})
	}
	return t.place(out, t.shapes[id], id, origin, cell, f)
}

// place puts shapes, in tile id's pixels, on the cell at origin, of size
// cell, flipped by f.
func (t *Tileset) place(out []geom.Rect, shapes []geom.Rect, id int, origin, cell geom.Vec2, f TileFlip) []geom.Rect {
	if len(shapes) == 0 {
		return out
	}
	_, r := t.image(id, 0)
	at, k := t.standing(id, origin, cell), t.scale(cell)
	for _, s := range shapes {
		if f&TileFlipX != 0 {
			s.X = r.W - s.X - s.W
		}
		if f&TileFlipY != 0 {
			s.Y = r.H - s.Y - s.H
		}
		out = append(out, geom.Rect{X: at.X + s.X*k, Y: at.Y + s.Y*k, W: s.W * k, H: s.H * k})
	}
	return out
}
func (t *Tileset) Tall(id int) bool { return t.tall[id] }

// Property is a property of tile id, such as one set in Tiled, as text.
func (t *Tileset) Property(id int, name string) (string, bool) {
	v, ok := t.props[id][name]
	if !ok {
		return "", false
	}
	if s, isString := v.(string); isString {
		return s, true
	}
	return fmt.Sprint(v), true
}

// Properties are tile id's properties, typed as Tiled saves them.
func (t *Tileset) Properties(id int) TiledProperties { return t.props[id] }

// Source is the tileset's own file (.tsj), as a path in the file system a
// Tiled map was read from: empty for a tileset embedded in its map or made
// in code. A map's unused tileset of its own file is only that: unread, no
// tiles (UsedTilesets).
func (t *Tileset) Source() string { return t.source }

// ImageFile is the file of tile id's picture, its path in the file system
// a Tiled tileset was read from: the tile's own picture, or the tileset's
// atlas. It is there whether or not the picture was loaded (UsedTilesets);
// a tileset made in code has none.
func (t *Tileset) ImageFile(id int) (string, bool) {
	if f, ok := t.files[id]; ok {
		return f, true
	}
	return t.file, t.file != "" && id >= 1
}

// ID is the tile whose name property is name.
func (t *Tileset) ID(name string) (int, bool) {
	if t.names == nil {
		t.names = map[string]int{}
		for id, p := range t.props {
			if n, ok := p.String("name"); ok {
				if old, taken := t.names[n]; !taken || id < old {
					t.names[n] = id
				}
			}
		}
	}
	id, ok := t.names[name]
	return id, ok
}

func (t *Tileset) setProperty(id int, name string, value any) {
	if t.props[id] == nil {
		t.props[id] = TiledProperties{}
	}
	t.props[id][name] = value
	t.names = nil
}

func (t *Tileset) columns() int {
	w := int(t.texture.Size().X) - 2*t.margin + t.spaceX
	return max(w/(t.tileW+t.spaceX), 1)
}

// Count is how many tiles the tileset holds: the atlas's grid, or up to
// its last tile of its own.
func (t *Tileset) Count() int {
	n := 0
	if t.texture != nil {
		h := int(t.texture.Size().Y) - 2*t.margin + t.spaceY
		n = t.columns() * max(h/(t.tileH+t.spaceY), 0)
	}
	for id := range t.tiles {
		n = max(n, id)
	}
	return n
}

// image is tile id's texture and rectangle in it, in pixels, at time
// seconds: its own picture, or its cell of the atlas.
func (t *Tileset) image(id int, time float32) (*Texture, geom.Rect) {
	if a, ok := t.anims[id]; ok && len(a.frames) > 0 && a.fps > 0 {
		id = a.frames[int(time*a.fps)%len(a.frames)]
	}
	if own, ok := t.tiles[id]; ok {
		return own.texture, own.rect
	}
	return t.texture, t.region(id)
}

// region is tile id's cell of the atlas, in pixels.
func (t *Tileset) region(id int) geom.Rect {
	if t.texture == nil {
		return geom.Rect{W: float32(t.tileW), H: float32(t.tileH)}
	}
	i := id - 1
	col, row := i%t.columns(), i/t.columns()
	return geom.Rect{
		X: float32(t.margin + col*(t.tileW+t.spaceX)), Y: float32(t.margin + row*(t.tileH+t.spaceY)),
		W: float32(t.tileW), H: float32(t.tileH),
	}
}

// MergeTilesets puts sets together as one tileset, each set's tiles
// numbered on from the last of the set before, as Tiled numbers a map's
// tilesets, every tile with its picture, solid, tall, properties and
// animation. The grid is the first set's, so every tile is drawn at the
// scale that fits it to a map's cells.
func MergeTilesets(sets ...*Tileset) *Tileset {
	firsts := make([]int, len(sets))
	next := 1
	for i, s := range sets {
		firsts[i] = next
		next += s.Count()
	}
	unit := Cell(1, 1)
	if len(sets) > 0 {
		unit = Cell(sets[0].tileW, sets[0].tileH)
	}
	return mergeTilesets(unit, firsts, sets)
}

// mergeTilesets is MergeTilesets with each set's first id given, on a
// grid of unit.
func mergeTilesets(unit Grid, firsts []int, sets []*Tileset) *Tileset {
	m := NewTileset(nil, unit)
	for k, s := range sets {
		m.merge(s, firsts[k]-1)
	}
	return m
}

// merge adds src's tiles to t, each id shifted by off.
func (t *Tileset) merge(src *Tileset, off int) {
	for id := 1; id <= src.Count(); id++ {
		tex, r := src.texture, src.region(id)
		if own, ok := src.tiles[id]; ok {
			tex, r = own.texture, own.rect
		}
		if tex != nil {
			t.tiles[off+id] = tileImage{tex, r}
		}
		if f, ok := src.ImageFile(id); ok {
			if t.files == nil {
				t.files = map[int]string{}
			}
			t.files[off+id] = f
		}
	}
	t.solid = shifted(t.solid, src.solid, off, same)
	t.shapes = shifted(t.shapes, src.shapes, off, slices.Clone)
	t.tall = shifted(t.tall, src.tall, off, same)
	t.shadows = shifted(t.shadows, src.shadows, off, slices.Clone)
	t.bodies = shifted(t.bodies, src.bodies, off, slices.Clone)
	t.noShadow = shifted(t.noShadow, src.noShadow, off, same)
	t.occluderGroup = shifted(t.occluderGroup, src.occluderGroup, off, same)
	t.height = shifted(t.height, src.height, off, same)
	t.emissive = shifted(t.emissive, src.emissive, off, same)
	if src.load != nil {
		if t.load == nil {
			t.load = &tiledLoad{}
		}
		for _, d := range src.load.defs {
			if _, ok := t.load.def(d.Name); !ok {
				t.load.defs = append(t.load.defs, d)
			}
		}
	}
	t.props = shifted(t.props, src.props, off, maps.Clone)
	for id, a := range src.anims {
		frames := make([]int, len(a.frames))
		for i, f := range a.frames {
			frames[i] = off + f
		}
		t.anims[off+id] = tileAnimation{fps: a.fps, frames: frames}
	}
}

// shifted puts cp of each of src's values into dst under its id plus off,
// making dst when it is nil and src has any.
func shifted[V any](dst, src map[int]V, off int, cp func(V) V) map[int]V {
	for id, v := range src {
		if dst == nil {
			dst = map[int]V{}
		}
		dst[off+id] = cp(v)
	}
	return dst
}

func same[V any](v V) V { return v }

func (*Tileset) ResourceType() string { return "twod.Tileset" }

type tilesetJSON struct {
	Texture       *Texture                `json:"texture"`
	TileW         int                     `json:"tile_w"`
	TileH         int                     `json:"tile_h"`
	SpaceX        int                     `json:"space_x"`
	SpaceY        int                     `json:"space_y"`
	Margin        int                     `json:"margin"`
	Solid         []int                   `json:"solid"`
	Tall          []int                   `json:"tall"`
	Props         map[int]TiledProperties `json:"properties"`
	Tiles         map[int]tileImageJSON   `json:"tiles,omitempty"`
	Shapes        map[int][]geom.Rect     `json:"collision,omitempty"`
	Shadows       map[int][]geom.Rect     `json:"shadow_shapes,omitempty"`
	Bodies        map[int][]geom.Rect     `json:"body_shapes,omitempty"`
	NoShadow      []int                   `json:"no_shadow,omitempty"`
	OccluderGroup map[int]int             `json:"occluder_group,omitempty"`
	Anims         map[int]struct {
		FPS    float32 `json:"fps"`
		Frames []int   `json:"frames"`
	} `json:"animations"`
}

type tileImageJSON struct {
	Texture *Texture  `json:"texture"`
	Rect    geom.Rect `json:"rect"`
}

func keys(m map[int]bool) []int {
	var out []int
	for k, on := range m {
		if on {
			out = append(out, k)
		}
	}
	slices.Sort(out)
	return out
}

func (t *Tileset) MarshalJSON() ([]byte, error) {
	d := tilesetJSON{
		Texture: t.texture,
		TileW:   t.tileW,
		TileH:   t.tileH,
		SpaceX:  t.spaceX,
		SpaceY:  t.spaceY,
		Margin:  t.margin,
		Solid:   keys(t.solid),
		Tall:    keys(t.tall),
		Props:   t.props,
	}
	if len(t.shapes) > 0 {
		d.Shapes = t.shapes
	}
	if len(t.shadows) > 0 {
		d.Shadows = t.shadows
	}
	if len(t.bodies) > 0 {
		d.Bodies = t.bodies
	}
	if len(t.occluderGroup) > 0 {
		d.OccluderGroup = t.occluderGroup
	}
	d.NoShadow = keys(t.noShadow)
	if len(t.tiles) > 0 {
		d.Tiles = map[int]tileImageJSON{}
		for id, own := range t.tiles {
			d.Tiles[id] = tileImageJSON{own.texture, own.rect}
		}
	}
	if len(t.anims) > 0 {
		d.Anims = map[int]struct {
			FPS    float32 `json:"fps"`
			Frames []int   `json:"frames"`
		}{}
		for id, a := range t.anims {
			d.Anims[id] = struct {
				FPS    float32 `json:"fps"`
				Frames []int   `json:"frames"`
			}{a.fps, a.frames}
		}
	}
	return json.Marshal(d)
}

func (t *Tileset) UnmarshalJSON(b []byte) error {
	var d tilesetJSON
	if err := json.Unmarshal(b, &d); err != nil {
		return err
	}
	*t = *NewTileset(
		d.Texture,
		Cell(d.TileW, d.TileH),
		Spacing(d.SpaceX, d.SpaceY),
		Margin(d.Margin),
		Solid(d.Solid...),
		Tall(d.Tall...),
	)
	for id, props := range d.Props {
		for k, v := range props {
			t.setProperty(id, k, v)
		}
	}
	for id, a := range d.Anims {
		t.anims[id] = tileAnimation{fps: a.FPS, frames: a.Frames}
	}
	for id, own := range d.Tiles {
		t.tiles[id] = tileImage{own.Texture, own.Rect}
	}
	for id, shapes := range d.Shapes {
		t.shapes[id] = shapes
	}
	for id, shapes := range d.Shadows {
		t.shadows[id] = shapes
	}
	for id, shapes := range d.Bodies {
		t.bodies[id] = shapes
	}
	for _, id := range d.NoShadow {
		t.noShadow[id] = true
	}
	for id, g := range d.OccluderGroup {
		t.occluderGroup[id] = g
	}
	return nil
}
