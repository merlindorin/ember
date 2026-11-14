package ui

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/lantervale/ember/ui/paint"
)

// A grid is a table of role grid, as WAI-ARIA's grid pattern: one Tab
// stop whose body rows (tr outside thead) the player selects, the
// selected one with aria-selected="true", those with aria-disabled="true"
// dimmed and skipped. A click or Up, Down, Home and End select a row,
// emitting Changed for the grid; Enter, Space, the pad's south button or
// a double click activate the selection, emitting Submitted for it. A
// header cell with aria-sort sorts the rows by its column when pressed,
// ascending then reversed: by each cell's data-sort, else its text, as
// numbers when both read as one, else ignoring case. The press emits
// Clicked for the header.

// Sorted is a grid sorted by a press on its header: Column is the
// header's place, Descending the order now shown. aria-sort says the same
// by the time it comes.
type Sorted struct {
	Node       *Node
	Column     int
	Descending bool
}

// SortBy has the grid sort its rows by compare in place of the default,
// told the column and the direction, which it applies itself (so a file
// list keeps its folders first both ways); nil goes back to the default.
func (d *Document) SortBy(grid *Node, compare func(a, b *Node, column int, descending bool) int) {
	if d.sorts == nil {
		d.sorts = map[*Node]func(a, b *Node, column int, descending bool) int{}
	}
	if compare == nil {
		delete(d.sorts, grid)
		return
	}
	d.sorts[grid] = compare
}

// doubleClick is how close, in seconds, two clicks on a row activate it.
const doubleClick = 0.4

func isGrid(n *Node) bool { return roleOf(n) == "grid" }

// gridOf is the grid n is in, or is, nil for none.
func gridOf(n *Node) *Node {
	for p := n; p != nil; p = p.Parent() {
		if isGrid(p) {
			return p
		}
	}
	return nil
}

func inHead(n *Node) bool { return inGroup(n, dom.Thead) }

// inGroup reports whether n is inside a row group of tag within its grid.
func inGroup(n *Node, tag dom.Tag) bool {
	for p := n.Parent(); p != nil && !isGrid(p); p = p.Parent() {
		if p.Tag() == tag {
			return true
		}
	}
	return false
}

// gridRows are grid's body rows, in tree order: its head's and foot's are not.
func gridRows(grid *Node) []*Node {
	var out []*Node
	grid.Walk(func(n *Node) bool {
		if n.IsText() || n != grid && isGrid(n) || n.Tag() == dom.Thead || n.Tag() == dom.Tfoot {
			return false
		}
		if n.Tag() == dom.Tr {
			out = append(out, n)
			return false
		}
		return true
	})
	return out
}

// isGridRow reports whether n is a grid's body row.
func isGridRow(n *Node) bool {
	return n.Tag() == dom.Tr && !inHead(n) && !inGroup(n, dom.Tfoot) && gridOf(n) != nil
}

// isSortHeader reports whether n is a grid's header cell that sorts.
func isSortHeader(n *Node) bool {
	return n.Tag() == dom.Th && has(n, "aria-sort") && gridOf(n) != nil
}

func ariaDisabled(n *Node) bool { v, _ := n.Attr("aria-disabled"); return v == attrTrue }
func ariaSelected(n *Node) bool { v, _ := n.Attr("aria-selected"); return v == attrTrue }

// selectRow selects row among its grid's, reporting whether that changed
// anything.
func selectRow(grid, row *Node) bool {
	if ariaDisabled(row) || ariaSelected(row) {
		return false
	}
	for _, r := range gridRows(grid) {
		if r != row && ariaSelected(r) {
			r.SetAttr("aria-selected", "false")
		}
	}
	row.SetAttr("aria-selected", attrTrue)
	return true
}

// gridClick is a click on n inside a grid: a sorting header sorts, a row
// is selected, then activated by a second click on it soon after. It
// reports whether n was either.
func (d *Document) gridClick(n *Node) bool {
	grid := gridOf(n)
	if grid == nil {
		return false
	}
	if isSortHeader(n) {
		d.sortGrid(grid, n)
		d.fire(n, control.Clicked)
		col, desc := cellIndex(n), false
		if v, _ := n.Attr("aria-sort"); v == "descending" {
			desc = true
		}
		d.events = append(d.events, Sorted{Node: grid, Column: col, Descending: desc})
		return true
	}
	if !isGridRow(n) {
		return false
	}
	if selectRow(grid, n) {
		d.fire(grid, control.Changed)
	} else if n == d.lastPress && d.clock-d.lastPressAt <= doubleClick && !ariaDisabled(n) {
		d.fire(grid, control.Submitted)
		d.lastPress = nil
		return true
	}
	d.lastPress, d.lastPressAt = n, d.clock
	return true
}

