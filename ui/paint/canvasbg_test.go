package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/control"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rootPage is html (400×50) holding body (380×30 at 10, 10), coloured as
// given, transparent for the zero colour.
func rootPage(htmlBg, bodyBg color.Color) *box.Tree {
	body := el(10, 10, 380, 30, bg(bodyBg))
	body.Node = dom.NewElement(dom.Body)
	html := el(0, 0, 400, 50, bg(htmlBg), body)
	html.Node = dom.NewElement(dom.HTML)
	return tree(html)
}

func fills(r *recorder) []op {
	var out []op
	for _, o := range r.of("rect", "round") {
		if o.Color.A > 0 {
			out = append(out, o)
		}
	}
	return out
}

func TestTheRootsBackgroundCoversTheWholeCanvas(t *testing.T) {
	r := newRecorder()
	Paint(r, rootPage(red, blue), Viewport(geom.Rect{W: 400, H: 300}))
	f := fills(r)
	require.Len(t, f, 2)
	assert.Equal(t, geom.Rect{W: 400, H: 300}, f[0].Rect, "the canvas, not html's 50 points")
	assert.Equal(t, red, f[0].Color)
	assert.Equal(t, blue, f[1].Color, "then body's own, in its box")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 380, H: 30}, f[1].Rect)
}

func TestWithoutARootBackgroundBodysPropagatesToTheCanvas(t *testing.T) {
	r := newRecorder()
	Paint(r, rootPage(color.Color{}, blue), Viewport(geom.Rect{W: 400, H: 300}))
	f := fills(r)
	require.Len(t, f, 1, "body's box doesn't paint it again")
	assert.Equal(t, geom.Rect{W: 400, H: 300}, f[0].Rect)
	assert.Equal(t, blue, f[0].Color)
}

func TestWithoutAViewportTheCanvasIsTheRootsBox(t *testing.T) {
	r := newRecorder()
	Paint(r, rootPage(red, color.Color{}))
	f := fills(r)
	require.Len(t, f, 1)
	assert.Equal(t, geom.Rect{W: 400, H: 50}, f[0].Rect)
}

func TestAHostedCanvasIsTheHostsArea(t *testing.T) {
	r := newRecorder()
	Paint(r, rootPage(red, color.Color{}), Viewport(geom.Rect{X: 30, Y: 40, W: 200, H: 100}))
	assert.Equal(t, geom.Rect{X: 30, Y: 40, W: 200, H: 100}, fills(r)[0].Rect)
}

func TestTheModalBackdropCoversTheViewport(t *testing.T) {
	b := ctl(dom.Dialog, 100, 60)
	b.Style.Position, b.Style.ZAuto, b.Style.Backdrop = style.FixedPosition, true, backdrop
	m := &control.Dialog{}
	m.ShowModal()
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 50, nil, b)),
		Controls(func(*dom.Node) any { return m }), Viewport(geom.Rect{W: 400, H: 300}))
	assert.Equal(t, geom.Rect{W: 400, H: 300}, r.of("rect")[0].Rect)
}
