package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func canvas(attrs map[string]string, edit func(*style.Computed)) *dom.Node {
	n := el(dom.Canvas, func(st *style.Computed) {
		st.Display = style.InlineBlock
		if edit != nil {
			edit(st)
		}
	})
	for k, v := range attrs {
		n.SetAttr(k, v)
	}
	return n
}

func layCanvas(c *dom.Node) geom.Vec2 {
	root := el(dom.Div, nil, c)
	return Layout(root, geom.V2(400, 300), ahemFonts{}).Of[c].Size
}

func TestACanvasIs300By150WithoutAttributes(t *testing.T) {
	assert.Equal(t, geom.V2(300, 150), layCanvas(canvas(nil, nil)), "as in HTML, with no images known")
}

func TestACanvasTakesItsWidthAndHeightAttributes(t *testing.T) {
	assert.Equal(t, geom.V2(120, 80), layCanvas(canvas(map[string]string{"width": "120", "height": "80"}, nil)))
	assert.Equal(t, geom.V2(120, 150),
		layCanvas(canvas(map[string]string{"width": "120"}, nil)), "the other side's default")
	assert.Equal(t, geom.V2(300, 150), layCanvas(canvas(map[string]string{"width": "wide"}, nil)),
		"a value that isn't a number is the default")
}

func TestCSSSizesACanvasKeepingItsRatio(t *testing.T) {
	c := canvas(map[string]string{"width": "120", "height": "80"}, func(s *style.Computed) {
		s.Size.Width = style.Px(60)
	})
	assert.Equal(t, geom.V2(60, 40), layCanvas(c))
}

func TestARatioHoldsBetweenContentBoxes(t *testing.T) {
	c := canvas(map[string]string{"width": "120", "height": "60"}, func(s *style.Computed) {
		s.Size.Width = style.Px(60)
		s.Padding = style.All(style.Px(4))
		s.Border = style.All[float32](2)
	})
	assert.Equal(t, geom.V2(60, 36), layCanvas(c), "border-box: 60 wide, 48 by 24 inside, padding and border around")
}

func TestAnImagesRatioHoldsBetweenContentBoxesToo(t *testing.T) {
	i := img("tile.png", func(s *style.Computed) {
		s.Size.Width = style.Px(60)
		s.Padding = style.All(style.Px(4))
		s.Border = style.All[float32](2)
	})
	assert.Equal(t, geom.V2(60, 36), layImages(el(dom.P, block, i))[i])
}
