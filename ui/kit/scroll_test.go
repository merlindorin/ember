package kit_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

// items are ten 20-point labels, more than a 60-point scroll shows.
func items() []ui.Arg {
	const n = 10
	args := make([]ui.Arg, n)
	for i := range n {
		args[i] = kit.Label(fmt.Sprint("item ", i), ui.Style("display: block; height: 20px"))
	}
	return args
}

func TestAScrollTurnsByTheWheel(t *testing.T) {
	sc := kit.Scroll(append([]ui.Arg{ui.Style("height: 60px")}, items()...)...)
	s := show(t, sc)
	first := sc.Children()[0]
	top := first.Box().Y
	s.in.SetMouse(sc.Box().Center())
	s.frames(1)
	s.in.AddWheel(geom.V2(0, 1))
	s.frames(2)
	assert.Less(t, first.Box().Y, top, "the content moved up")
}

func TestAScrollFillsTheWidthAndClipsItsContent(t *testing.T) {
	sc := kit.Scroll(append([]ui.Arg{ui.Style("height: 60px")}, items()...)...)
	show(t, sc)
	assert.Equal(t, float32(60), sc.Box().H)
	assert.Equal(t, float32(400), sc.Box().W)
}

func TestBarModesAreOverflows(t *testing.T) {
	for mode, want := range map[kit.BarMode]string{kit.BarAuto: "auto", kit.BarAlways: "scroll",
		kit.BarNever: "auto; scrollbar-width: none"} {
		assert.Equal(t, "overflow-y: "+want, kit.Scroll(kit.VerticalBar(mode)).Style())
		assert.Equal(t, "overflow-x: "+want, kit.Scroll(kit.HorizontalBar(mode)).Style())
	}
}

func TestBarNeverStillTurnsByTheWheel(t *testing.T) {
	sc := kit.Scroll(append([]ui.Arg{ui.Style("height: 60px"), kit.VerticalBar(kit.BarNever)}, items()...)...)
	s := show(t, sc)
	first := sc.Children()[0]
	top := first.Box().Y
	assert.Equal(t, float32(400), first.Box().W, "no gutter")
	s.in.SetMouse(sc.Box().Center())
	s.frames(1)
	s.in.AddWheel(geom.V2(0, 1))
	s.frames(2)
	assert.Less(t, first.Box().Y, top)
}

func TestAScrollInAColumnShrinksBelowItsContent(t *testing.T) {
	sc := kit.Scroll(append([]ui.Arg{ui.Class("kit-expand")}, items()...)...)
	col := kit.Column(ui.Style("height: 100px"), kit.Label("title"), sc)
	show(t, col)
	assert.LessOrEqual(t, sc.Box().Y+sc.Box().H, col.Box().Y+100, "min-height 0, as the old ember/ui's Scroll")
}
