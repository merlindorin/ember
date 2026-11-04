package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"image"
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// arrow is a 10×6 picture, like the select's.
//
//nolint:gochecknoglobals // the tests' fixed values; Go has no constant colours or pictures
var arrow = twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 10, 6)))

func pictures(b *box.Box) *recorder {
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, b)), Images(func(string) *twod.Texture { return arrow }))
	return r
}

func withImage(w, h float32, edit func(s *style.Computed)) *box.Box {
	return el(10, 10, w, h, func(s *style.Computed) {
		s.BackgroundImage = style.Image{Source: "arrow"}
		s.BackgroundNoRepeat = true
		if edit != nil {
			edit(s)
		}
	})
}

func TestABackgroundImageDrawsAtItsOwnSizeAtTheTopLeft(t *testing.T) {
	r := pictures(withImage(120, 24, nil))
	imgs := r.of("image")
	require.Len(t, imgs, 1)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 10, H: 6}, imgs[0].Rect, "never stretched over the box")
}

func TestABackgroundImageIsPlacedFromTheRightAndCentred(t *testing.T) {
	r := pictures(withImage(120, 24, func(s *style.Computed) {
		s.BackgroundPosition = style.BackgroundPosition{X: style.Px(6), Y: style.Percent(50), FromRight: true}
	}))
	assert.Equal(t, geom.Rect{X: 10 + 120 - 6 - 10, Y: 10 + (24-6)/2, W: 10, H: 6}, r.of("image")[0].Rect,
		"right 6px center, the select's arrow")
}

func TestABackgroundImageIsPlacedInThePaddingBox(t *testing.T) {
	b := withImage(120, 24, func(s *style.Computed) {
		s.BackgroundPosition = style.BackgroundPosition{X: style.Percent(100), Y: style.Px(0), FromBottom: true}
	})
	b.Border = box.Edges{Top: 2, Right: 2, Bottom: 2, Left: 2}
	assert.Equal(t, geom.Rect{X: 10 + 2 + 116 - 10, Y: 10 + 2 + 20 - 6, W: 10, H: 6}, pictures(b).of("image")[0].Rect,
		"100% is the right edge, 0 from the bottom is the bottom edge, both inside the border")
}

func TestARepeatedBackgroundImageTilesThePaddingBoxClipped(t *testing.T) {
	b := withImage(25, 10, func(s *style.Computed) { s.BackgroundNoRepeat = false })
	r := pictures(b)
	imgs := r.of("image")
	require.Len(t, imgs, 6, "three across, two down")
	for _, o := range imgs {
		assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 25, H: 10}, o.Clip)
		assert.Equal(t, float32(10), o.Rect.W)
	}
	assert.Equal(t, geom.V2(30, 16), geom.V2(imgs[5].Rect.X, imgs[5].Rect.Y))
}

func TestARepeatedImageTilesBothWaysFromItsPosition(t *testing.T) {
	b := withImage(25, 10, func(s *style.Computed) {
		s.BackgroundNoRepeat = false
		s.BackgroundPosition = style.BackgroundPosition{X: style.Px(4), Y: style.Px(0)}
	})
	xs := map[float32]bool{}
	for _, o := range pictures(b).of("image") {
		xs[o.Rect.X] = true
	}
	assert.True(t, xs[10+4-10], "a tile before the placed one fills the start")
	assert.True(t, xs[10+4], "the placed one")
}

func TestASelectDrawsNoBuiltInArrow(t *testing.T) {
	assert.Empty(t, paintOne(ctl(dom.Select, 120, 24)).of("triangles"), "its arrow is CSS's background image")
}

func TestAPlaceholderTakesItsPlaceholderColour(t *testing.T) {
	b := ctl(dom.Input, 100, 20, "type", "text", "placeholder", "Name")
	shadow := dom.NewText("Name")
	run := text(2, 0, "Name", b.Style)
	run.Node = shadow
	b.Append(run)
	dom.Data(b.Node).Shadow, dom.Data(b.Node).Placeholder = shadow, true
	grey := color.Hex("#757575")
	b.Style.PlaceholderColor, b.Style.HasPlaceholder = grey, true
	assert.Equal(t, grey, paintOne(b).of("text")[0].Color)

	b.Style.HasPlaceholder = false
	assert.InDelta(t, 0.54, paintOne(b).of("text")[0].Color.A, 1e-6, "no ::placeholder rule: 54% of the colour")
}

func TestABackgroundImageTakesItsBackgroundSize(t *testing.T) {
	r := pictures(withImage(120, 24, func(s *style.Computed) { s.BackgroundSize = [2]float32{5, 3} }))
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 5, H: 3}, r.of("image")[0].Rect, "a 2x picture drawn at its point size")
	r = pictures(withImage(120, 24, func(s *style.Computed) { s.BackgroundSize = [2]float32{20, 0} }))
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 20, H: 12}, r.of("image")[0].Rect,
		"one length keeps the picture's proportions")
	r = pictures(withImage(120, 24, func(s *style.Computed) { s.BackgroundSize = [2]float32{0, 3} }))
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 5, H: 3}, r.of("image")[0].Rect, "and so does a height alone")
}
