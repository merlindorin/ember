package twod

import (
	"math"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/world"
)

// InfiniteMap is a tile map without edges: chunks of cells a generator
// fills the first time anything reads them, the least recently used ones
// forgotten past KeepChunks. It draws only the cells in view, its solid
// tiles stop collide bodies (it is a collide.Solid), and its tall tiles
// stand as sprites at their feet, as TileMap's do.
type InfiniteMap struct {
	Item
	tileset  *Tileset
	tileSize geom.Vec2
	// roomScratch holds the shapes round a cell Room looks at.
	roomScratch []geom.Rect
	tallIn      Entity
	chunkSize   int
	keep        int
	generate    func(cx, cy int, cells [][]int)
	tint        func(x, y int) color.Color
	chunks      map[chunkKey]*chunk
	clock       uint64
	// keepFrom protects the chunks used since this frame's views were
	// prepared from being forgotten; none before the first frame.
	keepFrom uint64
	version  uint64
}

type chunkKey struct{ x, y int }

type chunk struct {
	cells [][]int
	// tints are its cells' CellTint, row by row; none without one.
	tints  []color.Color
	used   uint64
	edited bool
	talls  []*Sprite
}

// InfiniteMapOption sets up an InfiniteMap.
type InfiniteMapOption interface{ infiniteMap(*InfiniteMap) }

// MapOption sets up a TileMap or an InfiniteMap.
type MapOption interface {
	TileMapOption
	InfiniteMapOption
}

type infiniteMapOption func(*InfiniteMap)

func (o infiniteMapOption) infiniteMap(m *InfiniteMap) { o(m) }
func (o Option) infiniteMap(m *InfiniteMap)            { o(m) }

type mapOption struct {
	tile     func(*TileMap)
	infinite func(*InfiniteMap)
}

func (o mapOption) tileMap(m *TileMap)         { o.tile(m) }
func (o mapOption) infiniteMap(m *InfiniteMap) { o.infinite(m) }

// NewInfiniteMap returns an endless map of tileset's tiles, its chunks
// filled by the Generator, empty without one.
func NewInfiniteMap(tileset *Tileset, opts ...InfiniteMapOption) *InfiniteMap {
	m := Build(
		&InfiniteMap{
			Item:      Base(),
			tileset:   tileset,
			chunkSize: 16,
			keep:      64,
			chunks:    map[chunkKey]*chunk{},
			keepFrom:  math.MaxUint64,
		},
		"infinite-map",
	)
	if tileset != nil {
		m.tileSize = geom.V2(float32(tileset.tileW), float32(tileset.tileH))
	}
	for _, o := range opts {
		o.infiniteMap(m)
	}
	return m
}

// ChunkSize is how many cells square a chunk is, 16 by default.
func ChunkSize(n int) InfiniteMapOption {
	return infiniteMapOption(func(m *InfiniteMap) { m.chunkSize = max(n, 1) })
}

// Generator fills a chunk: cells[y][x] is map cell cx*size+x, cy*size+y.
// It must give a chunk the same cells every time, since a forgotten chunk
// is generated again.
func Generator(f func(cx, cy int, cells [][]int)) InfiniteMapOption {
	return infiniteMapOption(func(m *InfiniteMap) { m.generate = f })
}

// CellTint colours each cell's tile by f, white leaving it as it is,
// worked out once for a cell when its chunk is made, after the Generator:
// the floor of a region in its own shade. Tall tiles stand untinted.
func CellTint(f func(x, y int) color.Color) InfiniteMapOption {
	return infiniteMapOption(func(m *InfiniteMap) { m.tint = f })
}

// KeepChunks is how many chunks the map holds before forgetting the least
// recently used, 64 by default.
func KeepChunks(n int) InfiniteMapOption {
	return infiniteMapOption(func(m *InfiniteMap) { m.keep = max(n, 1) })
}

func (m *InfiniteMap) ChunkSize() int                             { return m.chunkSize }
func (m *InfiniteMap) Generator() func(cx, cy int, cells [][]int) { return m.generate }
func (m *InfiniteMap) KeepChunks() int                            { return m.keep }
func (m *InfiniteMap) CellTint() func(x, y int) color.Color       { return m.tint }
func (m *InfiniteMap) CellSize() geom.Vec2                        { return m.tileSize }

// Chunks is how many chunks the map holds.
func (m *InfiniteMap) Chunks() int { return len(m.chunks) }

