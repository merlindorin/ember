package kit

import "github.com/lantervale/ember/ui"

// ColorRect is a div of class "kit-color-rect" filled
// with c, a CSS colour. It is as large as its box: give it a size with a
// class or the style of its parent's layout.
func ColorRect(c string, args ...ui.Arg) *ui.Node {
	return ui.Div(append([]ui.Arg{ui.Class("kit-color-rect"), ui.Style("background-color: " + c)}, args...)...)
}
