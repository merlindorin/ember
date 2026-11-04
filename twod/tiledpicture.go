package twod

import (
	"cmp"
	"fmt"
	"image"
	"math"
	"os"
	"slices"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/render"
)

// TiledPicture is a large picture, endless if need be, made of tiles the
// game paints on demand (docs/design/tiled-picture.md): a big map, ground
// decals, or the ambient light of a world (light.AmbientMap). Its paint
// callback fills a tile the first time it is in view, and the texels
// MarkDirty names again. Tiles are kept in atlas pages, drawn a page a
// batch, and only what is painted is uploaded. Out of view, the least
// recently seen tiles past KeepTiles give their place up and are painted
// again if they come back. Pages are only ever added, so at most
// ceil((tiles in view + tiles painted ahead + KeepTiles) / slots a page,
// 49 at 256 texels a tile) of 16 MB each. Painting is the game's: its
// data lives on the CPU, and the GPU does the drawing.
type TiledPicture struct {
	Item
	paint  func(PictureTile)
	texel  geom.Vec2
	side   int
	keep   int
	ahead  int
	filter render.Filter

	pages []*picturePage
	tiles map[[2]int]*pictureTile
	free  []slotRef
	frame uint64
}

// PictureTile is a tile of a TiledPicture to paint: Area, in Image's
// coordinates, is the texels to fill, a one-texel border included, which
// stands for the neighbouring tiles' texels so smooth sampling meets them.
type PictureTile struct {
	// X, Y is which tile, counted from the world's origin in tiles.
	X, Y  int
	Image *image.RGBA
	Area  image.Rectangle
	// World is the rectangle the tile covers, in the picture's space.
	World geom.Rect
	// origin is where the tile's texel 0, 0 lies in Image.
	origin image.Point
	side   int
}

// Cell is the world texel image pixel x, y stands for, counted from the
// world's origin.
func (t PictureTile) Cell(x, y int) (int, int) {
	return t.X*t.side + x - t.origin.X, t.Y*t.side + y - t.origin.Y
}

// TiledPictureOption sets up a TiledPicture.
type TiledPictureOption interface{ tiledPicture(*TiledPicture) }

type tiledPictureOption func(*TiledPicture)

func (o tiledPictureOption) tiledPicture(p *TiledPicture) { o(p) }
func (o Option) tiledPicture(p *TiledPicture)             { o(p) }

// TexelSize is how many points of the picture's space a texel covers,
// 1×1 by default: 32×32 makes a texel a cell of a 32-point map.
func TexelSize(w, h float32) TiledPictureOption {
	return tiledPictureOption(func(p *TiledPicture) { p.texel = geom.V2(max(w, 1e-3), max(h, 1e-3)) })
}

// TileTexels is a tile's side in texels, 256 by default.
func TileTexels(n int) TiledPictureOption {
	return tiledPictureOption(func(p *TiledPicture) { p.side = min(max(n, 8), pageSide-2) })
}

// KeepTiles is how many tiles out of view stay painted, 64 by default.
func KeepTiles(n int) TiledPictureOption {
	return tiledPictureOption(func(p *TiledPicture) { p.keep = max(n, 0) })
}

// PaintAhead is how many tiles round the view are painted a frame before
// they show, 4 by default.
func PaintAhead(n int) TiledPictureOption {
	return tiledPictureOption(func(p *TiledPicture) { p.ahead = max(n, 0) })
}

// pageSide is an atlas page's side in pixels.
const pageSide = 2048

// NewTiledPicture returns a picture whose tiles paint calls for.
func NewTiledPicture(paint func(PictureTile), opts ...TiledPictureOption) *TiledPicture {
	p := Build(&TiledPicture{Item: Base(), paint: paint, texel: geom.V2(1, 1), side: 256, keep: 64, ahead: 4,
		tiles: map[[2]int]*pictureTile{}}, "tiled-picture")
	for _, o := range opts {
		o.tiledPicture(p)
	}
	return p
}

