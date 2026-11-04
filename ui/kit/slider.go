package kit

import "github.com/merlindorin/ember/ui"

// Slider is a range input of class "kit-slider" at value,
// between 0 and 100 unless Range bounds it and on its Step, dragged,
// clicked or stepped by the arrow keys.
func Slider(value float64, args ...ui.Arg) *ui.Node {
	return ui.Input(append([]ui.Arg{ui.Type("range"), ui.Class("kit-slider"), ui.Value(format(value))}, args...)...)
}
