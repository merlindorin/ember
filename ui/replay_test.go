package ui_test

import (
	"slices"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const replaySheet = `
button { background-color: #333333; color: #ffffff; }
button:hover { background-color: #aa3333; }
.fade { transition: background-color 1s linear; }
`

type replayed struct {
	w   *world.World
	win *platform.Window
	doc *ui.Document
}

func replayPage(t *testing.T, nodes ...*ui.Node) *replayed {
	t.Helper()
	p := &replayed{doc: ui.NewDocument(ui.Stylesheet(fstest.MapFS{"a.css": {Data: []byte(replaySheet)}}, "a.css")),
		win: platform.NewWindow(platform.Size(400, 300))}
	p.doc.Body().Append(nodes...)
	p.win.Add(p.doc)
	p.w = ember.New().AddEntities(p.win)
	p.win.Input().SetMouse(geom.V2(390, 290))
	p.w.RunFrames(3)
	return p
}

// counts is how many frames replayed and repainted since the last call.
func (p *replayed) frames(n int) (int, int) {
	var replays, repaints int
	before := p.counted()
	p.w.RunFrames(n)
	after := p.counted()
	for name, v := range after {
		if name == "ui replays" {
			replays += v - before[name]
		} else {
			repaints += v - before[name]
		}
	}
	return replays, repaints
}

func (p *replayed) counted() map[string]int {
	out := map[string]int{}
	for _, c := range p.w.Stats().Counted() {
		out[c.Name] = c.N
	}
	return out
}

func (p *replayed) vertices() []draw.Vertex {
	return slices.Clone(draw.FrontFor(p.w, p.win).Vertices)
}

func TestAStillDocumentReplaysWhatItPainted(t *testing.T) {
	p := replayPage(t, ui.Button(ui.Text("Save")), ui.P(ui.Text("Nothing moves here.")))
	painted := p.vertices()
	replays, repaints := p.frames(10)
	assert.Equal(t, 10, replays)
	assert.Zero(t, repaints)
	assert.Equal(t, painted, p.vertices(), "vertex for vertex")
}

func TestAColourOnlyHoverRepaints(t *testing.T) {
	b := ui.Button(ui.Text("Save"))
	p := replayPage(t, b)
	before := p.vertices()
	p.win.Input().SetMouse(b.Box().Center())
	_, repaints := p.frames(2)
	assert.NotZero(t, repaints)
	assert.NotEqual(t, before, p.vertices(), "the hover's background shows")
	replays, _ := p.frames(5)
	assert.Equal(t, 5, replays, "and then it rests")
}

func TestGameChangesRepaint(t *testing.T) {
	in := ui.Input(ui.Value("Heron"))
	p := replayPage(t, in)
	p.frames(2)
	p.doc.SetValue(in, "Crane")
	_, repaints := p.frames(1)
	assert.Equal(t, 1, repaints)
	in.SetStyle("width: 80px")
	_, repaints = p.frames(1)
	assert.Equal(t, 1, repaints)
}

func TestAFocusedFieldReplaysAndDrawsItsCaretOnTop(t *testing.T) {
	in := ui.Input(ui.Value("Heron"))
	p := replayPage(t, in)
	p.doc.Focus(in)
	p.frames(3)
	before := p.counted()["ui replays + live"]
	p.w.RunFrames(5)
	assert.Equal(t, 5, p.counted()["ui replays + live"]-before, "the rest replayed, the caret drawn again")
}

func TestAPopoverOverAFocusedFieldRepaintsItWhole(t *testing.T) {
	in := ui.Input(ui.Value("Heron"))
	pop := ui.Div(ui.Style("position: absolute; left: 0; top: 0; width: 200px; height: 60px; background-color: #222222"))
	p := replayPage(t, in, pop)
	p.doc.Focus(in)
	p.frames(3)
	before := p.counted()
	p.w.RunFrames(5)
	after := p.counted()
	assert.Equal(t, before["ui replays + live"], after["ui replays + live"], "never drawn on top of what covers it")
	assert.Equal(t, 5, after["ui repaints: live"]-before["ui repaints: live"])
}

func TestInputRepaints(t *testing.T) {
	sc := ui.Div(ui.Style("height: 100px; overflow: auto"), ui.Div(ui.Style("height: 1000px")))
	p := replayPage(t, sc)
	p.win.Input().SetMouse(geom.V2(50, 50))
	p.frames(3)
	p.win.Input().AddWheel(geom.V2(0, 2))
	_, repaints := p.frames(1)
	assert.Equal(t, 1, repaints)
	p.win.Input().SetKey(input.KeyTab, true)
	_, repaints = p.frames(1)
	assert.Equal(t, 1, repaints)
}

func TestARunningTransitionRepaintsUntilItEnds(t *testing.T) {
	b := ui.Button(ui.Class("fade"), ui.Text("Save"))
	p := replayPage(t, b)
	b.SetStyle("background-color: #0000ff")
	replays, _ := p.frames(5)
	assert.Zero(t, replays)
}

func TestAResizedWindowRepaints(t *testing.T) {
	p := replayPage(t, ui.P(ui.Text("x")))
	p.frames(2)
	p.win.SetSize(geom.V2(500, 300))
	_, repaints := p.frames(1)
	require.Equal(t, 1, repaints)
}

func TestAZoomedDocumentDrawsAtTheZoomedSize(t *testing.T) {
	box := ui.Div(ui.Style("position: absolute; left: 10px; top: 20px; width: 30px; height: 40px; " +
		"background-color: #ff0000"))
	p := replayPage(t, box)
	p.doc.SetZoom(2)
	p.frames(2)
	lo, hi := geom.V2(1e9, 1e9), geom.V2(-1e9, -1e9)
	for _, v := range p.vertices() {
		if v.Color.G == 0 && v.Color.B == 0 && v.Color.R > 0.9 {
			lo = geom.V2(min(lo.X, v.Pos.X), min(lo.Y, v.Pos.Y))
			hi = geom.V2(max(hi.X, v.Pos.X), max(hi.Y, v.Pos.Y))
		}
	}
	assert.Equal(t, geom.V2(20, 40), lo)
	assert.Equal(t, geom.V2(80, 120), hi)
}
