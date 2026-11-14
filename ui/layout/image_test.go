package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

type pictures map[string]geom.Vec2

func (p pictures) ImageSize(url string) (geom.Vec2, bool) {
	s, ok := p[url]
	return s, ok
}

func img(src string, edit func(*style.Computed)) *dom.Node {
	n := el(dom.Img, func(st *style.Computed) {
		st.Display = style.InlineBlock
		if edit != nil {
			edit(st)
		}
	})
	n.SetAttr("src", src)
	return n
}

func layImages(root *dom.Node) map[*dom.Node]geom.Vec2 {
	tr := Layout(root, geom.V2(400, 300), ahemFonts{}, Images(pictures{"tile.png": geom.V2(40, 20)}))
	out := map[*dom.Node]geom.Vec2{}
	root.Walk(func(n *dom.Node) bool {
		if b := tr.Of[n]; b != nil {
			out[n] = b.Size
		}
		return true
	})
	return out
}

func TestAnImageTakesItsNaturalSize(t *testing.T) {
	i := img("tile.png", nil)
	assert.Equal(t, geom.V2(40, 20), layImages(el(dom.P, block, i))[i])
}

func TestOneSideKeepsTheAspectRatio(t *testing.T) {
	w := img("tile.png", func(st *style.Computed) { st.Size.Width = style.Px(80) })
	h := img("tile.png", func(st *style.Computed) { st.Size.Height = style.Px(10) })
	sizes := layImages(el(dom.P, block, w, h))
	assert.Equal(t, geom.V2(80, 40), sizes[w])
	assert.Equal(t, geom.V2(20, 10), sizes[h])
}

func TestBothSidesStretchIt(t *testing.T) {
	i := img("tile.png", func(st *style.Computed) {
		st.Size = style.Size[style.Length]{Width: style.Px(30), Height: style.Px(30)}
	})
	assert.Equal(t, geom.V2(30, 30), layImages(el(dom.P, block, i))[i])
	flat := img("tile.png", func(st *style.Computed) {
		st.Size = style.Size[style.Length]{Width: style.Px(60), Height: style.Px(10)}
	})
	assert.Equal(t, geom.V2(60, 10), layImages(el(dom.P, block, flat))[flat],
		"the height given wins over the picture's ratio")
}

func TestMaxWidthScalesItDown(t *testing.T) {
	i := img("tile.png", func(st *style.Computed) { st.MaxSize.Width = style.Px(20) })
	assert.Equal(t, geom.V2(20, 10), layImages(el(dom.P, block, i))[i])
}

func TestAMissingImageIsSizedAsChromesBrokenImage(t *testing.T) {
	bare := img("nope.png", nil)
	assert.Equal(t, geom.V2(brokenIcon, brokenIcon), layImages(el(dom.P, block, bare))[bare], "its icon alone")
	named := img("nope.png", nil)
	named.SetAttr("alt", "door")
	got := layImages(el(dom.P, block, named))[named]
	assert.Equal(t, float32(brokenIcon+ahemFonts{}.Advance(styleOf(named).Font, "door")),
		got.X, "its icon and its alt text")
	assert.GreaterOrEqual(t, got.Y, float32(brokenIcon))
	empty := img("nope.png", nil)
	empty.SetAttr("alt", "")
	assert.Equal(t, geom.V2(0, 0), layImages(el(dom.P, block, empty))[empty], "an empty alt shows nothing")
	sized := img("nope.png", func(st *style.Computed) { st.Size.Width, st.Size.Height = style.Px(60), style.Px(40) })
	assert.Equal(t, geom.V2(60, 40), layImages(el(dom.P, block, sized))[sized], "a size it's given holds")
}

func TestAnImageInAFlexRow(t *testing.T) {
	i := img("tile.png", func(st *style.Computed) { st.Size.Height = style.Px(30) })
	row := el(dom.Div, func(st *style.Computed) { st.Display = style.FlexDisplay; st.AlignItems = style.AlignStart }, i)
	assert.Equal(t, geom.V2(60, 30), layImages(el(dom.Div, block, row))[i])
}
