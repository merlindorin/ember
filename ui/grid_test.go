package ui_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// files is a grid of Name and Size, sortable, its rows crypt (dimmed),
// walls.tsj, Props.tsj and floor.png.
func files() (*ui.Node, *ui.Node, *ui.Node, []*ui.Node) {
	var rows []*ui.Node
	name := ui.Th(ui.Attr("aria-sort", "none"), ui.Text("Name"))
	size := ui.Th(ui.Attr("aria-sort", "none"), ui.Text("Size"))
	body := ui.Tbody()
	for _, r := range [][2]string{{"crypt", ""}, {"walls.tsj", "24"}, {"Props.tsj", "9"}, {"floor.png", "120"}} {
		row := ui.Tr(ui.Td(ui.Text(r[0])), ui.Td(ui.Text(r[1])))
		if r[0] == "crypt" {
			row.SetAttr("aria-disabled", "true")
		}
		body.Append(row)
		rows = append(rows, row)
	}
	grid := ui.Table(ui.Attr("role", "grid"), ui.Thead(ui.Tr(name, size)), body)
	return grid, name, size, rows
}

func firstCells(grid *ui.Node) []string {
	var out []string
	for _, r := range grid.Children()[1].Children() {
		out = append(out, r.Children()[0].Text())
	}
	return out
}

func TestAGridSelectsARowAndActivatesIt(t *testing.T) {
	grid, _, _, rows := files()
	l := liveDoc(t, grid)
	assert.Equal(t, -1, l.doc.Selected(grid))

	l.clickAt(rows[1])
	assert.Equal(t, "true", attr(rows[1], "aria-selected"))
	assert.Equal(t, 1, l.doc.Selected(grid))
	assert.True(t, slices.Equal([]*ui.Node{grid}, l.changes))
	assert.True(t, l.doc.Focused(grid), "the grid takes the focus")

	l.clickAt(rows[0])
	assert.Equal(t, 1, l.doc.Selected(grid), "a dimmed row isn't selected")

	l.key(input.KeyDown)
	assert.Equal(t, 2, l.doc.Selected(grid))
	l.key(input.KeyUp)
	l.key(input.KeyUp)
	assert.Equal(t, 1, l.doc.Selected(grid), "Up stops at the first row that can be selected")
	l.key(input.KeyEnd)
	assert.Equal(t, 3, l.doc.Selected(grid))
	assert.Equal(t, "false", attr(rows[1], "aria-selected"))

	l.key(input.KeyEnter)
	require.Len(t, l.submits, 1, "Enter activates the selection")
	assert.Same(t, grid, l.submits[0].Node)

	l.clickAt(rows[2])
	l.clickAt(rows[2])
	assert.Len(t, l.submits, 2, "a double click activates the row")

	assert.Contains(t, l.roles(automation.RoleSelect), "walls.tsj")
}

func TestAGridSortsByAHeaderPress(t *testing.T) {
	grid, name, size, _ := files()
	l := liveDoc(t, grid)
	l.clickAt(name)
	assert.Equal(t, []string{"crypt", "floor.png", "Props.tsj", "walls.tsj"}, firstCells(grid), "by text, ignoring case")
	assert.Equal(t, "ascending", attr(name, "aria-sort"))
	assert.Contains(t, l.clicks, name)

	l.clickAt(name)
	assert.Equal(t, []string{"walls.tsj", "Props.tsj", "floor.png", "crypt"}, firstCells(grid), "again reverses")
	assert.Equal(t, "descending", attr(name, "aria-sort"))

	l.clickAt(size)
	assert.Equal(t, []string{"crypt", "Props.tsj", "walls.tsj", "floor.png"}, firstCells(grid), "numbers as numbers")
	assert.Equal(t, "none", attr(name, "aria-sort"))
	assert.Contains(t, l.roles(automation.RoleButton), "Size")
}