// Tileset is the tileset the map draws its tiles from.
func (m *InfiniteMap) Tileset() *Tileset { return m.tileset }

// Version changes whenever a cell is Set, or the map Reset.
func (m *InfiniteMap) Version() uint64 { return m.version }

func (m *InfiniteMap) Add(children ...world.BasicFace) *InfiniteMap {
	add(&m.Item, children)
	return m
}

// split is the chunk holding cell x, y and the cell's place in it.
func (m *InfiniteMap) split(x, y int) (chunkKey, int, int) {
	n := m.chunkSize
	cx, cy := floorDiv(x, n), floorDiv(y, n)
	return chunkKey{cx, cy}, x - cx*n, y - cy*n
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && a < 0 {
		q--
	}
	return q
}

// chunk is the chunk at k, generated if the map doesn't hold it.
func (m *InfiniteMap) chunk(k chunkKey) *chunk {
	m.clock++
	if c, ok := m.chunks[k]; ok {
		c.used = m.clock
		return c
	}
	c := &chunk{cells: make([][]int, m.chunkSize), used: m.clock}
	flat := make([]int, m.chunkSize*m.chunkSize)
	for y := range c.cells {
		c.cells[y] = flat[y*m.chunkSize : (y+1)*m.chunkSize]
	}
	if m.generate != nil {
		m.generate(k.x, k.y, c.cells)
	}
	if m.tint != nil {
		c.tints = make([]color.Color, len(flat))
		for i := range flat {
			c.tints[i] = m.tint(k.x*m.chunkSize+i%m.chunkSize, k.y*m.chunkSize+i/m.chunkSize)
		}
	}
	m.chunks[k] = c
	m.standTalls(k, c)
	m.forget()
	return c
}

// forget drops the least recently used chunks past KeepChunks, never an
// edited one or one the current draw needs.
func (m *InfiniteMap) forget() {
	for len(m.chunks) > m.keep {
		var oldest chunkKey
		var found *chunk
		for k, c := range m.chunks {
			if c.edited || c.used >= m.keepFrom || c.used == m.clock {
				continue
			}
			if found == nil || c.used < found.used {
				oldest, found = k, c
			}
		}
		if found == nil {
			return
		}
		for _, s := range found.talls {
			drop(s)
		}
		delete(m.chunks, oldest)
	}
}

// standTalls stands a sprite on each of the chunk's tall cells.
func (m *InfiniteMap) standTalls(k chunkKey, c *chunk) {
	for _, s := range c.talls {
		drop(s)
	}
	c.talls = c.talls[:0]
	if m.tileset == nil {
		return
	}
	parent := Entity(m)
	if m.tallIn != nil {
		parent = m.tallIn
	}
	for y, row := range c.cells {
		for x, id := range row {
			if !m.tileset.Tall(id) {
				continue
			}
			o := m.origin(k.x*m.chunkSize+x, k.y*m.chunkSize+y)
			if m.tallIn != nil {
				o = o.Add(m.position)
			}
			s := m.tileset.sprite(id, o, m.tileSize, 0)
			litAtBase(s, m.light.Mode, m.tileset.scale(m.tileSize), m.tileset.shadowsOf(nil, id, o, m.tileSize, 0))
			c.talls = append(c.talls, s)
			add(parent, []world.BasicFace{s})
		}
	}
}

// drop takes a tall tile's sprite out of its parent and out of the World.
func drop(s *Sprite) {
	if p := s.GetSceneNode().ParentNode(); p != nil {
		p.RemoveNode(s)
	}
	if p := s.Parent(); p != nil {
		p.RemoveChild(s.BasicEntity)
	}
	if w := world.Of(s); w != nil {
		w.RemoveEntity(s)
	}
}

// At is the tile id in cell x, y.
func (m *InfiniteMap) At(x, y int) int {
	k, lx, ly := m.split(x, y)
	return m.chunk(k).cells[ly][lx]
}

// Set puts tile id in cell x, y; its chunk is kept from then on.
func (m *InfiniteMap) Set(x, y, id int) {
	k, lx, ly := m.split(x, y)
	c := m.chunk(k)
	c.cells[ly][lx], c.edited = id, true
	m.version++
	m.standTalls(k, c)
}

