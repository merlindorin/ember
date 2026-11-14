package ui

import (
	"runtime"
	"strings"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// typed is a Document showing field in a window, field focused.
func typed(t *testing.T, field *Node) (*Document, *world.World, *input.State) {
	t.Helper()
	d := NewDocument()
	d.Body().Append(field)
	win := platform.NewWindow(platform.Size(400, 400))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(1)
	d.Focus(field)
	w.RunFrames(1)
	return d, w, win.Input()
}

func hit(w *world.World, in *input.State, keys ...input.Key) {
	for _, k := range keys {
		in.SetKey(k, true)
	}
	w.RunFrames(1)
	for _, k := range keys {
		in.SetKey(k, false)
	}
	w.RunFrames(1)
}

// caretShows reports whether field's caret lies inside its scrollport.
func caretShows(t *testing.T, d *Document, field *Node) bool {
	t.Helper()
	b := d.tree.Of[field]
	m := textOf(d.machine(field))
	require.NotNil(t, m)
	at, ok := caretBox(b, m.Layout, m.Editor.Caret())
	require.True(t, ok)
	port := b.ContentRect()
	at.X, at.Y = at.X-b.Scroll.X, at.Y-b.Scroll.Y
	return at.Y >= port.Y-0.5 && at.Y+at.H <= port.Y+port.H+0.5 && at.X >= port.X-0.5 && at.X+at.W <= port.X+port.W+0.5
}

// docEdge is the keys taking the caret to the text's start or end.
func docEdge(end bool) []input.Key {
	if runtime.GOOS == "darwin" {
		if end {
			return []input.Key{input.KeyLeftSuper, input.KeyDown}
		}
		return []input.Key{input.KeyLeftSuper, input.KeyUp}
	}
	if end {
		return []input.Key{input.KeyLeftControl, input.KeyEnd}
	}
	return []input.Key{input.KeyLeftControl, input.KeyHome}
}

func TestATextareaScrollsToKeepItsCaretInView(t *testing.T) {
	note := Textarea(Attr("rows", "12"), Text(strings.Repeat("a line\n", 10)+"end"))
	d, w, in := typed(t, note)
	hit(w, in, docEdge(true)...)
	for range 8 {
		hit(w, in, input.KeyEnter)
		assert.True(t, caretShows(t, d, note), "the new line's caret shows")
	}
	assert.Positive(t, d.tree.Of[note].Scroll.Y, "the text scrolled up")
	hit(w, in, docEdge(false)...)
	assert.Zero(t, d.tree.Of[note].Scroll.Y, "back at the top")
	assert.True(t, caretShows(t, d, note))
	hit(w, in, docEdge(true)...)
	assert.True(t, caretShows(t, d, note), "and down at the end again")
	assert.Positive(t, d.tree.Of[note].Scroll.Y)
}

func TestATextareasEmptyLinesHoldTheCaretWhereTheyStand(t *testing.T) {
	note := Textarea(Attr("rows", "6"), Text("a\n\nb"))
	d, w, _ := typed(t, note)
	w.RunFrames(1)
	m := textOf(d.machine(note))
	a, ok := m.Layout.(*area)
	require.True(t, ok, "a textarea is laid out line by line")
	require.Equal(t, 3, a.Lines())
	assert.Equal(t, 1, a.Line(2), "the empty line holds the position between its breaks")
	assert.Less(t, a.Row(0).Y, a.Row(1).Y)
	assert.Less(t, a.Row(1).Y, a.Row(2).Y, "and stands between the lines around it")
}

func TestATextInputScrollsAlongToKeepItsCaretInView(t *testing.T) {
	name := Input(Type("text"), Attr("size", "6"))
	d, w, in := typed(t, name)
	for _, r := range "a name far wider than its box" {
		in.AddText(r)
	}
	w.RunFrames(2)
	assert.True(t, caretShows(t, d, name), "the caret at the end shows")
	assert.Positive(t, d.tree.Of[name].Scroll.X)
	hit(w, in, input.KeyHome)
	assert.Zero(t, d.tree.Of[name].Scroll.X, "Home brings the start back")
}

func TestAClickPutsATextareasCaretOnTheLineUnderItAndADragPastItsEdgeScrolls(t *testing.T) {
	note := Textarea(Attr("rows", "4"), Text(strings.Repeat("row\n", 12)+"last"))
	d, w, in := typed(t, note)
	hit(w, in, docEdge(false)...)
	r := note.Box()
	m := textOf(d.machine(note))
	line := func() int { return m.Layout.Line(m.Editor.Caret()) }
	a, ok := m.Layout.(*area)
	if !ok {
		t.Fatalf("a textarea's layout is an area, not %T", m.Layout)
	}
	row := a.Row(2)
	in.SetMouse(geom.V2(r.X+20, r.Y+3+row.Y+row.H/2))
	w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, true)
	w.RunFrames(1)
	assert.Equal(t, 2, line(), "the click's line")
	in.SetMouse(geom.V2(r.X+20, r.Y+r.H+30))
	w.RunFrames(6)
	in.SetMouseButton(input.MouseLeft, false)
	w.RunFrames(1)
	assert.Greater(t, line(), 4, "held past the bottom, the selection reaches lines below")
	assert.Positive(t, d.tree.Of[note].Scroll.Y, "which scroll in")
	assert.True(t, caretShows(t, d, note))
}

