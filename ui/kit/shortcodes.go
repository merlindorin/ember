package kit

import (
	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/ui"
)

// ShortcodeField is the old ember/ui's Shortcodes for a TextInput or TextArea: a
// span of class "kit-shortcode-field" holding field, which then turns :name:
// into its emoji at the closing colon, and the list of class
// "kit-suggestions" under it, which offers up to six emoji whose names start
// with the one being typed and that the field's face draws. ui gives
// them WAI-ARIA's combobox behaviour: the focus stays in the field, Up
// and Down or the pad walk the list, Tab, Enter, South or a click choose,
// and Escape or East close it. The list's id is the field's name (its id,
// data-name, aria-label or placeholder's slug, else "shortcodes") then
// "-suggestions", the same at every run, so fields in one Document need
// names of their own.
func ShortcodeField(field *ui.Node) *ui.Node {
	id := nameOf(field, "shortcodes") + "-suggestions"
	for _, a := range [][2]string{{"data-shortcodes", ""}, {"role", "combobox"}, {"aria-autocomplete", "list"},
		{"aria-expanded", "false"}, {"aria-controls", id}} {
		field.SetAttr(a[0], a[1])
	}
	list := ui.Div(ui.ID(id), ui.Class("kit-suggestions"), ui.Attr("role", "listbox"), ui.Attr("hidden", ""))
	return ui.Span(ui.Class("kit-shortcode-field"), field, list)
}

// SuggestionList is a ShortcodeField's list.
func SuggestionList(field *ui.Node) *ui.Node { return field.Children()[1] }

// nameOf is n's own automation name as it is built, before it has a path:
// its id, data-name or aria-label, else its placeholder's slug, else
// fallback.
func nameOf(n *ui.Node, fallback string) string {
	if n.ID() != "" {
		return n.ID()
	}
	for _, a := range []string{"data-name", "aria-label"} {
		if v, _ := n.Attr(a); v != "" {
			return v
		}
	}
	if p, _ := n.Attr("placeholder"); automation.Slug(p) != "" {
		return automation.Slug(p)
	}
	return fallback
}
