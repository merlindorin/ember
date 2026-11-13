package twod

import (
	"bytes"
	"cmp"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"path"
	"slices"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
)

// Tiled is a map made in the Tiled editor: what the file says of itself,
// its layers in the file's order (tile layers as TileMaps, object layers as
// data for the game to spawn from) and its tilesets.
type Tiled struct {
	// dir is the map's folder in its file system.
	dir            string
	class          string
	columns, rows  int
	tileW, tileH   int
	scale          float32
	isometric      bool
	background     color.Color
	parallaxOrigin geom.Vec2
	layers         []*TiledLayer
	byID           map[int]TiledObject
	tilesets       []*Tileset
	properties     TiledProperties
	// merged is every tileset as one, numbered by global id, for layers
	// whose tiles come from several, made the first time one asks.
	merged *Tileset
	unit   Grid
	// used marks the tilesets a tile layer or a tile object takes a tile
	// from: the ones whose pictures were loaded.
	used []bool
}

// TiledObject is an object of an object layer: a spawn point, a trigger, a
// door, a picture placed off the grid. It is plain data; the game decides
// what it becomes.
type TiledObject struct {
	// ID is Tiled's id of the object, unique in its map.
	ID         int
	Name, Type string
	// Rect is the object's x, y, width and height as Tiled saves them, in
	// the map's pixels: a rectangle's or an ellipse's box from its top-left;
	// a tile object's picture with y at its bottom, where it stands; a
	// point's or a polygon's first point, sized 0. It is the box before
	// Rotation turns it.
	Rect geom.Rect
	// Rotation is how far Tiled turns the object, in radians, clockwise
	// about its x, y.
	Rotation float32
	// Visible is false for an object hidden in Tiled; it is listed all the
	// same.
	Visible bool
	Shape   TiledShape
	// Points are a polygon's or a polyline's points, in the map's pixels.
	Points []geom.Vec2
	// Tile is a tile object's tile, nil for any other object.
	Tile *TiledTile
	// Text is a text object's text, its Shape TiledText, and TextStyle how
	// Tiled sets it; nil for any other object.
	Text       string
	TextStyle  *TiledTextStyle
	Properties TiledProperties
}

// TiledTextStyle is how Tiled sets a text object's text, exposed for the
// game to draw it as it likes: the engine draws no text objects.
type TiledTextStyle struct {
	FontFamily string
	// PixelSize is the font's size in the map's pixels.
	PixelSize int
	Color     color.Color
	Wrap      bool
	Bold      bool
	Italic    bool
	Underline bool
	Strikeout bool
	Kerning   bool
	// HAlign is "left", "center", "right" or "justify"; VAlign "top",
	// "center" or "bottom".
	HAlign, VAlign string
}

// TiledShape is the shape of an object drawn in Tiled.
type TiledShape int

const (
	// TiledRectangle is a rectangle, and also a tile object.
	TiledRectangle TiledShape = iota
	TiledEllipse
	TiledPoint
	TiledPolygon
	TiledPolyline
	TiledText
)

// TiledTile is the tile a tile object shows.
type TiledTile struct {
	// GID is Tiled's global id of the tile, its flip bits included.
	GID uint32
	// Tileset is the map's tileset the tile is from, and ID the tile in it,
	// counted from 1 as a Tileset counts: Tiled's tile 5 is ID 6.
	Tileset *Tileset
	ID      int
	// Flip is how the object flips its picture.
	Flip TileFlip
}

// TiledProperties are the custom properties set in Tiled on a map, a layer
// or an object, each by its Tiled type: a string (also a color, or a file
// as its path from the file system's root), an int (also an object, by its
// id), a float64, a bool, or the TiledProperties of a custom class's
// members, as Tiled saves them: the members changed from their defaults,
// a number as a float64.
type TiledProperties map[string]any

// String is property name when it is text.
func (p TiledProperties) String(name string) (string, bool) {
	v, ok := p[name].(string)
	return v, ok
}

// Int is property name when it is a whole number.
func (p TiledProperties) Int(name string) (int, bool) {
	switch v := p[name].(type) {
	case int:
		return v, true
	case float64:
		if v == math.Trunc(v) {
			return int(v), true
		}
	}
	return 0, false
}

// Class is property name when it is a custom class: its members.
func (p TiledProperties) Class(name string) (TiledProperties, bool) {
	v, ok := p[name].(TiledProperties)
	return v, ok
}

// Float is property name when it is a number, whole or not.
func (p TiledProperties) Float(name string) (float64, bool) {
	switch v := p[name].(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	}
	return 0, false
}

// Bool is property name when it is true or false.
func (p TiledProperties) Bool(name string) (bool, bool) {
	v, ok := p[name].(bool)
	return v, ok
}

type tiledProperty struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

