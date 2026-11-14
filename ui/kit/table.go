package kit

import (
	"strconv"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
)

// TableColumn is a Table's column: a th reading title. ColumnWidth,
// ColumnAlign and Sortable set it up.
func TableColumn(title string, args ...ui.Arg) *ui.Node {
	return ui.Th(append([]ui.Arg{ui.Text(title)}, args...)...)
}

// ColumnWidth gives a column at least w points; the others share the
// rest.
func ColumnWidth(w float32) ui.Arg {
	return ui.Style("width: " + strconv.FormatFloat(float64(w), 'f', -1, 32) + "px")
}

// ColumnAlign aligns a column's title and cells: "right" for sizes, or
// "center".
func ColumnAlign(align string) ui.Arg { return ui.Class("kit-" + align) }

// Sortable lets a press on the column's title sort the rows by its cells,
// ascending then reversed: by each cell's SortValue, else its text, as
// numbers when both read as one, else ignoring case.
func Sortable() ui.Arg { return ui.Attr("aria-sort", "none") }

// NoHeader hides a Table's titles.
func NoHeader() ui.Arg { return ui.Class("kit-no-header") }

// TableRow is a row of a Table, one cell per text.
func TableRow(cells ...string) *ui.Node {
	r := ui.Tr()
	for _, c := range cells {
		r.Append(ui.Td(ui.Text(c)))
	}
	return r
}

// CellOf is row's cell in column i, the one spanning it when cells before
// span several; nil past the row's end. It counts the row's own colspans,
// not cells spanning down into it from the rows above.
func CellOf(row *ui.Node, i int) *ui.Node {
	col := 0
	for _, c := range row.Children() {
		if c.IsText() {
			continue
		}
		if col += colspan(c); i < col {
			return c
		}
	}
	return nil
}

// colspan and rowspan are how many columns and rows the cell c spans, as
// layout reads them; a rowspan of 0 counts as 1 here.
func colspan(c *ui.Node) int { return spanOf(c, "colspan", 1000) }
func rowspan(c *ui.Node) int { return spanOf(c, "rowspan", 65534) }

func spanOf(c *ui.Node, name string, most int) int {
	v, _ := c.Attr(name)
	n, err := strconv.Atoi(v)
	if err != nil {
		return 1
	}
	return min(max(n, 1), most)
}

// SortValue is what row sorts by in column i, rather than its text, such
// as a size in bytes for "24 KB"; for columns one cell spans, it is that
// cell's, the last set winning.
func SortValue(row *ui.Node, i int, value string) { CellOf(row, i).SetAttr("data-sort", value) }

// SetDimmed shows row greyed out, never selected.
func SetDimmed(row *ui.Node, on bool) {
	if on {
		row.SetAttr("aria-disabled", "true")
	} else {
		row.RemoveAttr("aria-disabled")
	}
}

// Table is a div of class "kit-table" that scrolls, holding
// a table of role grid with columns' titles over rows. ui gives it
// WAI-ARIA's grid behaviour: a click or the arrows select a row
// (ui.Changed for TableGrid), Enter or a double click activate it
// (ui.Submitted for TableGrid), and a Sortable title sorts the rows
// (ui.Clicked for the title); Document.Selected and SetSelected of
// TableGrid read and set the selected row. A ui.Caption or ui.Tfoot
// among args goes in the grid, the caption over it and the footer's rows
// under the body, aligned as its columns; every other arg goes on the
// div, such as NoHeader.
func Table(columns, rows []*ui.Node, args ...ui.Arg) *ui.Node {
	head := ui.Tr()
	for _, c := range columns {
		head.Append(c)
	}
	body := ui.Tbody()
	align(columns, rows)
	body.Append(rows...)
	grid := ui.Table(ui.Attr("role", "grid"))
	var foot *ui.Node
	div := []ui.Arg{ui.Class("kit-table"), grid}
	for _, a := range args {
		switch n, _ := a.(*ui.Node); {
		case n != nil && n.Tag() == dom.Caption:
			grid.Append(n)
		case n != nil && n.Tag() == dom.Tfoot:
			foot = n
			align(columns, n.Children())
		default:
			div = append(div, a)
		}
	}
	grid.Append(ui.Thead(head), body)
	if foot != nil {
		grid.Append(foot)
	}
	return ui.Div(div...)
}

// align gives each cell of rows its column's kit-right or kit-center, the
// columns found past titles spanning several and cells spanning down from
// the rows above.
func align(columns, rows []*ui.Node) {
	var heads []*ui.Node
	for _, c := range columns {
		for range colspan(c) {
			heads = append(heads, c)
		}
	}
	var busy []int
	for _, r := range rows {
		col := 0
		for _, cell := range r.Children() {
			if cell.IsText() {
				continue
			}
			for col < len(busy) && busy[col] > 0 {
				col++
			}
			for len(busy) < col+colspan(cell) {
				busy = append(busy, 0)
			}
			for k := col; k < col+colspan(cell); k++ {
				busy[k] = rowspan(cell)
			}
			if col < len(heads) {
				for _, a := range []string{"kit-right", "kit-center"} {
					if heads[col].HasClass(a) {
						cell.AddClass(a)
					}
				}
			}
			col += colspan(cell)
		}
		for k := range busy {
			busy[k] = max(busy[k]-1, 0)
		}
	}
}

// TableGrid is table's grid, whose events a Table's are.
func TableGrid(table *ui.Node) *ui.Node {
	for _, c := range table.Children() {
		if c.Tag() == dom.Table {
			return c
		}
	}
	return nil
}

// TableRows are table's rows, in the order they show.
func TableRows(table *ui.Node) []*ui.Node { return tableBody(TableGrid(table)).Children() }

// tableBody is grid's tbody, wherever its caption puts it.
func tableBody(grid *ui.Node) *ui.Node {
	for _, c := range grid.Children() {
		if c.Tag() == dom.Tbody {
			return c
		}
	}
	return nil
}

// SortedBy sorts table by column, descending or not, as a press on its
// title does but without an event, and shows that title's arrow, the
// others none: a table's first order, or the one a game keeps.
func SortedBy(d *ui.Document, table *ui.Node, column int, descending bool) {
	d.Sort(TableGrid(table), column, descending)
}