// gridIntent moves grid's selection for i, or activates it; false for
// intents a grid doesn't take.
func (d *Document) gridIntent(grid *Node, i control.Intent) bool {
	var live []*Node
	at := -1
	for _, r := range gridRows(grid) {
		if !ariaDisabled(r) {
			if ariaSelected(r) {
				at = len(live)
			}
			live = append(live, r)
		}
	}
	if len(live) == 0 {
		return i == control.Up || i == control.Down || i == control.Home || i == control.End
	}
	var next int
	switch i {
	case control.Activate:
		if at >= 0 {
			d.fire(grid, control.Submitted)
		}
		return true
	case control.Down:
		next = min(at+1, len(live)-1)
	case control.Up:
		next = max(at-1, 0)
	case control.Home:
		next = 0
	case control.End:
		next = len(live) - 1
	case control.None, control.Back, control.Left, control.Right, control.PageUp, control.PageDown:
		return false
	}
	if selectRow(grid, live[next]) {
		d.fire(grid, control.Changed)
	}
	d.revealRow(grid, live[next])
	return true
}

// sortGrid sorts grid's rows by the column of the header th: ascending,
// or reversed when it already was.
func (d *Document) sortGrid(grid, th *Node) {
	v, _ := th.Attr("aria-sort")
	d.Sort(grid, cellIndex(th), v == "ascending")
}

// Sort sorts grid's rows by column as a press on its header does, the
// grid's SortBy order if it has one, and shows the header's arrow, the
// others none; it emits nothing.
func (d *Document) Sort(grid *Node, column int, descending bool) {
	grid.Walk(func(n *Node) bool {
		if isSortHeader(n) {
			switch {
			case cellIndex(n) != column:
				n.SetAttr("aria-sort", "none")
			case descending:
				n.SetAttr("aria-sort", "descending")
			default:
				n.SetAttr("aria-sort", "ascending")
			}
		}
		return !n.IsText()
	})
	sorted := slices.Clone(gridRows(grid))
	compare := d.sorts[grid]
	slices.SortStableFunc(sorted, func(a, b *Node) int {
		if compare != nil {
			return compare(a, b, column, descending)
		}
		c := compareCells(sortKey(a, column), sortKey(b, column))
		if descending {
			return -c
		}
		return c
	})
	for _, r := range sorted {
		r.Parent().Append(r)
	}
}

// cellIndex is the column the cell n starts at in its row, past the
// columns the cells before it span; cells spanning down from the rows
// above aren't counted, so a sortable grid with rowspans sorts by the
// cell's place in its own row.
func cellIndex(n *Node) int {
	i := 0
	for _, c := range n.Parent().Children() {
		if c == n {
			return i
		}
		if !c.IsText() {
			i += colspan(c)
		}
	}
	return -1
}

// colspan is how many columns the cell c spans, as layout reads it.
func colspan(c *Node) int {
	v, _ := c.Attr("colspan")
	n, err := strconv.Atoi(v)
	if err != nil {
		return 1
	}
	return min(max(n, 1), 1000)
}

// sortKey is what row sorts by in column col: the data-sort of the cell
// spanning it, else its text.
func sortKey(row *Node, col int) string {
	i := 0
	for _, c := range row.Children() {
		if c.IsText() {
			continue
		}
		if i += colspan(c); col < i {
			if v, ok := c.Attr("data-sort"); ok {
				return v
			}
			return strings.TrimSpace(c.Text())
		}
	}
	return ""
}

func compareCells(a, b string) int {
	x, errA := strconv.ParseFloat(a, 64)
	y, errB := strconv.ParseFloat(b, 64)
	if errA == nil && errB == nil {
		return cmp.Compare(x, y)
	}
	return cmp.Compare(strings.ToLower(a), strings.ToLower(b))
}

// selectedRow is the place of grid's selected row among its rows, -1 for
// none.
func selectedRow(grid *Node) int {
	for i, r := range gridRows(grid) {
		if ariaSelected(r) {
			return i
		}
	}
	return -1
}

// firstCellText is how automation names a row: its first cell's text.
func firstCellText(row *Node, text func(*Node) string) string {
	for _, c := range row.Children() {
		if !c.IsText() {
			return strings.TrimSpace(say(c, text))
		}
	}
	return ""
}

// revealRow scrolls the containers holding grid's row, as reveal does,
// the innermost one's view starting below the grid's sticky header, so
// the header never hides the row.
func (d *Document) revealRow(grid, row *Node) {
	if d.tree == nil {
		return
	}
	b := d.tree.Of[row]
	if b == nil {
		return
	}
	header := float32(0)
	grid.Walk(func(n *Node) bool {
		if n.Tag() == dom.Th && inHead(n) {
			if hb := d.tree.Of[n]; hb != nil && hb.Style != nil && hb.Style.Position == style.Sticky {
				header = max(header, hb.Rect().H)
			}
		}
		return !n.IsText()
	})
	for i, s := range scrollContainers(b.Parent) {
		bars := paint.BarsOf(s, origin(s))
		r, v := b.Rect(), bars.View
		if i == 0 {
			v.Y, v.H = v.Y+header, max(v.H-header, 0)
		}
		off := bars.Scroll
		off.X += nearest(r.X, r.W, v.X, v.W)
		off.Y += nearest(r.Y, r.H, v.Y, v.H)
		off.X = min(max(off.X, 0), max(bars.Content.X-bars.View.W, 0))
		off.Y = min(max(off.Y, 0), max(bars.Content.Y-bars.View.H, 0))
		d.setScroll(s, off)
	}
}
