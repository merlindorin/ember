package twod

import (
	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
)

// press tracks a mouse press on r: clicked on the release over it, held
// while the button stays down after pressing it.
func (u *Canvas) press(r geom.Rect, key uint64) (bool, bool, bool) {
	var clicked, held, hover bool
	hover = u.Hovered(r)
	if u.in == nil {
		return false, false, hover
	}
	if hover {
		u.hot = key
		if u.in.Pressed(input.MouseLeft) {
			u.active = key
		}
		if u.active == key && u.in.Released(input.MouseLeft) {
			clicked = true
		}
	}
	held = u.active == key && u.in.Down(input.MouseLeft)
	if !u.in.Down(input.MouseLeft) && u.active == key {
		u.active = 0
	}
	return clicked, held, hover
}

// Slider is a horizontal slider over [lo, hi], snapped to step when it is

// TextField is a one-line text input. A click focuses it, and while
// focused it takes the keyboard: characters, backspace, left, right, and
// enter, which submits and releases it. It returns the text, whether it

// Interaction is what happened to a custom-drawn element this frame.
type Interaction struct {
	// Clicked is a click, or an automation client pressing the element.
	Clicked, Held, Hover bool
}

// Interact handles the mouse and automation presses of an element drawn
// by the caller, named name in the current scope. Call Record afterwards.
func (u *Canvas) Interact(r geom.Rect, name string) Interaction {
	var in Interaction
	in.Clicked, in.Held, in.Hover = u.press(r, id("interact", u.ID(name)))
	if r := u.registry(); r != nil {
		if a := r.Take(u.ID(name)); a != nil && a.Kind == automation.ActPress {
			in.Clicked = true
		}
	}
	return in
}

// Record publishes an element's final state for automation.
func (u *Canvas) Record(name, role, label string, value any, enabled bool, r geom.Rect, acts ...string) {
	if !u.registry().Recording() {
		return
	}
	u.record(
		automation.Element{
			ID:      u.ID(name),
			Role:    role,
			Label:   label,
			Value:   value,
			Enabled: enabled,
			Bounds:  r,
			Acts:    acts,
		},
	)
}

// SetScope replaces the ID scope with path, each part slugged as Push
// does, and returns the previous scope to restore.
func (u *Canvas) SetScope(path []string) []string {
	u.path()
	prev := u.scope
	u.scope = make([]string, len(path))
	for i, p := range path {
		u.scope[i] = automation.Slug(p)
	}
	return prev
}

// RestoreScope puts back a scope SetScope returned.
func (u *Canvas) RestoreScope(scope []string) { u.pending, u.scope = nil, scope }
