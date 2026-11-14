package twod

import (
	"fmt"
	"image"
	"maps"
	"slices"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
)

// The properties a tile, a tile object or a layer is drawn as a variant of
// its picture by (decision 0069), and the engine's class of a mask layer
// with the property naming the layer it cuts.
const (
	propTint     = "tint"
	propRecolour = "recolour"
	propMask     = "mask"
	maskClass    = "mask"
	propCuts     = "cuts"
)

// Palettes gives the Tiled loader the palettes the recolour property
// names, by name; a name it does not hold is an error naming the nearest
// one. Given twice, the later palette of a name wins.
func Palettes(pals map[string]*Palette) TiledOption {
	return tiledOption(func(l *tiledLoad) {
		if l.palettes == nil {
			l.palettes = map[string]*Palette{}
		}
		maps.Copy(l.palettes, pals)
	})
}

// Masks gives the Tiled loader its mask tileset: the mask property and a
// mask layer's tiles name its tiles by their name property; a name it does
// not hold is an error naming the nearest one.
func Masks(set *Tileset) TiledOption { return tiledOption(func(l *tiledLoad) { l.masks = set }) }

// variantProperties are the tile properties a tile is drawn as a variant
// of its picture by.
func variantProperties() []Property {
	return []Property{
		{Name: propTint, Kind: KindColor, Default: "", Doc: "multiplies its colours and opacity, when drawn"},
		{Name: propRecolour, Kind: KindString, Default: "",
			Doc: "the palette, given to the loader, its colours are swapped by"},
		{Name: propMask, Kind: KindString, Default: "", Doc: "the tile of the mask tileset, by name, that cuts it"},
	}
}

// elementProperties are the properties the engine reads on a layer or an
// object of any class, or of none: tint and recolour, and on an object
// mask.
func elementProperties(use ClassUse) []Property {
	var out []Property
	for _, d := range variantProperties() {
		if use&OnObject != 0 || use&OnLayer != 0 && d.Name != propMask {
			out = append(out, d)
		}
	}
	return out
}

// elementProperty is the property called name the engine reads on any
// element of use.
func elementProperty(use ClassUse, name string) (Property, bool) {
	switch name {
	case propTint, propRecolour, propMask:
		return propertyNamed(elementProperties(use), name)
	}
	return Property{}, false
}

// engineClasses are the classes the engine declares: a mask layer, and the
// classes a layer or an object takes in Tiled to show the engine's
// properties as typed fields.
func engineClasses() []Class {
	return []Class{
		{Name: "ember.Layer", On: OnLayer, Properties: elementProperties(OnLayer)},
		{Name: "ember.Object", On: OnObject, Properties: elementProperties(OnObject)},
		{Name: maskClass, On: OnLayer, Doc: "a tile layer of masks cutting the tiles under them, never drawn",
			Properties: []Property{{Name: propCuts, Kind: KindString, Default: "", Required: true,
				Doc: "the tile layer it cuts"}}},
	}
}

// palette is the palette called name.
func (l *tiledLoad) palette(name string) (*Palette, error) {
	if p, ok := l.palettes[name]; ok {
		return p, nil
	}
	if len(l.palettes) == 0 {
		return nil, fmt.Errorf("palette %s: the loader is given no palettes (twod.Palettes)", name)
	}
	return nil, fmt.Errorf("palette %s is not given; the nearest is %s", name, nearest(name, maps.Keys(l.palettes)))
}

// mask is the mask tile called name.
func (l *tiledLoad) mask(name string) (maskCut, error) {
	if l.masks == nil {
		return maskCut{}, fmt.Errorf("mask %s: the loader is given no mask tileset (twod.Masks)", name)
	}
	id, ok := l.masks.ID(name)
	if !ok {
		if len(l.masks.names) == 0 {
			return maskCut{}, fmt.Errorf("mask %s: the mask tileset names no tile", name)
		}
		return maskCut{}, fmt.Errorf("mask %s is not in the mask tileset; the nearest is %s", name,
			nearest(name, maps.Keys(l.masks.names)))
	}
	tex, r := l.masks.image(id, 0)
	if tex == nil || tex.Image == nil {
		return maskCut{}, fmt.Errorf("mask %s has no picture", name)
	}
	return maskCut{tex: tex, rect: r}, nil
}

// nearest is the one of names fewest edits from name, the first in order
// among equals.
func nearest(name string, names func(func(string) bool)) string {
	all := slices.Sorted(names)
	best := all[0]
	for _, n := range all[1:] {
		if edits(name, n) < edits(name, best) {
			best = n
		}
	}
	return best
}

