package kit

import "github.com/lantervale/ember/ui"

// PopUp is a select of class "kit-pop-up" showing items,
// item selected shown (-1 for none, which shows a ui.Placeholder), with
// The old ember/ui's chevron; a click, Enter, Space or the pad's south button
// opens the items under it, and choosing one emits ui.Edited, then
// ui.Changed.
func PopUp(selected int, items []string, args ...ui.Arg) *ui.Node {
	opts := []ui.Arg{ui.Class("kit-pop-up")}
	for i, it := range items {
		o := []ui.Arg{ui.Text(it)}
		if i == selected {
			o = append(o, ui.Selected())
		}
		opts = append(opts, ui.Option(o...))
	}
	return ui.Select(append(opts, args...)...)
}
