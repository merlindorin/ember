package fonts //nolint:testpackage // tests unexported internals

import (
	"fmt"
	"strings"
	"testing"

	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
)

func TestFamiliesFallBackToGoAndPickTheClosestWeight(t *testing.T) {
	s := Builtin()
	for _, c := range []struct {
		font   style.Font
		family string
		weight int
		italic bool
	}{
		{style.Font{Family: []string{"Nope", "sans-serif"}, Weight: 400}, "Go", 400, false},
		{style.Font{Family: []string{"monospace"}, Weight: 700}, "Go Mono", 400, false},
		{style.Font{Family: []string{"Go"}, Weight: 600}, "Go", 700, false},
		{style.Font{Weight: 400, Italic: true}, "Go", 400, true},
	} {
		f := s.Face(c.font)
		assert.Equal(t, c.family, f.Family)
		assert.Equal(t, c.weight, f.Weight)
		assert.Equal(t, c.italic, f.Italic)
	}
}

func TestMeasuresScaleWithTheSize(t *testing.T) {
	s := Builtin()
	small, big := style.Font{Family: []string{"Go"}, Size: 10,
		Weight: 400}, style.Font{Family: []string{"Go"}, Size: 20, Weight: 400}
	assert.InDelta(t, 2*s.Advance(small, "Hello"), s.Advance(big, "Hello"), 0.01)
	assert.Greater(t, s.Advance(small, "Hello"), s.Advance(small, "Hell"))
	m := s.Metrics(big)
	assert.Greater(t, m.Ascent, m.Descent)
	assert.Greater(t, m.Descent, float32(0))
	assert.GreaterOrEqual(t, m.LineGap, float32(0))
}

func TestMetricsRoundToDevicePixels(t *testing.T) {
	s := Builtin()
	f := style.Font{Family: []string{"Go"}, Size: 15, Weight: 400}
	one := s.Metrics(f)
	s.SetScale(2)
	two := s.Metrics(f)
	for _, v := range []float32{one.Ascent, one.Descent} {
		assert.Equal(t, v, float32(int(v)), "whole points at 1x")
	}
	for _, v := range []float32{two.Ascent, two.Descent} {
		assert.Equal(t, 2*v, float32(int(2*v)), "half points at 2x")
	}
	assert.InDelta(t, one.Ascent, two.Ascent, 0.5)
}

func TestWhatGoLacksComesFromGoMono(t *testing.T) {
	s := Builtin()
	f := style.Font{Family: []string{"Go"}, Size: 16, Weight: 400}
	mono := style.Font{Family: []string{"Go Mono"}, Size: 16, Weight: 400}
	assert.InDelta(t, s.Advance(f, "a")+s.Advance(mono, "▸"), s.Advance(f, "a▸"), 0.001)
	ranges := s.FallbackRanges(s.Face(f))
	assert.True(t, covers(ranges, '▸'))
	assert.True(t, covers(ranges, '✓'))
	assert.False(t, covers(ranges, '→'), "Go has its own arrow")
	assert.False(t, covers(ranges, 'a'))
	assert.Empty(t, s.FallbackRanges(s.Fallback()))

	m, ok := s.FallbackMetrics(f, "tile ▸")
	assert.True(t, ok)
	assert.Equal(t, s.Metrics(mono), m)
	_, ok = s.FallbackMetrics(f, "tile → 0")
	assert.False(t, ok)
}

// covers reports whether a CSS unicode-range holds r.
func covers(ranges string, r rune) bool {
	for _, part := range strings.Split(ranges, ", ") {
		var lo, hi rune
		if n, _ := fmt.Sscanf(part, "U+%X-%X", &lo, &hi); n == 2 {
			if r >= lo && r <= hi {
				return true
			}
		} else if n == 1 && r == lo {
			return true
		}
	}
	return false
}

func TestAFaceIsFoundWithoutAllocating(t *testing.T) {
	s := Builtin()
	f := style.Font{Family: []string{"Sans-Serif"}, Size: 15, Weight: 400}
	assert.Equal(t, "Go", s.Face(f).Family, "generic families in any case")
	assert.Zero(t, testing.AllocsPerRun(100, func() { s.Face(f) }))
}

func TestDrawsSaysWhatShowsWhole(t *testing.T) {
	s := Builtin()
	f := style.Font{Weight: 400, Size: 16}
	assert.True(t, s.Draws(f, "walls → ▸"), "Go, and the fallback for the arrows")
	assert.False(t, s.Draws(f, "🔥"), "no emoji without a set")
	assert.False(t, s.Draws(f, "3\uFE0F\u20E3"), "a keycap needs its picture, as twod says")
	assert.False(t, s.Draws(f, "👍\U0001F3FD"), "and so does a toned emoji")
	assert.True(t, s.Draws(f, "\u2764\uFE0F"), "a single character with a variation selector stands on its glyph")
	s.SetEmoji(func(c string) bool { return c == "🔥" })
	assert.True(t, s.Draws(f, "hot 🔥"))
}