// Reset forgets every chunk, edited ones too, and their tall tiles' sprites:
// the next draw or query generates them again, from the Generator and
// CellTint as they now read. A game whose generator's inputs changed, such
// as its regions retuned, calls it to see them. Version changes, as for a
// Set, so routes and flow fields over the map plan again.
func (m *InfiniteMap) Reset() {
	for _, c := range m.chunks {
		for _, s := range c.talls {
			drop(s)
		}
	}
	m.chunks = map[chunkKey]*chunk{}
	m.version++
}

// Picture is what cell x, y shows, in the map's parent's space, trimmed
// to its tile's Bounds: a Tall tile standing above its cell as the map
// stands it, any other the part of the cell its pixels cover. ok is false
// for an empty cell. The cell's chunk is made if it wasn't, as for
// collision.
func (m *InfiniteMap) Picture(x, y int) (geom.Rect, bool) {
	id := m.At(x, y)
	if id == 0 || m.tileset == nil {
		return geom.Rect{}, false
	}
	return m.tileset.picture(id, m.position.Add(m.origin(x, y)), m.tileSize)
}

// Solid reports whether cell x, y holds a solid tile, which blocks it for
// paths whatever its Collision shapes.
func (m *InfiniteMap) Solid(x, y int) bool {
	id := m.At(x, y)
	return id != 0 && m.tileset != nil && m.tileset.Solid(id)
}

func (m *InfiniteMap) origin(x, y int) geom.Vec2 {
	return geom.V2(float32(x)*m.tileSize.X, float32(y)*m.tileSize.Y)
}

// Center is the middle of cell x, y in the map's parent's space.
func (m *InfiniteMap) Center(x, y int) geom.Vec2 {
	return m.position.Add(m.origin(x, y)).Add(m.tileSize.Scale(0.5))
}

// Cell is the cell under point p, in the map's parent's space.
func (m *InfiniteMap) Cell(p geom.Vec2) (int, int) {
	p = p.Sub(m.position)
	return int(geom.Floor(p.X / m.tileSize.X)), int(geom.Floor(p.Y / m.tileSize.Y))
}

// span is the range of cells area, in the map's space, covers.
func (m *InfiniteMap) span(area geom.Rect) (int, int, int, int) {
	var x0, y0, x1, y1 int
	x0, y0 = int(geom.Floor(area.X/m.tileSize.X)), int(geom.Floor(area.Y/m.tileSize.Y))
	x1, y1 = int(geom.Floor((area.X+area.W)/m.tileSize.X)), int(geom.Floor((area.Y+area.H)/m.tileSize.Y))
	return x0, y0, x1, y1
}

// SolidIn lists the rectangles its tiles stop bodies with, whole cells or
// their Collision shapes, that area, in the map's parent's space, touches,
// generating the chunks it covers: the map is a collide.Solid.
func (m *InfiniteMap) SolidIn(area geom.Rect) []geom.Rect {
	if m.tileset == nil {
		return nil
	}
	local := geom.Rect{X: area.X - m.position.X, Y: area.Y - m.position.Y, W: area.W, H: area.H}
	x0, y0, x1, y1 := m.span(m.tileset.overhang(local, m.tileSize))
	var out []geom.Rect
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if id := m.At(x, y); id != 0 {
				out = m.tileset.solids(out, id, m.position.Add(m.origin(x, y)), m.tileSize, 0)
			}
		}
	}
	return touching(out, area)
}

// ShadowsIn are the rectangles the map's tiles block light with within
// area, as TileMap.ShadowsIn's.
func (m *InfiniteMap) ShadowsIn(area geom.Rect) []ShadowRect {
	if m.tileset == nil {
		return nil
	}
	local := geom.Rect{X: area.X - m.position.X, Y: area.Y - m.position.Y, W: area.W, H: area.H}
	x0, y0, x1, y1 := m.span(m.tileset.overhang(local, m.tileSize))
	var out []ShadowRect
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if id := m.At(x, y); id != 0 {
				out = m.tileset.shadowsOf(out, id, m.position.Add(m.origin(x, y)), m.tileSize, 0)
			}
		}
	}
	return out
}

// TallSprites are the sprites standing for the tall tiles of the chunks
// the map holds.
func (m *InfiniteMap) TallSprites() []*Sprite {
	var out []*Sprite
	for _, c := range m.chunks {
		out = append(out, c.talls...)
	}
	return out
}

