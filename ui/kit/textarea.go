package kit

import (
	"fmt"

	"github.com/merlindorin/ember/ui"
)

// TextArea is a textarea of class "kit-text-area"
// holding text, five lines high unless Rows says otherwise; NoWrap keeps
// its lines whole, scrolling sideways.
func TextArea(text string, args ...ui.Arg) *ui.Node {
	return ui.Textarea(append([]ui.Arg{ui.Class("kit-text-area"), ui.Text(text)}, args...)...)
}

// areaLine is a TextArea's line height, the old ember/ui's for its 15-point text,
// and areaPad the room above and below its lines.
const (
	areaLine = 18
	areaPad  = 12
)

// Rows makes a TextArea n lines high.
func Rows(n int) ui.Arg {
	return ui.Style(fmt.Sprintf("height: %dpx", max(n, 1)*areaLine+areaPad))
}

// NoWrap keeps a TextArea's lines whole, scrolling sideways instead;
// SetNoWrap changes it while the game runs, and Wraps reads it.
func NoWrap() ui.Arg { return ui.Class("kit-no-wrap") }

// SetNoWrap keeps area's lines whole, or wraps them again.
func SetNoWrap(area *ui.Node, on bool) {
	if on {
		area.AddClass("kit-no-wrap")
	} else {
		area.RemoveClass("kit-no-wrap")
	}
}

// Wraps reports whether area wraps its lines, as it does unless NoWrap.
func Wraps(area *ui.Node) bool { return !area.HasClass("kit-no-wrap") }