type tiledTileset struct {
	FirstGID         int             `json:"firstgid"`
	Source           string          `json:"source"`
	Class            string          `json:"class"`
	Properties       []tiledProperty `json:"properties"`
	ObjectAlignment  string          `json:"objectalignment"`
	TransparentColor string          `json:"transparentcolor"`
	TileRenderSize   string          `json:"tilerendersize"`
	FillMode         string          `json:"fillmode"`
	TileOffset       struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"tileoffset"`
	// abs is the tileset's file from the file system's root, for one a
	// template brings that the map doesn't name.
	abs        string
	Image      string `json:"image"`
	TileWidth  int    `json:"tilewidth"`
	TileHeight int    `json:"tileheight"`
	Spacing    int    `json:"spacing"`
	Margin     int    `json:"margin"`
	Grid       struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"grid"`
	Tiles []struct {
		ID         int             `json:"id"`
		Type       string          `json:"type"`
		Class      string          `json:"class"`
		Image      string          `json:"image"`
		X          int             `json:"x"`
		Y          int             `json:"y"`
		Width      int             `json:"width"`
		Height     int             `json:"height"`
		Properties []tiledProperty `json:"properties"`
		Animation  []struct {
			TileID   int `json:"tileid"`
			Duration int `json:"duration"`
		} `json:"animation"`
		ObjectGroup struct {
			Objects []tiledCollision `json:"objects"`
		} `json:"objectgroup"`
	} `json:"tiles"`
}

// tiledCollision is a collision shape drawn on a tile in Tiled's collision
// editor: a rectangle, an ellipse, a polygon or a point.
type tiledCollision struct {
	// Class (Type before Tiled 1.9) "shadow" makes the shape block light
	// only, "body" bodies only.
	Class    string  `json:"class"`
	Type     string  `json:"type"`
	Rotation float32 `json:"rotation"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	Width    float32 `json:"width"`
	Height   float32 `json:"height"`
	Polygon  []struct {
		X float32 `json:"x"`
		Y float32 `json:"y"`
	} `json:"polygon"`
}

// rect is the shape's bounds, in the tile's pixels; ok is false for a
// point, which covers nothing.
func (s tiledCollision) rect() (geom.Rect, bool) {
	if len(s.Polygon) == 0 {
		return geom.Rect{X: s.X, Y: s.Y, W: s.Width, H: s.Height}, s.Width > 0 && s.Height > 0
	}
	lo, hi := s.Polygon[0], s.Polygon[0]
	for _, p := range s.Polygon {
		lo.X, lo.Y, hi.X, hi.Y = min(lo.X, p.X), min(lo.Y, p.Y), max(hi.X, p.X), max(hi.Y, p.Y)
	}
	return geom.Rect{X: s.X + lo.X, Y: s.Y + lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}, hi.X > lo.X && hi.Y > lo.Y
}

type tiledLayer struct {
	ID          int             `json:"id"`
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Class       string          `json:"class"`
	Visible     *bool           `json:"visible"`
	Opacity     *float32        `json:"opacity"`
	OffsetX     float32         `json:"offsetx"`
	OffsetY     float32         `json:"offsety"`
	TintColor   string          `json:"tintcolor"`
	ParallaxX   *float32        `json:"parallaxx"`
	ParallaxY   *float32        `json:"parallaxy"`
	DrawOrder   string          `json:"draworder"`
	Width       int             `json:"width"`
	Height      int             `json:"height"`
	Data        json.RawMessage `json:"data"`
	Encoding    string          `json:"encoding"`
	Compression string          `json:"compression"`
	Properties  []tiledProperty `json:"properties"`
	Objects     []tiledObject   `json:"objects"`
	// Layers are a group's.
	Layers []tiledLayer `json:"layers"`
	// Image is an image layer's picture, from the map's folder.
	Image            string `json:"image"`
	RepeatX          bool   `json:"repeatx"`
	RepeatY          bool   `json:"repeaty"`
	TransparentColor string `json:"transparentcolor"`
	// gids are a tile layer's cells, read before the tilesets load.
	gids []uint32
}

// eachLayer calls fn on every layer of layers and of the groups among
// them, a group before what it holds, until fn fails.
func eachLayer(layers []tiledLayer, fn func(*tiledLayer) error) error {
	for i := range layers {
		if err := fn(&layers[i]); err != nil {
			return err
		}
		if err := eachLayer(layers[i].Layers, fn); err != nil {
			return err
		}
	}
	return nil
}

type tiledPoint struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type tiledObject struct {
	// raw is the object as the map writes it, kept for one made from a
	// template, whose values lie over the template's.
	raw        []byte
	ID         int             `json:"id"`
	Visible    *bool           `json:"visible"`
	Rotation   float32         `json:"rotation"`
	Template   string          `json:"template"`
	Text       *tiledText      `json:"text"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Class      string          `json:"class"`
	X          float32         `json:"x"`
	Y          float32         `json:"y"`
	Width      float32         `json:"width"`
	Height     float32         `json:"height"`
	GID        uint32          `json:"gid"`
	Ellipse    bool            `json:"ellipse"`
	Point      bool            `json:"point"`
	Polygon    []tiledPoint    `json:"polygon"`
	Polyline   []tiledPoint    `json:"polyline"`
	Properties []tiledProperty `json:"properties"`
}

// UnmarshalJSON reads o, keeping its JSON when it is made from a template.
func (o *tiledObject) UnmarshalJSON(b []byte) error {
	type plain tiledObject
	if err := json.Unmarshal(b, (*plain)(o)); err != nil {
		return err
	}
	if o.Template != "" {
		o.raw = append([]byte(nil), b...)
	}
	return nil
}

type tiledText struct {
	Text       string `json:"text"`
	FontFamily string `json:"fontfamily"`
	PixelSize  *int   `json:"pixelsize"`
	Color      string `json:"color"`
	Wrap       bool   `json:"wrap"`
	Bold       bool   `json:"bold"`
	Italic     bool   `json:"italic"`
	Underline  bool   `json:"underline"`
	Strikeout  bool   `json:"strikeout"`
	Kerning    *bool  `json:"kerning"`
	HAlign     string `json:"halign"`
	VAlign     string `json:"valign"`
}

