package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/stretchr/testify/assert"
)

func TestOptionsReadBackUnderTheirNames(t *testing.T) {
	text := NewText("Hi", Typeface(Bold()), FontSize(20), Wrap(100), TextShadow(color.Black), Pivot(0, 1), Overlay())
	assert.Equal(t, Bold(), text.Typeface())
	assert.Equal(t, float32(20), text.FontSize())
	assert.Equal(t, float32(100), text.Wrap())
	assert.Equal(t, color.Black, text.TextShadow())
	assert.Equal(t, geom.V2(0, 1), text.Pivot())
	assert.True(t, text.Overlay())

	line := NewLine(nil, Closed(), Joints(RoundJoint), Caps(Square, RoundCap))
	start, end := line.Caps()
	assert.True(t, line.Closed())
	assert.Equal(t, RoundJoint, line.Joints())
	assert.Equal(t, Square, start)
	assert.Equal(t, RoundCap, end)

	mask := NewLayer(1, AsMask())
	layer := NewLayer(2, Fixed(), Parallax(0.5), ClipTo(geom.Rect{W: 10, H: 10}), MaskedBy(mask))
	assert.True(t, mask.AsMask())
	assert.True(t, layer.Fixed())
	assert.Equal(t, float32(0.5), layer.Parallax())
	assert.Equal(t, geom.Rect{W: 10, H: 10}, layer.ClipTo())
	assert.Equal(t, mask, layer.MaskedBy())

	cam := NewCamera(
		Smoothing(4),
		Bounds(geom.Rect{W: 100, H: 50}),
		SnapToPixels(),
		Priority(3),
		Resolution(320, 180),
		Upscale(Linear),
		FitWindow(),
	)
	w, h := cam.Resolution()
	assert.Equal(t, float32(4), cam.Smoothing())
	assert.Equal(t, geom.Rect{W: 100, H: 50}, cam.Bounds())
	assert.True(t, cam.SnapToPixels())
	assert.Equal(t, 3, cam.Priority())
	assert.Equal(t, [2]int{320, 180}, [2]int{w, h})
	assert.Equal(t, Linear, cam.Upscale())
	assert.True(t, cam.FitWindow())

	s := NewSprite(nil, Flip(true, false), Pivot(0, 0), Speed(2))
	x, _ := s.Flip()
	assert.True(t, x)
	assert.Equal(t, geom.Vec2{}, s.Pivot())
	assert.Equal(t, float32(2), s.Speed())
}

func TestSettersChangeWhatIsDrawn(t *testing.T) {
	w, win, c := app()
	text := NewText("Hi", At(100, 100), Pivot(0, 0))
	win.Add(text)
	w.AddEntities(text)
	w.RunFrames(1)
	small := len(c.Unindexed())
	text.SetShadow(color.Black)
	w.RunFrames(1)
	assert.Equal(t, 2*small, len(c.Unindexed()), "the shadow draws the glyphs again")
	text.SetHidden(true)
	w.RunFrames(1)
	assert.Empty(t, c.Unindexed())
}
