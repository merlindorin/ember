package twod

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path"

	"github.com/merlindorin/ember/geom"
)

// Tiled is a map made in the Tiled editor: its tile layers as TileMaps,
// its tilesets, and its object layers as data for the game to spawn from.
type Tiled struct {
	layers     []*TileMap
	tilesets   []*Tileset
	objects    map[string][]TiledObject
	properties TiledProperties
	layerProps map[string]TiledProperties
	// merged is every tileset as one, numbered by global id, for layers
	// whose tiles come from several, made the first time one asks.
	merged *Tileset
	unit   Grid
}

// TiledObject is an object of an object layer: a spawn point, a trigger, a
// door. It is plain data; the game decides what it becomes.
type TiledObject struct {
	Name, Type string
	Rect       geom.Rect
	Properties TiledProperties
}

// TiledProperties are the custom properties set in Tiled on a map, a layer
// or an object, each by its Tiled type: a string (also a color or a file),
// an int, a float64 or a bool.
type TiledProperties map[string]any

// String is property name when it is text.
func (p TiledProperties) String(name string) (string, bool) {
	v, ok := p[name].(string)
	return v, ok
}

// Int is property name when it is a whole number.
func (p TiledProperties) Int(name string) (int, bool) {
	v, ok := p[name].(int)
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
	FirstGID   int    `json:"firstgid"`
	Source     string `json:"source"`
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
			Objects []tiledShape `json:"objects"`
		} `json:"objectgroup"`
	} `json:"tiles"`
}

// tiledShape is a collision shape drawn on a tile in Tiled's collision
// editor: a rectangle, an ellipse, a polygon or a point.
type tiledShape struct {
	// Class (Type before Tiled 1.9) "shadow" makes the shape block light
	// only, "body" bodies only.
	Class   string  `json:"class"`
	Type    string  `json:"type"`
	X       float32 `json:"x"`
	Y       float32 `json:"y"`
	Width   float32 `json:"width"`
	Height  float32 `json:"height"`
	Polygon []struct {
		X float32 `json:"x"`
		Y float32 `json:"y"`
	} `json:"polygon"`
}

// rect is the shape's bounds, in the tile's pixels; ok is false for a
// point, which covers nothing.
func (s tiledShape) rect() (geom.Rect, bool) {
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
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Width       int             `json:"width"`
	Height      int             `json:"height"`
	Data        json.RawMessage `json:"data"`
	Encoding    string          `json:"encoding"`
	Compression string          `json:"compression"`
	Properties  []tiledProperty `json:"properties"`
	Objects     []struct {
		Name       string          `json:"name"`
		Type       string          `json:"type"`
		Class      string          `json:"class"`
		X          float32         `json:"x"`
		Y          float32         `json:"y"`
		Width      float32         `json:"width"`
		Height     float32         `json:"height"`
		Properties []tiledProperty `json:"properties"`
	} `json:"objects"`
}

