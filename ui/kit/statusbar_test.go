package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/stretchr/testify/assert"
)

func TestAStatusBarReadsItsTextAndDotsUnsavedWork(t *testing.T) {
	zoom := kit.Label("100%")
	bar := kit.StatusBar("walls.tsj · 41 tiles · 3 shapes", zoom)
	bar.SetStyle("width: 300px")
	s := show(t, bar)
	assert.Equal(t, "walls.tsj · 41 tiles · 3 shapes", kit.Status(bar))
	assert.False(t, kit.Dot(bar))
	text := bar.Children()[0]
	assert.Greater(t, zoom.Box().X, text.Box().X, "added elements come after the text")

	kit.SetDot(bar, true)
	kit.SetStatus(bar, "props.tsj · 42 tiles · edited")
	s.frames(2)
	assert.True(t, kit.Dot(bar))
	dot := bar.Children()[2]
	assert.Greater(t, dot.Box().W, float32(0), "the dot shows while something is unsaved")
	assert.Greater(t, dot.Box().X, zoom.Box().X, "at the bar's right")
	assert.Equal(t, "props.tsj · 42 tiles · edited", kit.Status(bar))
	assert.InDelta(t, bar.Box().X+bar.Box().W, dot.Box().X+dot.Box().W+2, 0.01, "its margin box ends the bar")

	kit.SetDot(bar, false)
	s.frames(2)
	assert.Zero(t, bar.Children()[2].Box().W)
}

func TestALongStatusIsCutAtItsEnd(t *testing.T) {
	bar := kit.StatusBar("rooms-of-the-long-forgotten-ossuary.tsj · 412 tiles · 38 shapes · edited")
	bar.SetStyle("width: 200px")
	show(t, bar)
	text := bar.Children()[0]
	assert.LessOrEqual(t, text.Box().X+text.Box().W, bar.Box().X+bar.Box().W, "the text stays in the bar")
}

func TestAStatusBarTakesArgs(t *testing.T) {
	zoom := kit.Label("100%")
	bar := kit.StatusBar("ready", ui.ID("status"), ui.Class("kit-compact"), zoom)
	assert.Equal(t, "status", bar.ID())
	assert.True(t, bar.HasClass("kit-compact"))
	c := bar.Children()
	assert.Equal(t, zoom, c[1])
	assert.True(t, c[len(c)-1].HasClass("kit-status-dot"), "the dot stays last")
}
