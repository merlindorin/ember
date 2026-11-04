package ui

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestACaseMapTiesEachValuePositionToItsDrawnOne(t *testing.T) {
	shown, m := fieldCase("straße", style.Uppercase)
	assert.Equal(t, "STRASSE", shown)
	assert.Equal(t, []int{0, 1, 2, 3, 4, 6, 7}, m.toDrawn, "after ß is after SS")
	assert.Equal(t, 4, m.valueAt(4), "before SS is before ß")
	assert.Equal(t, 5, m.valueAt(6))
	assert.Equal(t, 6, m.valueAt(7))

	shown, m = fieldCase("éte ok", style.Uppercase)
	assert.Equal(t, "ÉTE OK", shown, "a combining mark keeps its letter")
	assert.Nil(t, nil, m)

	shown, _ = fieldCase("hello wORLD o'neil", style.Capitalize)
	assert.Equal(t, "Hello WORLD O'neil", shown, "capitalize titles each word's first letter only")

	shown, m = fieldCase("abc", style.NoTransform)
	assert.Equal(t, "abc", shown)
	assert.Nil(t, m)
}

func TestAnUppercaseFieldDrawsCasedTextAndKeepsTheValueAsTyped(t *testing.T) {
	field := Input(Type("text"), Style("text-transform: uppercase"), Value("straße"))
	d, w, in := typed(t, field)
	w.RunFrames(1)
	assert.Equal(t, "STRASSE", dom.Data(field).Shadow.Text(), "drawn cased")
	assert.Equal(t, "straße", d.Value(field), "the value as typed")
	in.AddText('x')
	w.RunFrames(2)
	assert.Equal(t, "straßex", d.Value(field))
	assert.Equal(t, "STRASSEX", dom.Data(field).Shadow.Text())
}

func TestAClickOnCasedTextPutsTheCaretWhereItFallsInTheValue(t *testing.T) {
	field := Input(Type("text"), Attr("size", "30"), Style("text-transform: uppercase"), Value("straße"))
	d, w, in := typed(t, field)
	w.RunFrames(1)
	m := textOf(d.machine(field))
	l, ok := m.Layout.(*line)
	require.True(t, ok)
	b := d.tree.Of[field]
	after := l.xs[6] // "STRASS"
	in.SetMouse(geom.V2(field.Box().X+b.ContentRect().X+after-b.Scroll.X, field.Box().Y+field.Box().H/2))
	w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, true)
	w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, false)
	w.RunFrames(1)
	assert.Equal(t, 5, m.Editor.Caret(), "after the drawn SS is after ß")
	in.AddText('x')
	w.RunFrames(2)
	assert.Equal(t, "straßxe", d.Value(field))
}

func TestASelectionOfCasedTextSpansTheDrawnText(t *testing.T) {
	field := Input(Type("text"), Attr("size", "30"), Style("text-transform: uppercase"), Value("straße"))
	d, w, _ := typed(t, field)
	w.RunFrames(1)
	m := textOf(d.machine(field))
	f := dom.Data(field).Style.Font
	assert.InDelta(t, d.fonts.Advance(f, "STRASSE"), m.Layout.X(6), 0.01, "the value's end is the drawn text's")
	assert.InDelta(t, d.fonts.Advance(f, "STRASS"), m.Layout.X(5), 0.01, "after ß the drawn SS's")
	start, end := m.Layout.Span(0)
	assert.Equal(t, [2]int{0, 6}, [2]int{start, end}, "spans in the value's positions")
}

func TestACapitalizedTextareaWalksItsLinesInTheValuesPositions(t *testing.T) {
	note := Textarea(Attr("rows", "4"), Style("text-transform: capitalize"), Text("hello world\nfoo"))
	d, w, _ := typed(t, note)
	w.RunFrames(1)
	assert.Equal(t, "Hello World\nFoo", dom.Data(note).Shadow.Text())
	assert.Equal(t, "hello world\nfoo", d.Value(note))
	m := textOf(d.machine(note))
	assert.Equal(t, 1, m.Layout.Line(13), "the second line's second letter")
	start, end := m.Layout.Span(1)
	assert.Equal(t, [2]int{12, 15}, [2]int{start, end})
}