// style is t as a TiledTextStyle, Tiled's defaults where it saved none.
func (t tiledText) style() (*TiledTextStyle, error) {
	s := &TiledTextStyle{
		FontFamily: t.FontFamily, PixelSize: 16, Color: color.Black, Wrap: t.Wrap, Bold: t.Bold,
		Italic: t.Italic, Underline: t.Underline, Strikeout: t.Strikeout, Kerning: t.Kerning == nil || *t.Kerning,
		HAlign: t.HAlign, VAlign: t.VAlign,
	}
	if t.PixelSize != nil {
		s.PixelSize = *t.PixelSize
	}
	if s.HAlign == "" {
		s.HAlign = alignLeft
	}
	if s.VAlign == "" {
		s.VAlign = "top"
	}
	if t.Color != "" {
		c, err := tiledColor(t.Color)
		if err != nil {
			return nil, err
		}
		s.Color = c
	}
	return s, nil
}

type tiledMap struct {
	Class           string          `json:"class"`
	Width           int             `json:"width"`
	Height          int             `json:"height"`
	BackgroundColor string          `json:"backgroundcolor"`
	ParallaxOriginX float32         `json:"parallaxoriginx"`
	ParallaxOriginY float32         `json:"parallaxoriginy"`
	Orientation     string          `json:"orientation"`
	Infinite        bool            `json:"infinite"`
	TileWidth       int             `json:"tilewidth"`
	TileHeight      int             `json:"tileheight"`
	Layers          []tiledLayer    `json:"layers"`
	Tilesets        []tiledTileset  `json:"tilesets"`
	Properties      []tiledProperty `json:"properties"`
}

// The high bits of a Tiled cell flip its tile across, upside down and
// across its diagonal; bit 28 turns hexagonal tiles, which aren't read.
const (
	tiledFlipX    = 0x80000000
	tiledFlipY    = 0x40000000
	tiledFlipDiag = 0x20000000
	tiledFlags    = 0xF0000000
)

// LoadTiled reads a Tiled map saved as JSON (.tmj), with its tilesets
// embedded or in .tsj files and their images, all from fsys. opts apply to
// every layer's TileMap, such as CellSize, PixelScale or TallIn. Cells keep
// their flips (TileMap.Flip), drawn as Tiled draws them, moved by their
// tileset's drawing offset; animated tiles play each frame for its own
// duration, and a tile's class is its Tileset's Class. Objects keep their id, shape, points, rotation
// and visibility and, for a tile object, its tile, and a text object its
// text and style; an object made from a template (.tj) is the template's
// object with its own values over it. Properties keep their Tiled type: a
// file is its path from fsys's root, a custom class its members. Layers
// keep the file's order and their settings (TiledLayer), a tile layer's
// TileMap and an image layer's Picture drawn as they say; a group holds
// its layers, its settings folded onto them. Orthogonal and isometric maps
// are read; staggered and hexagonal maps, infinite maps, repeated images
// and templates saved as XML are errors, as are two layers of one name in
// the same group. What only the editor uses (terrain and Wang sets,
// export and editor settings, a layer's lock, a tile's probability) is not
// read.
//
// Only the tilesets the map uses are loaded: those a tile layer or a tile
// object takes a tile from (UsedTilesets), as the map's global ids tell.
// An unused tileset of its own file is not read at all, so its .tsj and
// pictures needn't be in fsys; one embedded in the map keeps its
// properties and shapes but loads no pictures. A map can so hold a whole
// library of tilesets for editing in Tiled at no cost.
func LoadTiled(fsys fs.FS, name string, opts ...TileMapOption) (*Tiled, error) {
	var m tiledMap
	if err := readJSON(fsys, name, &m); err != nil {
		return nil, err
	}
	opts = cellsOfScale(opts, m)
	t, err := newTiled(m)
	if err != nil {
		return nil, fmt.Errorf("twod: %s: %w", name, err)
	}
	t.dir = path.Dir(name)
	t.properties = properties(m.Properties, t.dir)
	t.unit = Cell(m.TileWidth, m.TileHeight)
	if err = t.applyTemplates(fsys, &m); err != nil {
		return nil, fmt.Errorf("twod: %s: %w", name, err)
	}
	firsts := make([]int, len(m.Tilesets))
	for i, ts := range m.Tilesets {
		firsts[i] = ts.FirstGID
	}
	if err = t.readCells(&m, firsts); err != nil {
		return nil, fmt.Errorf("twod: %s: %w", name, err)
	}
	if err = t.loadTilesets(fsys, name, m, loadOf(opts)); err != nil {
		return nil, err
	}
	t.scale = scaleOf(m, opts)
	if t.layers, err = t.layersOf(fsys, m.Layers, nil, firsts, m, opts); err != nil {
		return nil, fmt.Errorf("twod: %s: %w", name, err)
	}
	return t, nil
}

