package input

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/lantervale/ember/geom"
)

// Actions binds a game's actions, values of its own type A, to buttons,
// so the game reads intent ("jump") and players can rebind it. A game
// builds one and keeps it; it reads any window's input, or one player's
// pad.
//
//	type Act int
//	const (Jump Act = iota; Left; Right)
//	controls := input.NewActions(input.Bind(Jump, input.KeySpace, input.KeyW))
//	if controls.Pressed(win.Input(), Jump) { … }
type Actions[A comparable] struct {
	order []A
	binds map[A][]Button
}

// Binding is an action's buttons, for NewActions.
type Binding[A comparable] struct {
	action  A
	buttons []Button
}

// Bind binds action to buttons: any of them triggers it.
func Bind[A comparable](action A, buttons ...Button) Binding[A] {
	return Binding[A]{action: action, buttons: buttons}
}

// NewActions returns the actions bound by binds.
func NewActions[A comparable](binds ...Binding[A]) *Actions[A] {
	a := &Actions[A]{binds: map[A][]Button{}}
	for _, b := range binds {
		a.Bind(b.action, b.buttons...)
	}
	return a
}

// Bind binds action to buttons, in place of what it was bound to: a
// rebinding menu calls it.
func (a *Actions[A]) Bind(action A, buttons ...Button) {
	if _, ok := a.binds[action]; !ok {
		a.order = append(a.order, action)
	}
	a.binds[action] = slices.Clone(buttons)
}

// Bindings is the buttons action is bound to.
func (a *Actions[A]) Bindings(action A) []Button { return slices.Clone(a.binds[action]) }

// Bound is the action b triggers, for a rebinding menu to warn about.
func (a *Actions[A]) Bound(b Button) (A, bool) {
	for _, action := range a.order {
		if slices.Contains(a.binds[action], b) {
			return action, true
		}
	}
	var zero A
	return zero, false
}

// Actions is every bound action, in the order they were first bound.
func (a *Actions[A]) Actions() []A { return slices.Clone(a.order) }

// Value is how far action is held, from 0 to 1: the most any of its
// buttons is, in r, a window's input or one player's pad.
func (a *Actions[A]) Value(r Reader, action A) float32 { return a.level(r, action, false) }

func (a *Actions[A]) level(r Reader, action A, before bool) float32 {
	var v float32
	for _, b := range a.binds[action] {
		v = max(v, b.level(r, before))
	}
	return v
}

// Down reports whether action is held in r: one of its buttons is, a
// trigger or stick direction past half way.
func (a *Actions[A]) Down(r Reader, action A) bool { return a.level(r, action, false) > 0.5 }

// Pressed reports whether action started this frame: one of its buttons
// went down while none was held before.
func (a *Actions[A]) Pressed(r Reader, action A) bool {
	return a.Down(r, action) && a.level(r, action, true) <= 0.5
}

// Released reports whether action ended this frame.
func (a *Actions[A]) Released(r Reader, action A) bool {
	return !a.Down(r, action) && a.level(r, action, true) > 0.5
}

// Axis is pos's value less neg's: -1, 0 or 1 by keys, in between by a
// stick.
func (a *Actions[A]) Axis(r Reader, neg, pos A) float32 {
	return a.Value(r, pos) - a.Value(r, neg)
}

// Direction is where four actions point, Y down: unit length by keys, a
// stick's lean kept, never longer than 1.
func (a *Actions[A]) Direction(r Reader, left, right, up, down A) geom.Vec2 {
	v := geom.Vec2{X: a.Axis(r, left, right), Y: a.Axis(r, up, down)}
	if v.Len() > 1 {
		return v.Normalize()
	}
	return v
}

// MarshalJSON saves the bindings by name: each action as it prints, its
// buttons as they print ("Space", "MouseLeft"), for a settings file.
func (a *Actions[A]) MarshalJSON() ([]byte, error) {
	out := map[string][]string{}
	for _, action := range a.order {
		names := []string{}
		for _, b := range a.binds[action] {
			names = append(names, b.String())
		}
		out[fmt.Sprint(action)] = names
	}
	return json.Marshal(out)
}

// UnmarshalJSON reads bindings MarshalJSON saved, for the actions already
// bound; others are ignored, so an old settings file stays readable.
func (a *Actions[A]) UnmarshalJSON(data []byte) error {
	var in map[string][]string
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	for _, action := range a.order {
		names, ok := in[fmt.Sprint(action)]
		if !ok {
			continue
		}
		var buttons []Button
		for _, n := range names {
			b, err := ParseButton(n)
			if err != nil {
				return err
			}
			buttons = append(buttons, b)
		}
		a.binds[action] = buttons
	}
	return nil
}

var byName = func() map[string]Button { //nolint:gochecknoglobals // built once from every button's name
	m := map[string]Button{}
	for k := Key(0); k <= keyLast; k++ {
		if n := k.String(); n != fmt.Sprintf("Key(%d)", int(k)) {
			m[n] = k
		}
	}
	for k := KeyShift; k <= KeySuper; k++ {
		m[k.String()] = k
	}
	for b := MouseButton(0); b < mouseButtonCount; b++ {
		m[b.String()] = b
	}
	for b := GamepadButton(0); b < gamepadButtonCount; b++ {
		m[b.String()] = b
	}
	return m
}()

// ParseButton is the button named name, as Key, MouseButton and
// GamepadButton print.
func ParseButton(name string) (Button, error) {
	if b, ok := byName[name]; ok {
		return b, nil
	}
	return nil, fmt.Errorf("input: no button is called %q", name)
}
