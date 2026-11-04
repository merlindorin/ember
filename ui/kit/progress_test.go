package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAProgressShowsItsValueAsAPercentage(t *testing.T) {
	p := kit.Progress(30)
	show(t, p)
	bar := kit.ProgressBar(p)
	require.NotNil(t, bar)
	v, _ := bar.Attr("value")
	m, _ := bar.Attr("max")
	assert.Equal(t, "30", v)
	assert.Equal(t, "100", m)
	assert.Equal(t, "30%", p.Text())
	assert.Equal(t, float32(200), p.Box().W)
	assert.Equal(t, float32(20), p.Box().H)
}

func TestARangedProgressCountsFromItsLowEnd(t *testing.T) {
	p := kit.Progress(70, kit.Range(10, 210))
	bar := kit.ProgressBar(p)
	v, _ := bar.Attr("value")
	m, _ := bar.Attr("max")
	assert.Equal(t, "60", v)
	assert.Equal(t, "200", m, "progress has no min: the range moves to 0")
	_, hasMin := bar.Attr("min")
	assert.False(t, hasMin)
	assert.Equal(t, "30%", p.Text())
}

func TestShowValueFalseLeavesThePercentageOff(t *testing.T) {
	p := kit.Progress(30, kit.ShowValue(false))
	assert.Empty(t, p.Text())
	_, marked := kit.ProgressBar(p).Attr("data-hide-value")
	assert.False(t, marked)
}

func TestThePercentageSitsInTheMiddleOfTheBar(t *testing.T) {
	p := kit.Progress(30)
	show(t, p)
	label := p.Children()[1]
	assert.InDelta(t, p.Box().Center().X, label.Box().Center().X, 1)
	assert.InDelta(t, p.Box().Center().Y, label.Box().Center().Y, 1)
}

func TestSetProgressMovesTheBarAndItsPercentage(t *testing.T) {
	p := kit.Progress(70, kit.Range(10, 210))
	s := show(t, p)
	kit.SetProgress(p, 160)
	s.frames(1)
	v, _ := kit.ProgressBar(p).Attr("value")
	assert.Equal(t, "150", v, "in the Progress's own units, from its low end")
	assert.Equal(t, "75%", p.Text())

	hidden := kit.Progress(30, kit.ShowValue(false))
	kit.SetProgress(hidden, 60)
	assert.Empty(t, hidden.Text(), "no percentage where it was left off")
}