// layersOf are layers as TiledLayers, in parent, nil at the top of the
// map; two of one name are an error.
func (t *Tiled) layersOf(
	fsys fs.FS, layers []tiledLayer, parent *TiledLayer, firsts []int, m tiledMap, opts []TileMapOption,
) ([]*TiledLayer, error) {
	var out []*TiledLayer
	for _, l := range layers {
		layer, err := t.layerOf(fsys, l, parent, firsts, m, opts)
		if err != nil {
			return nil, fmt.Errorf("layer %s: %w", l.Name, err)
		}
		if other, ok := layerNamed(out, l.Name); ok {
			return nil, fmt.Errorf("layers %d and %d are both called %q", other.ID, l.ID, l.Name)
		}
		out = append(out, layer)
	}
	return out, nil
}

// readCells reads every tile layer's cells, groups' included, first, so
// the map's used tilesets are known before any picture is loaded.
func (t *Tiled) readCells(m *tiledMap, firsts []int) error {
	t.used = make([]bool, len(m.Tilesets))
	use := func(g uint32) {
		if g &^= tiledFlags; g != 0 && len(firsts) > 0 {
			t.used[tilesetOf(firsts, g)] = true
		}
	}
	return eachLayer(m.Layers, func(l *tiledLayer) error {
		switch l.Type {
		case tileLayerType:
			cells, err := cellsOf(*l)
			if err != nil {
				return fmt.Errorf("layer %s: %w", l.Name, err)
			}
			l.gids = cells
			for _, g := range cells {
				use(g)
			}
		case "objectgroup":
			for _, o := range l.Objects {
				use(o.GID)
			}
		case groupType, imageLayerType:
		default:
			return fmt.Errorf("layer %s: %s layers are not supported", l.Name, l.Type)
		}
		return nil
	})
}

// The kinds of layer a map's file names.
const (
	tileLayerType  = "tilelayer"
	groupType      = "group"
	imageLayerType = "imagelayer"
)

// loadTilesets loads map name's tilesets: the used ones whole, an unused
// one of its own file not read at all, so the file needn't even be there.
func (t *Tiled) loadTilesets(fsys fs.FS, name string, m tiledMap, load *tiledLoad) error {
	dir := path.Dir(name)
	for i, ts := range m.Tilesets {
		base, source := dir, ""
		if ts.Source != "" {
			source = path.Join(dir, ts.Source)
			if ts.abs != "" {
				source = ts.abs
			}
			if !t.used[i] {
				unread := NewTileset(nil, t.unit)
				unread.source = source
				t.tilesets = append(t.tilesets, unread)
				continue
			}
			if err := readJSON(fsys, source, &ts); err != nil {
				return err
			}
			base = path.Dir(source)
		}
		tileset, err := tiledTilesetOf(fsys, base, ts, m.TileWidth, m.TileHeight, load, t.used[i])
		if err != nil {
			if src := m.Tilesets[i].Source; src != "" {
				return fmt.Errorf("twod: %s: %s: %w", name, src, err)
			}
			return fmt.Errorf("twod: %s: %w", name, err)
		}
		tileset.source = source
		t.tilesets = append(t.tilesets, tileset)
	}
	return nil
}

// PixelScale makes each of a Tiled map's pixels take n points: its cells
// are the map's tile size times n, whatever it is, as CellSize would set
// them, the later of the two winning. Tiled.Scale is n.
func PixelScale(n float32) TiledOption { return pixelScale(n) }

type pixelScale float32

func (pixelScale) tiled(*tiledLoad) {}
func (pixelScale) tileMap(*TileMap) {}

// cellsOfScale is opts with each PixelScale a CellSize of map m's tiles.
func cellsOfScale(opts []TileMapOption, m tiledMap) []TileMapOption {
	out := slices.Clone(opts)
	for i, o := range out {
		if n, ok := o.(pixelScale); ok {
			out[i] = CellSize(float32(m.TileWidth)*float32(n), float32(m.TileHeight)*float32(n))
		}
	}
	return out
}

// scaleOf is how many points a map pixel takes: the cell opts give, by
// CellSize, over the map's, 1 without one.
func scaleOf(m tiledMap, opts []TileMapOption) float32 {
	if m.TileWidth <= 0 {
		return 1
	}
	var cell TileMap
	cell.tileSize = geom.V2(float32(m.TileWidth), float32(m.TileHeight))
	for _, o := range opts {
		if size, ok := o.(mapOption); ok && size.tile != nil {
			size.tile(&cell)
		}
	}
	return cell.tileSize.X / float32(m.TileWidth)
}

// newTiled is what map m says of itself, its layers and tilesets yet to
// read; an error for what it can't read.
func newTiled(m tiledMap) (*Tiled, error) {
	switch {
	case m.Orientation != orthogonal && m.Orientation != isometric:
		return nil, fmt.Errorf("%s maps are not supported", m.Orientation)
	case m.Infinite:
		return nil, errors.New("infinite maps are not supported")
	}
	t := &Tiled{
		class:          m.Class,
		columns:        m.Width,
		rows:           m.Height,
		tileW:          m.TileWidth,
		tileH:          m.TileHeight,
		isometric:      m.Orientation == isometric,
		parallaxOrigin: geom.V2(m.ParallaxOriginX, m.ParallaxOriginY),
		byID:           map[int]TiledObject{},
	}
	if m.BackgroundColor != "" {
		c, err := tiledColor(m.BackgroundColor)
		if err != nil {
			return nil, fmt.Errorf("background: %w", err)
		}
		t.background = c
	}
	return t, nil
}

// The orientations of the maps the loader reads.
const (
	orthogonal = "orthogonal"
	isometric  = "isometric"
)