// specOf is what properties p of a tile or a tile object draw it through.
func (l *tiledLoad) specOf(p TiledProperties) (lookSpec, error) {
	s := lookSpec{tint: color.White}
	if v, _ := p.String(propTint); v != "" {
		c, err := tiledColor(v)
		if err != nil {
			return s, fmt.Errorf("property %s: %w", propTint, err)
		}
		s.tint = c
	}
	if v, _ := p.String(propRecolour); v != "" {
		pal, err := l.palette(v)
		if err != nil {
			return s, fmt.Errorf("property %s: %w", propRecolour, err)
		}
		s.pal = pal
	}
	if v, _ := p.String(propMask); v != "" {
		m, err := l.mask(v)
		if err != nil {
			return s, fmt.Errorf("property %s: %w", propMask, err)
		}
		s.mask = m
	}
	return s, nil
}

// maskCut is a mask: rect of tex, turned by turn over a tile turned by
// under, as a mask layer's cell and the cell it cuts each are.
type maskCut struct {
	tex         *Texture
	rect        geom.Rect
	turn, under TileFlip
}

// lookSpec is what a tile is drawn through, in this order: a palette, a
// mask, a tint.
type lookSpec struct {
	pal  *Palette
	mask maskCut
	tint color.Color
}

// plain is whether s draws a tile as it is.
func (s lookSpec) plain() bool {
	return s.pal == nil && s.mask.tex == nil && (s.tint == color.White || s.tint == color.Color{})
}

// tileLook is a tile drawn as a variant of its picture: each frame's
// picture recoloured and masked, made at load, and a tint.
type tileLook struct {
	images map[int]tileImage
	tint   color.Color
}

// lookKey is a tile drawn through a spec.
type lookKey struct {
	id   int
	spec lookSpec
}

// look is tile id drawn through s, its frames' pictures made now; nil when
// s changes nothing. A palette holding no colour of a frame's picture, or a
// mask of another size than the tile, is an error.
func (t *Tileset) look(id int, s lookSpec) (*tileLook, error) {
	if s.plain() {
		return nil, nil //nolint:nilnil // a plain tile has no look
	}
	l := &tileLook{tint: s.tint}
	if l.tint == (color.Color{}) {
		l.tint = color.White
	}
	if s.pal == nil && s.mask.tex == nil {
		return l, nil
	}
	l.images = map[int]tileImage{}
	for _, f := range t.framesOf(id) {
		tex, r := t.image(f, 0)
		if tex == nil || tex.Image == nil {
			continue
		}
		var err error
		if s.pal != nil {
			if tex, err = Recolour(tex, s.pal); err != nil {
				return nil, err
			}
		}
		if s.mask.tex != nil {
			mask, mr, turnErr := s.mask.oriented()
			if turnErr != nil {
				return nil, turnErr
			}
			if tex, err = Masked(tex, r, mask, mr); err != nil {
				return nil, err
			}
			r = geom.Rect{W: r.W, H: r.H}
		}
		l.images[f] = tileImage{tex, r}
	}
	return l, nil
}

// recolours is whether pal holds a colour of every frame's picture of
// tile id.
func (t *Tileset) recolours(id int, pal *Palette) bool {
	for _, f := range t.framesOf(id) {
		tex, _ := t.image(f, 0)
		if tex == nil || tex.Image == nil {
			return false
		}
		if _, err := Recolour(tex, pal); err != nil {
			return false
		}
	}
	return true
}

// framesOf are the tiles tile id shows: its animation's frames, or itself.
func (t *Tileset) framesOf(id int) []int {
	if a, ok := t.anims[id]; ok && a.playable() {
		frames := slices.Clone(a.frames)
		slices.Sort(frames)
		return slices.Compact(frames)
	}
	return []int{id}
}

// lookOf is tile id's own look, nil for none.
func (t *Tileset) lookOf(id int) *tileLook {
	if id >= 0 && id < len(t.looks) {
		return t.looks[id]
	}
	return nil
}

// setLook makes s tile id's own look.
func (t *Tileset) setLook(id int, s lookSpec) error {
	l, err := t.look(id, s)
	if err != nil || l == nil {
		return err
	}
	if t.specs == nil {
		t.specs = map[int]lookSpec{}
	}
	t.specs[id] = s
	t.putLook(id, l)
	return nil
}

