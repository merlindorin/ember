package kit

import (
	"strconv"

	"github.com/merlindorin/ember/ui"
)

// Grid places its children in rows of columns cells, the old ember/ui's Grid: a CSS grid of class
// "kit-grid", each column as wide as its widest cell, 4 points apart, the cells of a row on
// one baseline, so a label lines up with its value. Columns gives other tracks, so a column can take the room left.
func Grid(columns int, args ...ui.Arg) *ui.Node {
	tracks := declaration{gridTemplateColumns, "repeat(" + strconv.Itoa(max(columns, 1)) + ", max-content)"}
	return ui.Div(append([]ui.Arg{ui.Class("kit-grid"), tracks}, args...)...)
}

// Columns sets a grid's column tracks as CSS writes them, such as "max-content 1fr". It
// adds to the style attribute, so it goes after a ui.Style.
// gridTemplateColumns is the property a grid's tracks are set by.
const gridTemplateColumns = "grid-template-columns"

func Columns(tracks string) ui.Arg { return declaration{gridTemplateColumns, tracks} }
