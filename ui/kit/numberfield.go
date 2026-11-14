package kit

import (
	"strconv"

	"github.com/lantervale/ember/ui"
)

// NumberField is an input of type number and
// class "kit-number-field" holding value. Up and Down step it, ten steps
// with Shift, as the wheel does over it while it has the focus; what is
// typed is taken on Enter or when the focus leaves, kept within its
// Range on its Step, and text that isn't a number gives the value back.
// Each step, by key or wheel, emits ui.Edited, and ui.Changed comes
// on Enter or when the focus leaves, as a range input's do.
func NumberField(value float64, args ...ui.Arg) *ui.Node {
	return ui.Input(append([]ui.Arg{ui.Type("number"), ui.Class("kit-number-field"), ui.Value(format(value))}, args...)...)
}

// Range keeps a NumberField's, a Slider's or a Progress's value within lo
// and hi; a Progress counts from lo, as a progress element has no min.
func Range(lo, hi float64) ui.Arg {
	return args{ui.Attr("min", format(lo)), ui.Attr("max", format(hi))}
}

// Step is what Up and Down add to a NumberField or a Slider, and where
// their values snap, counted from the Range's low end.
func Step(s float64) ui.Arg { return ui.Attr("step", format(s)) }

func format(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
