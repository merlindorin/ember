package draw

import (
	_ "embed"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
	"github.com/rivo/uniseg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/gofont/gosmallcaps"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

//go:embed fonts/SourceCodePro-Regular.ttf
var sourceCodePro []byte

// MonoTTF is the embedded SourceCodePro font file, for other text
// renderers such as the ImGui layer.
func MonoTTF() []byte { return sourceCodePro }

// Emoji is a set of pictures for grapheme clusters, drawn inline instead
// of glyphs.
type Emoji interface {
	Picture(cluster string) (*render.Texture, bool)
}

// Face is a parsed font file, with the faces that draw what it lacks and
// the pictures that replace emoji. Faces are rasterized on demand per
// size.
type Face struct {
	name      string
	font      *opentype.Font
	fallbacks []*Face
	emoji     Emoji
	sizes     map[float32]*Font
	// pixelSizes are the face's pixelated fonts (PixelSize), apart, so
	// smoothed text never shares their atlas.
	pixelSizes map[float32]*Font
}

// The engine's faces.
//
//nolint:gochecknoglobals // one value each, parsed once, that every text shares
var (
	Regular   = mustFace("go-regular", goregular.TTF)
	Bold      = mustFace("go-bold", gobold.TTF)
	Italic    = mustFace("go-italic", goitalic.TTF)
	SmallCaps = mustFace("go-smallcaps", gosmallcaps.TTF)
	Mono      = mustFace("source-code-pro", sourceCodePro)
)

// ParseFace parses ttf, the bytes of a TrueType or OpenType font file
// called name in errors.
func ParseFace(name string, ttf []byte) (*Face, error) {
	f, err := opentype.Parse(ttf)
	if err != nil {
		return nil, fmt.Errorf("twod: parse font %s: %w", name, err)
	}
	return &Face{name: name, font: f, sizes: map[float32]*Font{}}, nil
}

func mustFace(name string, ttf []byte) *Face {
	f, err := ParseFace(name, ttf)
	if err != nil {
		panic(err)
	}
	return f
}

// With is f drawing what it lacks with fallbacks, in order, and the
// clusters emoji has as pictures.
func (f *Face) With(fallbacks []*Face, emoji Emoji) *Face {
	return &Face{name: f.name, font: f.font, fallbacks: fallbacks, emoji: emoji, sizes: map[float32]*Font{}}
}

// Draws reports whether the face draws text as written: a picture for
// each cluster of several characters (a flag, a ZWJ family, a keycap),
// else a glyph in the face or a fallback for a single character.
func (f *Face) Draws(text string) bool {
	var buf sfnt.Buffer
	ok := true
	eachCluster(text, func(c string, first rune) {
		if !ok || first == '\n' {
			return
		}
		if f.emoji != nil && len(c) > 1 {
			if _, pic := f.emoji.Picture(c); pic {
				return
			}
		}
		if utf8.RuneCountInString(strings.ReplaceAll(c, "\ufe0f", "")) > 1 {
			ok = false
			return
		}
		for _, src := range append([]*Face{f}, f.fallbacks...) {
			if idx, err := src.font.GlyphIndex(&buf, first); err == nil && idx != 0 {
				return
			}
		}
		ok = false
	})
	return ok
}

// oversample rasterizes glyphs at twice the logical size so text stays
// crisp on high-density displays.
const oversample = 2

// Size returns the font rasterized at px logical pixels.
func (f *Face) Size(px float32) *Font {
	px = float32(math.Round(float64(px)))
	if fnt, ok := f.sizes[px]; ok {
		return fnt
	}
	fnt, err := newFont(f, px, false)
	if err != nil {
		panic(err)
	}
	f.sizes[px] = fnt
	return fnt
}

// PixelSize returns the font at px logical pixels rasterized without
// smoothing: each glyph pixel covered or not, at half its coverage, as
// Chrome draws -webkit-font-smoothing: none, for pixel fonts.
func (f *Face) PixelSize(px float32) *Font {
	px = float32(math.Round(float64(px)))
	if fnt, ok := f.pixelSizes[px]; ok {
		return fnt
	}
	fnt, err := newFont(f, px, true)
	if err != nil {
		panic(err)
	}
	if f.pixelSizes == nil {
		f.pixelSizes = map[float32]*Font{}
	}
	f.pixelSizes[px] = fnt
	return fnt
}

type glyph struct {
	advance float32
	offset  geom.Vec2
	size    geom.Vec2
	// rect is where the glyph is in the atlas, in pixels.
	rect image.Rectangle
	// face is which of the font's faces draws it and idx its index
	// there, for kerning; ready marks an ASCII slot filled.
	face  int
	idx   sfnt.GlyphIndex
	r     rune
	ready bool
}

// phases is how many horizontal positions within an atlas pixel a glyph
// is rasterized at, so text keeps its exact advances yet each glyph is
// drawn 1:1 on whole pixels.
const phases = 4

// glyphKey is a rune rasterized at a phase, in 1/phases of a pixel.
type glyphKey struct {
	r     rune
	phase int
}

// Font is a face at one size, backed by a glyph atlas that grows as new
// characters are drawn.
type Font struct {
	Atlas   *image.Alpha
	version uint64
	glyphs  map[glyphKey]glyph
	// ascii holds the ASCII glyphs at every phase, looked up without a
	// map, as most text is ASCII.
	ascii [128 * phases]glyph
	// kerns caches the non-ASCII pairs met, by face and glyph indices
	// packed for a fast map; asciiKerns is each face's ASCII table.
	kerns      map[uint64]float32
	asciiKerns []*[128 * 128]fixed.Int26_6
	faces      []font.Face
	sources    []*opentype.Font
	emoji      Emoji
	buf        sfnt.Buffer
	cell       int
	penX, penY int
	ascent     float32
	lineHeight float32
	size       float32
	// pixel rasterizes glyphs without smoothing (PixelSize).
	pixel bool
}

const atlasRunes = "ÀÂÄÇÉÈÊËÎÏÔÖÙÛÜàâäçéèêëîïôöùûüœ·•×÷°±…—–’“”«»€£¥©®™←↑→↓★♥"

func newFont(f *Face, px float32, pixel bool) (*Font, error) {
	out := &Font{glyphs: map[glyphKey]glyph{}, kerns: map[uint64]float32{}, emoji: f.emoji, size: px, pixel: pixel}
	for _, src := range append([]*Face{f}, f.fallbacks...) {
		face, err := opentype.NewFace(
			src.font,
			&opentype.FaceOptions{Size: float64(px * oversample), DPI: 72, Hinting: font.HintingNone},
		)
		if err != nil {
			return nil, err
		}
		out.faces, out.sources = append(out.faces, face), append(out.sources, src.font)
		out.asciiKerns = append(out.asciiKerns, asciiKerning(src.font))
	}
	m := out.faces[0].Metrics()
	out.ascent = float32(m.Ascent.Ceil()) / oversample
	out.lineHeight = float32(m.Height.Ceil()) / oversample
	out.cell = (m.Ascent + m.Descent).Ceil() + 4
	out.Atlas = image.NewAlpha(image.Rect(0, 0, nextPow2(16*(out.cell+2)), nextPow2(8*(out.cell+2))))
	for r := rune(32); r < 127; r++ {
		out.glyph(r)
	}
	for _, r := range atlasRunes {
		out.glyph(r)
	}
	return out, nil
}

func nextPow2(v int) int {
	p := 64
	for p < v {
		p *= 2
	}
	return p
}

// Version changes whenever a glyph is added to the atlas, so the GPU knows
// to upload it again.
func (f *Font) Version() uint64 { return f.version }

func (f *Font) LineHeight() float32 { return f.lineHeight }

func (f *Font) Ascent() float32 { return f.ascent }

// Measure returns the size of a single- or multi-line string.
func (f *Font) Measure(s string) geom.Vec2 { return f.measure(s, 0) }

// measure is the size of s with tracking after each cluster.
func (f *Font) measure(s string, tracking float32) geom.Vec2 {
	var w, line float32
	lines := 1
	prev := none
	eachCluster(s, func(c string, first rune) {
		if first == '\n' || first == '\r' {
			w = max(w, line)
			line = 0
			lines++
			prev = none
			return
		}
		if _, ok := f.picture(c); ok {
			line += f.size + tracking
			prev = none
			return
		}
		g := f.glyph(first)
		line += f.kern(prev, g) + g.advance + tracking
		prev = g
	})
	return geom.V2(max(w, line), float32(lines)*f.lineHeight)
}

// picture is the emoji picture for cluster c, when the face has one.
func (f *Font) picture(c string) (*render.Texture, bool) {
	if f.emoji == nil || len(c) == 1 {
		return nil, false
	}
	return f.emoji.Picture(c)
}

// advance is glyph idx's advance in face, in points, measured in font
// units as layout measures it, so the two agree to the last fraction.
func (f *Font) advance(face int, idx sfnt.GlyphIndex) float32 {
	src := f.sources[face]
	upem := src.UnitsPerEm()
	a, err := src.GlyphAdvance(&f.buf, idx, fixed.I(int(upem)), font.HintingNone)
	if err != nil {
		return 0
	}
	return float32(a) / 64 * f.size / float32(upem)
}

// kern is the room between glyphs p and g, drawn one after the other,
// from their face's kerning table when both come from the same face; 0
// after nothing (p's face < 0). ASCII pairs come from the face's table
// read once; others are looked up and cached.
func (f *Font) kern(p, g glyph) float32 {
	if p.face < 0 || p.face != g.face || p.idx == 0 || g.idx == 0 {
		return 0
	}
	src := f.sources[g.face]
	upem := float32(src.UnitsPerEm())
	if p.r < 128 && g.r < 128 {
		t := f.asciiKerns[g.face]
		if t == nil {
			return 0
		}
		return float32(t[p.r*128+g.r]) / 64 * f.size / upem
	}
	key := uint64(g.face)<<32 | uint64(p.idx)<<16 | uint64(g.idx) //nolint:gosec // a face index and two 16-bit glyph ids
	if k, ok := f.kerns[key]; ok {
		return k
	}
	k, err := src.Kern(&f.buf, p.idx, g.idx, fixed.I(int(upem)), font.HintingNone)
	if err != nil {
		k = 0
	}
	v := float32(k) / 64 * f.size / upem
	f.kerns[key] = v
	return v
}

//nolint:gochecknoglobals // a cache every font shares, under its mutex
var (
	kerning   = map[*opentype.Font]*[128 * 128]fixed.Int26_6{}
	kerningMu sync.Mutex
)

// asciiKerning is src's kerning between ASCII characters, in font units
// as 26.6, read once for every size; nil when it kerns none of them.
func asciiKerning(src *opentype.Font) *[128 * 128]fixed.Int26_6 {
	kerningMu.Lock()
	defer kerningMu.Unlock()
	if t, ok := kerning[src]; ok {
		return t
	}
	var buf sfnt.Buffer
	var idx [128]sfnt.GlyphIndex
	for r := range idx {
		idx[r], _ = src.GlyphIndex(&buf, rune(r))
	}
	t := new([128 * 128]fixed.Int26_6)
	kerns := false
	upem := fixed.I(int(src.UnitsPerEm()))
	for a := 32; a < 128; a++ {
		for b := 32; b < 128; b++ {
			if idx[a] == 0 || idx[b] == 0 {
				continue
			}
			if k, err := src.Kern(&buf, idx[a], idx[b], upem, font.HintingNone); err == nil && k != 0 {
				t[a*128+b], kerns = k, true
			} else if errors.Is(err, sfnt.ErrNotFound) {
				kerning[src] = nil
				return nil
			}
		}
	}
	if !kerns {
		t = nil
	}
	kerning[src] = t
	return t
}

// none is what comes before the first glyph: nothing to kern against.
//
//nolint:gochecknoglobals // a fixed value; Go has no constant structs
var none = glyph{face: -1}

// glyph is r's glyph at phase 0: its advance and face.
func (f *Font) glyph(r rune) glyph { return f.glyphAt(r, 0) }

// glyphAt is r's glyph rasterized phase/phases of a pixel to the right,
// from the first face that has it, the first time it is asked for; '?'
// when none has it. Its advance is the font's, unrounded.
func (f *Font) glyphAt(r rune, phase int) glyph {
	if r >= 0 && r < 128 {
		if g := f.ascii[int(r)*phases+phase]; g.ready {
			return g
		}
	} else if g, ok := f.glyphs[glyphKey{r, phase}]; ok {
		return g
	}
	g := f.raster(r, phase)
	g.ready = true
	if r >= 0 && r < 128 {
		f.ascii[int(r)*phases+phase] = g
	} else {
		f.glyphs[glyphKey{r, phase}] = g
	}
	return g
}

// raster rasterizes r at phase into the atlas, from the first face that
// has it; '?' when none has it.
func (f *Font) raster(r rune, phase int) glyph {
	face, idx := -1, sfnt.GlyphIndex(0)
	for i, src := range f.sources {
		if x, err := src.GlyphIndex(&f.buf, r); err == nil && x != 0 {
			face, idx = i, x
			break
		}
	}
	if face < 0 {
		if r == '?' {
			return glyph{}
		}
		return f.glyphAt('?', phase)
	}
	g := glyph{advance: f.advance(face, idx), face: face, idx: idx, r: r}
	x := fixed.Int26_6(phase * 64 / phases) //nolint:gosec // phase < phases, so at most 64
	dr, mask, mp, _, ok := f.faces[face].Glyph(fixed.Point26_6{X: x}, r)
	if ok && !dr.Empty() {
		dst := f.place(dr.Dx()+2, dr.Dy()+2)
		draw.Draw(f.Atlas, dst.Inset(1), mask, mp, draw.Src)
		if f.pixel {
			threshold(f.Atlas, dst)
		} else {
			embolden(f.Atlas, dst, darkening(f.size*oversample))
		}
		f.version++
		g.offset = geom.V2(float32(dr.Min.X-1), float32(dr.Min.Y-1)).Scale(1.0 / oversample)
		g.size = geom.V2(float32(dst.Dx()), float32(dst.Dy())).Scale(1.0 / oversample)
		g.rect = dst
	}
	return g
}

// darkening is how far stem darkening moves a glyph's edges out, in
// pixels of a raster px pixels tall: 0.005 of it, fitted to Chrome's text
// on macOS, which reads heavier than exact coverage.
func darkening(px float32) float32 { return 0.005 * px }

// embolden moves the edges of the shapes in r out by d pixels: every
// pixel partly covered, or beside one more than half covered, gains d of
// coverage, as an outline offset by d would give it.
func embolden(a *image.Alpha, r image.Rectangle, d float32) {
	add := int(math.Round(float64(d) * 255))
	if add <= 0 {
		return
	}
	src := image.NewAlpha(r)
	draw.Draw(src, r, a, r.Min, draw.Src)
	covered := func(x, y int) bool { return image.Pt(x, y).In(r) && src.AlphaAt(x, y).A > 127 }
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			v := int(src.AlphaAt(x, y).A)
			if v > 0 || covered(x-1, y) || covered(x+1, y) || covered(x, y-1) || covered(x, y+1) {
				a.SetAlpha(x, y, color.Alpha{A: uint8(min(255, v+add))}) //nolint:gosec // capped at 255, add ≥ 0
			}
		}
	}
}

