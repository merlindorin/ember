package twod

import (
	"strconv"
	"strings"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/world"
)

//nolint:gochecknoinits // registers the package's node types on import
func init() {
	node.Register(node.Type{New: func() node.Noded { return NewTileMap(nil) }})
}

// TileMap is a grid of tiles from a Tileset: a level's floor and walls. It
// draws only the cells in view, its solid tiles stop collide bodies (it is
// a collide.Solid), and its tall tiles stand as sprites at their feet to
// sort with characters.
type TileMap struct {
	Item
	tileset   *Tileset  `node:"prop"`
	columns   int       `node:"prop"`
	cells     []int     `node:"prop"`
	tileSize  geom.Vec2 `node:"prop"`
	isometric bool      `node:"prop"`
	// offset moves every cell within the map, so placing the map keeps it.
	offset  geom.Vec2 `node:"prop"`
	tallIn  Entity
	talls   map[int]*Sprite
	version uint64
	// flips is each cell's flip, nil while none is flipped.
	flips []TileFlip
}

// TileFlip is how a map cell turns its tile: mirrored across, upside down,
// or across its diagonal (a transpose); together they make the eight ways
// a square tile can face. Tiled maps keep them, LoadTiled reads them.
type TileFlip uint8

const (
	TileFlipX TileFlip = 1 << iota
	TileFlipY
	TileFlipDiagonal
)

// Flip is how cell x, y turns its tile.
func (m *TileMap) Flip(x, y int) TileFlip {
	i, ok := m.index(x, y)
	if !ok || m.flips == nil {
		return 0
	}
	return m.flips[i]
}

// SetFlip turns cell x, y's tile by f; a Tall tile takes the mirroring
// across and upside down, not the diagonal.
func (m *TileMap) SetFlip(x, y int, f TileFlip) {
	i, ok := m.index(x, y)
	if !ok {
		return
	}
	if m.flips == nil {
		if f == 0 {
			return
		}
		m.flips = make([]TileFlip, len(m.cells))
	}
	m.flips[i] = f
	m.version++
	m.syncTalls()
}

func (m *TileMap) index(x, y int) (int, bool) {
	if x < 0 || y < 0 || x >= m.Columns() || y >= m.Rows() {
		return 0, false
	}
	return y*m.columns + x, true
}

// flipped is the texture points of the corners of a cell showing uv turned
// by f: top-left, top-right, bottom-right, bottom-left, as Tiled turns
// tiles, the diagonal first, then across, then upside down.
func flipped(uv geom.Rect, f TileFlip) [4]geom.Vec2 {
	var out [4]geom.Vec2
	for i, c := range [4][2]float32{{0, 0}, {1, 0}, {1, 1}, {0, 1}} {
		x, y := c[0], c[1]
		if f&TileFlipX != 0 {
			x = 1 - x
		}
		if f&TileFlipY != 0 {
			y = 1 - y
		}
		if f&TileFlipDiagonal != 0 {
			x, y = y, x
		}
		out[i] = geom.V2(uv.X+x*uv.W, uv.Y+y*uv.H)
	}
	return out
}

// TileMapOption sets up a TileMap.
type TileMapOption interface{ tileMap(*TileMap) }

type tileMapOption func(*TileMap)

func (o tileMapOption) tileMap(m *TileMap) { o(m) }
func (o Option) tileMap(m *TileMap)        { o(m) }

// NewTileMap returns a map of tileset's tiles, empty until Cells, CSV or
// ASCII fill it.
func NewTileMap(tileset *Tileset, opts ...TileMapOption) *TileMap {
	m := Build(&TileMap{Item: Base(), tileset: tileset, talls: map[int]*Sprite{}}, "tile-map")
	if tileset != nil {
		m.tileSize = geom.V2(float32(tileset.tileW), float32(tileset.tileH))
	}
	for _, o := range opts {
		o.tileMap(m)
	}
	m.syncTalls()
	return m
}

// Cells fills the map from a grid of tile ids, row by row.
func Cells(grid [][]int) TileMapOption {
	return tileMapOption(func(m *TileMap) {
		m.columns, m.cells, m.flips = 0, nil, nil
		for _, row := range grid {
			m.columns = max(m.columns, len(row))
		}
		for _, row := range grid {
			m.cells = append(m.cells, row...)
			m.cells = append(m.cells, make([]int, m.columns-len(row))...)
		}
	})
}

// CSV fills the map from rows of comma-separated tile ids.
func CSV(text string) TileMapOption {
	var grid [][]int
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		var row []int
		for _, field := range strings.Split(line, ",") {
			id, _ := strconv.Atoi(strings.TrimSpace(field))
			row = append(row, id)
		}
		grid = append(grid, row)
	}
	return Cells(grid)
}

// ASCII fills the map from a picture of it, one character per cell, legend
// giving each character's tile id; other characters are empty.
func ASCII(text string, legend map[rune]int) TileMapOption {
	var grid [][]int
	for _, line := range strings.Split(strings.Trim(text, "\n"), "\n") {
		var row []int
		for _, r := range line {
			row = append(row, legend[r])
		}
		grid = append(grid, row)
	}
	return Cells(grid)
}

