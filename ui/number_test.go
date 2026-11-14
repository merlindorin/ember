package ui_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/stretchr/testify/assert"
)

func (l *live) typed(s string) {
	in := l.win.Input()
	for _, r := range s {
		in.AddText(r)
	}
	l.frames(2)
}

func (l *live) shifted(k input.Key) {
	in := l.win.Input()
	in.SetKey(input.KeyLeftShift, true)
	l.key(k)
	in.SetKey(input.KeyLeftShift, false)
	l.frames(1)
}

func TestANumberSteps(t *testing.T) {
	n := ui.Input(ui.Type("number"), ui.Attr("min", "0"), ui.Attr("max", "100"), ui.Attr("step", "5"), ui.Value("10"))
	l := liveDoc(t, n)
	l.doc.Focus(n)
	l.frames(1)

	l.key(input.KeyUp)
	assert.Equal(t, "15", l.doc.Value(n), "Up steps by step")
	l.shifted(input.KeyUp)
	assert.Equal(t, "65", l.doc.Value(n), "Shift+Up steps ten steps")
	l.shifted(input.KeyUp)
	assert.Equal(t, "100", l.doc.Value(n), "held at max")
	l.key(input.KeyDown)
	assert.Equal(t, "95", l.doc.Value(n))

	in := l.win.Input()
	in.SetMouse(n.Box().Center())
	l.frames(1)
	in.AddWheel(geom.V2(0, -1))
	l.frames(2)
	assert.Equal(t, "90", l.doc.Value(n), "the wheel over a focused number steps it, down for down")
}

func TestANumberSettlesInItsRangeWhenCommitted(t *testing.T) {
	n := ui.Input(ui.Type("number"), ui.Attr("min", "0"), ui.Attr("max", "100"), ui.Attr("step", "5"), ui.Value("10"))
	after := ui.Button(ui.Text("After"))
	l := liveDoc(t, n, after)
	l.doc.Focus(n)
	l.frames(1)

	l.key(input.KeyBackspace)
	l.key(input.KeyBackspace)
	l.typed("1000")
	l.key(input.KeyEnter)
	assert.Equal(t, "100", l.doc.Value(n), "kept within max")
	assert.Len(t, l.changes, 1)

	l.key(input.KeyBackspace)
	l.key(input.KeyBackspace)
	l.key(input.KeyBackspace)
	l.typed("12")
	l.doc.Focus(after)
	l.frames(2)
	assert.Equal(t, "10", l.doc.Value(n), "snapped to the step when the focus leaves")
	assert.Len(t, l.changes, 2)

	l.doc.Focus(n)
	l.frames(1)
	l.key(input.KeyBackspace)
	l.key(input.KeyBackspace)
	l.typed("-")
	l.key(input.KeyEnter)
	assert.Equal(t, "10", l.doc.Value(n), "what isn't a number goes back to the value")
	assert.Len(t, l.changes, 2, "and changes nothing")
}