func (p *TiledPicture) TexelSize() geom.Vec2 { return p.texel }
func (p *TiledPicture) TileTexels() int      { return p.side }
func (p *TiledPicture) KeepTiles() int       { return p.keep }
func (p *TiledPicture) PaintAhead() int      { return p.ahead }

// Pixelated reports whether the picture is sampled nearest, its texels
// sharp; it is smooth between them otherwise.
func (p *TiledPicture) Pixelated() bool { return p.filter == render.FilterNearest }

// Pages are the atlas pages the picture holds its tiles in.
func (p *TiledPicture) Pages() int { return len(p.pages) }

// Tiles is how many tiles the picture holds painted.
func (p *TiledPicture) Tiles() int { return len(p.tiles) }

type picturePage struct {
	tex *Texture
	img *image.RGBA
}

type slotRef struct{ page, slot int }

type pictureTile struct {
	at    slotRef
	dirty image.Rectangle
	seen  uint64
}

// slot is the side of a tile's place in a page: the tile and its border.
func (p *TiledPicture) slot() int    { return p.side + 2 }
func (p *TiledPicture) perRow() int  { return max(pageSide/p.slot(), 1) }
func (p *TiledPicture) perPage() int { return p.perRow() * p.perRow() }

// slotRect is where s lies in its page, border included.
func (p *TiledPicture) slotRect(s slotRef) image.Rectangle {
	n, k := p.perRow(), p.slot()
	x, y := s.slot%n*k, s.slot/n*k
	return image.Rect(x, y, x+k, y+k)
}

// whole is a tile's texels and border, in tile texels.
func (p *TiledPicture) whole() image.Rectangle { return image.Rect(-1, -1, p.side+1, p.side+1) }

// MarkDirty repaints the texels under area, in the picture's space, in
// the tiles that hold them (borders included), as soon as each is next
// drawn.
func (p *TiledPicture) MarkDirty(area geom.Rect) {
	x0, y0 := int(math.Floor(float64(area.X/p.texel.X))), int(math.Floor(float64(area.Y/p.texel.Y)))
	x1, y1 := int(math.Ceil(float64((area.X+area.W)/p.texel.X))), int(math.Ceil(float64((area.Y+area.H)/p.texel.Y)))
	texels := image.Rect(x0, y0, x1, y1)
	for key, t := range p.tiles {
		local := texels.Sub(image.Pt(key[0]*p.side, key[1]*p.side)).Intersect(p.whole())
		if !local.Empty() {
			t.dirty = t.dirty.Union(local)
		}
	}
}

// Forget repaints every tile, whole, as soon as each is next drawn.
func (p *TiledPicture) Forget() {
	for _, t := range p.tiles {
		t.dirty = p.whole()
	}
}

// tileAt is the tile of key, given a place and painted whole if it had
// none, its dirty texels painted, and seen this frame.
func (p *TiledPicture) tileAt(key [2]int) *pictureTile {
	t := p.tiles[key]
	if t == nil {
		t = &pictureTile{at: p.place(), dirty: p.whole()}
		p.tiles[key] = t
	}
	t.seen = p.frame
	if !t.dirty.Empty() {
		p.paintTile(key, t)
	}
	return t
}

// paintTile has the game paint t's dirty texels and marks them for upload.
func (p *TiledPicture) paintTile(key [2]int, t *pictureTile) {
	page := p.pages[t.at.page]
	r := p.slotRect(t.at)
	origin := r.Min.Add(image.Pt(1, 1))
	area := t.dirty.Add(origin).Intersect(r)
	t.dirty = image.Rectangle{}
	if area.Empty() {
		return
	}
	p.paint(PictureTile{X: key[0], Y: key[1], Image: page.img, Area: area, origin: origin, side: p.side,
		World: geom.Rect{X: float32(key[0]*p.side) * p.texel.X, Y: float32(key[1]*p.side) * p.texel.Y,
			W: float32(p.side) * p.texel.X, H: float32(p.side) * p.texel.Y}})
	page.tex.TouchRect(area)
}