type tiledMap struct {
	Orientation string          `json:"orientation"`
	Infinite    bool            `json:"infinite"`
	TileWidth   int             `json:"tilewidth"`
	TileHeight  int             `json:"tileheight"`
	Layers      []tiledLayer    `json:"layers"`
	Tilesets    []tiledTileset  `json:"tilesets"`
	Properties  []tiledProperty `json:"properties"`
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
// every layer's TileMap, such as CellSize or TallIn. Cells keep their
// flips (TileMap.Flip), drawn as Tiled draws them. Orthogonal and
// isometric maps are read; staggered and hexagonal maps, infinite maps and
// image layers are errors.
func LoadTiled(fsys fs.FS, name string, opts ...TileMapOption) (*Tiled, error) {
	var m tiledMap
	if err := readJSON(fsys, name, &m); err != nil {
		return nil, err
	}
	switch {
	case m.Orientation != "orthogonal" && m.Orientation != "isometric":
		return nil, fmt.Errorf("twod: %s: %s maps are not supported", name, m.Orientation)
	case m.Infinite:
		return nil, fmt.Errorf("twod: %s: infinite maps are not supported", name)
	}
	t := &Tiled{
		objects:    map[string][]TiledObject{},
		properties: properties(m.Properties),
		layerProps: map[string]TiledProperties{},
	}
	dir := path.Dir(name)
	load := loadOf(opts)
	t.unit = Cell(m.TileWidth, m.TileHeight)
	firsts := make([]int, len(m.Tilesets))
	for i, ts := range m.Tilesets {
		firsts[i] = ts.FirstGID
		base := dir
		if ts.Source != "" {
			source := path.Join(dir, ts.Source)
			if err := readJSON(fsys, source, &ts); err != nil {
				return nil, err
			}
			base = path.Dir(source)
		}
		tileset, err := tiledTilesetOf(fsys, base, ts, m.TileWidth, m.TileHeight, load)
		if err != nil {
			if ts.Source != "" {
				return nil, fmt.Errorf("twod: %s: %s: %w", name, ts.Source, err)
			}
			return nil, fmt.Errorf("twod: %s: %w", name, err)
		}
		t.tilesets = append(t.tilesets, tileset)
	}
	for _, l := range m.Layers {
		t.layerProps[l.Name] = properties(l.Properties)
		switch l.Type {
		case "tilelayer":
			layer, err := t.tileLayer(l, firsts, m, opts)
			if err != nil {
				return nil, fmt.Errorf("twod: %s: layer %s: %w", name, l.Name, err)
			}
			t.layers = append(t.layers, layer)
		case "objectgroup":
			for _, o := range l.Objects {
				kind := o.Type
				if kind == "" {
					kind = o.Class
				}
				t.objects[l.Name] = append(
					t.objects[l.Name],
					TiledObject{
						Name:       o.Name,
						Type:       kind,
						Rect:       geom.Rect{X: o.X, Y: o.Y, W: o.Width, H: o.Height},
						Properties: properties(o.Properties),
					},
				)
			}
		default:
			return nil, fmt.Errorf("twod: %s: %s layers are not supported", name, l.Type)
		}
	}
	return t, nil
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

func properties(list []tiledProperty) TiledProperties {
	if len(list) == 0 {
		return nil
	}
	out := TiledProperties{}
	for _, p := range list {
		v := p.Value
		switch p.Type {
		case typeInt:
			if f, ok := v.(float64); ok {
				v = int(f)
			}
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
	t, err := tiledTilesetOf(fsys, path.Dir(name), ts, 0, 0, load)
	if err != nil {
		return nil, fmt.Errorf("twod: %s: %w", name, err)
	}
	return t, nil
}

// tiledTilesetOf makes ts's tileset, its images found from dir. A
// collection's grid is its own, else unitW by unitH, the map's cell in
// pixels, else its largest tile.
func tiledTilesetOf(fsys fs.FS, dir string, ts tiledTileset, unitW, unitH int, props *tiledLoad) (*Tileset, error) {
	images := map[string]*Texture{}
	load := func(name string) (*Texture, error) {
		if tex, ok := images[name]; ok {
			return tex, nil
		}
		tex, err := LoadTexture(fsys, path.Join(dir, name), Pixelated())
		images[name] = tex
		return tex, err
	}
	var tileset *Tileset
	if ts.Image != "" {
		tex, err := load(ts.Image)
		if err != nil {
			return nil, err
		}
		tileset = NewTileset(tex, Cell(ts.TileWidth, ts.TileHeight), Spacing(ts.Spacing, ts.Spacing), Margin(ts.Margin))
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
			r := geom.Rect{X: float32(tile.X), Y: float32(tile.Y), W: float32(tile.Width), H: float32(tile.Height)}
			if r.W == 0 || r.H == 0 {
				size := tex.Size()
				r = geom.Rect{W: size.X, H: size.Y}
			}
			tileset.tiles[id] = tileImage{tex, r}
		}
		for _, p := range tile.Properties {
			if err := tileset.setTiledProperty(id, p, props); err != nil {
				return nil, fmt.Errorf("tile %d: property %s: %w", tile.ID, p.Name, err)
			}
		}
		for _, shape := range tile.ObjectGroup.Objects {
			tileset.addTiledShape(id, shape)
		}
		if len(tile.Animation) > 0 && tile.Animation[0].Duration > 0 {
			var frames []int
			for _, f := range tile.Animation {
				frames = append(frames, f.TileID+1)
			}
			tileset.anims[id] = tileAnimation{fps: 1000 / float32(tile.Animation[0].Duration), frames: frames}
		}
	}
	tileset.load = props
	return tileset, nil
}

// setTiledProperty sets tile id's property p, checked against the
// declarations in props; undeclared ones are kept as they are.
func (t *Tileset) setTiledProperty(id int, p tiledProperty, props *tiledLoad) error {
	d, known := props.def(p.Name)
	if !known {
		for name, v := range properties([]tiledProperty{p}) {
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
func (t *Tileset) addTiledShape(id int, shape tiledShape) {
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
func (t *Tiled) tileLayer(l tiledLayer, firsts []int, m tiledMap, opts []TileMapOption) (*TileMap, error) {
	gids, err := cellsOf(l)
	if err != nil {
		return nil, err
	}
	set, mixed := -1, false
	cells := make([]int, len(gids))
	flips := make([]TileFlip, len(gids))
	flipped := false
	for i, g := range gids {
		if g&tiledFlipX != 0 {
			flips[i] |= TileFlipX
		}
		if g&tiledFlipY != 0 {
			flips[i] |= TileFlipY
		}
		if g&tiledFlipDiag != 0 {
			flips[i] |= TileFlipDiagonal
		}
		flipped = flipped || flips[i] != 0
		g &^= tiledFlags
		if g == 0 {
			continue
		}
		s := 0
		for k, first := range firsts {
			if int(g) >= first {
				s = k
			}
		}
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
	if m.Orientation == "isometric" {
		all = append(all, Isometric())
	}
	layer := NewTileMap(tileset, append(all, opts...)...)
	if flipped {
		layer.flips = flips
		layer.syncTalls()
	}
	layer.SetName(l.Name)
	return layer, nil
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

// Layers are the map's tile layers, in order, each named after its layer.
func (t *Tiled) Layers() []*TileMap { return t.layers }

// Layer is the tile layer called name.
func (t *Tiled) Layer(name string) (*TileMap, bool) {
	for _, l := range t.layers {
		if l.Name() == name {
			return l, true
		}
	}
	return nil, false
}

// Tilesets are the map's tilesets, in order.
func (t *Tiled) Tilesets() []*Tileset { return t.tilesets }

// Properties are the map's own custom properties, such as a room's name,
// tier or weight.
func (t *Tiled) Properties() TiledProperties { return t.properties }

// LayerProperties are the custom properties of the layer called name, a
// tile layer or an object layer.
func (t *Tiled) LayerProperties(name string) TiledProperties { return t.layerProps[name] }

// Objects are the objects of the object layer called layer.
func (t *Tiled) Objects(layer string) []TiledObject { return t.objects[layer] }
