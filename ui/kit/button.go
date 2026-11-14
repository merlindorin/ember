package kit

import "github.com/lantervale/ember/ui"

// Button is a button of class "kit-button" showing text.
// Classes "kit-primary" and "kit-destructive" are the old ember/ui's call-to-action and
// dangerous ones; ui.Disabled() greys it out.
func Button(text string, args ...ui.Arg) *ui.Node {
	return ui.Button(append([]ui.Arg{ui.Class("kit-button"), ui.Text(text)}, args...)...)
}