// CellSize is a cell's size on screen in points (default the tile's size
// in pixels).
func CellSize(w, h float32) MapOption {
	return mapOption{
		tile:     func(m *TileMap) { m.tileSize = geom.V2(w, h) },
		infinite: func(m *InfiniteMap) { m.tileSize = geom.V2(w, h) },
	}
}

// CellOffset moves every cell of the map by x, y points within it: its
// pictures, its tall tiles, its solids and shadows, and Center and Cell.
// The map's position adds to it, so placing the map keeps the offset, as
// LoadTiled does for a Tiled layer's.
func CellOffset(x, y float32) TileMapOption {
	return tileMapOption(func(m *TileMap) { m.offset = geom.V2(x, y) })
}

// CellOffset is how far every cell stands from where the map's position
// puts it.
func (m *TileMap) CellOffset() geom.Vec2 { return m.offset }

// SetPosition places the map, its tall tiles moving with it wherever
// TallIn stands them.
func (m *TileMap) SetPosition(at geom.Vec2) {
	delta := at.Sub(m.position)
	m.Placement.SetPosition(at)
	m.moveTalls(delta)
}

// Translate moves the map by delta, its tall tiles with it wherever TallIn
// stands them.
func (m *TileMap) Translate(delta geom.Vec2) {
	m.Placement.Translate(delta)
	m.moveTalls(delta)
}

// moveTalls moves the tall tiles standing outside the map by delta; those
// inside it follow it already.
func (m *TileMap) moveTalls(delta geom.Vec2) {
	if m.tallIn == nil || delta == (geom.Vec2{}) {
		return
	}
	for _, s := range m.talls {
		s.Translate(delta)
	}
}

// Isometric lays the cells out as diamonds, drawn back to front.
func Isometric() TileMapOption { return tileMapOption(func(m *TileMap) { m.isometric = true }) }

// TallIn stands the map's tall tiles in parent rather than in the map, such
// as a SortByY layer holding the characters they sort with.
func TallIn(parent Entity) MapOption {
	return mapOption{
		tile:     func(m *TileMap) { m.tallIn = parent },
		infinite: func(m *InfiniteMap) { m.tallIn = parent },
	}
}

// Columns is how many cells wide the map is.
func (m *TileMap) Columns() int { return m.columns }

// Rows is how many cells tall the map is.
func (m *TileMap) Rows() int {
	if m.columns == 0 {
		return 0
	}
	return len(m.cells) / m.columns
}

// At is the tile id in cell x, y, 0 outside the map.
func (m *TileMap) At(x, y int) int {
	columns, rows := m.Columns(), m.Rows()
	if x < 0 || y < 0 || x >= columns || y >= rows {
		return 0
	}
	return m.cells[y*m.columns+x]
}

// Set puts tile id in cell x, y; cells outside the map are ignored.
func (m *TileMap) Set(x, y, id int) {
	columns, rows := m.Columns(), m.Rows()
	if x < 0 || y < 0 || x >= columns || y >= rows {
		return
	}
	m.cells[y*m.columns+x] = id
	m.version++
	m.syncTalls()
}

// Picture is what cell x, y shows, in the map's parent's space, trimmed
// to its tile's Bounds: a Tall tile standing above its cell as the map
// stands it, any other the part of the cell its pixels cover. ok is false
// for an empty cell, outside the map, or on an isometric map. An arrow
// that hits only what is drawn asks it.
func (m *TileMap) Picture(x, y int) (geom.Rect, bool) {
	id := m.At(x, y)
	if id == 0 || m.tileset == nil || m.isometric {
		return geom.Rect{}, false
	}
	return m.tileset.picture(id, m.position.Add(m.origin(x, y)), m.tileSize)
}

// Tileset is the tileset the map draws its tiles from.
func (m *TileMap) Tileset() *Tileset { return m.tileset }

// Version changes whenever a cell is Set, so a path planned over the map
// knows to plan again.
func (m *TileMap) Version() uint64 { return m.version }

// Origin is the top-left corner of cell x, y in the map's own space, the
// top corner of its diamond when isometric, moved by the cells' offset.
func (m *TileMap) origin(x, y int) geom.Vec2 {
	if m.isometric {
		return m.offset.Add(geom.V2(float32(x-y)*m.tileSize.X/2, float32(x+y)*m.tileSize.Y/2))
	}
	return m.offset.Add(geom.V2(float32(x)*m.tileSize.X, float32(y)*m.tileSize.Y))
}

// Center is the middle of cell x, y in the map's parent's space.
func (m *TileMap) Center(x, y int) geom.Vec2 {
	o := m.origin(x, y)
	if m.isometric {
		return m.position.Add(o).Add(geom.V2(0, m.tileSize.Y/2))
	}
	return m.position.Add(o).Add(m.tileSize.Scale(0.5))
}

