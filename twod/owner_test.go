package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/inputfeed"
	"github.com/merlindorin/ember/internal/wm"
	"github.com/merlindorin/ember/node"
)

type inputProbe struct {
	Item
	hovered, key, clicked bool
	mouse, canvasMouse    geom.Vec2
	drawing               node.Noded
	interacted            bool
}

func (p *inputProbe) Draw(c *Canvas) {
	in := c.Input()
	p.hovered = c.Hovered(geom.Rect{X: -1000, Y: -1000, W: 2000, H: 2000})
	p.key = inputfeed.Of(in).RawPressed(int(input.KeyA))
	p.clicked = in.Pressed(input.MouseLeft)
	p.mouse = in.Mouse()
	p.canvasMouse = c.Mouse()
	p.interacted = c.Interact(geom.Rect{X: -1000, Y: -1000, W: 2000, H: 2000}, "probe").Held
	p.drawing = c.Drawing()
}

func TestOnlyTheOwnerOfAWindowTakesItsInput(t *testing.T) {
	w, win, _ := app()
	shell, modal := &inputProbe{Item: Base()}, &inputProbe{Item: Base()}
	win.Add(Build(shell, "shell"), Build(modal, "modal"))
	w.RunFrames(1)
	assert.Equal(t, node.Noded(modal), modal.drawing, "the canvas says what it draws")

	wm.Own(w, win, modal)
	win.Input().SetMouse(geom.V2(10, 10))
	win.Input().SetMouseButton(input.MouseLeft, true)
	win.Input().SetKey(input.KeyA, true)
	w.RunFrames(1)
	assert.True(t, modal.hovered && modal.key && modal.clicked, "the owner hovers, gets keys and clicks")
	assert.False(
		t,
		shell.hovered || shell.key || shell.clicked,
		"what is under the owner neither hovers nor gets keys or clicks",
	)
	assert.NotEqual(t, geom.V2(10, 10), shell.mouse, "nor sees the pointer")

	wm.Release(w, win, modal)
	win.Input().SetMouseButton(input.MouseLeft, false)
	win.Input().SetKey(input.KeyA, false)
	w.RunFrames(1)
	win.Input().SetMouseButton(input.MouseLeft, true)
	win.Input().SetKey(input.KeyA, true)
	w.RunFrames(1)
	assert.True(t, shell.hovered && shell.key && shell.clicked, "released, the window's input is everyone's again")
}

func TestWhatALayerCoversTakesNoPointerTheNextFrameButKeepsItsKeys(t *testing.T) {
	w, win, _ := app()
	widget, menu := &inputProbe{Item: Base()}, &inputProbe{Item: Base()}
	win.Add(Build(widget, "widget"), Build(menu, "menu"))
	win.Input().SetMouse(geom.V2(10, 10))
	w.RunFrames(1)
	wm.Cover(w, win, menu)
	win.Input().SetMouseButton(input.MouseLeft, true)
	win.Input().SetKey(input.KeyA, true)
	w.RunFrames(1)
	assert.False(t, widget.hovered || widget.interacted, "under the cover the widget neither hovers nor presses")
	assert.NotEqual(t, geom.V2(10, 10), widget.canvasMouse, "nor sees the pointer")
	assert.True(t, widget.key, "but keeps its keys")
	assert.True(t, menu.hovered, "the cover keeps the pointer")
	win.Input().SetMouseButton(input.MouseLeft, false)
	w.RunFrames(1)
	assert.True(t, widget.hovered, "not covered again, the widget has the pointer back")
}
