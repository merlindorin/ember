package kit

import "github.com/lantervale/ember/ui"

// Switch is the old ember/ui's Check drawn as a switch: a label of class "kit-switch"
// holding a checkbox with the switch attribute, of class "kit-switch-track",
// and its text, so a click on either flips it. args go to the checkbox; a
// disabled one greys its text too.
func Switch(text string, args ...ui.Arg) *ui.Node {
	track := ui.Input(append([]ui.Arg{ui.Type("checkbox"), ui.Attr("switch", ""), ui.Class("kit-switch-track")},
		args...)...)
	label := ui.Label(ui.Class("kit-switch"), track, ui.Span(ui.Class("kit-switch-text"), ui.Text(text)))
	if _, off := track.Attr("disabled"); off {
		label.AddClass("kit-disabled")
	}
	return label
}

// SwitchTrack is the checkbox of a Switch, for reading or setting it.
func SwitchTrack(sw *ui.Node) *ui.Node {
	for _, c := range sw.Children() {
		if c.HasClass("kit-switch-track") {
			return c
		}
	}
	return nil
}
