package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/stretchr/testify/assert"
)

func TestTrackingAddsRoomAfterEachCharacter(t *testing.T) {
	plain := Measure("Hello", FontSize(20))
	tracked := Measure("Hello", FontSize(20), Tracking(3))
	assert.InDelta(t, plain.X+5*3, tracked.X, 0.01, "five characters, three points each")
	assert.Equal(t, plain.Y, tracked.Y)
	assert.InDelta(
		t,
		Measure("e👍🏽", Tracking(3)).X-Measure("e👍🏽").X,
		2*3,
		0.01,
		"an emoji with its skin tone is one character",
	)
	assert.InDelta(t, plain.X-5, Measure("Hello", FontSize(20), Tracking(-1)).X, 0.01, "negative tightens")
	two := Measure("ab\nabc", Tracking(2))
	assert.InDelta(t, Measure("abc").X+3*2, two.X, 0.01, "per line")
}

func TestATrackedPrefixEndsWhereTheNextCharacterStarts(t *testing.T) {
	text := "Tracking"
	for i := 1; i < len(text); i++ {
		whole := Measure(text[:i+1], Tracking(4)).X
		prefix := Measure(text[:i], Tracking(4)).X
		next := Measure(text[i:i+1], Tracking(4)).X
		assert.InDelta(t, whole, prefix+next, 0.01, "carets and selections measure prefixes")
	}
}

func TestATextEntityAndTheCanvasTrackAlike(t *testing.T) {
	w, _, _ := app()
	w.RunFrames(1)
	c := CanvasFor(w, nil)
	text := NewText("Wide", FontSize(18), Tracking(5))
	assert.Equal(t, float32(5), text.Tracking())
	assert.Equal(t, Measure("Wide", FontSize(18), Tracking(5)).X, text.Measure().X)
	assert.Equal(t, text.Measure().X, c.Measure("Wide", FontSize(18), Tracking(5)).X)
	drawn := c.Text(geom.Vec2{}, "Wide", FontSize(18), Tracking(5))
	assert.Equal(t, text.Measure().X, drawn.X, "drawn as measured")
	text.SetTracking(0)
	assert.Equal(t, Measure("Wide", FontSize(18)).X, text.Measure().X)

	words := NewText("one two three", FontSize(16), Wrap(Measure("one two", FontSize(16)).X+1))
	assert.Equal(t, "one two", words.Lines()[0])
	words.SetTracking(4)
	assert.Equal(t, "one", words.Lines()[0], "tracked, fewer words fit a line")
}
