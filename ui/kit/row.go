package kit

import (
	"strconv"

	"github.com/lantervale/ember/ui"
)

// Row lines its children up left to right, the old ember/ui's Row: a flex div of class "kit-row",
// 4 points apart, stretched to its height. Classes change it: "kit-justify-start",
// "kit-justify-center", "kit-justify-end" and "kit-justify-between" place the children along it,
// "kit-align-start", "kit-align-center" and "kit-align-end" across it, and "kit-toolbar", "kit-body" and
// "kit-footer" are the old ember/ui's paddings and gaps. A child of class "kit-expand" grows into the
// room left, and one of class "kit-fill" stretches across.
func Row(args ...ui.Arg) *ui.Node {
	return ui.Div(append([]ui.Arg{ui.Class("kit-row")}, args...)...)
}

// Column stacks its children top to bottom, the old ember/ui's Column: Row's classes, down.
func Column(args ...ui.Arg) *ui.Node {
	return ui.Div(append([]ui.Arg{ui.Class("kit-column")}, args...)...)
}

// Gap keeps px points between a row's, column's or grid's children, the old ember/ui's Gap.
// Gap and Padding add to the style attribute, so they go after a ui.Style, which
// replaces it.
func Gap(px float32) ui.Arg { return declare("gap", px) }

// Padding keeps px points between a container's edge and its children, the old ember/ui's Padding.
func Padding(px float32) ui.Arg { return declare("padding", px) }

// declaration adds one property to an element's style attribute, beside what it has.
type declaration struct{ property, value string }

func declare(property string, px float32) declaration {
	return declaration{property, pxs(px)}
}

func pxs(v float32) string { return strconv.FormatFloat(float64(v), 'f', -1, 32) + "px" }

func (d declaration) Apply(n *ui.Node) {
	s := n.Style()
	if s != "" {
		s += "; "
	}
	n.SetStyle(s + d.property + ": " + d.value)
}
