package ui

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
	"github.com/stretchr/testify/assert"
)

func TestAMinContentBoxBreaksItsInlineBlocksAtTheirNarrowest(t *testing.T) {
	css := `.fit { width: min-content; font-size: 14px; } .chip { display: inline-block; }
		.one { white-space: nowrap; font-size: 14px; }`
	d := NewDocument(Stylesheet(fstest.MapFS{"a.css": {Data: []byte(css)}}, "a.css"))
	fit := Div(Class("fit"), Span(Class("chip"), Text("bbbb cccc")))
	whole := Span(Class("one"), Text("bbbb cccc"))
	d.Body().Append(fit, Div(whole))
	d.Layout(geom.V2(600, 400))
	f, _ := d.BoxOf(fit)
	w, _ := d.BoxOf(whole)
	assert.Less(t, f.W, w.W*0.75, "min-content takes the chip at its narrowest, one word a line")
}

// BenchmarkAStatusLineChange is a HUD's corner: an absolutely placed column
// of a status line, a bar, a second line and a canvas, the status line
// changing every frame.
func BenchmarkAStatusLineChange(b *testing.B) {
	css := `.hud { position: absolute; left: 16px; top: 12px; display: flex; flex-direction: column;
		align-items: flex-start; gap: 6px; }
.s { font-size: 15px; white-space: pre; } .x { width: 240px; height: 8px; } .d { height: 20px; }`
	d := NewDocument(Stylesheet(fstest.MapFS{"a.css": {Data: []byte(css)}}, "a.css"))
	status := Span(Class("s"), Text("Area 1 · 0:00 · 0 kills"))
	d.Body().Append(Div(Class("hud"), status, Progress(Class("x")),
		Span(Class("s"), Text("Bow 1 · — · — · —   No torch")), Canvas(func(*twod.Canvas) {}, Class("d"))))
	size := geom.V2(1280, 800)
	d.Layout(size)
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		i++
		status.SetText(fmt.Sprintf("Area 1 · %d:%02d · %d kills", i/60, i%60, i%7))
		d.Layout(size)
	}
}
