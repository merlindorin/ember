package kit

import "github.com/merlindorin/ember/ui"

// Tooltip is tip, any element such as a Label or a
// Column of them, shows beside its element while the pointer is over it,
// or while it has the focus from the keys or a pad, above the rest. It
// becomes the element's child, of class "kit-tooltip" and role tooltip, so
// the element can't be a void one (an input or an img): give it to the
// label holding one. Class "kit-left" puts the tip on the left, "kit-below" under
// it. It hangs past its element on purpose, marked data-overflow-ok for
// align's spill rule.
func Tooltip(tip *ui.Node) ui.Arg { return tooltip{tip} }

type tooltip struct{ tip *ui.Node }

func (t tooltip) Apply(n *ui.Node) {
	n.AddClass("kit-has-tooltip")
	t.tip.AddClass("kit-tooltip")
	t.tip.SetAttr("data-overflow-ok", "")
	t.tip.SetAttr("role", "tooltip")
	n.Append(t.tip)
}
