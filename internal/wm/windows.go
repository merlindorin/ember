// Package wm is how engine packages find a World's windows: the
// primary one, the one being presented, and its input. Games hold their
// windows instead.
package wm

import (
	"math"

	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/inputfeed"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/world"
)

type registry struct {
	world.Group[platform.Windowed]
	current *platform.Window
	owners  map[*platform.Window][]Entity
	covers  map[*platform.Window]cover
}

func of(w *world.World) *registry {
	if r, ok := world.TryResource[registry](w); ok {
		return r
	}
	r := world.SetResource(w, &registry{})
	w.AddView(r)
	return r
}

// Primary is the first window added to the World, nil when it has none.
func Primary(w *world.World) *platform.Window {
	for e := range of(w).All() {
		return e.GetWindow()
	}
	return nil
}

// Resolve is win, or the primary window when win is nil.
func Resolve(w *world.World, win *platform.Window) *platform.Window {
	if win != nil {
		return win
	}
	return Primary(w)
}

// Of is the window e is under, the nearest window among its parents, nil
// when it is under none.
func Of(w *world.World, e interface{ GetBasicEntity() *world.BasicEntity }) *platform.Window {
	for p := e.GetBasicEntity().Parent(); p != nil; p = p.Parent() {
		if parent, ok := w.Entity(p.ID()); ok {
			if windowed, isWindow := parent.(platform.Windowed); isWindow {
				return windowed.GetWindow()
			}
		}
	}
	return nil
}

// Home is the window e belongs to: win when set, else the window e is
// under, else the primary.
func Home(w *world.World, e interface{ GetBasicEntity() *world.BasicEntity }, win *platform.Window) *platform.Window {
	if win != nil {
		return win
	}
	if under := Of(w, e); under != nil {
		return under
	}
	return Primary(w)
}

// SetCurrent is set by the runner around each window's presentation.
func SetCurrent(w *world.World, win *platform.Window) { of(w).current = win }

// Presenting reports whether a runner is presenting a window right now.
func Presenting(w *world.World) bool { return of(w).current != nil }

// Target is the window being presented, else the primary window.
func Target(w *world.World) *platform.Window {
	if c := of(w).current; c != nil {
		return c
	}
	return Primary(w)
}

var none input.State //nolint:gochecknoglobals // the one empty input a windowless entity reads

// Input is the primary window's input, which also holds the gamepads, or
// an idle state when the World has no window.
func Input(w *world.World) *input.State {
	if p := Primary(w); p != nil {
		return p.Input()
	}
	return &none
}

// Plugin begins every window's input frame before any other system runs,
// so a World run by RunFrames or Tick sees what tests and backends set,
// and emits the pads that connected and disconnected and the touches that
// began, moved and lifted.
type Plugin struct{}

func (Plugin) Build(w *world.World) { w.AddSystems(&inputFrames{}) }

type inputFrames struct{}

func (*inputFrames) Phase() world.Phase { return world.PreUpdate }
func (*inputFrames) Priority() int      { return math.MaxInt }

func (*inputFrames) Update(f world.Frame) {
	for e := range of(f.World).All() {
		in := e.GetWindow().Input()
		feed := inputfeed.Of(in)
		feed.BeginFrame()
		connected, disconnected := feed.GamepadChanges()
		for _, id := range connected {
			world.Emit(f.World, input.GamepadConnected{ID: id, Name: in.Gamepad(id).Name()})
		}
		for _, id := range disconnected {
			world.Emit(f.World, input.GamepadDisconnected{ID: id})
		}
		for _, t := range in.Touches() {
			switch t.Phase {
			case input.TouchBegan:
				world.Emit(f.World, input.TouchStarted{Touch: t})
			case input.TouchMoving:
				world.Emit(f.World, input.TouchMoved{Touch: t})
			case input.TouchLifted:
				world.Emit(f.World, input.TouchEnded{Touch: t})
			case input.TouchResting:
			}
		}
	}
	of(f.World).owned(f.World)
}
