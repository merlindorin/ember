package kit

import "github.com/merlindorin/ember/ui"

// Scroll is a div of class "kit-scroll" that clips its content and scrolls
// it, with bars while it overflows. It may shrink below its content in a row or column.
func Scroll(args ...ui.Arg) *ui.Node {
	return ui.Div(append([]ui.Arg{ui.Class("kit-scroll")}, args...)...)
}

// BarMode is when a scrollbar shows, the old ember/ui's.
type BarMode int

const (
	// BarAuto shows the bar when the content overflows that way.
	BarAuto BarMode = iota
	// BarAlways shows it, and reserves its room, even when everything fits.
	BarAlways
	// BarNever hides it while the content still turns by the wheel and keys: overflow
	// auto with scrollbar-width: none, which hides both axes' bars, so BarNever on one
	// axis hides the other's too.
	BarNever
)

// VerticalBar and HorizontalBar set when a Scroll's bars show, as overflow-y and
// overflow-x.
func VerticalBar(m BarMode) ui.Arg   { return bar("overflow-y", m) }
func HorizontalBar(m BarMode) ui.Arg { return bar("overflow-x", m) }

func bar(property string, m BarMode) ui.Arg {
	switch m {
	case BarAlways:
		return declaration{property, "scroll"}
	case BarNever:
		return declarations{{property, "auto"}, {"scrollbar-width", "none"}}
	case BarAuto:
		// auto, as below
	}
	return declaration{property, "auto"}
}

type declarations []declaration

func (ds declarations) Apply(n *ui.Node) {
	for _, d := range ds {
		d.Apply(n)
	}
}