// layerOf is l as a TiledLayer, its tile layer's TileMap made and its
// objects read.
func (t *Tiled) layerOf(
	fsys fs.FS, l tiledLayer, parent *TiledLayer, firsts []int, m tiledMap, opts []TileMapOption,
) (*TiledLayer, error) {
	layer := &TiledLayer{
		ID: l.ID, Name: l.Name, Class: l.Class, Properties: properties(l.Properties, t.dir),
		Visible: l.Visible == nil || *l.Visible, Opacity: 1, Offset: geom.V2(l.OffsetX, l.OffsetY),
		Tint: color.White, Parallax: geom.V2(1, 1), parent: parent,
	}
	if l.Opacity != nil {
		layer.Opacity = *l.Opacity
	}
	if l.ParallaxX != nil {
		layer.Parallax.X = *l.ParallaxX
	}
	if l.ParallaxY != nil {
		layer.Parallax.Y = *l.ParallaxY
	}
	if l.TintColor != "" {
		c, err := tiledColor(l.TintColor)
		if err != nil {
			return nil, err
		}
		layer.Tint = c
	}
	var err error
	switch l.Type {
	case tileLayerType:
		layer.Kind = TiledTiles
		layer.tileMap = t.tileLayer(l, l.gids, firsts, m, append(layer.mapOptions(t.scale), opts...))
	case groupType:
		layer.Kind = TiledGroup
		layer.layers, err = t.layersOf(fsys, l.Layers, layer, firsts, m, opts)
	case imageLayerType:
		layer.Kind = TiledImage
		layer.picture, err = t.pictureOf(fsys, l, layer)
	default:
		layer.Kind = TiledObjects
		if l.DrawOrder == "index" {
			layer.DrawOrder = TiledIndex
		}
		offset := layer.FoldedOffset()
		for _, o := range l.Objects {
			obj, objErr := t.objectOf(o, firsts, t.dir)
			if objErr != nil {
				return nil, objErr
			}
			obj.shift(offset)
			layer.objects = append(layer.objects, obj)
			t.byID[obj.ID] = obj
		}
	}
	if err != nil {
		return nil, err
	}
	return layer, nil
}

// pictureOf is image layer l's picture, nil when it names none: its
// top-left corner at the layer's offset, in the map's points, drawn as the
// layer's folded settings say.
func (t *Tiled) pictureOf(fsys fs.FS, l tiledLayer, layer *TiledLayer) (*Sprite, error) {
	switch {
	case l.RepeatX || l.RepeatY:
		return nil, errors.New("a repeated image is not supported")
	case l.TransparentColor != "":
		return nil, errors.New("an image layer's transparent colour is not supported")
	case l.Image == "":
		return nil, nil //nolint:nilnil // an image layer may hold no picture
	}
	tex, err := LoadTexture(fsys, path.Join(t.dir, l.Image), Pixelated())
	if err != nil {
		return nil, err
	}
	size, at := tex.Size().Scale(t.scale), layer.FoldedOffset().Scale(t.scale)
	tint := layer.FoldedTint()
	tint.A *= layer.FoldedOpacity()
	opts := []SpriteOption{Pivot(0, 0), At(at.X, at.Y), Size(size.X, size.Y), Tint(tint)}
	if !layer.Shown() {
		opts = append(opts, Hidden())
	}
	return NewSprite(tex, opts...), nil
}

// objectOf is o as a TiledObject, its tile found among the map's tilesets.
func (t *Tiled) objectOf(o tiledObject, firsts []int, dir string) (TiledObject, error) {
	kind := o.Type
	if kind == "" {
		kind = o.Class
	}
	obj := TiledObject{
		ID:         o.ID,
		Name:       o.Name,
		Type:       kind,
		Rect:       geom.Rect{X: o.X, Y: o.Y, W: o.Width, H: o.Height},
		Rotation:   o.Rotation * math.Pi / 180,
		Visible:    o.Visible == nil || *o.Visible,
		Properties: properties(o.Properties, dir),
	}
	points := func(list []tiledPoint) []geom.Vec2 {
		out := make([]geom.Vec2, len(list))
		for i, p := range list {
			out[i] = geom.V2(o.X+p.X, o.Y+p.Y)
		}
		return out
	}
	switch {
	case o.Point:
		obj.Shape = TiledPoint
	case o.Ellipse:
		obj.Shape = TiledEllipse
	case len(o.Polygon) > 0:
		obj.Shape, obj.Points = TiledPolygon, points(o.Polygon)
	case len(o.Polyline) > 0:
		obj.Shape, obj.Points = TiledPolyline, points(o.Polyline)
	case o.Text != nil:
		style, err := o.Text.style()
		if err != nil {
			return TiledObject{}, fmt.Errorf("object %d (%s): %w", o.ID, o.Name, err)
		}
		obj.Shape, obj.Text, obj.TextStyle = TiledText, o.Text.Text, style
	}
	if g := o.GID &^ tiledFlags; g != 0 && len(firsts) > 0 {
		s := tilesetOf(firsts, g)
		obj.Tile = &TiledTile{GID: o.GID, Tileset: t.tilesets[s], ID: int(g) - firsts[s] + 1, Flip: flipOf(o.GID)}
	}
	return obj, nil
}

