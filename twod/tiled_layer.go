package twod

import (
	"fmt"
	"strings"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
)

// TiledLayerKind is what a Tiled layer holds.
type TiledLayerKind int

const (
	// TiledTiles is a tile layer: a TileMap.
	TiledTiles TiledLayerKind = iota
	// TiledObjects is an object layer: objects for the game to read.
	TiledObjects
	// TiledImage is an image layer: one picture.
	TiledImage
	// TiledGroup is a group: layers of its own.
	TiledGroup
)

// TiledDrawOrder is how Tiled orders an object layer's objects.
type TiledDrawOrder int

const (
	// TiledTopDown sorts them by y, as Tiled draws them by default.
	TiledTopDown TiledDrawOrder = iota
	// TiledIndex keeps the file's order.
	TiledIndex
)

// TiledLayer is a layer of a Tiled map as the file writes it: what it
// holds, by Kind, and its settings, the layer's own values; a group's
// reach what it holds, as Shown, FoldedOpacity, FoldedTint and
// FoldedOffset fold them. A tile layer's TileMap is hidden unless Shown,
// tinted by FoldedTint with its alpha times FoldedOpacity and moved by
// FoldedOffset, its tall tiles too wherever TallIn stands them; hidden is
// only not drawn, so the map still collides and casts. An image layer's
// Picture is drawn the same way, and an object layer's objects are moved
// by FoldedOffset. Parallax is read and not applied.
type TiledLayer struct {
	// ID is Tiled's id of the layer, unique in its map.
	ID int
	// Name is the layer's name, unique among its siblings.
	Name       string
	Class      string
	Kind       TiledLayerKind
	Properties TiledProperties
	Visible    bool
	// Opacity is 0 to 1.
	Opacity float32
	// Offset is in the map's pixels.
	Offset geom.Vec2
	// Tint is white unless the layer sets one.
	Tint     color.Color
	Parallax geom.Vec2
	// DrawOrder is an object layer's.
	DrawOrder TiledDrawOrder

	tileMap *TileMap
	picture *Sprite
	objects []TiledObject
	layers  []*TiledLayer
	parent  *TiledLayer
}

// TileMap is a tile layer's map, else nil.
func (l *TiledLayer) TileMap() *TileMap { return l.tileMap }

// Objects are an object layer's objects, in the file's order.
func (l *TiledLayer) Objects() []TiledObject { return l.objects }

// Picture is an image layer's picture, its top-left corner at the layer's
// folded offset in the map's points, else nil. The game adds it where it
// draws the map.
func (l *TiledLayer) Picture() *Sprite { return l.picture }

// Layers are a group's layers, bottom first; nil for any other layer.
func (l *TiledLayer) Layers() []*TiledLayer { return l.layers }

// Layer is the group's layer called name.
func (l *TiledLayer) Layer(name string) (*TiledLayer, bool) { return layerNamed(l.layers, name) }

// Parent is the group the layer is in, nil at the top of the map.
func (l *TiledLayer) Parent() *TiledLayer { return l.parent }

// Shown is whether Tiled shows the layer: it and every group above it
// visible.
func (l *TiledLayer) Shown() bool {
	for g := l; g != nil; g = g.parent {
		if !g.Visible {
			return false
		}
	}
	return true
}

// FoldedOpacity is the layer's opacity times every group's above it.
func (l *TiledLayer) FoldedOpacity() float32 {
	o := float32(1)
	for g := l; g != nil; g = g.parent {
		o *= g.Opacity
	}
	return o
}

// FoldedTint is the layer's tint times every group's above it.
func (l *TiledLayer) FoldedTint() color.Color {
	c := color.White
	for g := l; g != nil; g = g.parent {
		c = color.Color{R: c.R * g.Tint.R, G: c.G * g.Tint.G, B: c.B * g.Tint.B, A: c.A * g.Tint.A}
	}
	return c
}

