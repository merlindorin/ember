package control_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/control"

	"github.com/merlindorin/ember/geom"
	"github.com/stretchr/testify/assert"
)

// grid is three buttons in a row over a wide one:
//
//	[a] [b] [c]
//	[    wide  ]
func grid() []control.Target[string] {
	return []control.Target[string]{
		{Key: "a", Rect: geom.Rect{X: 0, Y: 0, W: 50, H: 20}},
		{Key: "b", Rect: geom.Rect{X: 60, Y: 0, W: 50, H: 20}},
		{Key: "c", Rect: geom.Rect{X: 120, Y: 0, W: 50, H: 20}},
		{Key: "wide", Rect: geom.Rect{X: 0, Y: 30, W: 170, H: 20}},
	}
}

func TestTabWalksTheTreeOrderAndWraps(t *testing.T) {
	f := &control.Focus[string]{}
	f.SetTargets(grid())
	assert.True(t, f.Tab(false))
	assert.Equal(t, "a", f.Key())
	assert.True(t, f.Active())
	f.Tab(false)
	f.Tab(false)
	f.Tab(false)
	assert.Equal(t, "wide", f.Key())
	f.Tab(false)
	assert.Equal(t, "a", f.Key(), "wraps")
	f.Tab(true)
	assert.Equal(t, "wide", f.Key(), "Shift+Tab goes back")
}

func TestTabIndexOrdersPositiveFirstAndSkipsNegative(t *testing.T) {
	f := &control.Focus[string]{}
	f.SetTargets([]control.Target[string]{
		{Key: "first"}, {Key: "skipped", TabIndex: -1}, {Key: "two", TabIndex: 2},
		{Key: "one", TabIndex: 1}, {Key: "off", Disabled: true},
	})
	var order []string
	for range 4 {
		f.Tab(false)
		order = append(order, f.Key())
	}
	assert.Equal(t, []string{"one", "two", "first", "one"}, order)
	f.Set("skipped", false)
	assert.Equal(t, "skipped", f.Key(), "tabindex -1 still takes focus by click")
	f.Set("off", false)
	assert.Equal(t, "skipped", f.Key(), "a disabled control never does")
}

func TestArrowsMoveToTheNearestByEdges(t *testing.T) {
	f := &control.Focus[string]{}
	f.SetTargets(grid())
	f.Set("b", true)
	assert.True(t, f.Move(geom.V2(1, 0)))
	assert.Equal(t, "c", f.Key())
	f.Move(geom.V2(0, 1))
	assert.Equal(t, "wide", f.Key(), "the wide one is right below c by its edges")
	f.Move(geom.V2(0, -1))
	assert.Equal(t, "a", f.Key(), "back up, the nearest edge-on is the first, aside counting double")
	assert.False(t, f.Move(geom.V2(-1, 0)), "nothing left of a")
	assert.Equal(t, "a", f.Key())
}

func TestAClickFocusesQuietly(t *testing.T) {
	f := &control.Focus[string]{}
	f.SetTargets(grid())
	f.Set("b", false)
	assert.Equal(t, "b", f.Key())
	assert.False(t, f.Active(), "no :focus-visible outline after a click")
	assert.False(t, f.Move(geom.V2(1, 0)), "quiet focus doesn't steer by arrows")
	f.Tab(false)
	assert.True(t, f.Active())
	assert.Equal(t, "c", f.Key())
}

func TestAModalScopeTrapsTheFocus(t *testing.T) {
	f := &control.Focus[string]{}
	f.SetTargets([]control.Target[string]{
		{Key: "page"},
		{Key: "ok", Scope: 1},
		{Key: "cancel", Scope: 1},
	})
	f.Set("page", true)
	f.Trap(1)
	assert.Equal(t, "ok", f.Key(), "trapping moves the focus inside")
	f.Tab(false)
	f.Tab(false)
	assert.Equal(t, "ok", f.Key(), "Tab cycles inside the dialog")
	f.Set("page", false)
	assert.Equal(t, "ok", f.Key(), "outside can't take focus while trapped")
	f.Release()
	assert.Equal(t, "page", f.Key(), "the focus goes back where it was")
}

func TestFocusLeavesATargetThatIsGone(t *testing.T) {
	f := &control.Focus[string]{}
	f.SetTargets(grid())
	f.Set("c", true)
	f.SetTargets(grid()[:2])
	assert.Equal(t, "", f.Key())
	_, ok := f.Focused()
	assert.False(t, ok)
	assert.False(t, f.Active())
}