func TestATextareasLastBreakLeadsToAnEmptyLineHoldingNoText(t *testing.T) {
	note := Textarea(Attr("rows", "4"), Text("one\ntwo\n"))
	d, w, in := typed(t, note)
	hit(w, in, docEdge(true)...)
	m := textOf(d.machine(note))
	a, ok := m.Layout.(*area)
	require.True(t, ok)
	require.Equal(t, 3, a.Lines(), "the line after the last break")
	assert.Equal(t, 8, m.Editor.Caret(), "End of text is after the break")
	assert.Equal(t, 2, a.Line(8), "on that empty line")
	assert.Zero(t, a.X(8), "at its start")
	start, end := a.Span(2)
	assert.Equal(t, [2]int{8, 8}, [2]int{start, end}, "holding no text")
	start, end = a.Span(1)
	assert.Equal(t, [2]int{4, 7}, [2]int{start, end}, "the line before ends at its break")
	assert.Equal(t, "one\ntwo\n", d.Value(note), "the value never holds what layout drew there")
	assert.True(t, caretShows(t, d, note))
}

func TestAPasswordScrollsAlongItsBulletsAndAPlaceholderScrollsNothing(t *testing.T) {
	hint := Input(Type("text"), Attr("size", "4"), Attr("placeholder", "a long hint wider than the box"))
	d, w, _ := typed(t, hint)
	w.RunFrames(1)
	b := d.tree.Of[hint]
	assert.LessOrEqual(t, b.Content.X, b.Size.X, "a placeholder adds no scroll width")

	secret := Input(Type("password"), Attr("size", "4"))
	d, w, in := typed(t, secret)
	for _, r := range "iiiiiiiiiiiiiiiiiiii" {
		in.AddText(r)
	}
	w.RunFrames(2)
	b = d.tree.Of[secret]
	bullets := d.fonts.Advance(dom.Data(dom.Data(secret).Shadow).Style.Font, strings.Repeat("•", 20))
	assert.InDelta(t, bullets+1, b.Content.X-b.Padding.Horizontal(), 0.5, "the bullets' width, not the narrow letters'")
	assert.True(t, caretShows(t, d, secret))
}

func TestUpAndDownKeepTheCaretsColumnThroughAnEmptyLine(t *testing.T) {
	note := Textarea(Attr("rows", "4"), Text("abcdef\n\nabcdef"))
	d, w, in := typed(t, note)
	hit(w, in, docEdge(false)...)
	for range 4 {
		hit(w, in, input.KeyRight)
	}
	hit(w, in, input.KeyDown)
	m := textOf(d.machine(note))
	assert.Equal(t, 7, m.Editor.Caret(), "the empty line")
	hit(w, in, input.KeyDown)
	assert.Equal(t, 12, m.Editor.Caret(), "and back to the fourth column under it")
	hit(w, in, input.KeyUp)
	hit(w, in, input.KeyUp)
	assert.Equal(t, 4, m.Editor.Caret())
}
