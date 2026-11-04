package kit

import "github.com/merlindorin/ember/ui"

// StatusBar is a div of class "kit-status-bar" along a
// window's bottom reading text in a muted Label of class "kit-status", cut
// with an ellipsis at its end, then children, then an accent dot of class
// "kit-status-dot" while something is unsaved (SetDot). args go on the bar:
// children, classes, an ID.
func StatusBar(text string, args ...ui.Arg) *ui.Node {
	b := ui.Div(ui.Class("kit-status-bar"), Label(text, ui.Class("kit-status", "kit-truncate")))
	for _, a := range args {
		a.Apply(b)
	}
	b.Append(ui.Span(ui.Class("kit-status-dot"), ui.Attr("hidden", "")))
	return b
}

// Status is the text bar reads.
func Status(bar *ui.Node) string { return bar.Children()[0].Text() }

// SetStatus makes bar read text.
func SetStatus(bar *ui.Node, text string) { bar.Children()[0].SetText(text) }

// Dot reports whether bar's dot shows: something is unsaved.
func Dot(bar *ui.Node) bool { _, hidden := dotOf(bar).Attr("hidden"); return !hidden }

// SetDot shows bar's dot, or hides it.
func SetDot(bar *ui.Node, on bool) {
	if on {
		dotOf(bar).RemoveAttr("hidden")
	} else {
		dotOf(bar).SetAttr("hidden", "")
	}
}

func dotOf(bar *ui.Node) *ui.Node { c := bar.Children(); return c[len(c)-1] }