// putLook makes l tile id's own look.
func (t *Tileset) putLook(id int, l *tileLook) {
	if id >= len(t.looks) {
		t.looks = slices.Grow(t.looks, id+1-len(t.looks))[:id+1]
	}
	t.looks[id] = l
}

// shifted is l with its frames numbered off further, as a merged tileset
// numbers them.
func (l *tileLook) shifted(off int) *tileLook {
	out := &tileLook{tint: l.tint}
	if l.images != nil {
		out.images = shifted(nil, l.images, off, same)
	}
	return out
}

// setLoad keeps what the tileset was loaded with, and makes the looks of
// the tiles their tint, recolour or mask property draws other than as
// their picture.
func (t *Tileset) setLoad(load *tiledLoad) error {
	t.load = load
	var ids []int
	for id, p := range t.props {
		if p[propTint] != nil || p[propRecolour] != nil || p[propMask] != nil {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	for _, id := range ids {
		s, err := load.specOf(t.props[id])
		if err == nil {
			err = t.setLook(id, s)
		}
		if err != nil {
			return fmt.Errorf("tile %d: %w", id-1, err)
		}
	}
	return nil
}

// drawn is the picture tile id shows at time seconds through look, nil
// for its own picture.
func (t *Tileset) drawn(id int, time float32, look *tileLook) (*Texture, geom.Rect) {
	if look != nil && look.images != nil {
		frame := id
		if a, ok := t.anims[id]; ok && a.playable() {
			frame = a.frame(time)
		}
		if img, ok := look.images[frame]; ok {
			return img.texture, img.rect
		}
	}
	return t.image(id, time)
}

// tintOf is the tint look draws with, white for none.
func tintOf(look *tileLook) color.Color {
	if look == nil {
		return color.White
	}
	return look.tint
}

// oriented is the mask as it lies over the tile's own picture: itself, or
// a picture of its own, made once, when the mask and the tile are turned
// differently.
func (m maskCut) oriented() (*Texture, geom.Rect, error) {
	if m.turn == m.under {
		return m.tex, m.rect, nil
	}
	w, h := int(m.rect.W), int(m.rect.H)
	if (m.turn|m.under)&TileFlipDiagonal != 0 && w != h {
		return nil, geom.Rect{}, fmt.Errorf("a mask of %dx%d turns across its diagonal only when square", w, h)
	}
	op := fmt.Sprintf("turn %v %d %d", m.rect, m.turn, m.under)
	tex, err := variant(variantKey{source: m.tex, op: op}, func() (*Texture, error) {
		src, err := pixels(m.tex, m.rect)
		if err != nil {
			return nil, err
		}
		out := image.NewNRGBA(image.Rect(0, 0, w, h))
		for y := range h {
			for x := range w {
				u, v := drawnAt(m.under, (float32(x)+0.5)/float32(w), (float32(y)+0.5)/float32(h))
				u, v = pictureAt(m.turn, u, v)
				out.Set(x, y, m.tex.Image.At(src.Min.X+int(u*float32(w)), src.Min.Y+int(v*float32(h))))
			}
		}
		return derived(m.tex, &Texture{Texture: render.Texture{Image: out}}), nil
	})
	return tex, geom.Rect{W: m.rect.W, H: m.rect.H}, err
}

// pictureAt is the point of a picture, 0 to 1, a cell turned by f shows at
// its point x, y, as flipped turns the corners.
func pictureAt(f TileFlip, x, y float32) (float32, float32) {
	if f&TileFlipX != 0 {
		x = 1 - x
	}
	if f&TileFlipY != 0 {
		y = 1 - y
	}
	if f&TileFlipDiagonal != 0 {
		x, y = y, x
	}
	return x, y
}

// drawnAt is where a cell turned by f shows its picture's point x, y: the
// inverse of pictureAt.
func drawnAt(f TileFlip, x, y float32) (float32, float32) {
	if f&TileFlipDiagonal != 0 {
		x, y = y, x
	}
	if f&TileFlipY != 0 {
		y = 1 - y
	}
	if f&TileFlipX != 0 {
		x = 1 - x
	}
	return x, y
}

// ownLooks gives the map a look per cell, its tile's own, so cells can
// take their own.
func (m *TileMap) ownLooks() {
	if m.looks != nil {
		return
	}
	m.looks = make([]*tileLook, len(m.cells))
	for i, id := range m.cells {
		m.looks[i] = m.tileset.lookOf(id)
	}
}

// cellLook is the look cell i, holding tile id, draws through.
func (m *TileMap) cellLook(i, id int) *tileLook {
	if m.looks != nil {
		return m.looks[i]
	}
	return m.tileset.lookOf(id)
}

// lookAt works out cell i's look: its tile's own, the layer's recolour over
// the tile's where the palette holds the tile's colours, and the cell's
// cut over the tile's mask.
func (m *TileMap) lookAt(i int) (*tileLook, error) {
	id := m.cells[i]
	if id == 0 {
		return nil, nil //nolint:nilnil // an empty cell has no look
	}
	s, own := m.tileset.specs[id]
	if !own {
		s = lookSpec{tint: color.White}
	}
	changed := false
	if m.recolour != nil {
		held, known := m.held[id]
		if !known {
			held = m.tileset.recolours(id, m.recolour)
			m.held[id] = held
		}
		if held {
			s.pal, changed = m.recolour, true
		}
	}
	if m.cuts != nil && m.cuts[i].tex != nil {
		s.mask, changed = m.cuts[i], true
		s.mask.under = m.flipAt(i)
	}
	if !changed {
		return m.tileset.lookOf(id), nil
	}
	key := lookKey{id, s}
	if l, ok := m.made[key]; ok {
		return l, nil
	}
	l, err := m.tileset.look(id, s)
	if err != nil {
		return nil, err
	}
	if m.made == nil {
		m.made = map[lookKey]*tileLook{}
	}
	m.made[key] = l
	return l, nil
}

// relook works out cell i's look again after its tile or flip changed: a
// variant that can't be made of the new tile, such as a cut of another
// size, leaves it drawn through its own look.
func (m *TileMap) relook(i int) {
	if m.looks == nil {
		return
	}
	l, err := m.lookAt(i)
	if err != nil {
		l = m.tileset.lookOf(m.cells[i])
	}
	m.looks[i] = l
}

func (m *TileMap) flipAt(i int) TileFlip {
	if m.flips == nil {
		return 0
	}
	return m.flips[i]
}

// setRecolour draws the map's tiles through pal, as a Tiled layer's
// recolour property does: a tile whose pictures pal holds no colour of
// keeps its own colours, and a map none of whose tiles it recolours is an
// error.
func (m *TileMap) setRecolour(pal *Palette) error {
	m.recolour, m.held = pal, map[int]bool{}
	m.ownLooks()
	held, filled := false, false
	for i, id := range m.cells {
		if id == 0 {
			continue
		}
		l, err := m.lookAt(i)
		if err != nil {
			return err
		}
		m.looks[i] = l
		held, filled = held || m.held[id], true
	}
	if filled && !held {
		return fmt.Errorf("palette (%s) matches no colour of the layer's tiles", pal.key)
	}
	m.syncTalls()
	return nil
}

// cut cuts cell i's tile by c, as a mask layer's cell does.
func (m *TileMap) cut(i int, c maskCut) error {
	if m.cuts == nil {
		m.cuts = make([]maskCut, len(m.cells))
	}
	m.ownLooks()
	m.cuts[i] = c
	l, err := m.lookAt(i)
	if err != nil {
		return err
	}
	m.looks[i] = l
	return nil
}

// variantOf reads layer l's tint and recolour properties: its tint
// multiplies Tiled's own.
func (t *Tiled) variantOf(l *TiledLayer) error {
	l.tinted = color.White
	if v, _ := l.Properties.String(propTint); v != "" {
		c, err := tiledColor(v)
		if err != nil {
			return fmt.Errorf("property %s: %w", propTint, err)
		}
		l.tinted, l.Tint = c, l.Tint.Mul(c)
	}
	if v, _ := l.Properties.String(propRecolour); v != "" {
		pal, err := t.load.palette(v)
		if err != nil {
			return fmt.Errorf("property %s: %w", propRecolour, err)
		}
		l.recolour = pal
	}
	return nil
}

// recolouring is the palette the layer's recolour, else its nearest
// group's, names; nil for none.
func (l *TiledLayer) recolouring() *Palette {
	for g := l; g != nil; g = g.parent {
		if g.recolour != nil {
			return g.recolour
		}
	}
	return nil
}

// foldedTinted is the layer's tint property times every group's above it.
func (l *TiledLayer) foldedTinted() color.Color {
	c := color.White
	for g := l; g != nil; g = g.parent {
		if g.tinted != (color.Color{}) {
			c = c.Mul(g.tinted)
		}
	}
	return c
}

// objectLook makes the look of tile object o of layer: its tile's own, the
// layer's recolour over the tile's where it holds the tile's colours, the
// object's recolour and mask over both, and its tint times the tile's and
// the layers'. tint, recolour and mask on any other object are an error.
func (t *Tiled) objectLook(o *TiledObject, layer *TiledLayer) error {
	own, err := t.load.specOf(o.Properties)
	if err != nil {
		return err
	}
	if o.Tile == nil || o.Tile.Tileset == nil {
		if !own.plain() {
			return fmt.Errorf("%s, %s and %s are a tile object's", propTint, propRecolour, propMask)
		}
		return nil
	}
	set, id := o.Tile.Tileset, o.Tile.ID
	s, has := set.specs[id]
	if !has {
		s = lookSpec{tint: color.White}
	}
	changed := false
	if pal := layer.recolouring(); pal != nil && set.recolours(id, pal) {
		s.pal, changed = pal, true
	}
	if own.pal != nil {
		s.pal, changed = own.pal, true
	}
	if own.mask.tex != nil {
		s.mask, changed = own.mask, true
	}
	if tint := own.tint.Mul(layer.foldedTinted()); tint != color.White {
		s.tint, changed = s.tint.Mul(tint), true
	}
	if !changed {
		return nil
	}
	o.look, err = set.look(id, s)
	return err
}

// cutMasks cuts, cell by cell, the tile layer each mask layer names by its
// cuts property.
func (t *Tiled) cutMasks(firsts []int) error {
	var err error
	t.Walk(func(l *TiledLayer) bool {
		if l.Kind == TiledMask {
			if err = t.cutBy(l, firsts); err != nil {
				err = fmt.Errorf("layer %s: %w", l.Name, err)
			}
		}
		return err == nil
	})
	return err
}

// cutBy cuts the layer mask layer l names by l's cells: each the tile of
// the mask tileset its tile's name property names, turned as the cell is.
// A mask over an empty cell cuts nothing.
func (t *Tiled) cutBy(l *TiledLayer, firsts []int) error {
	name, _ := l.Properties.String(propCuts)
	target, err := t.cutLayer(l, name)
	if err != nil {
		return err
	}
	m := target.tileMap
	for i, g := range l.masks {
		gid := g &^ tiledFlags
		if gid == 0 || i >= len(m.cells) || m.cells[i] == 0 || len(firsts) == 0 {
			continue
		}
		x, y := i%max(m.columns, 1), i/max(m.columns, 1)
		s := tilesetOf(firsts, gid)
		id := int(gid) - firsts[s] + 1
		maskName, ok := t.tilesets[s].Property(id, "name")
		if !ok {
			return fmt.Errorf("cell %d, %d: tile %d has no name to find it in the mask tileset", x, y, id-1)
		}
		c, cutErr := t.load.mask(maskName)
		if cutErr == nil {
			c.turn = flipOf(g)
			cutErr = m.cut(i, c)
		}
		if cutErr != nil {
			return fmt.Errorf("cell %d, %d: %w", x, y, cutErr)
		}
	}
	m.syncTalls()
	return nil
}

// cutLayer is the tile layer mask layer l cuts, called name: among its
// siblings, else the one layer of the map so called.
func (t *Tiled) cutLayer(l *TiledLayer, name string) (*TiledLayer, error) {
	siblings := t.layers
	if l.parent != nil {
		siblings = l.parent.layers
	}
	target, found := layerNamed(siblings, name)
	if !found {
		var named []*TiledLayer
		var tileLayers []string
		t.Walk(func(o *TiledLayer) bool {
			if o.Name == name {
				named = append(named, o)
			}
			if o.Kind == TiledTiles {
				tileLayers = append(tileLayers, o.Name)
			}
			return true
		})
		switch {
		case len(named) > 1:
			return nil, fmt.Errorf("cuts %s: %d layers are called so; put the mask layer beside the one it cuts", name,
				len(named))
		case len(named) == 0 && len(tileLayers) == 0:
			return nil, fmt.Errorf("cuts %s: the map has no tile layer", name)
		case len(named) == 0:
			return nil, fmt.Errorf("cuts %s: no layer is called so; the nearest is %s", name,
				nearest(name, slices.Values(tileLayers)))
		}
		target = named[0]
	}
	if target.Kind != TiledTiles || target.tileMap == nil {
		return nil, fmt.Errorf("cuts %s: it is not a tile layer", name)
	}
	return target, nil
}
