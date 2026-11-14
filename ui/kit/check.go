package kit

import "github.com/lantervale/ember/ui"

// Check is a label of class "kit-check" holding a checkbox
// of class "kit-check-box" and its text, so a click on either toggles it. args
// go to the checkbox, where ui.Checked and ui.Disabled belong; a
// disabled one greys its text too.
func Check(text string, args ...ui.Arg) *ui.Node {
	box := ui.Input(append([]ui.Arg{ui.Type("checkbox"), ui.Class("kit-check-box")}, args...)...)
	label := ui.Label(ui.Class("kit-check"), box, ui.Span(ui.Class("kit-check-text"), ui.Text(text)))
	if _, off := box.Attr("disabled"); off {
		label.AddClass("kit-disabled")
	}
	return label
}

// CheckBox is the checkbox of a Check, for reading or setting it.
func CheckBox(check *ui.Node) *ui.Node {
	for _, c := range check.Children() {
		if c.HasClass("kit-check-box") {
			return c
		}
	}
	return nil
}
