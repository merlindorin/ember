package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
)

func TestShadesAreChromesForAnInsetOrOutsetSide(t *testing.T) {
	near := func(want [3]float32, got color.Color, msg string) {
		t.Helper()
		s := got.SRGB()
		for k := range 3 {
			assert.InDelta(t, want[k], s[k]*255, 1, msg)
		}
	}
	for _, c := range []struct {
		name          string
		c             color.Color
		darker, light [3]float32
	}{
		{"gray", color.RGB8(128, 128, 128), [3]float32{44, 44, 44}, [3]float32{212, 212, 212}},
		{"red", color.RGB8(224, 60, 49), [3]float32{140, 37, 30}, [3]float32{255, 68, 55}},
		{"black", color.RGB8(0, 0, 0), [3]float32{84, 84, 84}, [3]float32{168, 168, 168}},
	} {
		d, l := shades(c.c)
		near(c.darker, d, c.name+": darker, as Chrome draws it")
		near(c.light, l, c.name+": lighter")
	}
}

func TestInsetDarkensTopAndLeftAndOutsetTheOthers(t *testing.T) {
	gray := color.RGB8(128, 128, 128)
	darker, lighter := shades(gray)
	for _, c := range []struct {
		s         style.BorderStyle
		darkSides []int
		name      string
	}{{style.BorderInset, []int{0, 3}, "inset"}, {style.BorderOutset, []int{1, 2}, "outset"}} {
		b := el(10, 10, 40, 30, func(s *style.Computed) {
			s.BorderStyle = style.All(c.s)
			s.BorderColor = style.All(gray)
			s.Border = style.All[float32](4)
		})
		b.Border = box.Edges{Top: 4, Right: 4, Bottom: 4, Left: 4}
		r := newRecorder()
		Paint(r, tree(b))
		sides := r.of("triangles")
		if !assert.Len(t, sides, 4, "%s: four mitered sides", c.name) {
			continue
		}
		for i, o := range sides {
			want := lighter
			for _, d := range c.darkSides {
				if d == i {
					want = darker
				}
			}
			assert.Equal(t, want, o.Color, "%s side %d (top, right, bottom, left)", c.name, i)
		}
	}
}
