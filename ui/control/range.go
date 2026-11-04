package control

import "math"

// Range is an input of type range: a value in [Min, Max] on Step from
// Min (0 for any value), stepped by Left and Right, set by Home, End and
// the pages, or dragged along its track.
type Range struct {
	Min, Max, Step, Value float64
	Disabled              bool

	dragging bool
	start    float64
}

// NewRange is a range as HTML makes one: 0 to 100 by 1, at 50.
func NewRange() *Range { return &Range{Max: 100, Step: 1, Value: 50} }

// Handles reports whether i steps the range; Up and Down move the focus.
func (r *Range) Handles(i Intent) bool {
	switch i {
	case Left, Right, Home, End, PageUp, PageDown:
		return !r.Disabled
	case None, Activate, Back, Up, Down:
	}
	return false
}

func (r *Range) Do(i Intent) Events {
	if !r.Handles(i) {
		return 0
	}
	page := max(r.Step, (r.Max-r.Min)/10)
	v := r.Value
	switch i {
	case Left:
		v -= r.step()
	case Right:
		v += r.step()
	case Home:
		v = r.Min
	case End:
		v = r.Max
	case PageUp:
		v += page
	case PageDown:
		v -= page
	case None, Activate, Back, Up, Down: // Handles lets only the steps through
	}
	return r.SetValue(v)
}

// step is how far an arrow moves: Step, or a hundredth of the range for
// any value.
func (r *Range) step() float64 {
	if r.Step > 0 {
		return r.Step
	}
	return (r.Max - r.Min) / 100
}

// SetValue puts the range at v, on its step and within its bounds.
func (r *Range) SetValue(v float64) Events {
	v = snap(v, r.Min, r.Max, r.Step)
	changed := v != r.Value
	r.Value = v
	return when(changed, Input|Changed)
}

// Fraction is where the value lies along the track, 0 to 1.
func (r *Range) Fraction() float64 {
	if r.Max <= r.Min {
		return 0
	}
	return (r.Value - r.Min) / (r.Max - r.Min)
}

// Press starts a drag at fraction f of the track.
func (r *Range) Press(f float64) Events {
	if r.Disabled {
		return 0
	}
	r.dragging, r.start = true, r.Value
	return r.Drag(f)
}

// Drag moves the value to fraction f of the track while dragging.
func (r *Range) Drag(f float64) Events {
	if !r.dragging {
		return 0
	}
	return r.SetValue(r.Min+f*(r.Max-r.Min)) &^ Changed
}

// Release ends a drag, Changed if it moved the value.
func (r *Range) Release() Events {
	if !r.dragging {
		return 0
	}
	r.dragging = false
	return when(r.Value != r.start, Changed)
}

func (r *Range) Dragging() bool { return r.dragging }

// snap is v on step from lo, within [lo, hi], rounded to the step's
// decimals so 0.1 + 0.1 stays 0.2. A bound that is NaN doesn't hold.
func snap(v, lo, hi, step float64) float64 {
	if step > 0 {
		base := lo
		if math.IsNaN(base) {
			base = 0
		}
		v = base + math.Round((v-base)/step)*step
		if !math.IsNaN(hi) && v > hi {
			v -= math.Ceil((v-hi)/step-1e-9) * step
		}
		v = round(v, decimals(step)+decimals(base))
	}
	if !math.IsNaN(lo) {
		v = max(v, lo)
	}
	if !math.IsNaN(hi) {
		v = min(v, hi)
	}
	return v
}

func decimals(f float64) int {
	for n := 0; n < 12; n++ {
		if s := f * math.Pow10(n); math.Abs(s-math.Round(s)) < 1e-9 {
			return n
		}
	}
	return 12
}

func round(v float64, n int) float64 {
	p := math.Pow10(n)
	return math.Round(v*p) / p
}