// tilesetOf is the index of the tileset global id g is from: the last whose
// first id is at most g.
func tilesetOf(firsts []int, g uint32) int {
	s := 0
	for k, first := range firsts {
		if int(g) >= first {
			s = k
		}
	}
	return s
}

// flipOf is the flips Tiled's high bits on g ask for.
func flipOf(g uint32) TileFlip {
	var f TileFlip
	if g&tiledFlipX != 0 {
		f |= TileFlipX
	}
	if g&tiledFlipY != 0 {
		f |= TileFlipY
	}
	if g&tiledFlipDiag != 0 {
		f |= TileFlipDiagonal
	}
	return f
}

func readJSON(fsys fs.FS, name string, v any) error {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("twod: %w", err)
	}
	if err = json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("twod: %s: %w", name, err)
	}
	return nil
}

// properties are list's values by name, each by its Tiled type: a
// property pointing to an object is the object's id, one naming a file its
// path from the file system's root, dir being the folder of the file that
// holds list, one of a custom class its members.
func properties(list []tiledProperty, dir string) TiledProperties {
	if len(list) == 0 {
		return nil
	}
	out := TiledProperties{}
	for _, p := range list {
		v := p.Value
		switch p.Type {
		case typeInt, "object":
			if f, ok := v.(float64); ok {
				v = int(f)
			}
		case "file":
			if f, ok := v.(string); ok && f != "" {
				v = path.Join(dir, f)
			}
		case "class":
			v = members(v)
		case typeBool, typeFloat:
		default:
			if _, ok := v.(string); !ok && v != nil && p.Type != "" {
				v = fmt.Sprint(v)
			}
		}
		out[p.Name] = v
	}
	return out
}

// members are a custom class's member values as Tiled saves them, a
// nested class's as their own TiledProperties.
func members(v any) TiledProperties {
	m, _ := v.(map[string]any)
	out := TiledProperties{}
	for name, value := range m {
		if nested, ok := value.(map[string]any); ok {
			out[name] = members(nested)
			continue
		}
		out[name] = value
	}
	return out
}

// LoadTiledTileset reads a Tiled tileset saved as JSON (.tsj) and its
// images from fsys: an atlas cut into a grid, or a collection of tiles,
// each a picture of its own or, as Tiled 1.9 saves them, a rectangle of a
// shared one, at its own size. Tiles keep their properties, solid and tall
// among them, their animations, and the shapes drawn in Tiled's collision
// editor as their Collision, a polygon or an ellipse by its bounds. A
// shape of class "shadow" blocks light only (ShadowShapes), one of class
// "body" bodies only (BodyShapes); a tile with the property shadow
// "none" blocks no light at all (CastShadow(false)), and occluder_group
// sets its occluder group (OccluderGroup). Tiled counts tiles from 0 and a
// Tileset from 1: Tiled's tile 5 is the Tileset's tile 6.
//
// Tile properties are checked against EngineTileProperties and those
// TileProperties declares: a wrong kind, an enum value or a number out of
// range is an error naming the tileset, the tile and the property; any
// other property is kept, read with Value.
func LoadTiledTileset(fsys fs.FS, name string, opts ...TiledOption) (*Tileset, error) {
	var ts tiledTileset
	if err := readJSON(fsys, name, &ts); err != nil {
		return nil, err
	}
	load := &tiledLoad{}
	for _, o := range opts {
		o.tiled(load)
	}
	t, err := tiledTilesetOf(fsys, path.Dir(name), ts, 0, 0, load, true)
	if err != nil {
		return nil, fmt.Errorf("twod: %s: %w", name, err)
	}
	return t, nil
}

// tiledTilesetOf makes ts's tileset, its images found from dir and loaded
// when pictures is true; without, it keeps everything but the pictures. A
// collection's grid is its own, else unitW by unitH, the map's cell in
// pixels, else its largest tile.
func tiledTilesetOf(
	fsys fs.FS,
	dir string,
	ts tiledTileset,
	unitW, unitH int,
	props *tiledLoad,
	pictures bool,
) (*Tileset, error) {
	images := map[string]*Texture{}
	load := func(name string) (*Texture, error) {
		if tex, ok := images[name]; ok || !pictures {
			return tex, nil
		}
		tex, err := LoadTexture(fsys, path.Join(dir, name), Pixelated())
		images[name] = tex
		return tex, err
	}
	if err := refusedTileset(ts); err != nil {
		return nil, err
	}
	var tileset *Tileset
	if ts.Image != "" {
		tex, err := load(ts.Image)
		if err != nil {
			return nil, err
		}
		tileset = NewTileset(tex, Cell(ts.TileWidth, ts.TileHeight), Spacing(ts.Spacing, ts.Spacing), Margin(ts.Margin))
		tileset.file = path.Join(dir, ts.Image)
	} else {
		w, h := ts.Grid.Width, ts.Grid.Height
		if w == 0 || h == 0 {
			w, h = unitW, unitH
		}
		if w == 0 || h == 0 {
			w, h = ts.TileWidth, ts.TileHeight
		}
		tileset = NewTileset(nil, Cell(w, h))
	}
	for _, tile := range ts.Tiles {
		id := tile.ID + 1
		if tile.Image != "" {
			tex, err := load(tile.Image)
			if err != nil {
				return nil, err
			}
			region := geom.Rect{X: float32(tile.X), Y: float32(tile.Y), W: float32(tile.Width), H: float32(tile.Height)}
			tileset.setTileImage(id, path.Join(dir, tile.Image), tex, region)
		}
		for _, p := range tile.Properties {
			if err := tileset.setTiledProperty(id, p, props, dir); err != nil {
				return nil, fmt.Errorf("tile %d: property %s: %w", tile.ID, p.Name, err)
			}
		}
		for _, shape := range tile.ObjectGroup.Objects {
			tileset.addTiledShape(id, shape)
		}
		if len(tile.Animation) > 0 {
			a := tileAnimation{}
			for _, f := range tile.Animation {
				a.frames = append(a.frames, f.TileID+1)
				a.durations = append(a.durations, float32(f.Duration)/1000)
			}
			tileset.anims[id] = a
		}
		if class := cmp.Or(tile.Class, tile.Type); class != "" {
			if tileset.classes == nil {
				tileset.classes = map[int]string{}
			}
			tileset.classes[id] = class
		}
	}
	tileset.load = props
	tileset.objectAlign = ts.ObjectAlignment
	tileset.offset = geom.V2(float32(ts.TileOffset.X), float32(ts.TileOffset.Y))
	return tileset, nil
}

