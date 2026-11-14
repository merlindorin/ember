package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/lantervale/ember/render"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnImageFillsItsContentBoxByDefault(t *testing.T) {
	b := ctl(dom.Img, 100, 50, "src", "arrow")
	r := pictures(b)
	imgs := r.of("image")
	require.Len(t, imgs, 1)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 100, H: 50}, imgs[0].Rect)
}

func TestAnImageWithoutAPictureDrawsAsChromesBrokenImage(t *testing.T) {
	broken := func(attrs ...string) *recorder {
		b := ctl(dom.Img, 100, 50, append([]string{"src", "missing"}, attrs...)...)
		r := newRecorder()
		Paint(r, tree(el(0, 0, 400, 300, nil, b)), Images(func(string) *twod.Texture { return nil }))
		return r
	}
	r := broken()
	assert.Empty(t, r.of("image"), "no picture")
	frames := r.of("border")
	require.NotEmpty(t, frames)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 100, H: 50}, frames[0].Rect,
		"without alt text, a silver frame round the box")
	assert.Equal(t, brokenFrame, frames[0].Color)
	assert.Equal(t, geom.Rect{X: 12, Y: 12, W: BrokenIcon, H: BrokenIcon}, frames[1].Rect, "and the icon inside it")

	r = broken("alt", "A crypt door")
	texts := r.of("text")
	require.Len(t, texts, 1)
	assert.Equal(t, "A crypt door", texts[0].Text, "with alt text, the text")
	assert.Equal(t, geom.V2(10+BrokenIcon, 10), texts[0].At, "after the icon at the top-left")
	assert.NotEqual(t, brokenFrame, r.of("border")[0].Color, "and no frame")

	r = broken("alt", "")
	assert.Empty(t, r.ops, "an empty alt draws nothing")
}

func TestObjectFitPlacesTheNaturalPicture(t *testing.T) {
	box := geom.Rect{X: 10, Y: 10, W: 100, H: 50}
	natural := geom.V2(10, 6)
	centre := style.BackgroundPosition{X: style.Percent(50), Y: style.Percent(50)}
	for _, c := range []struct {
		fit  style.ObjectFit
		want geom.Rect
		clip bool
	}{
		{style.FitFill, box, false},
		{style.FitContain, geom.Rect{X: 10 + (100-50/6.0*10)/2, Y: 10, W: 50 / 6.0 * 10, H: 50}, false},
		{style.FitCover, geom.Rect{X: 10, Y: 10 + (50-60)/2, W: 100, H: 60}, true},
		{style.FitNone, geom.Rect{X: 10 + 45, Y: 10 + 22, W: 10, H: 6}, false},
		{style.FitScaleDown, geom.Rect{X: 10 + 45, Y: 10 + 22, W: 10, H: 6}, false},
	} {
		got, clip := objectFit(c.fit, box, natural, centre)
		assert.InDelta(t, c.want.X, got.X, 1e-3, "fit %d", c.fit)
		assert.InDelta(t, c.want.Y, got.Y, 1e-3, "fit %d", c.fit)
		assert.InDelta(t, c.want.W, got.W, 1e-3, "fit %d", c.fit)
		assert.InDelta(t, c.want.H, got.H, 1e-3, "fit %d", c.fit)
		assert.Equal(t, c.clip, clip, "fit %d", c.fit)
	}
	small, _ := objectFit(style.FitScaleDown, geom.Rect{W: 5, H: 3}, natural, centre)
	assert.InDelta(t, 5, small.W, 1e-3, "scale-down shrinks like contain when the picture is bigger")
	big, clip := objectFit(style.FitNone, geom.Rect{W: 5, H: 3}, natural, centre)
	assert.True(t, clip, "none overflowing a small box is clipped")
	assert.Equal(t, float32(10), big.W)
	topLeft, _ := objectFit(style.FitNone, box, natural, style.BackgroundPosition{X: style.Px(0), Y: style.Px(0)})
	assert.Equal(t, geom.V2(10, 10), geom.V2(topLeft.X, topLeft.Y), "object-position 0 0")
	right, _ := objectFit(style.FitNone, box, natural,
		style.BackgroundPosition{X: style.Px(4), Y: style.Percent(50), FromRight: true})
	assert.Equal(t, float32(10+100-4-10), right.X, "right 4px")
}

func TestAnImageFollowsItsObjectFit(t *testing.T) {
	b := ctl(dom.Img, 100, 50, "src", "arrow")
	b.Style.ObjectFit = style.FitNone
	b.Style.ObjectPosition = style.BackgroundPosition{X: style.Percent(50), Y: style.Percent(50)}
	imgs := pictures(b).of("image")
	require.Len(t, imgs, 1)
	assert.Equal(t, geom.Rect{X: 10 + 45, Y: 10 + 22, W: 10, H: 6}, imgs[0].Rect, "its natural 10×6, centred")
}

func TestPixelatedPicturesSampleTheirNearestPixels(t *testing.T) {
	b := ctl(dom.Img, 100, 50, "src", "arrow")
	smooth := pictures(b).of("image")[0].Tex
	assert.Same(t, arrow, smooth, "auto: the texture as given")

	b.Style.ImageRendering = style.RenderPixelated
	sharp := pictures(b).of("image")[0].Tex
	assert.NotSame(t, arrow, sharp)
	assert.Equal(t, render.FilterNearest, sharp.Filter)
	assert.Equal(t, render.FilterLinear, arrow.Filter, "the given texture untouched")
	assert.Same(t, arrow.Image, sharp.Image, "the same picture")
	assert.Same(t, sharp, pictures(b).of("image")[0].Tex, "one sharp copy, kept")
}

func TestABackgroundImageIsPixelatedToo(t *testing.T) {
	bg := withImage(40, 40, func(s *style.Computed) { s.ImageRendering = style.RenderPixelated })
	assert.Equal(t, render.FilterNearest, pictures(bg).of("image")[0].Tex.Filter)
}

func TestATouchedPictureReachesItsSharpCopy(t *testing.T) {
	b := ctl(dom.Img, 100, 50, "src", "arrow")
	b.Style.ImageRendering = style.RenderPixelated
	before := pictures(b).of("image")[0].Tex.Version()
	arrow.Touch()
	assert.Greater(t, pictures(b).of("image")[0].Tex.Version(), before)
}