// prepare makes the chunks the views need, rectangles of the map's own
// space, and the ring of chunks around each, before anything is drawn: the
// twod system gives it what every camera of the map's window shows, drawing
// the map or not. Until the next frame's, these chunks are not forgotten.
func (m *InfiniteMap) prepare(views []geom.Rect) {
	if m.tileSize.X <= 0 || m.tileSize.Y <= 0 {
		return
	}
	m.keepFrom = m.clock + 1
	for _, view := range views {
		x0, y0, x1, y1 := m.span(view)
		k0, _, _ := m.split(x0, y0)
		k1, _, _ := m.split(x1, y1)
		for cy := k0.y - 1; cy <= k1.y+1; cy++ {
			for cx := k0.x - 1; cx <= k1.x+1; cx++ {
				m.chunk(chunkKey{cx, cy})
			}
		}
	}
	m.forget()
}

// Draw draws the cells in view; prepare has made their chunks.
func (m *InfiniteMap) Draw(c *Canvas) {
	if m.tileset == nil || m.tileSize.X <= 0 || m.tileSize.Y <= 0 {
		return
	}
	view := m.tileset.overhang(viewed(c), m.tileSize)
	x0, y0, x1, y1 := m.span(view)
	time := c.Time()
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			k, lx, ly := m.split(x, y)
			ch := m.chunk(k)
			id := ch.cells[ly][lx]
			if id == 0 || m.tileset.Tall(id) {
				continue
			}
			tint := color.White
			if ch.tints != nil {
				tint = ch.tints[ly*m.chunkSize+lx]
			}
			m.tileset.drawLit(c, id, m.origin(x, y), m.origin(x, y), m.tileSize, time, 0, tint)
		}
	}
}

// Area is a window of the map's cells, columns by rows from cell x, y: a
// finite grid, for nav.
func (m *InfiniteMap) Area(x, y, columns, rows int) *TileArea {
	return &TileArea{m: m, x: x, y: y, columns: columns, rows: rows}
}

// AreaAround is the Area of columns by rows cells centred on the cell
// under at: what a nav.FlowField follows the hero with, through
// nav.Around(m.AreaAround, columns, rows).
func (m *InfiniteMap) AreaAround(at geom.Vec2, columns, rows int) *TileArea {
	x, y := m.Cell(at)
	return m.Area(x-columns/2, y-rows/2, columns, rows)
}

// TileArea is a window of an InfiniteMap's cells, numbered from its own
// top-left: a finite grid nav finds paths over.
type TileArea struct {
	m                   *InfiniteMap
	x, y, columns, rows int
	// ids are the area's tiles and a border round it, as of idsVersion.
	ids        []int
	idsVersion uint64
	idsFilled  bool
}

func (a *TileArea) Columns() int    { return a.columns }
func (a *TileArea) Rows() int       { return a.rows }
func (a *TileArea) Version() uint64 { return a.m.Version() }

// Origin is the map cell of the area's cell 0, 0.
func (a *TileArea) Origin() (int, int) { return a.x, a.y }

// Solid reports whether the area's cell x, y is solid; outside the area
// counts as solid.
func (a *TileArea) Solid(x, y int) bool {
	if x < 0 || y < 0 || x >= a.columns || y >= a.rows {
		return true
	}
	return a.m.Solid(a.x+x, a.y+y)
}

// FillSolid sets dst, columns by rows row by row, to which of the area's
// cells are solid, reading each chunk once.
func (a *TileArea) FillSolid(dst []bool) {
	m := a.m
	if m.tileset == nil {
		clear(dst)
		return
	}
	for y := range a.rows {
		row := dst[y*a.columns : (y+1)*a.columns]
		for x := 0; x < a.columns; {
			k, lx, ly := m.split(a.x+x, a.y+y)
			cells := m.chunk(k).cells[ly]
			for ; lx < m.chunkSize && x < a.columns; lx, x = lx+1, x+1 {
				id := cells[lx]
				row[x] = id != 0 && m.tileset.Solid(id)
			}
		}
	}
}

// Cell is the area's cell under p.
func (a *TileArea) Cell(p geom.Vec2) (int, int, bool) {
	var x int
	var y int
	mx, my := a.m.Cell(p)
	x, y = mx-a.x, my-a.y
	return x, y, x >= 0 && y >= 0 && x < a.columns && y < a.rows
}

// Center is the middle of the area's cell x, y.
func (a *TileArea) Center(x, y int) geom.Vec2 { return a.m.Center(a.x+x, a.y+y) }
