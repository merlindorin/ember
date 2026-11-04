package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/merlindorin/ember/world"
)

// game is the example as main builds it, headless.
type game struct {
	w   *world.World
	s   *settings
	win *platform.Window
	in  *input.State
}

func play(t *testing.T) game {
	t.Helper()
	win := platform.NewWindow(platform.Size(640, 440))
	s := newSettings()
	win.Add(s.doc)
	w := ember.New().
		AddEntities(win).
		AddSystems(&Loading{doc: s.doc, bar: s.loading}, &Reopen{settings: s, in: win.Input()})
	w.RunFrames(2)
	return game{w, s, win, win.Input()}
}

func (g game) box(n *ui.Node) geom.Rect { r, _ := g.s.doc.BoxOf(n); return r }

func (g game) key(k input.Key) {
	g.in.SetKey(k, true)
	g.w.RunFrames(1)
	g.in.SetKey(k, false)
	g.w.RunFrames(2)
}

func (g game) click(at geom.Vec2) {
	g.in.SetMouse(at)
	g.w.RunFrames(1)
	g.in.SetMouseButton(input.MouseLeft, true)
	g.w.RunFrames(1)
	g.in.SetMouseButton(input.MouseLeft, false)
	g.w.RunFrames(2)
}

func (g game) drag(from, by geom.Vec2) {
	g.in.SetMouse(from)
	g.in.SetMouseButton(input.MouseLeft, true)
	g.w.RunFrames(1)
	for i := 1; i <= 10; i++ {
		g.in.SetMouse(from.Add(by.Scale(float32(i) / 10)))
		g.w.RunFrames(1)
	}
	g.in.SetMouseButton(input.MouseLeft, false)
	g.w.RunFrames(2)
}

func (g game) status() string { return g.s.status.Text() }

func TestTheStatusLineSaysWhatEachControlReported(t *testing.T) {
	g := play(t)
	g.click(g.box(kit.CheckBox(g.s.music)).Center())
	if g.status() != "Music is off" {
		t.Errorf("a check: %q", g.status())
	}
	g.s.doc.Focus(g.s.volume)
	g.key(input.KeyLeft)
	if g.status() != "Volume 65%" {
		t.Errorf("the slider: %q", g.status())
	}
	g.s.doc.Focus(g.s.name)
	g.in.AddText('x')
	g.w.RunFrames(3)
	if g.status() != "Typing Adax" {
		t.Errorf("typing: %q", g.status())
	}
	g.key(input.KeyEnter)
	if g.status() != "Hello, Adax" {
		t.Errorf("Enter in the name: %q", g.status())
	}
	g.click(g.box(g.s.slots.Children()[0]).Center())
	if g.status() != "Save slots open" {
		t.Errorf("the slots unfolded: %q", g.status())
	}
	g.click(g.box(g.s.slots.Children()[0]).Center())
	if g.status() != "Save slots folded" {
		t.Errorf("the slots folded: %q", g.status())
	}
}

func TestTheLoadingBarFillsAndStartsAgain(t *testing.T) {
	g := play(t)
	l := &Loading{doc: g.s.doc, bar: g.s.loading}
	w := ember.New().AddSystems(l)
	w.RunFrames(120)
	if l.value < 0.49 || l.value > 0.51 {
		t.Fatalf("half full after two of its four seconds: %v", l.value)
	}
	w.RunFrames(150)
	if l.value > 0.2 {
		t.Fatalf("full, then started again: %v", l.value)
	}
}

