package control

import "sync/atomic"

// Dialog is a dialog element: shown, or shown modal, when Back closes it
// with an empty return value and ui traps the focus inside it.
type Dialog struct {
	Open, Modal bool
	ReturnValue string
	// Layer orders modal dialogs in the top layer: the one shown modal
	// last is the highest.
	Layer uint64
}

var layers atomic.Uint64 //nolint:gochecknoglobals // the one count that orders every dialog shown

func (d *Dialog) Show() { d.Open, d.Modal = true, false }

func (d *Dialog) ShowModal() { d.Open, d.Modal, d.Layer = true, true, layers.Add(1) }

// Close closes the dialog with value as its return value: Closed, once.
func (d *Dialog) Close(value string) Events {
	if !d.Open {
		return 0
	}
	d.Open, d.Modal, d.ReturnValue = false, false, value
	return Closed
}

func (d *Dialog) Handles(i Intent) bool { return i == Back && d.Open && d.Modal }

func (d *Dialog) Do(i Intent) Events {
	if !d.Handles(i) {
		return 0
	}
	return d.Close("")
}

// Progress is a progress element: Value of Max (1 when 0), or
// indeterminate when it has no value.
type Progress struct {
	Value, Max    float64
	Indeterminate bool
}

func (p *Progress) Fraction() float64 {
	if p.Indeterminate {
		return 0
	}
	m := p.Max
	if m <= 0 {
		m = 1
	}
	return min(max(p.Value/m, 0), 1)
}

// Region is where a meter's value lies for its optimum, which sets its
// colour.
type Region int

const (
	Optimum Region = iota
	Suboptimal
	EvenLessGood
)

// Meter is a meter element: Value in [Min, Max], with Low, High and
// Optimum splitting it into regions as HTML does.
type Meter struct {
	Value, Min, Max, Low, High, Optimum float64
}

// NewMeter is a meter at value with HTML's defaults: 0 to 1, Low and High
// at the ends, Optimum in the middle.
func NewMeter(value float64) *Meter {
	return &Meter{Value: value, Max: 1, High: 1, Optimum: 0.5}
}

func (m *Meter) Fraction() float64 {
	if m.Max <= m.Min {
		return 0
	}
	return min(max((m.Value-m.Min)/(m.Max-m.Min), 0), 1)
}

func (m *Meter) Region() Region {
	v := min(max(m.Value, m.Min), m.Max)
	low, high := max(m.Low, m.Min), min(m.High, m.Max)
	switch {
	case m.Optimum < low:
		return pickRegion(v < low, v <= high)
	case m.Optimum > high:
		return pickRegion(v > high, v >= low)
	case v < low || v > high:
		return Suboptimal
	}
	return Optimum
}

func pickRegion(best, middle bool) Region {
	switch {
	case best:
		return Optimum
	case middle:
		return Suboptimal
	}
	return EvenLessGood
}
