package draw //nolint:testpackage // white-box: reaches draw's unexported state

import (
	"testing"
	"unicode"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/render"
	"golang.org/x/image/font/sfnt"
)

func TestEmojiSequencesAreOneClusterEach(t *testing.T) {
	for text, want := range map[string][]string{
		"👨‍👩‍👧":  {"👨‍👩‍👧"},
		"👍🏽!":    {"👍🏽", "!"},
		"🇫🇷🇯🇵":   {"🇫🇷", "🇯🇵"},
		"1️⃣":    {"1️⃣"},
		"❤️ ok":  {"❤️", " ", "o", "k"},
		"été":   {"é", "t", "é"},
		"a\r\nb": {"a", "\r\n", "b"},
		"🧙‍♂️":   {"🧙‍♂️"},
	} {
		var got []string
		eachCluster(text, func(c string, _ rune) { got = append(got, c) })
		if len(got) != len(want) {
			t.Errorf("%q: %q, want %q", text, got, want)
			continue
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%q: %q, want %q", text, got, want)
			}
		}
	}
}

// onlyIn is a rune Mono draws and Regular does not.
func onlyIn(t *testing.T) rune {
	t.Helper()
	var buf sfnt.Buffer
	for r := rune(0x2000); r < 0x2c00; r++ {
		a, _ := Regular.font.GlyphIndex(&buf, r)
		b, _ := Mono.font.GlyphIndex(&buf, r)
		if a == 0 && b != 0 && unicode.IsGraphic(r) && !unicode.IsSpace(r) {
			return r
		}
	}
	t.Skip("no rune only Mono has")
	return 0
}

func TestAFallbackDrawsWhatTheFaceLacks(t *testing.T) {
	r := onlyIn(t)
	plain := Regular.Size(16)
	question := plain.glyph('?')
	if g := plain.glyph(r); g.rect != question.rect {
		t.Fatalf("without a fallback, %q is a question mark", r)
	}
	chained := Regular.With([]*Face{Mono}, nil).Size(16)
	before := chained.Version()
	g := chained.glyph(r)
	if g.rect == chained.glyph('?').rect || g.size.X == 0 {
		t.Fatalf("with Mono behind it, %q is drawn from Mono", r)
	}
	if chained.Version() == before {
		t.Fatal("a glyph added on demand changes the atlas version")
	}
	if w := chained.Measure(string(r)).X; w != g.advance {
		t.Fatalf("measured by its fallback glyph: %v, want %v", w, g.advance)
	}
}

func TestTheAtlasGrowsAsGlyphsAreAdded(t *testing.T) {
	f := Mono.With(nil, nil).Size(40)
	h := f.Atlas.Rect.Dy()
	first := f.uv(f.glyph('A'))
	for r := rune(0x100); r < 0x500; r++ {
		f.glyph(r)
	}
	if f.Atlas.Rect.Dy() <= h {
		t.Fatalf("hundreds more glyphs need a taller atlas: %d", f.Atlas.Rect.Dy())
	}
	if uv := f.uv(f.glyph('A')); uv != first || uv[3] > float32(h) {
		t.Fatalf(
			"the first glyphs keep their texels, so their UVs, in texels, stay as they were: %v, then %v",
			first,
			uv,
		)
	}
}

type pictures map[string]*render.Texture

func (p pictures) Picture(c string) (*render.Texture, bool) { t, ok := p[c]; return t, ok }

func TestAnEmojiPictureIsOneSquareAtTheTextSize(t *testing.T) {
	family := "👨‍👩‍👧"
	f := Regular.With(nil, pictures{family: &render.Texture{}}).Size(20)
	if w := f.Measure("a" + family + "a").X; w != 2*f.glyph('a').advance+20 {
		t.Fatalf("one 20-point square between two a's: %v", w)
	}
	var l List
	l.Text(geom.Vec2{}, family, Style{Face: Regular.With(nil, pictures{family: &render.Texture{}}), Size: 20})
	images := 0
	for _, v := range l.Unindexed() {
		if Kind(v.Shape.X) == KindImage {
			images++
		}
	}
	if images != 6 {
		t.Fatalf("drawn as one picture, two triangles: %d vertices", images)
	}
}