// FoldedOffset is the layer's offset plus every group's above it, in map
// pixels.
func (l *TiledLayer) FoldedOffset() geom.Vec2 {
	var o geom.Vec2
	for g := l; g != nil; g = g.parent {
		o = o.Add(g.Offset)
	}
	return o
}

func layerNamed(layers []*TiledLayer, name string) (*TiledLayer, bool) {
	for _, l := range layers {
		if l.Name == name {
			return l, true
		}
	}
	return nil, false
}

// walkLayers visits layers and what each group holds, a group before its
// own layers, bottom first, until visit returns false; it reports whether
// it went through them all.
func walkLayers(layers []*TiledLayer, visit func(*TiledLayer) bool) bool {
	for _, l := range layers {
		if !visit(l) || !walkLayers(l.layers, visit) {
			return false
		}
	}
	return true
}

// tiledColor reads one of Tiled's colours, #RRGGBB or #AARRGGBB.
func tiledColor(s string) (color.Color, error) {
	h := strings.TrimPrefix(s, "#")
	if len(h) == 8 {
		h = h[2:] + h[:2]
	}
	c, err := color.ParseHex(h)
	if err != nil {
		return color.Color{}, fmt.Errorf("colour %q: %w", s, err)
	}
	return c, nil
}

// Sprite is tile object o's picture where it stands, in the map's points:
// its tile's picture at the object's width and height, which a tile
// object resized in Tiled changes, moved by its tileset's drawing offset,
// turned by its rotation and flipped as Tiled shows it, about the object's
// anchor. The anchor is the tileset's
// object alignment: the bottom-left corner of an orthogonal map's object
// unless the tileset says otherwise, the bottom middle of an isometric
// map's. It is hidden when the object is, see-through as its opacity.
// opts come after, so they can
// change any of it. o must be a tile object of this map.
func (t *Tiled) Sprite(o TiledObject, opts ...SpriteOption) *Sprite {
	tile := o.Tile
	if tile == nil || tile.Tileset == nil {
		panic(fmt.Sprintf("twod: object %d (%s) is not a tile object", o.ID, o.Name))
	}
	tex, region := tile.Tileset.image(tile.ID, 0)
	px, py := tile.Tileset.anchor(t.isometric)
	all := []SpriteOption{
		Region(region),
		Size(o.Rect.W*t.scale, o.Rect.H*t.scale),
		Pivot(px, py),
		At((o.Rect.X+tile.Tileset.offset.X)*t.scale, (o.Rect.Y+tile.Tileset.offset.Y)*t.scale),
		Rotation(o.Rotation),
		Flip(tile.Flip&TileFlipX != 0, tile.Flip&TileFlipY != 0),
	}
	if !o.Visible {
		all = append(all, Hidden())
	}
	if o.Opacity != 1 {
		all = append(all, Tint(color.RGBA(1, 1, 1, o.Opacity)))
	}
	return NewSprite(tex, append(all, opts...)...)
}

// mapOptions are what the layer's folded settings make of its TileMap: not
// drawn when hidden, tinted by its tint and opacity, its cells moved by its
// offset of map pixels, scale points each, wherever the map is placed.
func (l *TiledLayer) mapOptions(scale float32) []TileMapOption {
	var out []TileMapOption
	if !l.Shown() {
		out = append(out, Hidden())
	}
	if tint, opacity := l.FoldedTint(), l.FoldedOpacity(); tint != color.White || opacity != 1 {
		tint.A *= opacity
		out = append(out, Tint(tint))
	}
	if offset := l.FoldedOffset(); offset != (geom.Vec2{}) {
		out = append(out, CellOffset(offset.X*scale, offset.Y*scale))
	}
	return out
}

// shift moves o by offset, in map pixels: its box and its points.
func (o *TiledObject) shift(offset geom.Vec2) {
	if offset == (geom.Vec2{}) {
		return
	}
	o.Rect.X, o.Rect.Y = o.Rect.X+offset.X, o.Rect.Y+offset.Y
	for i := range o.Points {
		o.Points[i] = o.Points[i].Add(offset)
	}
}
