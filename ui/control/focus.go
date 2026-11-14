package control

import (
	"slices"

	"github.com/lantervale/ember/geom"
)

// Target is something that can take a window's focus, as laid out this
// frame: its key, its box in window points, its tabindex (-1 for click
// only) and the modal scope it is in, 0 for the page.
type Target[K comparable] struct {
	Key      K
	Rect     geom.Rect
	TabIndex int
	Disabled bool
	Scope    int
}

// Focus is one window's focus, moved as the old ember/ui moves it: Tab in tab
// order, arrows to the nearest by edges, active when reached by keyboard
// or pad (:focus-visible) and quiet after a click.
type Focus[K comparable] struct {
	targets []Target[K]
	at      int
	active  bool
	// traps are the modal scopes open, the innermost last, and before
	// where the focus was when each opened.
	traps  []int
	before []K
	had    []bool
}

// SetTargets gives the focus this frame's targets, in tree order; the
// focus leaves a target no longer among them.
func (f *Focus[K]) SetTargets(ts []Target[K]) {
	k, ok := f.Focused()
	f.targets, f.at = ts, -1
	if ok {
		f.at = f.index(k)
	}
	if f.at < 0 {
		f.active = false
	}
}

// Focused is the focused key, if any.
func (f *Focus[K]) Focused() (K, bool) {
	if f.at < 0 || f.at >= len(f.targets) {
		var zero K
		return zero, false
	}
	return f.targets[f.at].Key, true
}

// Key is the focused key, the zero key for none.
func (f *Focus[K]) Key() K { k, _ := f.Focused(); return k }

// Active reports whether the focus shows: reached by Tab, arrows or pad.
func (f *Focus[K]) Active() bool { return f.active }

// Set focuses k, active or quiet, unless it can't take focus now.
func (f *Focus[K]) Set(k K, active bool) bool {
	i := f.index(k)
	if i < 0 || !f.takes(i, false) {
		return false
	}
	f.at, f.active = i, active
	return true
}

// Blur drops the focus.
func (f *Focus[K]) Blur() { f.at, f.active = -1, false }

// Tab moves the focus to the next target in tab order, or the previous
// when back, wrapping, within the innermost trap.
func (f *Focus[K]) Tab(back bool) bool {
	order := f.order()
	if len(order) == 0 {
		return false
	}
	i := slices.Index(order, f.at)
	switch {
	case i < 0 && back:
		i = len(order) - 1
	case i < 0:
		i = 0
	case back:
		i = (i + len(order) - 1) % len(order)
	default:
		i = (i + 1) % len(order)
	}
	f.at, f.active = order[i], true
	return true
}

// order is the targets Tab reaches: positive tabindexes ascending, then
// the zeros in tree order.
func (f *Focus[K]) order() []int {
	var out []int
	for i := range f.targets {
		if f.takes(i, true) {
			out = append(out, i)
		}
	}
	slices.SortStableFunc(out, func(a, b int) int {
		ta, tb := f.targets[a].TabIndex, f.targets[b].TabIndex
		switch {
		case ta == tb:
			return 0
		case ta == 0:
			return 1
		case tb == 0:
			return -1
		}
		return ta - tb
	})
	return out
}

func (f *Focus[K]) takes(i int, byTab bool) bool {
	t := f.targets[i]
	if t.Disabled || byTab && t.TabIndex < 0 {
		return false
	}
	return len(f.traps) == 0 || t.Scope == f.traps[len(f.traps)-1]
}

// Move moves an active focus to the nearest target towards dir: the one
// whose near edge is closest past the focused one's far edge, distance
// aside counting double.
func (f *Focus[K]) Move(dir geom.Vec2) bool {
	if f.at < 0 || !f.active {
		return false
	}
	from := f.targets[f.at].Rect
	best, bestScore := -1, float32(0)
	for i, t := range f.targets {
		if i == f.at || !f.takes(i, false) || t.TabIndex < 0 {
			continue
		}
		if t.Rect.Center().Sub(from.Center()).Dot(dir) <= 1 {
			continue
		}
		along, aside := gaps(from, t.Rect, dir)
		if score := along + 2*aside; best < 0 || score < bestScore {
			best, bestScore = i, score
		}
	}
	if best < 0 {
		return false
	}
	f.at = best
	return true
}

// gaps are how far apart from and to are along dir, then across it.
func gaps(from, to geom.Rect, dir geom.Vec2) (float32, float32) {
	gap := func(a0, a1, b0, b1 float32) float32 { return max(b0-a1, a0-b1, 0) }
	if dir.X != 0 {
		return gap(from.X, from.X+from.W, to.X, to.X+to.W), gap(from.Y, from.Y+from.H, to.Y, to.Y+to.H)
	}
	return gap(from.Y, from.Y+from.H, to.Y, to.Y+to.H), gap(from.X, from.X+from.W, to.X, to.X+to.W)
}

// Trap keeps the focus within scope, a modal dialog's, moving it to the
// scope's first target, until Release.
func (f *Focus[K]) Trap(scope int) {
	k, ok := f.Focused()
	f.traps, f.before, f.had = append(f.traps, scope), append(f.before, k), append(f.had, ok)
	if order := f.order(); len(order) > 0 {
		f.at, f.active = order[0], true
	} else {
		f.at = -1
	}
}

// Release ends the innermost trap, the focus going back where it was.
func (f *Focus[K]) Release() {
	n := len(f.traps)
	if n == 0 {
		return
	}
	k, had := f.before[n-1], f.had[n-1]
	f.traps, f.before, f.had = f.traps[:n-1], f.before[:n-1], f.had[:n-1]
	f.at = -1
	if had {
		f.Set(k, f.active)
	}
}

func (f *Focus[K]) index(k K) int {
	return slices.IndexFunc(f.targets, func(t Target[K]) bool { return t.Key == k })
}
