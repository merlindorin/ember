// Package fonts holds a Document's font families: it measures text for
// layout from the font files' own units, and gives export the same files
// for Chrome.
package fonts

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/rivo/uniseg"

	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/style"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Face is one font file of a family.
type Face struct {
	Family string
	Weight int
	Italic bool
	TTF    []byte
	font   *sfnt.Font
	upem   float32
}

// Of is a ui.Document's set, for export.
var Of func(doc any) *Set //nolint:gochecknoglobals // a hook ui sets, as Of in assets

// Set is the families a Document knows.
type Set struct {
	// emoji reports the grapheme clusters drawn as emoji pictures.
	emoji func(cluster string) bool
	faces []*Face
	buf   sfnt.Buffer
	// scale is device pixels per point, 0 for 1.
	scale float32
}

// SetScale sets the device pixels per point the text is drawn at, which
// metrics round to, as Chrome rounds them to device pixels.
func (s *Set) SetScale(pixels float32) { s.scale = pixels }

// Scale is the device pixels per point SetScale set, 0 before it.
func (s *Set) Scale() float32 { return s.scale }

// goMono is the monospace family Builtin adds and Fallback draws with.
const goMono = "Go Mono"

// Builtin is "Go" (regular, bold, italic, bold italic) and "Go Mono".
func Builtin() *Set {
	s := &Set{}
	for _, f := range []struct {
		family string
		weight int
		italic bool
		ttf    []byte
	}{
		{"Go", 400, false, goregular.TTF},
		{"Go", 700, false, gobold.TTF},
		{"Go", 400, true, goitalic.TTF},
		{"Go", 700, true, gobolditalic.TTF},
		{goMono, 400, false, draw.MonoTTF()},
	} {
		if err := s.Add(f.family, f.weight, f.italic, f.ttf); err != nil {
			panic(err)
		}
	}
	return s
}

// Add parses a font file into family.
func (s *Set) Add(family string, weight int, italic bool, ttf []byte) error {
	f, err := sfnt.Parse(ttf)
	if err != nil {
		return err
	}
	s.faces = append(s.faces,
		&Face{Family: family, Weight: weight, Italic: italic, TTF: ttf, font: f, upem: float32(f.UnitsPerEm())})
	return nil
}

// Faces is every face, in the order added.
func (s *Set) Faces() []*Face { return s.faces }

// generic are the generic families and the faces drawing them.
//
//nolint:gochecknoglobals // a lookup table
var generic = [...]struct{ name, face string }{
	{"sans-serif", "Go"}, {"serif", "Go"}, {"system-ui", "Go"}, {"monospace", goMono},
}

// genericFace is the face drawing name when it is a generic family, in
// any case, without allocating: Face runs for every text run every frame.
func genericFace(name string) (string, bool) {
	for _, g := range generic {
		if strings.EqualFold(name, g.name) {
			return g.face, true
		}
	}
	return "", false
}

// Face is the face drawing f: the first family of its list the set has,
// else Go, then the closest weight in the asked style.
func (s *Set) Face(f style.Font) *Face {
	family := "Go"
	for _, name := range f.Family {
		if g, ok := genericFace(name); ok {
			name = g
		}
		if s.has(name) {
			family = name
			break
		}
	}
	var best *Face
	score := func(c *Face) int {
		d := c.Weight - f.Weight
		if d < 0 {
			d = -d
		}
		if c.Italic != f.Italic {
			d += 10000
		}
		return d
	}
	for _, c := range s.faces {
		if c.Family == family && (best == nil || score(c) < score(best)) {
			best = c
		}
	}
	return best
}

func (s *Set) has(family string) bool {
	for _, c := range s.faces {
		if c.Family == family {
			return true
		}
	}
	return false
}

// Metrics are the face's hhea metrics at f.Size, ascent and descent
// rounded to whole device pixels as Chrome rounds them.
func (s *Set) Metrics(f style.Font) box.Metrics {
	c := s.Face(f)
	m, err := c.font.Metrics(&s.buf, fixed.I(int(c.upem)), font.HintingNone)
	if err != nil {
		return box.Metrics{}
	}
	scale := f.Size / c.upem
	ascent := float32(m.Ascent) / 64 * scale
	descent := float32(m.Descent) / 64 * scale
	gap := float32(m.Height)/64*scale - ascent - descent
	px := s.scale
	if px <= 0 {
		px = 1
	}
	return box.Metrics{Ascent: round(ascent*px) / px, Descent: round(descent*px) / px, LineGap: max(gap, 0)}
}

func round(v float32) float32 { return float32(int(v + 0.5)) }

// Advance is the width of text in f, kerning included; a character f's
// face lacks takes the Fallback face's advance.
func (s *Set) Advance(f style.Font, text string) float32 {
	if s.emoji == nil || isASCII(text) {
		return s.advance(f, text)
	}
	var w float32
	start := 0
	rest, state := text, -1
	var c string
	for at := 0; len(rest) > 0; {
		c, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		if len(c) > 1 && s.emoji(c) {
			w += s.advance(f, text[start:at]) + f.Size
			start = at + len(c)
		}
		at += len(c)
	}
	return w + s.advance(f, text[start:])
}