// setTileImage gives tile id its own picture, file in the fsys: tex, or
// region of it when that has a size, tex nil when pictures aren't loaded.
func (t *Tileset) setTileImage(id int, file string, tex *Texture, region geom.Rect) {
	if t.files == nil {
		t.files = map[int]string{}
	}
	t.files[id] = file
	if tex == nil {
		return
	}
	if region.W == 0 || region.H == 0 {
		size := tex.Size()
		region = geom.Rect{W: size.X, H: size.Y}
	}
	t.tiles[id] = tileImage{tex, region}
}

// refusedTileset is an error for what of ts the loader can't read yet: a
// render size or fill mode other than Tiled's default, a
// transparent colour, a class or properties on the tileset, and a
// collision shape turned.
func refusedTileset(ts tiledTileset) error {
	switch {
	case ts.TileRenderSize != "" && ts.TileRenderSize != "tile":
		return fmt.Errorf("render size %s is not supported", ts.TileRenderSize)
	case ts.FillMode != "" && ts.FillMode != "stretch":
		return fmt.Errorf("fill mode %s is not supported", ts.FillMode)
	case ts.TransparentColor != "":
		return errors.New("a transparent colour is not supported")
	case ts.Class != "":
		return errors.New("a tileset's class is not supported")
	case len(ts.Properties) > 0:
		return errors.New("a tileset's own properties are not supported")
	}
	for _, tile := range ts.Tiles {
		for _, shape := range tile.ObjectGroup.Objects {
			if shape.Rotation != 0 {
				return fmt.Errorf("tile %d: a turned collision shape is not supported", tile.ID)
			}
		}
	}
	return nil
}

// setTiledProperty sets tile id's property p, checked against the
// declarations in props; undeclared ones are kept as they are, a file's
// path from dir, the tileset's folder.
func (t *Tileset) setTiledProperty(id int, p tiledProperty, props *tiledLoad, dir string) error {
	d, known := props.def(p.Name)
	if !known {
		for name, v := range properties([]tiledProperty{p}, dir) {
			t.setProperty(id, name, v)
		}
		return nil
	}
	v, err := d.check(p.Type, p.Value)
	if err != nil {
		return err
	}
	switch p.Name {
	case "solid":
		if v == true {
			t.solid[id] = true
		}
	case "tall":
		if v == true {
			t.tall[id] = true
		}
	case propShadow:
		if v == "none" {
			t.noShadow[id] = true
		}
	case "occluder_group":
		t.occluderGroup[id], _ = v.(int) // typed by the property's declaration
	case propHeight:
		h, _ := v.(float64) // typed by the property's declaration
		t.height[id] = float32(h)
	case "emissive":
		e, _ := v.(float64) // typed by the property's declaration
		t.emissive[id] = float32(e)
	default:
		t.setProperty(id, p.Name, v)
	}
	return nil
}

// addTiledShape adds a collision shape drawn on tile id: light-only, body-only
// or both, by its class.
func (t *Tileset) addTiledShape(id int, shape tiledCollision) {
	r, ok := shape.rect()
	switch {
	case !ok:
	case shape.Class == propShadow || shape.Type == propShadow:
		t.shadows[id] = append(t.shadows[id], r)
	case shape.Class == "body" || shape.Type == "body":
		t.bodies[id] = append(t.bodies[id], r)
	default:
		t.shapes[id] = append(t.shapes[id], r)
	}
}

// tileLayer turns a layer's global tile ids into a TileMap of the one
// tileset they come from, or of all the map's tilesets merged when they
// come from several, numbered by global id.
func (t *Tiled) tileLayer(l tiledLayer, gids []uint32, firsts []int, m tiledMap, opts []TileMapOption) *TileMap {
	set, mixed := -1, false
	cells := make([]int, len(gids))
	flips := make([]TileFlip, len(gids))
	flipped := false
	for i, g := range gids {
		flips[i] = flipOf(g)
		flipped = flipped || flips[i] != 0
		g &^= tiledFlags
		if g == 0 {
			continue
		}
		s := tilesetOf(firsts, g)
		mixed = mixed || set >= 0 && s != set
		set = s
		cells[i] = int(g)
	}
	var tileset *Tileset
	switch {
	case mixed:
		if t.merged == nil {
			t.merged = mergeTilesets(t.unit, firsts, t.tilesets)
		}
		tileset = t.merged
	case set >= 0:
		tileset = t.tilesets[set]
		for i, g := range cells {
			if g != 0 {
				cells[i] = g - firsts[set] + 1
			}
		}
	case len(t.tilesets) > 0:
		tileset = t.tilesets[0]
	}
	grid := make([][]int, l.Height)
	for y := range grid {
		grid[y] = cells[y*l.Width : (y+1)*l.Width]
	}
	all := []TileMapOption{Cells(grid), CellSize(float32(m.TileWidth), float32(m.TileHeight))}
	if m.Orientation == isometric {
		all = append(all, Isometric())
	}
	layer := NewTileMap(tileset, append(all, opts...)...)
	if flipped {
		layer.flips = flips
		layer.syncTalls()
	}
	layer.SetName(l.Name)
	return layer
}

