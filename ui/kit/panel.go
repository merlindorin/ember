package kit

import "github.com/lantervale/ember/ui"

// Panel is a section of class "kit-panel" on the panel colour, with a
// border and rounded corners. Title gives it a title bar. Classes "kit-checked" (the old ember/ui's
// toggled look) and "kit-glow" (its Glow) change it.
func Panel(args ...ui.Arg) *ui.Node {
	return ui.Section(append([]ui.Arg{ui.Class("kit-panel")}, args...)...)
}

// Title is a panel's title bar, the old ember/ui's Title: a header of class "kit-panel-title". It goes
// first, before the panel's children.
func Title(text string) ui.Arg {
	return ui.Header(ui.Class("kit-panel-title"), ui.Text(text))
}

// CollapsiblePanel is the old ember/ui's Collapsible titled panel: a details of class "kit-panel"
// whose summary is its title bar. A click on the bar, or Enter or Space on it, folds the
// panel to its bar or opens it, emitting ui.Toggled for the panel; a click also emits
// ui.Clicked for the bar.
func CollapsiblePanel(title string, open bool, args ...ui.Arg) *ui.Node {
	head := []ui.Arg{ui.Class("kit-panel"), ui.Summary(ui.Class("kit-panel-title"), ui.Text(title))}
	if open {
		head = append(head, ui.Open())
	}
	return ui.Details(append(head, args...)...)
}

// Card is the old ember/ui's Pressable panel of class "kit-card": a button of class "panel card" on
// the window colour, with a glow, that a click, Enter or Space presses (ui.Clicked).
func Card(args ...ui.Arg) *ui.Node {
	return ui.Button(append([]ui.Arg{ui.Class("kit-panel", "kit-card")}, args...)...)
}

// Glow is the old ember/ui's Glow(c, size): a halo of colour c, a CSS colour, full at the box's
// edge and fading to nothing size px out. ui fades as (1 - d/size)², which a CSS shadow
// spread by size/4.5 and blurred by size/1.5 follows within a few percent.
func Glow(c string, size float32) ui.Arg {
	return declaration{"box-shadow", "0 0 " + pxs(size/1.5) + " " + pxs(size/4.5) + " " + c}
}
