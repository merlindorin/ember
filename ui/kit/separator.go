package kit

import "github.com/lantervale/ember/ui"

// Separator is an hr of class "kit-separator", a 1-point
// line in the border colour, inset 6 points at its ends, in a 9-point
// slot, as between a menu's items. Class "kit-divider" makes it the plain
// line across the old ember/ui's forms and dialogs, and "kit-vertical" stands it up,
// one line of text tall, between the items of a row.
func Separator(args ...ui.Arg) *ui.Node {
	return ui.Hr(append([]ui.Arg{ui.Class("kit-separator")}, args...)...)
}
