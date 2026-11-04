package kit_test

import (
	"slices"
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestATableSelectsSortsAndActivatesRows(t *testing.T) {
	name := kit.TableColumn("Name", kit.Sortable())
	size := kit.TableColumn("Size", kit.ColumnWidth(70), kit.ColumnAlign("right"), kit.Sortable())
	crypt := kit.TableRow("crypt", "—")
	kit.SetDimmed(crypt, true)
	walls, props := kit.TableRow("walls.tsj", "24 KB"), kit.TableRow("props.tsj", "9 KB")
	kit.SortValue(walls, 1, "24")
	kit.SortValue(props, 1, "9")
	table := kit.Table([]*ui.Node{name, size}, []*ui.Node{crypt, walls, props})
	table.SetStyle("width: 300px")
	s := show(t, table)
	grid := kit.TableGrid(table)
	changes, submits := watch[ui.Changed](s), watch[ui.Submitted](s)

	assert.Equal(t, "kit-right", kit.CellOf(walls, 1).Classes()[0], "a column's alignment reaches its cells")
	assert.GreaterOrEqual(t, kit.CellOf(walls, 1).Box().W, float32(70))

	s.click(walls)
	require.Len(t, *changes, 1)
	assert.Same(t, grid, (*changes)[0].Node)
	assert.Equal(t, 1, s.doc.Selected(grid))
	s.key(input.KeyDown)
	assert.Equal(t, 2, s.doc.Selected(grid))
	s.key(input.KeyEnter)
	require.Len(t, *submits, 1, "Enter activates the selected row")

	s.click(size)
	assert.True(t, slices.Equal([]*ui.Node{props, walls, crypt}, kit.TableRows(table)),
		"by the sort values, 9 before 24, and a dash as text after them")
	assert.Equal(t, automation.RoleButton, s.element(t, "Size").Role)
	assert.Equal(t, automation.RoleSelect, s.element(t, "walls.tsj").Role)
}

func TestATableWithoutHeader(t *testing.T) {
	table := kit.Table([]*ui.Node{kit.TableColumn("Name")}, []*ui.Node{kit.TableRow("one")}, kit.NoHeader())
	show(t, table)
	assert.Zero(t, kit.TableGrid(table).Children()[0].Box().H, "its titles are hidden")
}

func TestSortedByOrdersATableAndShowsItsArrow(t *testing.T) {
	name, size := kit.TableColumn("Name", kit.Sortable()), kit.TableColumn("Size", kit.Sortable())
	a, b := kit.TableRow("a.tsj", "9"), kit.TableRow("b.tsj", "24")
	table := kit.Table([]*ui.Node{name, size}, []*ui.Node{a, b})
	s := show(t, table)
	kit.SortedBy(s.doc, table, 1, true)
	assert.True(t, slices.Equal([]*ui.Node{b, a}, kit.TableRows(table)))
	v, _ := size.Attr("aria-sort")
	assert.Equal(t, "descending", v)
	v, _ = name.Attr("aria-sort")
	assert.Equal(t, "none", v)
}

func TestATablesTitlesAreSeparatedAndKeepTheirColumns(t *testing.T) {
	name, size := kit.TableColumn("Name"), kit.TableColumn("Size", kit.ColumnWidth(70))
	row := kit.TableRow("walls.tsj", "24 KB")
	table := kit.Table([]*ui.Node{name, size}, []*ui.Node{row})
	table.SetStyle("width: 300px")
	show(t, table)
	assert.Equal(t, size.Box().X, kit.CellOf(row, 1).Box().X, "a title stands over its column")
	assert.Equal(t, size.Box().W, kit.CellOf(row, 1).Box().W)
	assert.Equal(t, float32(70), size.Box().W, "a sized column keeps its width, separator included")
}

func TestCellOfAndAlignmentFollowASpanningCell(t *testing.T) {
	total := ui.Tr(ui.Td(ui.Attr("colspan", "2"), ui.Text("total")), ui.Td(ui.Text("153 KB")))
	plain := kit.TableRow("walls.tsj", "Tileset", "24 KB")
	kit.Table([]*ui.Node{kit.TableColumn("Name"), kit.TableColumn("Kind"),
		kit.TableColumn("Size", kit.ColumnAlign("right"))}, []*ui.Node{plain, total})
	assert.Same(t, total.Children()[0], kit.CellOf(total, 1), "column 1 is the spanning cell's")
	assert.Same(t, total.Children()[1], kit.CellOf(total, 2), "and Size the cell after it")
	assert.Nil(t, kit.CellOf(total, 3), "none past the row")
	assert.True(t, total.Children()[1].HasClass("kit-right"), "Size's alignment goes on its cell")
	assert.False(t, total.Children()[0].HasClass("kit-right"))
	assert.True(t, kit.CellOf(plain, 2).HasClass("kit-right"), "as in a plain row")
}

func TestAColumnUnderASpanningTitleKeepsItsAlignment(t *testing.T) {
	row := kit.TableRow("walls.tsj", "Tileset", "24 KB")
	kit.Table([]*ui.Node{kit.TableColumn("File", ui.Attr("colspan", "2")),
		kit.TableColumn("Size", kit.ColumnAlign("right"))}, []*ui.Node{row})
	assert.True(t, kit.CellOf(row, 2).HasClass("kit-right"), "Size is the third column, under the second title")
	assert.False(t, kit.CellOf(row, 1).HasClass("kit-right"))
}

func TestACellUnderARowspanTakesItsColumnsAlignment(t *testing.T) {
	first := ui.Tr(ui.Td(ui.Attr("rowspan", "2"), ui.Text("walls.tsj")), ui.Td(ui.Text("collision")),
		ui.Td(ui.Text("24 KB")))
	second := ui.Tr(ui.Td(ui.Text("light")), ui.Td(ui.Text("6 KB")))
	kit.Table([]*ui.Node{kit.TableColumn("Tileset", ui.Attr("colspan", "2")),
		kit.TableColumn("Size", kit.ColumnAlign("right"))}, []*ui.Node{first, second})
	assert.True(t, second.Children()[1].HasClass("kit-right"), "6 KB stands in Size's column, past the cell spanning down")
	assert.False(t, second.Children()[0].HasClass("kit-right"))
}

func TestATablesCaptionAndFooterGoInItsGridAndTheRestOnItsDiv(t *testing.T) {
	caption := ui.Caption(ui.Text("Tilesets"))
	total := ui.Tr(ui.Td(ui.Attr("colspan", "2"), ui.Text("Total")), ui.Td(ui.Text("51 KB")))
	rows := []*ui.Node{kit.TableRow("walls.tsj", "Tileset", "24 KB"), kit.TableRow("floor.tsj", "Tileset", "9 KB")}
	table := kit.Table([]*ui.Node{kit.TableColumn("Name"), kit.TableColumn("Kind"),
		kit.TableColumn("Size", kit.ColumnAlign("right"))}, rows, caption, ui.Tfoot(total), kit.NoHeader())
	grid := kit.TableGrid(table)
	require.NotNil(t, grid)
	assert.Same(t, caption, grid.Children()[0], "the caption heads the grid")
	assert.Equal(t, rows, kit.TableRows(table), "the rows are still the body's")
	assert.Same(t, total, grid.Children()[len(grid.Children())-1].Children()[0], "the footer comes under the body")
	assert.True(t, total.Children()[1].HasClass("kit-right"),
		"Size's alignment reaches the footer, past its spanning cell")
	assert.True(t, table.HasClass("kit-no-header"), "other args go on the div")
}

func TestBelowPutsATablesCaptionUnderItsRows(t *testing.T) {
	caption := ui.Caption(ui.Text("Under"), kit.Below())
	rows := []*ui.Node{kit.TableRow("walls.tsj")}
	d := ui.NewDocument(kit.Theme())
	d.Body().Append(kit.Table([]*ui.Node{kit.TableColumn("Name")}, rows, caption))
	d.Layout(geom.V2(400, 300))
	assert.Greater(t, caption.Box().Y, rows[0].Box().Y)
}