func TestASortEmitsSortedAndAGridMaySortItsOwnWay(t *testing.T) {
	grid, name, size, rows := files()
	l := liveDoc(t, grid)
	l.clickAt(size)
	require.Len(t, l.sorted, 1)
	assert.Same(t, grid, l.sorted[0].Node)
	assert.Equal(t, 1, l.sorted[0].Column)
	assert.False(t, l.sorted[0].Descending)
	assert.Contains(t, l.clicks, size, "the header is still clicked")

	crypt := rows[0]
	foldersFirst := func(a, b *ui.Node, column int, descending bool) int {
		af, bf := a == crypt, b == crypt
		switch {
		case af && !bf:
			return -1
		case bf && !af:
			return 1
		}
		c := strings.Compare(strings.ToLower(a.Children()[column].Text()), strings.ToLower(b.Children()[column].Text()))
		if descending {
			return -c
		}
		return c
	}
	l.doc.SortBy(grid, foldersFirst)
	l.clickAt(name)
	assert.Equal(t, []string{"crypt", "floor.png", "Props.tsj", "walls.tsj"}, firstCells(grid))
	l.clickAt(name)
	assert.Equal(t, []string{"crypt", "walls.tsj", "Props.tsj", "floor.png"}, firstCells(grid),
		"folders first both ways: the comparator is told the direction")
	last := l.sorted[len(l.sorted)-1]
	assert.Same(t, grid, last.Node)
	assert.Equal(t, 0, last.Column)
	assert.True(t, last.Descending)
}

func TestASortIsStable(t *testing.T) {
	body := ui.Tbody()
	for _, n := range []string{"b1", "a1", "b2", "a2", "b3"} {
		body.Append(ui.Tr(ui.Td(ui.Attr("data-sort", n[:1]), ui.Text(n))))
	}
	head := ui.Th(ui.Attr("aria-sort", "none"), ui.Text("Name"))
	grid := ui.Table(ui.Attr("role", "grid"), ui.Thead(ui.Tr(head)), body)
	l := liveDoc(t, grid)
	l.clickAt(head)
	assert.Equal(t, []string{"a1", "a2", "b1", "b2", "b3"}, firstCells(grid), "equal keys keep their order")
	l.clickAt(head)
	assert.Equal(t, []string{"b1", "b2", "b3", "a1", "a2"}, firstCells(grid), "reversed by key, still stable")
}

func TestAGridLeavesCommandArrowsToTheGameAndCanBeCleared(t *testing.T) {
	grid, _, _, rows := files()
	l := liveDoc(t, grid)
	l.clickAt(rows[2])
	require.Equal(t, 2, l.doc.Selected(grid))

	in := l.win.Input()
	in.SetKey(input.KeyLeftSuper, true)
	in.SetKey(input.KeyUp, true)
	l.frames(1)
	assert.True(t, in.Down(input.KeyUp), "the game reads Command-Up")
	in.SetKey(input.KeyUp, false)
	in.SetKey(input.KeyLeftSuper, false)
	l.frames(1)
	assert.Equal(t, 2, l.doc.Selected(grid), "the selection stays")

	l.doc.SetSelected(grid, -1)
	assert.Equal(t, -1, l.doc.Selected(grid), "-1 clears it")
	assert.Equal(t, "false", attr(rows[2], "aria-selected"))
}

func TestSortSortsAGridAsItsHeaderWould(t *testing.T) {
	grid, name, size, _ := files()
	l := liveDoc(t, grid)
	l.doc.Sort(grid, 0, true)
	assert.Equal(t, []string{"walls.tsj", "Props.tsj", "floor.png", "crypt"}, firstCells(grid))
	assert.Equal(t, "descending", attr(name, "aria-sort"))
	assert.Equal(t, "none", attr(size, "aria-sort"))
	l.frames(2)
	assert.Empty(t, l.sorted, "a game's sort emits nothing")
	l.clickAt(name)
	assert.Equal(t, "ascending", attr(name, "aria-sort"), "a press goes on from there")
}

