package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"image"
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
)

func TestANinePatchSamplesEachSliceWithinItself(t *testing.T) {
	tex := twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 24, 24)))
	r := newRecorder()
	(&painter{s: r}).ninePatch(geom.Rect{X: 10, Y: 10, W: 200, H: 60}, tex,
		style.Sides[float32]{Top: 8, Right: 8, Bottom: 8, Left: 8}, [2]style.ImageRepeat{}, color.White)
	slices := [4]float32{0, 8, 16, 24}
	within := func(a, b float32) bool {
		for i := range 3 {
			lo, hi := (slices[i]+0.5)/24, (slices[i+1]-0.5)/24
			if a >= lo-1e-6 && b <= hi+1e-6 {
				return true
			}
		}
		return false
	}
	area := float32(0)
	for _, o := range r.of("image") {
		area += o.Rect.W * o.Rect.H
		assert.True(t, within(o.UV.X, o.UV.X+o.UV.W),
			"u %v..%v within one slice, half a texel in from its edges", o.UV.X*24, (o.UV.X+o.UV.W)*24)
		assert.True(t, within(o.UV.Y, o.UV.Y+o.UV.H), "v %v..%v within one slice", o.UV.Y*24, (o.UV.Y+o.UV.H)*24)
	}
	assert.InDelta(t, 200*60, area, 1e-2, "covering the box once")
}

func TestANinePatchSliceMapsItsMiddleLinearly(t *testing.T) {
	tex := twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 24, 24)))
	r := newRecorder()
	(&painter{s: r}).ninePatch(geom.Rect{W: 200, H: 60}, tex,
		style.Sides[float32]{Top: 8, Right: 8, Bottom: 8, Left: 8}, [2]style.ImageRepeat{}, color.White)
	for _, o := range r.of("image") {
		if o.Rect.X > 8 && o.Rect.X+o.Rect.W < 192 && o.Rect.Y > 8 && o.Rect.Y+o.Rect.H < 52 {
			assert.InDelta(t, 7.0/(184-184.0/8), o.UV.W*24/o.Rect.W, 1e-4,
				"the centre's middle: 7 texels over what is left of 184 points after half a texel each end")
		}
	}
}

func TestTilesLayASliceByEachRepeat(t *testing.T) {
	cases := []struct {
		repeat style.ImageRepeat
		want   [][2]float32
		cut    bool
	}{
		{style.RepeatStretch, [][2]float32{{0, 25}}, false},
		// One copy centred on 12.5, then whole copies out to both ends.
		{style.RepeatTile, [][2]float32{{-2.5, 7.5}, {7.5, 17.5}, {17.5, 27.5}}, true},
		// 2.5 copies round half away from zero, as Blink's roundf does.
		{style.RepeatRound, [][2]float32{{0, 25.0 / 3}, {25.0 / 3, 50.0 / 3}, {50.0 / 3, 25}}, false},
		{style.RepeatSpace, [][2]float32{{5.0 / 3, 35.0 / 3}, {40.0 / 3, 70.0 / 3}}, false},
	}
	for _, c := range cases {
		got, cut := tiles(0, 25, 10, c.repeat, nil)
		assert.Equal(t, c.cut, cut, "repeat %d cut", c.repeat)
		assert.Len(t, got, len(c.want), "repeat %d", c.repeat)
		for k := range min(len(got), len(c.want)) {
			assert.InDelta(t, c.want[k][0], got[k][0], 1e-4, "repeat %d copy %d", c.repeat, k)
			assert.InDelta(t, c.want[k][1], got[k][1], 1e-4, "repeat %d copy %d", c.repeat, k)
		}
	}
	none, _ := tiles(0, 5, 10, style.RepeatSpace, nil)
	assert.Empty(t, none, "space draws nothing where one copy won't fit")
	one, _ := tiles(0, 4, 10, style.RepeatRound, nil)
	assert.Equal(t, [][2]float32{{0, 4}}, one, "round keeps at least one copy")
}

func TestANinePatchTilesItsEdgesAndCentreByRepeat(t *testing.T) {
	tex := twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 24, 24)))
	r := newRecorder()
	box := geom.Rect{W: 46, H: 30}
	(&painter{s: r}).ninePatch(box, tex, style.Sides[float32]{Top: 8, Right: 8, Bottom: 8, Left: 8},
		[2]style.ImageRepeat{style.RepeatTile, style.RepeatRound}, color.White)
	top := 0
	for _, o := range r.of("image") {
		if o.Rect.Y >= 8 || o.Rect.X+o.Rect.W <= 8 || o.Rect.X >= 38 {
			continue
		}
		// The top edge: 30 points of 8-point copies, centred: cut at both ends.
		top++
		assert.Equal(t, geom.Rect{X: 8, Y: 0, W: 30, H: 8}, o.Clip, "copies cut by the edge's ends")
	}
	assert.Positive(t, top)
	sum := float32(0)
	for _, o := range r.of("image") {
		// The left edge's first span across, once per span down.
		if o.Rect.X == 0 && o.Rect.Y >= 8 && o.Rect.Y < 22 {
			sum += o.Rect.H
		}
	}
	assert.InDelta(t, 14, sum, 1e-3, "the left edge rounds: whole copies fill its 14 points exactly")
}