// cellsOf reads a layer's cells, as an array or base64, raw or compressed.
func cellsOf(l tiledLayer) ([]uint32, error) {
	if l.Width*l.Height == 0 {
		return nil, nil
	}
	if l.Encoding != "base64" {
		var cells []uint32
		if err := json.Unmarshal(l.Data, &cells); err != nil {
			return nil, err
		}
		if len(cells) != l.Width*l.Height {
			return nil, fmt.Errorf("%d cells for %d×%d", len(cells), l.Width, l.Height)
		}
		return cells, nil
	}
	var text string
	if err := json.Unmarshal(l.Data, &text); err != nil {
		return nil, err
	}
	raw, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil, err
	}
	var r io.Reader = bytes.NewReader(raw)
	switch l.Compression {
	case "":
	case "zlib":
		if r, err = zlib.NewReader(r); err != nil {
			return nil, err
		}
	case "gzip":
		if r, err = gzip.NewReader(r); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%s compression is not supported", l.Compression)
	}
	cells := make([]uint32, l.Width*l.Height)
	if err = binary.Read(r, binary.LittleEndian, cells); err != nil {
		return nil, err
	}
	return cells, nil
}

// Class is the map's class, as Tiled names it.
func (t *Tiled) Class() string { return t.class }

// Columns is how many cells wide the map is.
func (t *Tiled) Columns() int { return t.columns }

// Rows is how many cells tall the map is.
func (t *Tiled) Rows() int { return t.rows }

// TilePixels is a cell's width and height in the map's pixels.
func (t *Tiled) TilePixels() (int, int) { return t.tileW, t.tileH }

// Scale is how many points a map pixel takes, from the map's CellSize:
// objects are in map pixels.
func (t *Tiled) Scale() float32 { return t.scale }

// Background is the map's background colour, for a game's clear colour;
// clear when the map sets none.
func (t *Tiled) Background() color.Color { return t.background }

// ParallaxOrigin is where the map's layers' parallax is measured from, in
// map pixels.
func (t *Tiled) ParallaxOrigin() geom.Vec2 { return t.parallaxOrigin }

// Layers are the map's layers, bottom first, as the file orders them.
func (t *Tiled) Layers() []*TiledLayer { return t.layers }

// Layer is the map's layer called name.
func (t *Tiled) Layer(name string) (*TiledLayer, bool) { return layerNamed(t.layers, name) }

// Walk visits every layer of the map in draw order, a group before what it
// holds, until visit returns false.
func (t *Tiled) Walk(visit func(*TiledLayer) bool) { walkLayers(t.layers, visit) }

// TileLayers are every tile layer's map, in draw order, each named after
// its layer.
func (t *Tiled) TileLayers() []*TileMap {
	var out []*TileMap
	t.Walk(func(l *TiledLayer) bool {
		if l.tileMap != nil {
			out = append(out, l.tileMap)
		}
		return true
	})
	return out
}

// TileLayer is the map of the tile layer called name.
func (t *Tiled) TileLayer(name string) (*TileMap, bool) {
	l, ok := t.Layer(name)
	if !ok || l.tileMap == nil {
		return nil, false
	}
	return l.tileMap, true
}

// Object is the object Tiled numbers id, in any object layer.
func (t *Tiled) Object(id int) (TiledObject, bool) {
	o, ok := t.byID[id]
	return o, ok
}

// Tilesets are the map's tilesets, in order, so their global ids hold;
// only the used ones are loaded (UsedTilesets). An unused one of its own
// file is an empty Tileset that knows its Source.
func (t *Tiled) Tilesets() []*Tileset { return t.tilesets }

// UsedTilesets are the map's tilesets a tile layer or a tile object takes a
// tile from, in order: the ones whose pictures were loaded.
func (t *Tiled) UsedTilesets() []*Tileset {
	var out []*Tileset
	for i, ts := range t.tilesets {
		if t.used[i] {
			out = append(out, ts)
		}
	}
	return out
}

// Properties are the map's own custom properties, such as a room's name,
// tier or weight.
func (t *Tiled) Properties() TiledProperties { return t.properties }

// LayerProperties are the custom properties of the layer called name, a
// tile layer or an object layer.
func (t *Tiled) LayerProperties(name string) TiledProperties {
	if l, ok := t.Layer(name); ok {
		return l.Properties
	}
	return nil
}

// Objects are the objects of the object layer called layer.
func (t *Tiled) Objects(layer string) []TiledObject {
	if l, ok := t.Layer(layer); ok {
		return l.objects
	}
	return nil
}
