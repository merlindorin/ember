package ui

import (
	"fmt"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAMachineMovedWithoutTheDomRepaints(t *testing.T) {
	box := Input(Type("checkbox"))
	sheet := fstest.MapFS{"a.css": {Data: []byte("input:checked { background-color: #ff0000; }")}}
	d := NewDocument(Stylesheet(sheet, "a.css"))
	d.Body().Append(box)
	win := platform.NewWindow(platform.Size(200, 100))
	win.Add(d)
	w := ember.New().AddEntities(win)
	win.Input().SetMouse(geom.V2(190, 90))
	w.RunFrames(4)
	require.True(t, d.replay.recorded)

	m, ok := d.machine(box).(*control.Checkbox)
	require.True(t, ok)
	m.Checked = true
	repaints := func() int {
		n := 0
		for _, c := range w.Stats().Counted() {
			if c.Name != replayed {
				n += c.N
			}
		}
		return n
	}
	before := repaints()
	w.RunFrames(1)
	assert.Equal(t, before+1, repaints(), "the sync dirtied the tree after the check")
}

// quietDoc is a Document in a 300×200 window, run until its frames replay.
func quietDoc(t *testing.T, at geom.Vec2, nodes ...*Node) (*Document, *platform.Window, func(int)) {
	t.Helper()
	d := NewDocument()
	d.Body().Append(nodes...)
	win := platform.NewWindow(platform.Size(300, 200))
	win.Add(d)
	w := ember.New().AddEntities(win)
	win.Input().SetMouse(at)
	w.RunFrames(4)
	require.True(t, d.replay.replayedLast, "still frames replay")
	return d, win, w.RunFrames
}

func TestContentScrolledUnderAStillPointerTakesTheHover(t *testing.T) {
	sc := Div(Style("height: 100px; overflow: auto"))
	var rows []*Node
	for i := range 20 {
		r := Div(Style("height: 20px"), Text(fmt.Sprint(i)))
		rows = append(rows, r)
		sc.Append(r)
	}
	d, win, frames := quietDoc(t, geom.V2(50, 50), sc)
	require.True(t, rows[2].State().Has(dom.Hover))

	dom.Data(sc).Scroll = geom.V2(0, 40)
	d.scrolled = true
	frames(1)
	assert.True(t, rows[4].State().Has(dom.Hover), "the row now under the pointer, that frame")
	assert.False(t, rows[2].State().Has(dom.Hover))

	frames(3)
	win.Input().AddWheel(geom.V2(0, 1))
	frames(2)
	assert.False(t, rows[4].State().Has(dom.Hover), "a wheel turn moves it on too, the frame after it scrolls")
}

func TestAHiddenButtonLeavesTheTabOrderTheNextFrame(t *testing.T) {
	a, b, c := Button(Text("a")), Button(Text("b")), Button(Text("c"))
	d, win, frames := quietDoc(t, geom.V2(290, 190), a, b, c)
	d.Focus(a)
	frames(3)
	b.SetStyle("display: none")
	frames(1)
	win.Input().SetKey(input.KeyTab, true)
	frames(1)
	win.Input().SetKey(input.KeyTab, false)
	frames(1)
	assert.True(t, d.Focused(c), "Tab skips the hidden button")
}

func TestAPinnedStateStillReplays(t *testing.T) {
	b := Button(Text("Save"))
	d := NewDocument()
	d.Body().Append(b)
	d.pinned = map[*Node]dom.State{b: dom.Hover | dom.FocusVisible}
	win := platform.NewWindow(platform.Size(300, 200))
	win.Add(d)
	w := ember.New().AddEntities(win)
	win.Input().SetMouse(geom.V2(290, 190))
	w.RunFrames(4)
	assert.True(t, b.State().Has(dom.Hover|dom.FocusVisible))
	assert.True(t, d.replay.replayedLast, "a pin held through states() leaves the tree clean")
}

func TestALiveFrameDrawsWhatAWholePaintWould(t *testing.T) {
	in := Input(Value("Heron"))
	d := NewDocument()
	d.Body().Append(in)
	win := platform.NewWindow(platform.Size(300, 200))
	win.Add(d)
	w := ember.New().AddEntities(win)
	win.Input().SetMouse(geom.V2(290, 190))
	frames := w.RunFrames
	frames(4)
	d.Focus(in)
	frames(3)
	for range 20 {
		frames(1)
		require.False(t, d.replay.inPlace)
		live := slices.Clone(draw.FrontFor(w, win).Vertices)
		clock := d.clock
		d.replay.repaint = true
		frames(1)
		if int(clock*2)%2 != int(d.clock*2)%2 {
			continue
		}
		assert.Equal(t, draw.FrontFor(w, win).Vertices, live, "the caret on top of the replay, where a whole paint draws it")
		return
	}
	t.Fatal("no two frames in one blink")
}

func TestAFocusedFieldsFrameAllocatesNothingFromUI(t *testing.T) {
	in := Input(Value("Heron"))
	d, _, frames := quietDoc(t, geom.V2(290, 190), in)
	d.Focus(in)
	frames(3)
	_, _, empty := quietDoc(t, geom.V2(290, 190), Div())
	world := testing.AllocsPerRun(50, func() { empty(1) })
	assert.Equal(t, world, testing.AllocsPerRun(50, func() { frames(1) }), "only the World's own")
}

func TestHoverLeavesTheElementsItLeftAndTouchesNoOthers(t *testing.T) {
	a, b := Button(Text("A")), Button(Text("B"))
	d, win, frames := quietDoc(t, geom.V2(290, 190), Div(a, b))
	win.Input().SetMouse(a.Box().Center())
	frames(2)
	require.True(t, a.State().Has(dom.Hover))
	assert.False(t, b.State().Has(dom.Hover))

	win.Input().SetMouse(b.Box().Center())
	frames(2)
	assert.False(t, a.State().Has(dom.Hover), "the element left loses its hover")
	assert.True(t, b.State().Has(dom.Hover))
	assert.LessOrEqual(t, len(d.stated), 8, "states() keeps only the chains, not the tree")
}

func TestArrowsReachWhatAScrollBroughtBeside(t *testing.T) {
	from := Button(Text("from"), Style("position: absolute; left: 0; top: 10px; width: 60px; height: 20px"))
	near, far := Button(Text("near"), Style("display: block; height: 20px")), Button(Text("far"),
		Style("display: block; height: 20px; margin-top: 200px"))
	sc := Div(Style("position: absolute; left: 100px; top: 0; width: 100px; height: 100px; overflow: auto"), near, far)
	d, win, frames := quietDoc(t, geom.V2(290, 190), from, sc)
	d.Focus(from)
	frames(2)

	dom.Data(sc).Scroll = geom.V2(0, 220)
	d.scrolled = true
	frames(2)
	win.Input().SetKey(input.KeyRight, true)
	frames(1)
	win.Input().SetKey(input.KeyRight, false)
	frames(1)
	assert.True(t, d.Focused(far), "the button the scroll brought beside, not the one it took away")
}

func TestAnElementGivenATabindexJoinsTheTabOrder(t *testing.T) {
	a, card := Button(Text("a")), Div(Text("card"), Style("height: 20px"))
	d, win, frames := quietDoc(t, geom.V2(290, 190), a, card)
	d.Focus(a)
	frames(3)
	card.SetAttr("tabindex", "0")
	frames(2)
	win.Input().SetKey(input.KeyTab, true)
	frames(1)
	win.Input().SetKey(input.KeyTab, false)
	frames(1)
	assert.True(t, d.Focused(card), "Tab reaches it, with nothing else changed")
}

func TestARepaintPassesItsFrameWithoutMakingOptions(t *testing.T) {
	d, _, frames := quietDoc(t, geom.V2(290, 190), Div(Text("a")))
	frames(2)
	opts := d.frameOptions()
	d.replay.repaint = true
	frames(1)
	assert.Same(t, &opts[0], &d.frameOptions()[0], "the same options, frame after frame")
	assert.Equal(t, d.clock, d.replay.frame.Time, "with this frame's clock")
}
