package ui_test

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/twod"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/stretchr/testify/assert"
)

func TestADocumentInAWindowLaysOutAtTheWindowsSize(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 360))
	doc := ui.NewDocument()
	card := ui.Div(ui.Style("height: 50px; margin: 10px"))
	doc.Body().Append(card)
	win.Add(doc)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)

	assert.Equal(t, geom.Rect{X: 18, Y: 10, W: 604, H: 50}, card.Box(),
		"inside body's 8px margin, its top margin collapsing with the card's")

	win.SetSize(geom.V2(800, 360))
	w.RunFrames(1)
	assert.Equal(t, float32(764), card.Box().W)
}

func TestTransitionsRunOnTheWorldsRealTime(t *testing.T) {
	win := platform.NewWindow(platform.Size(320, 200))
	doc := ui.NewDocument()
	card := ui.Div(ui.Style("height: 20px; background-color: #000000; transition: background-color 1s linear"))
	doc.Body().Append(card)
	win.Add(doc)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)
	w.Time().Scale = 0

	card.SetStyle("height: 20px; background-color: #ffffff; transition: background-color 1s linear")
	w.RunFrames(30)
	mid := ui.ShownBackground(card)
	assert.Greater(t, mid.R, float32(0.05))
	assert.Less(t, mid.R, float32(0.95))

	w.RunFrames(60)
	assert.InDelta(t, 1, ui.ShownBackground(card).R, 0.001)
}

// host is a game's own 2D entity drawing a Document in a box of its own.
type host struct {
	twod.Item
	doc  *ui.Document
	size geom.Vec2
}

func (h *host) BeginDraw(c *twod.Canvas) func() {
	c.SetSize(h.size)
	return func() {}
}

func (h *host) Draw(c *twod.Canvas) { h.doc.Draw(c) }

func TestA2DEntityHostsADocumentInItsOwnSpace(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 360))
	doc := ui.NewDocument()
	card := ui.Div(ui.ID("card"), ui.Style("height: 20px; margin: 4px"))
	doc.Body().Append(card)
	h := twod.Build(&host{Item: twod.Base(), doc: doc, size: geom.V2(200, 120)}, "host", twod.At(100, 50))
	win.Add(h)
	auto := automation.New(automation.Recording())
	w := ember.New(ember.Automation(auto)).AddEntities(win)
	w.RunFrames(3)

	assert.Equal(t, geom.Rect{X: 12, Y: 8, W: 176, H: 20}, card.Box(), "laid out at the host's width, in its space")
	_, els := auto.Snapshot("", "")
	var got geom.Rect
	for _, e := range els {
		if strings.HasSuffix(e.ID, "card") {
			got = e.Bounds
		}
	}
	assert.Equal(t, geom.Rect{X: 112, Y: 58, W: 176, H: 20}, got, "automation sees it in the window")
}
