package kit

import (
	"strconv"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
)

// Choice is one of its items selected, as tabs, cards
// or swatches. It is a div of class "kit-choice" and role radiogroup, each
// item in a button of class "kit-choice-item" and role radio, item selected
// checked (-1 for none); a click or Enter, Space or the pad's south button
// selects one, the arrows move the selection, and each change emits
// ui.Changed for the item's button, also when it was already selected.
// Class "kit-segmented" joins the items as tabs. An item made by ChoiceItem
// is used as it is, with its args, such as a name.
func Choice(selected int, items []*ui.Node, args ...ui.Arg) *ui.Node {
	c := ui.Div(ui.Class("kit-choice"), ui.Attr("role", "radiogroup"))
	for i, it := range items {
		checked := strconv.FormatBool(i == selected)
		if !it.HasClass("kit-choice-item") {
			it = ChoiceItem(it)
		}
		it.SetAttr("aria-checked", checked)
		c.Append(it)
	}
	for _, a := range args {
		a.Apply(c)
	}
	return c
}

// Chosen is the place of choice's selected item, -1 for none.
func Chosen(choice *ui.Node) int {
	i := 0
	for _, c := range choice.Children() {
		if c.IsText() || c.Tag() != dom.Button {
			continue
		}
		if v, _ := c.Attr("aria-checked"); v == "true" {
			return i
		}
		i++
	}
	return -1
}

// ChoiceItem is an item of a Choice holding content, a button of class
// "kit-choice-item" and role radio taking args, such as ui.ID or a title.
func ChoiceItem(content *ui.Node, args ...ui.Arg) *ui.Node {
	return ui.Button(append([]ui.Arg{ui.Class("kit-choice-item"), ui.Attr("role", "radio"), content}, args...)...)
}

// SetChosen selects choice's item i, -1 for none, without an event.
func SetChosen(choice *ui.Node, i int) {
	k := 0
	for _, c := range choice.Children() {
		if c.IsText() || c.Tag() != dom.Button {
			continue
		}
		if k == i {
			c.SetAttr("aria-checked", "true")
		} else {
			c.SetAttr("aria-checked", "false")
		}
		k++
	}
}
