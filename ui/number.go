package ui

import (
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/inputfeed"
	"github.com/lantervale/ember/ui/control"
)

// settle readies a number input's commit, as the old ember/ui's NumberField
// takes its text: kept within min and max on a step, and what doesn't
// read as a number back to its value. Chrome keeps such text and reports
// it invalid; a game's field holds a number.
func settle(m any) {
	if num, ok := m.(*control.Number); ok {
		num.Settle()
	}
}

// stepNumber steps the number input n one step by Up or Down, ten with
// Shift as the old ember/ui's NumberField does.
func (d *Document) stepNumber(n *Node, num *control.Number, i control.Intent, shift bool) {
	times := 1
	if shift {
		times = 10
	}
	for range times {
		d.fire(n, num.Do(i))
	}
}

// wheelNumber steps the focused number input by the wheel while the
// pointer is over it, as Chrome does, Shift ten steps at a time. It
// reports whether the wheel went to it rather than to scrolling.
func (d *Document) wheelNumber(in *input.State) bool {
	w := in.Wheel().Y
	n := d.focused()
	num, ok := d.machine(n).(*control.Number)
	if w == 0 || !ok || !contains(d.hovered, n) {
		return false
	}
	i := control.Up
	if w < 0 {
		i = control.Down
	}
	d.stepNumber(n, num, i, inputfeed.Of(in).RawDown(int(input.KeyShift)))
	return true
}