// threshold makes each pixel in r covered or not, at half coverage: a
// glyph unsmoothed.
func threshold(a *image.Alpha, r image.Rectangle) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			v := uint8(0)
			if a.AlphaAt(x, y).A >= 128 {
				v = 255
			}
			a.SetAlpha(x, y, color.Alpha{A: v})
		}
	}
}

// place finds room for a w×h glyph in the atlas, row by row, doubling the
// atlas's height when it is full.
func (f *Font) place(w, h int) image.Rectangle {
	row := max(f.cell, h) + 2
	if f.penX+w+2 > f.Atlas.Rect.Dx() {
		f.penX, f.penY = 0, f.penY+row
	}
	for f.penY+row > f.Atlas.Rect.Dy() {
		grown := image.NewAlpha(image.Rect(0, 0, f.Atlas.Rect.Dx(), f.Atlas.Rect.Dy()*2))
		draw.Draw(grown, f.Atlas.Rect, f.Atlas, image.Point{}, draw.Src)
		f.Atlas = grown
	}
	at := image.Rect(f.penX+1, f.penY+1, f.penX+1+w, f.penY+1+h)
	f.penX += w + 2
	return at
}

// uv is g's rectangle in the atlas, in texels: the atlas only grows,
// downwards, so it stays right however much it grows, in the same frame
// or later; the shader divides by the atlas's size as it samples.
func (f *Font) uv(g glyph) [4]float32 {
	return [4]float32{float32(g.rect.Min.X), float32(g.rect.Min.Y), float32(g.rect.Max.X), float32(g.rect.Max.Y)}
}

// eachCluster calls fn with each grapheme cluster of s and its first rune:
// a ZWJ family, a flag or a letter with its accents is one.
func eachCluster(s string, fn func(cluster string, first rune)) {
	state := -1
	for len(s) > 0 {
		var c string
		c, s, _, state = uniseg.FirstGraphemeClusterInString(s, state)
		r := []rune(c)[0]
		if strings.ContainsRune(c, '\n') {
			r = '\n'
		}
		fn(c, r)
	}
}