// longGrid is a 60-row grid under a sticky header, in a 120-point
// scroller.
func longGrid() (*ui.Node, *ui.Node, *ui.Node, []*ui.Node) {
	var rows []*ui.Node
	head := ui.Th(ui.Style("position: sticky; top: 0; height: 30px; background-color: #222"), ui.Text("Name"))
	body := ui.Tbody()
	for range 60 {
		r := ui.Tr(ui.Td(ui.Style("height: 20px"), ui.Text("row")))
		body.Append(r)
		rows = append(rows, r)
	}
	grid := ui.Table(ui.Attr("role", "grid"), ui.Style("border-spacing: 0; width: 200px"), ui.Thead(ui.Tr(head)), body)
	sc := ui.Div(ui.Style("height: 120px; overflow: auto"), grid)
	return sc, grid, head, rows
}

// shows reports whether row is wholly inside sc's view, below head.
func shows(sc, head, row *ui.Node) bool {
	v, h, r := sc.Box(), head.Box(), row.Box()
	return r.Y >= h.Y+h.H-0.01 && r.Y+r.H <= v.Y+v.H+0.01
}

func TestTheSelectionScrollsIntoViewBelowAStickyHeader(t *testing.T) {
	sc, grid, head, rows := longGrid()
	l := liveDoc(t, sc)
	l.clickAt(rows[0])
	l.key(input.KeyEnd)
	require.Equal(t, 59, l.doc.Selected(grid))
	assert.True(t, shows(sc, head, rows[59]), "End brings the last row into view")
	assert.InDelta(t, sc.Box().Y, head.Box().Y, 0.01, "under the header stuck at the top")
	l.key(input.KeyHome)
	assert.True(t, shows(sc, head, rows[0]), "Home brings the first back, not under the header")

	for range 4 {
		l.key(input.KeyDown)
	}
	assert.True(t, shows(sc, head, rows[4]), "down a row at a time, scrolled no more than needed")

	l.doc.SetSelected(grid, 40)
	l.frames(2)
	assert.True(t, shows(sc, head, rows[40]), "a game's selection shows too")
	l.doc.SetSelected(grid, 2)
	l.frames(2)
	assert.True(t, shows(sc, head, rows[2]), "even upwards, below the header")
}

func TestAGridSortsByTheColumnPastASpanningHeaderAndLeavesItsFootOut(t *testing.T) {
	size := ui.Th(ui.Attr("aria-sort", "none"), ui.Text("Size"))
	body := ui.Tbody()
	var rows []*ui.Node
	for _, r := range [][3]string{{"walls", "tsj", "24"}, {"floor", "png", "120"}, {"props", "tsj", "9"}} {
		row := ui.Tr(ui.Td(ui.Text(r[0])), ui.Td(ui.Text(r[1])), ui.Td(ui.Text(r[2])))
		body.Append(row)
		rows = append(rows, row)
	}
	total := ui.Tr(ui.Td(ui.Attr("colspan", "2"), ui.Text("total")), ui.Td(ui.Text("153")))
	grid := ui.Table(ui.Attr("role", "grid"),
		ui.Thead(ui.Tr(ui.Th(ui.Attr("colspan", "2"), ui.Text("File")), size)), body, ui.Tfoot(total))
	l := liveDoc(t, grid)
	l.clickAt(size)
	require.Len(t, l.sorted, 1)
	assert.Equal(t, 2, l.sorted[0].Column, "Size is the third column, past File's two")
	assert.Equal(t, []string{"props", "walls", "floor"}, firstCells(grid), "sorted by its numbers")
	l.clickAt(rows[0])
	l.key(input.KeyEnd)
	assert.Equal(t, "true", attr(rows[1], "aria-selected"), "End stops at the last body row, not the foot's")
	assert.NotEqual(t, "true", attr(total, "aria-selected"))
}