// place is a free place for a tile: a free slot, else the slot of the
// tile seen longest ago, out of view and past the newest KeepTiles out of
// view, else one in a new page.
func (p *TiledPicture) place() slotRef {
	if n := len(p.free); n > 0 {
		s := p.free[n-1]
		p.free = p.free[:n-1]
		return s
	}
	var out [][2]int
	for key, t := range p.tiles {
		if t.seen < p.frame {
			out = append(out, key)
		}
	}
	if len(out) > p.keep {
		slices.SortFunc(out, func(a, b [2]int) int { return cmp.Compare(p.tiles[a].seen, p.tiles[b].seen) })
		key := out[0]
		s := p.tiles[key].at
		delete(p.tiles, key)
		return s
	}
	p.addPage()
	return p.place()
}

// addPage adds an atlas page of free slots, and says so past the first
// two: pages are only ever added, so a view that keeps growing shows.
func (p *TiledPicture) addPage() {
	img := image.NewRGBA(image.Rect(0, 0, pageSide, pageSide))
	tex := NewTexture(img)
	tex.Filter = p.filter
	p.pages = append(p.pages, &picturePage{tex: tex, img: img})
	page := len(p.pages) - 1
	for s := p.perPage() - 1; s >= 0; s-- {
		p.free = append(p.free, slotRef{page, s})
	}
	if len(p.pages) > 2 {
		fmt.Fprintf(
			os.Stderr,
			"twod: tiled picture %q now holds %d atlas pages of 16 MB: its view and KeepTiles need %d tiles\n",
			p.GetSceneNode().Name(),
			len(p.pages),
			len(p.tiles)+1,
		)
	}
}

// Draw draws the tiles the canvas shows, painting those it hasn't yet,
// and paints up to PaintAhead tiles round them.
func (p *TiledPicture) Draw(c *Canvas) {
	p.drawTiles(c.list, viewed(c), c.Transform())
}

// drawTiles draws the tiles of p within view, in p's space, onto list
// through t, p's space to the list's: those not painted yet are painted
// first, and up to PaintAhead tiles round them after.
func (p *TiledPicture) drawTiles(list *draw.List, view geom.Rect, t geom.Transform2D) {
	p.frame++
	span := geom.V2(float32(p.side)*p.texel.X, float32(p.side)*p.texel.Y)
	x0, y0 := int(math.Floor(float64(view.X/span.X))), int(math.Floor(float64(view.Y/span.Y)))
	x1, y1 := int(math.Floor(float64((view.X+view.W)/span.X))), int(math.Floor(float64((view.Y+view.H)/span.Y)))
	if (x1-x0+1)*(y1-y0+1) > 4096 {
		return
	}
	saved := list.Transform()
	list.SetTransform(t)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			tile := p.tileAt([2]int{x, y})
			page := p.pages[tile.at.page]
			r := p.slotRect(tile.at)
			src := geom.Rect{X: float32(r.Min.X + 1), Y: float32(r.Min.Y + 1), W: float32(p.side), H: float32(p.side)}
			dst := geom.Rect{X: float32(x) * span.X, Y: float32(y) * span.Y, W: span.X, H: span.Y}
			list.Image(dst, &page.tex.Texture, page.tex.uv(src), color.White)
		}
	}
	list.SetTransform(saved)
	ahead := p.ahead
	for y := y0 - 1; y <= y1+1 && ahead > 0; y++ {
		for x := x0 - 1; x <= x1+1 && ahead > 0; x++ {
			if x >= x0 && x <= x1 && y >= y0 && y <= y1 {
				continue
			}
			if t, ok := p.tiles[[2]int{x, y}]; ok && t.dirty.Empty() {
				continue
			}
			p.tileAt([2]int{x, y})
			ahead--
		}
	}
}
