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

// Tileset is a texture atlas cut into tiles, with what each tile is: tall
// to sort with characters, animated, or carrying properties and shapes the
// systems read (collide/tiles, twod/light). Tiles are numbered from 1, left to right then top to bottom;
// 0 is an empty cell. Maps share it. A tileset can also hold tiles of
// their own, each a part of any texture at any size, as a Tiled
// collection does or as MergeTilesets puts several tilesets together.
type Tileset struct {
	texture        *Texture
	tileW, tileH   int
	spaceX, spaceY int
	margin         int
	tall           map[int]bool
	// shapes are the tiles' shapes, in their pixels, with their class.
	shapes map[int][]TileShape
	// version changes with any tile's property or shape; cached are the
	// systems' tables of the tiles, made at a version.
	version uint64
	cached  map[any]cachedTable
	cacheMu sync.Mutex
	// faces caches, by tile, cell size and flip, the box a tile that
	// blocks light takes its light round.
	faces map[faceKey]geom.Rect
	// faceList holds the faces of tiles unflipped on cells of faceCell,
	// by id: what a map's every cell asks each frame.
	faceCell geom.Vec2
	faceList []faceEntry
	// faceVersion is the version faces were worked out at.
	faceVersion uint64
	// load are the properties declared when they were read.
	load  *tiledLoad
	props map[int]TiledProperties
	anims map[int]tileAnimation
	// classes are the tiles' classes, as Tiled names them.
	classes map[int]string
	// tiles are the tiles with a picture of their own rather than a cell of
	// the grid.
	tiles map[int]tileImage
	// file is the atlas's image file and files the tiles' own, as read
	// from Tiled, kept whether or not the pictures were loaded; source is
	// the tileset's own .tsj file, when a map names one.
	file   string
	files  map[int]string
	source string
	// objectAlign is Tiled's object alignment: where a tile object of the
	// tileset is anchored.
	objectAlign string
	// offset moves every tile's picture by that many pixels: Tiled's
	// drawing offset.
	offset geom.Vec2
	// bounds caches each tile's opaque bounds, found the first time asked;
	// names each tile's name property, made the first time asked.
	bounds map[int]geom.Rect
	names  map[string]int
	// largest is the size of its largest tile, in pixels, found the first
	// time asked.
	largest *geom.Vec2
	// specs are what the tiles' own tint, recolour and mask draw them
	// through, and looks the looks made of them, by id (decision 0069).
	specs map[int]lookSpec
	looks []*tileLook
}

// tileImage is a tile's own picture: rect, in pixels, of texture.
type tileImage struct {
	texture *Texture
	rect    geom.Rect
}

// tileAnimation plays frames at fps, or each for its own of durations, in
// seconds, when it has them: a Tiled tileset's.
type tileAnimation struct {
	fps       float32
	frames    []int
	durations []float32
}

// frame is the animation's frame at time seconds.
func (a tileAnimation) frame(time float32) int {
	if len(a.durations) == 0 {
		return a.frames[int(time*a.fps)%len(a.frames)]
	}
	var total float32
	for _, d := range a.durations {
		total += d
	}
	at := time - total*geom.Floor(time/total)
	for i, d := range a.durations {
		if at < d {
			return a.frames[i]
		}
		at -= d
	}
	return a.frames[len(a.frames)-1]
}

// playable is whether the animation has frames to show and time to show
// them in.
func (a tileAnimation) playable() bool {
	if len(a.frames) == 0 {
		return false
	}
	if a.durations == nil {
		return a.fps > 0
	}
	var total float32
	for _, d := range a.durations {
		total += d
	}
	return total > 0
}

// TilesetOption sets up a Tileset.
type TilesetOption func(*Tileset)