func TestAPadStepsTheVolumeFromTheStart(t *testing.T) {
	g := play(t)
	g.in.ConnectGamepad(0, "pad")
	g.w.RunFrames(2)
	if !g.s.doc.Focused(g.s.volume) {
		t.Fatal("the volume has focus from the start")
	}
	press := func(b input.GamepadButton) {
		g.in.SetGamepadButton(0, b, true)
		g.w.RunFrames(1)
		g.in.SetGamepadButton(0, b, false)
		g.w.RunFrames(2)
	}
	press(input.GamepadRight)
	if v := g.s.doc.Value(g.s.volume); v != "0.75" {
		t.Fatalf("right steps it up: %v", v)
	}
	press(input.GamepadDown)
	press(input.GamepadLeft)
	press(input.GamepadSouth)
	if g.s.doc.Checked(kit.CheckBox(g.s.music)) {
		t.Fatal("down to Fullscreen, left to the music, then south, turns it off")
	}
}

func TestTheSaveSlotsScrollByTheWheel(t *testing.T) {
	g := play(t)
	g.s.doc.SetOpen(g.s.slots, true)
	g.w.RunFrames(2)
	list := g.s.slots.Children()[1]
	first := list.Children()[0].Children()[0]
	top := g.box(first).Y
	g.in.SetMouse(g.box(list).Center())
	g.in.AddWheel(geom.V2(0, 3))
	g.w.RunFrames(3)
	if g.box(first).Y >= top {
		t.Fatalf("the wheel scrolls the slots up: %v, was %v", g.box(first).Y, top)
	}
}

func TestTheSlotsFoldAndEscapeClosesTheSettingsThatEnterReopens(t *testing.T) {
	g := play(t)
	folded, top, wide := g.box(g.s.window).H, g.box(g.s.window).Y, g.box(g.s.window).W
	g.s.doc.SetOpen(g.s.slots, true)
	g.w.RunFrames(2)
	if r := g.box(g.s.window); r.H <= folded+90 || r.Y != top || r.W != wide {
		t.Fatalf(
			"open, the slots take their room below, the window's top and width staying put: %v, folded %v at %v, %v wide",
			r,
			folded,
			top,
			wide,
		)
	}

	g.key(input.KeyEscape)
	if g.s.doc.Open(g.s.window) || g.status() != "Settings closed: Enter opens them again" {
		t.Fatalf("Escape with the focus inside closes the settings: %q", g.status())
	}
	g.s.doc.SetValue(g.s.volume, "0.3")
	g.key(input.KeyEnter)
	if !g.s.doc.Open(g.s.window) || g.s.doc.Value(g.s.volume) != "0.3" || g.status() != "Settings open again" {
		t.Fatalf(
			"Enter opens them again as they were, and says so: %v, %v, %q",
			g.s.doc.Open(g.s.window),
			g.s.doc.Value(g.s.volume),
			g.status(),
		)
	}
}

func TestTheTitleBarDragsTheSettings(t *testing.T) {
	g := play(t)
	from := g.box(g.s.window)
	g.drag(geom.V2(from.X+40, from.Y+10), geom.V2(-80, 60))
	if to := g.box(g.s.window); to.X != from.X-80 || to.Y != from.Y+60 || g.status() != "Settings moved to 50, 90" {
		t.Fatalf("dragged by its bar: %v, was %v, %q", to, from, g.status())
	}
}

func TestTheCornerResizesTheSettingsUpToTheirMaximum(t *testing.T) {
	g := play(t)
	from := g.box(g.s.window)
	g.drag(geom.V2(from.X+from.W-2, from.Y+from.H-2), geom.V2(300, 80))
	if to := g.box(g.s.window); to.W > 560 || to.W <= from.W || to.H <= from.H {
		t.Fatalf("wider up to its 560 maximum, and taller: %v, was %v", to, from)
	}
	g.s.doc.SetOpen(g.s.slots, true)
	g.w.RunFrames(2)
	to := g.box(g.s.window)
	g.drag(geom.V2(to.X+to.W-2, to.Y+to.H-2), geom.V2(0, 200))
	if w, s := g.box(g.s.window), g.box(g.s.status); w.Overlaps(s) {
		t.Fatalf("resized as tall as it goes, the window %v keeps clear of the status line %v", w, s)
	}
}