// Cell is the cell under point p, in the map's parent's space.
func (m *TileMap) Cell(p geom.Vec2) (int, int, bool) {
	var x int
	var y int
	p = p.Sub(m.position).Sub(m.offset)
	var fx, fy float32
	if m.isometric {
		u, v := p.X/(m.tileSize.X/2), p.Y/(m.tileSize.Y/2)
		fx, fy = (u+v)/2, (v-u)/2
	} else {
		fx, fy = p.X/m.tileSize.X, p.Y/m.tileSize.Y
	}
	x, y = int(geom.Floor(fx)), int(geom.Floor(fy))
	columns, rows := m.Columns(), m.Rows()
	return x, y, x >= 0 && y >= 0 && x < columns && y < rows
}

// span is the range of cells area, in the map's space, covers, clipped to
// the map.
func (m *TileMap) span(area geom.Rect) (int, int, int, int) {
	area.X, area.Y = area.X-m.offset.X, area.Y-m.offset.Y
	var x0, y0, x1, y1 int
	columns, rows := m.Columns(), m.Rows()
	x0, y0 = int(geom.Floor(area.X/m.tileSize.X)), int(geom.Floor(area.Y/m.tileSize.Y))
	x1, y1 = int(geom.Floor((area.X+area.W)/m.tileSize.X)), int(geom.Floor((area.Y+area.H)/m.tileSize.Y))
	return max(x0, 0), max(y0, 0), min(x1, columns-1), min(y1, rows-1)
}

// TallSprites are the sprites standing for the map's tall tiles.
func (m *TileMap) TallSprites() []*Sprite {
	var out []*Sprite
	for i := range m.cells {
		if s, ok := m.talls[i]; ok {
			out = append(out, s)
		}
	}
	return out
}

// syncTalls stands a sprite on every tall cell and removes the others. A
// tall tile keeps its shape, scaled to the cell's width, so one taller
// than wide rises above its cell.
func (m *TileMap) syncTalls() {
	if m.tileset == nil {
		return
	}
	for i, id := range m.cells {
		s, has := m.talls[i]
		switch {
		case m.tileset.Tall(id) && !has:
			s = m.tallSprite(i, id)
			m.talls[i] = s
			parent := Entity(m)
			if m.tallIn != nil {
				parent = m.tallIn
			}
			add(parent, []world.BasicFace{s})
		case !m.tileset.Tall(id) && has:
			drop(s)
			delete(m.talls, i)
		case has:
			tex, r := m.tileset.image(id, 0)
			s.SetTexture(tex)
			s.SetRegion(r)
			f := m.Flip(i%m.columns, i/m.columns)
			s.SetFlip(f&TileFlipX != 0, f&TileFlipY != 0)
		}
	}
}

func (m *TileMap) tallSprite(i, id int) *Sprite {
	x, y := i%m.columns, i/m.columns
	o, cell := m.cellCorner(x, y), m.origin(x, y)
	if m.tallIn != nil {
		o, cell = o.Add(m.position), cell.Add(m.position)
	}
	s := m.tileset.sprite(id, o, m.tileSize, m.Flip(x, y))
	if m.tallIn != nil {
		s.SetHidden(m.hidden)
		s.SetTint(m.tint)
	}
	litAtBase(
		s,
		m.light.Mode,
		m.tileset.scale(m.tileSize),
		m.tileset.blocksOf(nil, id, cell, m.tileSize, m.Flip(x, y)),
	)
	return s
}

// cellCorner is the top-left corner of cell x, y's square in the map's
// space: an isometric cell's diamond is centred on its origin.
func (m *TileMap) cellCorner(x, y int) geom.Vec2 {
	o := m.origin(x, y)
	if m.isometric {
		o = o.Sub(geom.V2(m.tileSize.X/2, 0))
	}
	return o
}

// SetHidden hides the map, its tall tiles included wherever TallIn stands
// them. A hidden map is only not drawn: it still collides and casts its
// shadows.
func (m *TileMap) SetHidden(hidden bool) {
	m.Look.SetHidden(hidden)
	if m.tallIn != nil {
		for _, s := range m.talls {
			s.SetHidden(hidden)
		}
	}
}

// SetTint tints the map, its tall tiles included wherever TallIn stands
// them.
func (m *TileMap) SetTint(c color.Color) {
	m.Look.SetTint(c)
	if m.tallIn != nil {
		for _, s := range m.talls {
			s.SetTint(c)
		}
	}
}

// Draw draws the cells in view, the tall ones aside.
func (m *TileMap) Draw(c *Canvas) {
	if m.tileset == nil {
		return
	}
	view := m.tileset.overhang(viewed(c), m.tileSize)
	time := c.Time()
	cell := func(x, y int) {
		id := m.At(x, y)
		if id == 0 || m.tileset.Tall(id) {
			return
		}
		m.tileset.drawLit(c, id, m.cellCorner(x, y), m.origin(x, y), m.tileSize, time, m.Flip(x, y), color.White)
	}
	columns, rows := m.Columns(), m.Rows()
	if m.isometric {
		for d := 0; d < columns+rows-1; d++ {
			for x := max(0, d-rows+1); x <= min(d, columns-1); x++ {
				cell(x, d-x)
			}
		}
		return
	}
	x0, y0, x1, y1 := m.span(view)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			cell(x, y)
		}
	}
}
