package twod

import (
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"sync"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/node"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

//nolint:gochecknoinits // registers the package's node types on import
func init() {
	node.Register(node.Type{New: func() node.Noded { return NewBitmapText("") }})
}

// BitmapFont is a font of fixed cells, each character a picture of lit
// pixels: retro HUDs and pixel art. One font pixel is one point.
type BitmapFont struct {
	cell   geom.Vec2
	mu     sync.Mutex
	glyphs map[rune]*bitmapGlyph
	glyph  func(r rune) *bitmapGlyph
}

// bitmapGlyph stores a glyph as horizontal runs of lit pixels.
type bitmapGlyph struct {
	runs []geom.Rect
}

//nolint:gochecknoglobals // the engine's built-in bitmap font, one value every text shares
var defaultBitmapFont = &BitmapFont{cell: geom.V2(7, 13), glyphs: map[rune]*bitmapGlyph{}, glyph: basicGlyph}

// DefaultBitmapFont is the 7×13 font of x/image.
func DefaultBitmapFont() *BitmapFont { return defaultBitmapFont }

func basicGlyph(r rune) *bitmapGlyph {
	face := basicfont.Face7x13
	dr, mask, mp, _, ok := face.Glyph(fixed.P(0, face.Ascent), r)
	if !ok {
		dr, mask, mp, _, _ = face.Glyph(fixed.P(0, face.Ascent), '?')
	}
	return runsOf(
		mask,
		image.Rect(mp.X, mp.Y, mp.X+dr.Dx(), mp.Y+dr.Dy()),
		geom.V2(float32(dr.Min.X), float32(dr.Min.Y)),
	)
}

// BitmapFontOption describes the picture of a bitmap font: its Cell, and
// the Charset in its cells.
type BitmapFontOption interface{ bitmapFont(*bitmapFontSpec) }

type bitmapFontSpec struct {
	cellW, cellH int
	charset      string
}

// Grid is the size in pixels of the cells of a picture cut into a grid:
// a sprite sheet, a tileset, a bitmap font. Cell makes one.
type Grid struct{ w, h int }

// Cell is a grid of w×h pixel cells.
func Cell(w, h int) Grid { return Grid{w, h} }

// Size is a cell's width and height in pixels.
func (g Grid) Size() (int, int) { return g.w, g.h }

func (g Grid) bitmapFont(s *bitmapFontSpec) { s.cellW, s.cellH = g.w, g.h }

type charset string

func (c charset) bitmapFont(s *bitmapFontSpec) { s.charset = string(c) }

// Charset is the characters of the picture's cells, in order, left to
// right then top to bottom (default the printable ASCII characters, from
// the space to ~).
func Charset(chars string) BitmapFontOption { return charset(chars) }

// LoadBitmapFont reads the PNG name in fsys, a grid of character cells
// described by Cell and Charset; a pixel is lit when it is more than half
// opaque.
func LoadBitmapFont(fsys fs.FS, name string, opts ...BitmapFontOption) (*BitmapFont, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, fmt.Errorf("twod: %w", err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("twod: %s: %w", name, err)
	}
	font, err := NewBitmapFont(img, opts...)
	if err != nil {
		return nil, fmt.Errorf("twod: %s: %w", name, err)
	}
	return font, nil
}

// NewBitmapFont cuts img, a grid of character cells described by Cell and
// Charset, into a bitmap font.
func NewBitmapFont(img image.Image, opts ...BitmapFontOption) (*BitmapFont, error) {
	spec := bitmapFontSpec{charset: printableASCII}
	for _, o := range opts {
		o.bitmapFont(&spec)
	}
	cellW, cellH := spec.cellW, spec.cellH
	if cellW <= 0 || cellH <= 0 || img.Bounds().Dx() < cellW {
		return nil, fmt.Errorf("cells of %d×%d do not fit", cellW, cellH)
	}
	columns := img.Bounds().Dx() / cellW
	font := &BitmapFont{cell: geom.V2(float32(cellW), float32(cellH)), glyphs: map[rune]*bitmapGlyph{}}
	i := 0
	for _, r := range spec.charset {
		at := img.Bounds().Min.Add(image.Pt(i%columns*cellW, i/columns*cellH))
		font.glyphs[r] = runsOf(img, image.Rectangle{Min: at, Max: at.Add(image.Pt(cellW, cellH))}, geom.Vec2{})
		i++
	}
	font.glyph = func(rune) *bitmapGlyph { return font.glyphs['?'] }
	return font, nil
}

//nolint:gochecknoglobals // a read-only table
var printableASCII = func() string {
	b := make([]byte, 0, '~'-' '+1)
	for c := byte(' '); c <= '~'; c++ {
		b = append(b, c)
	}
	return string(b)
}()

// runsOf turns the lit pixels of area in m into runs, offset by at.
func runsOf(m image.Image, area image.Rectangle, at geom.Vec2) *bitmapGlyph {
	g := &bitmapGlyph{}
	for y := area.Min.Y; y < area.Max.Y; y++ {
		start := -1
		for x := area.Min.X; x <= area.Max.X; x++ {
			on := x < area.Max.X && lit(m, image.Pt(x, y))
			switch {
			case on && start < 0:
				start = x
			case !on && start >= 0:
				g.runs = append(
					g.runs,
					geom.Rect{
						X: at.X + float32(start-area.Min.X),
						Y: at.Y + float32(y-area.Min.Y),
						W: float32(x - start),
						H: 1,
					},
				)
				start = -1
			}
		}
	}
	return g
}

func lit(m image.Image, p image.Point) bool {
	_, _, _, a := m.At(p.X, p.Y).RGBA()
	return a > 0x7fff
}

func (f *BitmapFont) glyphOf(r rune) *bitmapGlyph {
	f.mu.Lock()
	defer f.mu.Unlock()
	if g, ok := f.glyphs[r]; ok {
		return g
	}
	g := f.glyph(r)
	if g == nil {
		g = &bitmapGlyph{}
	}
	f.glyphs[r] = g
	return g
}

// Measure is the size of text in font pixels.
func (f *BitmapFont) Measure(text string) geom.Vec2 {
	lines, cols, widest := 1, 0, 0
	for _, r := range text {
		if r == '\n' {
			lines, cols = lines+1, 0
			continue
		}
		cols++
		widest = max(widest, cols)
	}
	return geom.V2(float32(widest)*f.cell.X, float32(lines)*f.cell.Y)
}

// Draw draws text from at, its top-left corner, every font pixel a
// scale×scale square snapped to that grid so it stays crisp.
func (f *BitmapFont) Draw(c *Canvas, at geom.Vec2, text string, scale float32, col color.Color) {
	scale = max(scale, 1e-3)
	at = geom.V2(geom.Floor(at.X/scale)*scale, geom.Floor(at.Y/scale)*scale)
	x, y := at.X, at.Y
	for _, r := range text {
		if r == '\n' {
			x, y = at.X, y+f.cell.Y*scale
			continue
		}
		for _, run := range f.glyphOf(r).runs {
			c.Rect(geom.Rect{X: x + run.X*scale, Y: y + run.Y*scale, W: run.W * scale, H: scale}, col)
		}
		x += f.cell.X * scale
	}
}

// BitmapText draws a string in a bitmap font, one font pixel per point
// before Scale, centred on its position unless its Pivot says otherwise.
type BitmapText struct {
	Item
	text   string      `node:"prop"`
	color  color.Color `node:"prop"`
	shadow color.Color `node:"prop"`
	pivot  geom.Vec2   `node:"prop"`
	font   *BitmapFont
}

// NewBitmapText returns white text in the default bitmap font.
func NewBitmapText(text string, opts ...BitmapTextOption) *BitmapText {
	t := Build(
		&BitmapText{Item: Base(), text: text, color: color.White, pivot: geom.V2(0.5, 0.5), font: DefaultBitmapFont()},
		"bitmap-text",
	)
	for _, o := range opts {
		o.bitmapText(t)
	}
	return t
}

// Glyphs draws a bitmap text with the glyphs of f.
func Glyphs(f *BitmapFont) BitmapTextOption {
	return bitmapTextOption(func(t *BitmapText) {
		if f != nil {
			t.font = f
		}
	})
}

// TextShadowOption sets the shadow of a text or a bitmap text.
type TextShadowOption interface {
	TextOption
	BitmapTextOption
}

type textShadowOption color.Color

func (o textShadowOption) text(t *Text)             { t.shadow = color.Color(o) }
func (o textShadowOption) bitmapText(t *BitmapText) { t.shadow = color.Color(o) }

// TextShadow draws a text's shadow in c, a little down and right: by one
// font pixel for a bitmap text.
func TextShadow(c color.Color) TextShadowOption { return textShadowOption(c) }

func (t *BitmapText) Text() string           { return t.text }
func (t *BitmapText) SetText(s string)       { t.text = s }
func (t *BitmapText) Color() color.Color     { return t.color }
func (t *BitmapText) SetColor(c color.Color) { t.color = c }

// Measure is the text's size in font pixels, before Scale.
func (t *BitmapText) Measure() geom.Vec2 { return t.font.Measure(t.text) }

func (t *BitmapText) Draw(c *Canvas) {
	size := t.Measure()
	at := geom.V2(-size.X*t.pivot.X, -size.Y*t.pivot.Y)
	if t.shadow != (color.Color{}) {
		t.font.Draw(c, at.Add(geom.V2(1, 1)), t.text, 1, t.shadow)
	}
	t.font.Draw(c, at, t.text, 1, t.color)
}