// NewTileset cuts tex into tiles of cell.
func NewTileset(tex *Texture, cell Grid, opts ...TilesetOption) *Tileset {
	t := &Tileset{
		texture: tex,
		tileW:   max(cell.w, 1),
		tileH:   max(cell.h, 1),
		tall:    map[int]bool{},
		props:   map[int]TiledProperties{},
		anims:   map[int]tileAnimation{},
		tiles:   map[int]tileImage{},
		shapes:  map[int][]TileShape{},
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
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
	return geom.Rect{X: area.X - right, Y: area.Y, W: area.W + right, H: area.H + up}
}

// draw draws tile id on the cell at origin, of size cell, at time,
// flipped by f, tinted, standing as picture says, through look, nil for
// its picture as it is.
func (t *Tileset) draw(
	c *Canvas, id int, origin, cell geom.Vec2, time float32, f TileFlip, tint color.Color, look *tileLook,
) {
	tex, r := t.drawn(id, time, look)
	if tex == nil || tex.Size() == (geom.Vec2{}) {
		return
	}
	if look != nil {
		tint = tint.Mul(look.tint)
	}
	at, k := t.standing(id, origin, cell), t.scale(cell)
	at = at.Add(t.offset.Scale(k))
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
// base. It shows the tile through look, nil for its picture as it is.
func (t *Tileset) sprite(id int, origin, cell geom.Vec2, f TileFlip, look *tileLook) *Sprite {
	tex, r := t.drawn(id, 0, look)
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
		At(origin.X+cell.X/2+t.offset.X*k, origin.Y+cell.Y-(r.H-foot)*k+t.offset.Y*k),
		Scale(k, k),
		Flip(f&TileFlipX != 0, f&TileFlipY != 0),
		Tint(tintOf(look)),
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

// Tall is whether tile id stands as a sprite of its own, sorted with
// characters.
func (t *Tileset) Tall(id int) bool { return t.tall[id] }

// Class is tile id's class, as Tiled names it (its type before Tiled 1.9),
// empty when it has none.
func (t *Tileset) Class(id int) string { return t.classes[id] }

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
	t.version++
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
	if a, ok := t.anims[id]; ok && a.playable() {
		id = a.frame(time)
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
	t.shapes = shifted(t.shapes, src.shapes, off, slices.Clone)
	t.tall = shifted(t.tall, src.tall, off, same)
	t.classes = shifted(t.classes, src.classes, off, same)
	if src.load != nil {
		if t.load == nil {
			t.load = &tiledLoad{}
		}
		for _, d := range src.load.defs {
			if _, ok := t.load.def(d.Name); !ok {
				t.load.defs = append(t.load.defs, d)
			}
		}
		for _, c := range src.load.classes {
			if _, ok := t.load.class(c.Name); !ok {
				t.load.classes = append(t.load.classes, c)
			}
		}
	}
	t.props = shifted(t.props, src.props, off, maps.Clone)
	t.specs = shifted(t.specs, src.specs, off, same)
	for id, l := range src.looks {
		if l != nil {
			t.putLook(off+id, l.shifted(off))
		}
	}
	for id, a := range src.anims {
		frames := make([]int, len(a.frames))
		for i, f := range a.frames {
			frames[i] = off + f
		}
		t.anims[off+id] = tileAnimation{fps: a.fps, frames: frames, durations: a.durations}
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
	Texture *Texture                `json:"texture"`
	TileW   int                     `json:"tile_w"`
	TileH   int                     `json:"tile_h"`
	SpaceX  int                     `json:"space_x"`
	SpaceY  int                     `json:"space_y"`
	Margin  int                     `json:"margin"`
	Tall    []int                   `json:"tall"`
	Props   map[int]TiledProperties `json:"properties"`
	Tiles   map[int]tileImageJSON   `json:"tiles,omitempty"`
	Shapes  map[int][]TileShape     `json:"shapes,omitempty"`
	Anims   map[int]animationJSON   `json:"animations"`
	Classes map[int]string          `json:"classes,omitempty"`
}

type animationJSON struct {
	FPS       float32   `json:"fps"`
	Frames    []int     `json:"frames"`
	Durations []float32 `json:"durations,omitempty"`
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
		Tall:    keys(t.tall),
		Props:   t.props,
	}
	if len(t.shapes) > 0 {
		d.Shapes = t.shapes
	}
	if len(t.tiles) > 0 {
		d.Tiles = map[int]tileImageJSON{}
		for id, own := range t.tiles {
			d.Tiles[id] = tileImageJSON{own.texture, own.rect}
		}
	}
	if len(t.anims) > 0 {
		d.Anims = map[int]animationJSON{}
		for id, a := range t.anims {
			d.Anims[id] = animationJSON{a.fps, a.frames, a.durations}
		}
	}
	if len(t.classes) > 0 {
		d.Classes = t.classes
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
		Tall(d.Tall...),
	)
	for id, props := range d.Props {
		for k, v := range props {
			t.setProperty(id, k, v)
		}
	}
	for id, a := range d.Anims {
		t.anims[id] = tileAnimation{fps: a.FPS, frames: a.Frames, durations: a.Durations}
	}
	if len(d.Classes) > 0 {
		t.classes = d.Classes
	}
	for id, own := range d.Tiles {
		t.tiles[id] = tileImage{own.Texture, own.Rect}
	}
	for id, shapes := range d.Shapes {
		t.shapes[id] = shapes
	}
	return nil
}

// alignLeft is Tiled's word for the left, in alignments and text.
const alignLeft = "left"

// anchor is where a tile object of the tileset is anchored, as a share of
// its picture: Tiled's object alignment, by default the bottom-left corner
// of an orthogonal map's object and the bottom middle of an isometric
// map's.
func (t *Tileset) anchor(isometric bool) (float32, float32) {
	switch t.objectAlign {
	case "topleft":
		return 0, 0
	case "top":
		return 0.5, 0
	case "topright":
		return 1, 0
	case alignLeft:
		return 0, 0.5
	case "center":
		return 0.5, 0.5
	case "right":
		return 1, 0.5
	case "bottom":
		return 0.5, 1
	case "bottomright":
		return 1, 1
	case "bottomleft":
		return 0, 1
	}
	if isometric {
		return 0.5, 1
	}
	return 0, 1
}
