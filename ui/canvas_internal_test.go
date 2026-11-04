package ui

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestACanvasDrawsAgainOnlyOnceInvalidated(t *testing.T) {
	draws := 0
	cv := Canvas(func(*twod.Canvas) { draws++ }, Attr("width", "80"), Attr("height", "40"))
	d, _, frames := quietDoc(t, geom.V2(290, 190), cv)
	require.Positive(t, draws, "drawn when first painted")
	n := draws
	frames(3)
	assert.Equal(t, n, draws, "replayed frames draw nothing")
	d.Invalidate(cv)
	frames(1)
	assert.Equal(t, n+1, draws, "invalidated, drawn on the next frame")
	assert.True(t, d.replay.replayedLast, "over a replay of the rest")
	frames(2)
	assert.Equal(t, n+1, draws, "and then replayed again")
}

func TestInvalidatingAnythingButACanvasDoesNothing(t *testing.T) {
	div := Div(Text("x"))
	d, _, frames := quietDoc(t, geom.V2(290, 190), div)
	d.Invalidate(div)
	d.Invalidate(nil)
	frames(1)
	assert.True(t, d.replay.replayedLast, "the frame still replays")
}

func TestAHiddenCanvasNeverDraws(t *testing.T) {
	draws := 0
	f := func(*twod.Canvas) { draws++ }
	hidden, closed := Canvas(f, Style("display: none")), Canvas(f)
	d, _, frames := quietDoc(t, geom.V2(290, 190), hidden, Dialog(closed))
	d.Invalidate(hidden)
	d.Invalidate(closed)
	frames(2)
	assert.Zero(t, draws, "neither under display: none nor in a closed dialog")
}

func TestPointerOnIsInTheCanvasContentBox(t *testing.T) {
	cv := Canvas(func(*twod.Canvas) {}, Attr("width", "100"), Attr("height", "50"),
		Style("position: absolute; left: 20px; top: 30px; border: 2px solid; padding: 3px"))
	d, win, frames := quietDoc(t, geom.V2(35, 45), cv)
	at, ok := d.PointerOn(cv)
	assert.True(t, ok)
	assert.Equal(t, geom.V2(10, 10), at, "from the content box's top-left, past border and padding")

	win.Input().SetMouse(geom.V2(250, 150))
	frames(1)
	_, ok = d.PointerOn(cv)
	assert.False(t, ok, "off it")
	_, ok = d.PointerOn(Div())
	assert.False(t, ok, "and never on what isn't a canvas")
}

func TestACoveredCanvasRepaintsTheDocumentOnlyWhenInvalidated(t *testing.T) {
	draws := 0
	cv := Canvas(func(*twod.Canvas) { draws++ }, Attr("width", "80"), Attr("height", "40"))
	over := Div(Style("position: absolute; left: 20px; top: 10px; width: 40px; height: 40px; background-color: #ff0000"))
	d, _, frames := quietDoc(t, geom.V2(290, 190), cv, over)
	n := draws
	frames(3)
	assert.Equal(t, n, draws, "replayed, drawn in its place")
	d.Invalidate(cv)
	frames(1)
	assert.Equal(t, n+1, draws)
	assert.False(t, d.replay.replayedLast, "the box over it is painted again over the new drawing")
	frames(2)
	assert.True(t, d.replay.replayedLast, "and then it replays again")
	assert.Equal(t, n+1, draws)
}

func TestAStillCanvasFrameAllocatesNothingFromUI(t *testing.T) {
	cv := Canvas(func(c *twod.Canvas) { c.Rect(geom.Rect{W: 10, H: 10}, color.Hex("#ff0000")) },
		Attr("width", "80"), Attr("height", "40"))
	_, _, frames := quietDoc(t, geom.V2(290, 190), cv)
	_, _, empty := quietDoc(t, geom.V2(290, 190), Div())
	world := testing.AllocsPerRun(50, func() { empty(1) })
	assert.Equal(t, world, testing.AllocsPerRun(50, func() { frames(1) }), "only the World's own")
}

func TestACanvasStaysDrawnWhenSomethingComesOverIt(t *testing.T) {
	ink := color.Hex("#123456")
	cv := Canvas(func(c *twod.Canvas) { c.Rect(geom.Rect{W: c.Size().X, H: c.Size().Y}, ink) },
		Attr("width", "80"), Attr("height", "40"))
	over := Div(Style("position: absolute; left: 200px; top: 150px; width: 40px; height: 20px; background-color: #ff0000"))
	d := NewDocument()
	d.Body().Append(cv, over)
	win := platform.NewWindow(platform.Size(300, 200))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(4)
	drawn := func() bool {
		for _, v := range draw.FrontFor(w, win).Vertices {
			if v.Color == ink {
				return true
			}
		}
		return false
	}
	require.True(t, drawn(), "drawn while nothing covers it")
	over.SetStyle("position: absolute; left: 20px; top: 10px; width: 40px; height: 20px; background-color: #ff0000")
	for f := range 4 {
		w.RunFrames(1)
		assert.True(t, drawn(), "frame %d after a box came over it", f)
	}
	assert.True(t, d.replay.replayedLast, "and it replays again")
}
