package kit

import "github.com/merlindorin/ember/ui"

// Alert is a dialog of class "kit-alert" asking title,
// explained by message, answered by buttons, with the old ember/ui's icon above.
// Show it with Document.ShowModal: the page behind it is inert and the
// focus stays in it, starting on its "kit-primary" button, which Enter
// presses, else on its first button that isn't "kit-destructive", so Enter
// never destroys by default; Escape cancels. Its title is its aria-label,
// so automation names it by its heading. A button answers it: it closes and
// ui.Closed{Node: alert, Value} reports the button's value (its text
// unless ui.Value sets one), Escape an empty one. "kit-destructive" buttons
// stand on the left, the others on the right.
func Alert(title, message string, buttons ...*ui.Node) *ui.Node {
	left := ui.Div(ui.Class("kit-alert-buttons-left"))
	right := ui.Div(ui.Class("kit-alert-buttons-right"))
	var first, primary *ui.Node
	for _, b := range buttons {
		b.SetAttr("formmethod", "dialog")
		if _, ok := b.Attr("value"); !ok {
			b.SetAttr("value", b.Text())
		}
		if b.HasClass("kit-primary") && primary == nil {
			primary = b
		}
		if !b.HasClass("kit-destructive") && first == nil {
			first = b
		}
		if b.HasClass("kit-destructive") {
			left.Append(b)
		} else {
			right.Append(b)
		}
	}
	if primary == nil {
		primary = first
	}
	if primary != nil {
		primary.SetAttr("autofocus", "")
	}
	return ui.Dialog(ui.Class("kit-alert"), ui.Attr("aria-label", title),
		ui.Div(ui.Class("kit-alert-icon"), ui.Text("!")),
		ui.H2(ui.Class("kit-alert-title"), ui.Text(title)),
		ui.P(ui.Class("kit-alert-message"), ui.Text(message)),
		ui.Div(ui.Class("kit-alert-buttons"), left, right),
	)
}

// SetAlert changes an Alert's title, its aria-label with it, and its
// message, so one alert asks again about something else.
func SetAlert(alert *ui.Node, title, message string) {
	alert.SetAttr("aria-label", title)
	for _, k := range alert.Children() {
		switch {
		case k.HasClass("kit-alert-title"):
			k.SetText(title)
		case k.HasClass("kit-alert-message"):
			k.SetText(message)
		}
	}
}
