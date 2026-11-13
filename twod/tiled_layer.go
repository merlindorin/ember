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
// holds, by Kind, and its settings, the layer's own values. A tile
// layer's TileMap is hidden when the layer isn't Visible, tinted by Tint
// with its alpha times Opacity and moved by Offset, its tall tiles too
// wherever TallIn stands them; hidden is only not drawn, so the map still
// collides and casts. An object layer's objects are moved by Offset.
// Parallax is read and not applied.
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
	objects []TiledObject
	layers  []*TiledLayer
	parent  *TiledLayer
}

// TileMap is a tile layer's map, else nil.
func (l *TiledLayer) TileMap() *TileMap { return l.tileMap }

// Objects are an object layer's objects, in the file's order.
func (l *TiledLayer) Objects() []TiledObject { return l.objects }

// Picture is an image layer's picture, else nil. Image layers are not read
// yet, so it is always nil.
func (l *TiledLayer) Picture() *Sprite { return nil }

// Layers are a group's layers, bottom first; nil for any other layer.
// Group layers are not read yet, so it is always nil.
func (l *TiledLayer) Layers() []*TiledLayer { return l.layers }

// Layer is the group's layer called name.
func (l *TiledLayer) Layer(name string) (*TiledLayer, bool) { return layerNamed(l.layers, name) }

// Parent is the group the layer is in, nil at the top of the map.
func (l *TiledLayer) Parent() *TiledLayer { return l.parent }

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
// map's. It is hidden when the object is. opts come after, so they can
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
	return NewSprite(tex, append(all, opts...)...)
}

// mapOptions are what the layer's settings make of its TileMap: not drawn
// when hidden, tinted by its tint and opacity, moved by its offset of map
// pixels, scale points each.
func (l *TiledLayer) mapOptions(scale float32) []TileMapOption {
	var out []TileMapOption
	if !l.Visible {
		out = append(out, Hidden())
	}
	if tint := l.Tint; tint != color.White || l.Opacity != 1 {
		tint.A *= l.Opacity
		out = append(out, Tint(tint))
	}
	if l.Offset != (geom.Vec2{}) {
		out = append(out, At(l.Offset.X*scale, l.Offset.Y*scale))
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
