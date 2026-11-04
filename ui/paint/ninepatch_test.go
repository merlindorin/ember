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
		style.Sides[float32]{Top: 8, Right: 8, Bottom: 8, Left: 8}, color.White)
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
		style.Sides[float32]{Top: 8, Right: 8, Bottom: 8, Left: 8}, color.White)
	for _, o := range r.of("image") {
		if o.Rect.X > 8 && o.Rect.X+o.Rect.W < 192 && o.Rect.Y > 8 && o.Rect.Y+o.Rect.H < 52 {
			assert.InDelta(t, 7.0/(184-184.0/8), o.UV.W*24/o.Rect.W, 1e-4,
				"the centre's middle: 7 texels over what is left of 184 points after half a texel each end")
		}
	}
}
