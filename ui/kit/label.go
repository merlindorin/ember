package kit

import "github.com/merlindorin/ember/ui"

// Label is a line of text, the old ember/ui's Label: a span of class "kit-label",
// kept on one line. Looks change it: Wrap wraps it in its box, Truncate
// cuts it there with an ellipsis, Center and Right align it, and Caps,
// FormLabel, AlertTitle and AlertMessage are the old ember/ui's text styles.
func Label(text string, args ...ui.Arg) *ui.Node {
	return ui.Span(append([]ui.Arg{ui.Class("kit-label"), ui.Text(text)}, args...)...)
}
