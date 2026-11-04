package kit_test

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/merlindorin/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shown is a Document of kit's theme holding nodes, in a 400×300 window,
// with automation recording.
type shown struct {
	doc  *ui.Document
	w    *world.World
	in   *input.State
	win  *platform.Window
	auto *automation.Automation
	// watchers read events after every frame, so none is missed.
	watchers []func()
}

func show(t *testing.T, nodes ...*ui.Node) *shown {
	t.Helper()
	return showIn(t, 400, 300, nodes...)
}

// showIn is show in a window of w×h points.
func showIn(t *testing.T, w, h int, nodes ...*ui.Node) *shown {
	t.Helper()
	s := &shown{doc: ui.NewDocument(kit.Theme())}
	s.doc.Body().Append(nodes...)
	win := platform.NewWindow(platform.Size(w, h))
	win.Add(s.doc)
	s.auto = automation.New(automation.Recording())
	s.w = ember.New(ember.Automation(s.auto)).AddEntities(win)
	s.in, s.win = win.Input(), win
	s.frames(2)
	require.Empty(t, s.doc.Warnings())
	return s
}

// frames runs n frames, the watchers reading after each.
func (s *shown) frames(n int) {
	for range n {
		s.w.RunFrames(1)
		for _, w := range s.watchers {
			w()
		}
	}
}

// watch collects every T emitted from now on.
func watch[T any](s *shown) *[]T {
	var r world.Reader[T]
	var got []T
	s.watchers = append(s.watchers, func() {
		for e := range r.Read(s.w) {
			got = append(got, e)
		}
	})
	return &got
}

func (s *shown) element(t *testing.T, label string) automation.Element {
	t.Helper()
	_, all := s.auto.Snapshot("", "")
	for _, e := range all {
		if e.Label == label {
			return e
		}
	}
	t.Fatalf("no element labelled %q", label)
	return automation.Element{}
}

func (s *shown) click(n *ui.Node) { s.clickAt(n.Box().Center()) }

// clickAt clicks the left button at p.
func (s *shown) clickAt(p geom.Vec2) {
	s.in.SetMouse(p)
	s.frames(1)
	s.in.SetMouseButton(input.MouseLeft, true)
	s.frames(1)
	s.in.SetMouseButton(input.MouseLeft, false)
	s.frames(2)
}

func (s *shown) key(k input.Key) {
	s.in.SetKey(k, true)
	s.frames(1)
	s.in.SetKey(k, false)
	s.frames(2)
}

func TestAButtonIsAPressableButtonForAutomation(t *testing.T) {
	s := show(t, kit.Button("Save"), kit.Button("Delete", ui.Class("kit-destructive")), kit.Button("Off", ui.Disabled()))
	save := s.element(t, "Save")
	assert.Equal(t, automation.RoleButton, save.Role)
	assert.True(t, save.Enabled)
	assert.Contains(t, save.Acts, automation.ActPress)
	assert.Equal(t, automation.RoleButton, s.element(t, "Delete").Role, "a class changes the look only")
	assert.False(t, s.element(t, "Off").Enabled)
}

func TestAButtonIsClickedByThePointerAndByEnterOrSpace(t *testing.T) {
	b := kit.Button("Save", ui.Class("kit-primary"))
	s := show(t, b)
	clicks := watch[ui.Clicked](s)
	s.click(b)
	assert.Len(t, *clicks, 1)
	s.doc.Focus(b)
	s.frames(1)
	s.key(input.KeyEnter)
	s.key(input.KeySpace)
	assert.Len(t, *clicks, 3)
	assert.Same(t, b, (*clicks)[2].Node)
}

func TestADisabledButtonIsNeverClicked(t *testing.T) {
	b := kit.Button("Off", ui.Disabled())
	s := show(t, b)
	clicks := watch[ui.Clicked](s)
	s.click(b)
	s.doc.Focus(b)
	s.frames(1)
	s.key(input.KeyEnter)
	assert.Empty(t, *clicks)
}

func TestAButtonsSizeIsItsTextAndEmberUIsPadding(t *testing.T) {
	b, l := kit.Button("Save"), kit.Label("Save")
	show(t, b, l)
	assert.InDelta(t, l.Box().W+2*10+2, b.Box().W, 0.5, "10 each side and a 1 px border")
	assert.InDelta(t, l.Box().H+2*4+2, b.Box().H, 0.5, "4 above and below and a 1 px border")
}

func TestKitControlsShowTheHandAndFieldsTheText(t *testing.T) {
	button, off := kit.Button("Save"), kit.Button("Off", ui.Disabled())
	check, slider, field := kit.Check("Music"), kit.Slider(40), kit.TextInput("Ayla")
	s := show(t, ui.Div(ui.Style("display: flex; flex-direction: column; gap: 8px"), button, off, check, slider, field))
	for n, want := range map[*ui.Node]platform.Cursor{
		button: platform.CursorHand, off: platform.CursorArrow, check: platform.CursorHand,
		slider: platform.CursorHand, field: platform.CursorText,
	} {
		s.in.SetMouse(n.Box().Center())
		s.frames(2)
		assert.Equal(t, want, s.win.Cursor(), "%v", n.Classes())
	}
}
