package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/control"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/merlindorin/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// canvasElement is a canvas element at (20, 20), 100 by 60 with a border of 2 and a
// padding of 3, and the machine whose Draw notes what it was given.
func canvasElement(t *testing.T) (*box.Box, *control.Canvas, *[]geom.Vec2) {
	t.Helper()
	b := ctl(dom.Canvas, 100, 60)
	b.Border, b.Padding = box.Edges{Top: 2, Right: 2, Bottom: 2, Left: 2}, box.Edges{Top: 3, Right: 3, Bottom: 3, Left: 3}
	b.Pos = geom.V2(20, 20)
	sizes := &[]geom.Vec2{}
	m := &control.Canvas{Draw: func(c *twod.Canvas) {
		*sizes = append(*sizes, c.Size())
		c.Rect(geom.Rect{X: 0, Y: 0, W: 10, H: 10}, red)
	}}
	return b, m, sizes
}

func TestACanvasDrawsInItsContentBox(t *testing.T) {
	w := world.New()
	win := platform.NewWindow(platform.Size(320, 200))
	w.AddEntities(win)
	w.AddPlugins(twod.Plugin{})
	w.Frame(1.0 / 60)
	c := twod.CanvasFor(w, win)
	require.NotNil(t, c)
	b, m, sizes := canvasElement(t)
	before, xf := c.Size(), c.Transform()

	Paint(c, tree(el(0, 0, 320, 200, nil, b)), Controls(func(*dom.Node) any { return m }))

	assert.Equal(t, []geom.Vec2{geom.V2(90, 50)}, *sizes, "its content box's size, once")
	l := draw.FrontFor(w, win)
	require.NotNil(t, l)
	require.NotEmpty(t, l.Vertices)
	assert.Equal(t, geom.V2(25, 25), l.Vertices[len(l.Vertices)-4].Pos, "its origin at the content box's top-left")
	last := l.Batches[len(l.Batches)-1]
	assert.True(t, last.Clipped)
	assert.Equal(t, geom.Rect{X: 25, Y: 25, W: 90, H: 50}, last.Clip, "clipped to it")
	assert.Equal(t, before, c.Size(), "the canvas's size given back")
	assert.Equal(t, xf, c.Transform(), "and its transform")
}

func TestAHitTestDrawsNoCanvas(t *testing.T) {
	b, m, sizes := canvasElement(t)
	b.Style.Background = white
	got := HitTest(tree(el(0, 0, 320, 200, nil, b)), geom.V2(50, 40), Controls(func(*dom.Node) any { return m }))
	assert.Same(t, b, got, "the canvas, one box")
	assert.Empty(t, *sizes, "and its drawing never asked for")
}

func TestACanvasWithoutDrawingIsItsBoxAlone(t *testing.T) {
	b, _, _ := canvasElement(t)
	b.Style.Background = white
	r := newRecorder()
	Paint(r, tree(b), Controls(func(*dom.Node) any { return &control.Canvas{} }))
	assert.Len(t, r.of("rect"), 1, "its background")
}

// onCanvas is a twod.Canvas on a 320 by 200 window, and its draw list.
func onCanvas(t *testing.T) (*twod.Canvas, func() *draw.List) {
	t.Helper()
	w := world.New()
	win := platform.NewWindow(platform.Size(320, 200))
	w.AddEntities(win)
	w.AddPlugins(twod.Plugin{})
	w.Frame(1.0 / 60)
	c := twod.CanvasFor(w, win)
	require.NotNil(t, c)
	return c, func() *draw.List { return draw.FrontFor(w, win) }
}

func TestACanvasReplaysItsDrawingUntilInvalidated(t *testing.T) {
	c, list := onCanvas(t)
	b, m, sizes := canvasElement(t)
	find := Controls(func(*dom.Node) any { return m })
	page := tree(el(0, 0, 320, 200, nil, b))
	Paint(c, page, find)
	first := len(list().Vertices)
	Paint(c, page, find)
	assert.Len(t, *sizes, 1, "drawn once")
	assert.Equal(t, 2*first, len(list().Vertices), "its drawing replayed the second time")

	m.Invalidate()
	Paint(c, page, find)
	assert.Len(t, *sizes, 2, "drawn again once invalidated")
	b.Pos = geom.V2(40, 20)
	Paint(c, page, find)
	assert.Len(t, *sizes, 3, "and once moved")
}

func TestACanvasIsDrawnApartOverAReplay(t *testing.T) {
	c, list := onCanvas(t)
	b, m, sizes := canvasElement(t)
	find := Controls(func(*dom.Node) any { return m })
	var set LiveSet
	Paint(c, tree(el(0, 0, 320, 200, nil, b)), find, Live(&set, true))
	assert.Empty(t, list().Vertices, "left out of the paint")
	require.True(t, set.Any())
	assert.False(t, set.Covered())
	assert.False(t, set.Timed(), "nothing in it moves with the clock")
	PaintLive(c, &set, 0, find)
	assert.Len(t, *sizes, 1)
	require.NotEmpty(t, list().Vertices)
	assert.Equal(t, geom.V2(25, 25), list().Vertices[0].Pos, "drawn in its content box on top")
	PaintLive(c, &set, 0, find)
	assert.Len(t, *sizes, 1, "replayed on the next")
}

func TestABoxOverACanvasCoversIt(t *testing.T) {
	b, m, _ := canvasElement(t)
	over := el(30, 30, 20, 20, func(s *style.Computed) {
		s.Background, s.Position, s.ZAuto, s.ZIndex = white, style.Relative, false, 1
	})
	var set LiveSet
	Paint(newRecorder(), tree(el(0, 0, 320, 200, nil, b, over)),
		Controls(func(*dom.Node) any { return m }), Live(&set, true))
	assert.True(t, set.Covered())
}