// SetEmoji has Advance measure each grapheme cluster has reports, and
// twod draws as an emoji picture, one em wide, as twod does.
func (s *Set) SetEmoji(has func(cluster string) bool) { s.emoji = has }

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// fallbackAdvance is r's advance at size in the fallback face, when c lacks r and the
// fallback draws it.
func (s *Set) fallbackAdvance(c *Face, r rune, size float32) (float32, bool) {
	fb := s.Fallback()
	if fb == nil || fb == c {
		return 0, false
	}
	g, err := fb.font.GlyphIndex(&s.buf, r)
	if err != nil || g == 0 {
		return 0, false
	}
	a, err := fb.font.GlyphAdvance(&s.buf, g, fixed.I(int(fb.upem)), font.HintingNone)
	if err != nil {
		return 0, true
	}
	return float32(a) / 64 * size / fb.upem, true
}

// advance is text's width in f's face, the fallback's characters in it.
func (s *Set) advance(f style.Font, text string) float32 {
	c := s.Face(f)
	ppem := fixed.I(int(c.upem))
	var w fixed.Int26_6
	var other float32
	prev := sfnt.GlyphIndex(0)
	for _, r := range text {
		g, err := c.font.GlyphIndex(&s.buf, r)
		if err != nil {
			continue
		}
		if g == 0 {
			if a, ok := s.fallbackAdvance(c, r, f.Size); ok {
				other += a
				prev = 0
				continue
			}
		}
		if prev != 0 {
			if k, kerr := c.font.Kern(&s.buf, prev, g, ppem, font.HintingNone); kerr == nil {
				w += k
			}
		}
		a, err := c.font.GlyphAdvance(&s.buf, g, ppem, font.HintingNone)
		if err == nil {
			w += a
		}
		prev = g
	}
	return float32(w)/64*f.Size/c.upem + other
}

// Fallback is the face that draws what a face lacks: Go Mono, whose
// arrows, triangles and ticks Go has not.
func (s *Set) Fallback() *Face {
	for _, c := range s.faces {
		if c.Family == goMono {
			return c
		}
	}
	return nil
}

// FallbackRanges are the characters c lacks that the Fallback face has,
// below U+3000, as a CSS unicode-range; "" for none.
func (s *Set) FallbackRanges(c *Face) string {
	fb := s.Fallback()
	if fb == nil || fb == c {
		return ""
	}
	var ranges []string
	start := rune(-1)
	flush := func(end rune) {
		if start < 0 {
			return
		}
		if start == end {
			ranges = append(ranges, fmt.Sprintf("U+%X", start))
		} else {
			ranges = append(ranges, fmt.Sprintf("U+%X-%X", start, end))
		}
		start = -1
	}
	for r := rune(0x20); r < 0x3000; r++ {
		g, _ := c.font.GlyphIndex(&s.buf, r)
		fg, _ := fb.font.GlyphIndex(&s.buf, r)
		if g == 0 && fg != 0 {
			if start < 0 {
				start = r
			}
			continue
		}
		flush(r - 1)
	}
	flush(0x2FFF)
	return strings.Join(ranges, ", ")
}

// FallbackMetrics are the Fallback face's metrics at f's size when it
// draws some of text, as box.FallbackFonts says.
func (s *Set) FallbackMetrics(f style.Font, text string) (box.Metrics, bool) {
	c, fb := s.Face(f), s.Fallback()
	if fb == nil || fb == c {
		return box.Metrics{}, false
	}
	for _, r := range text {
		if g, _ := c.font.GlyphIndex(&s.buf, r); g != 0 {
			continue
		}
		if g, _ := fb.font.GlyphIndex(&s.buf, r); g != 0 {
			return s.Metrics(style.Font{Family: []string{fb.Family}, Size: f.Size, Weight: fb.Weight}), true
		}
	}
	return box.Metrics{}, false
}

var _ box.FallbackFonts = (*Set)(nil)

// Draws reports whether text shows whole in f, as twod's Face.Draws
// says: each grapheme cluster an emoji picture, or, when it is a single
// character (a variation selector aside), a glyph of f's face or of the
// fallback. A keycap, a toned emoji or a flag needs its picture.
func (s *Set) Draws(f style.Font, text string) bool {
	c, fb := s.Face(f), s.Fallback()
	rest, state := text, -1
	var g string
	for len(rest) > 0 {
		g, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		if s.emoji != nil && s.emoji(g) {
			continue
		}
		bare := strings.ReplaceAll(g, "\ufe0f", "")
		if utf8.RuneCountInString(bare) != 1 {
			return false
		}
		r, _ := utf8.DecodeRuneInString(bare)
		if !s.hasGlyph(c, r) && (fb == nil || !s.hasGlyph(fb, r)) {
			return false
		}
	}
	return true
}

func (s *Set) hasGlyph(c *Face, r rune) bool {
	g, err := c.font.GlyphIndex(&s.buf, r)
	return err == nil && g != 0
}
