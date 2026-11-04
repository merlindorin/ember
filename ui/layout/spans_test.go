package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

func spanning(n *dom.Node, name, v string) *dom.Node {
	n.SetAttr(name, v)
	return n
}

func TestCellsTakeTheSlotsThatCellsAboveLeave(t *testing.T) {
	a, b := spanning(cell("A"), "rowspan", "2"), cell("B")
	c, d := cell("C"), spanning(cell("D"), "rowspan", "9")
	e, f := spanning(cell("E"), "rowspan", "0"), cell("F")
	g := cell("G")
	head := el(dom.Tbody, nil, row(a, b), row(c, d))
	foot := el(dom.Tbody, nil, row(e, f), row(g))
	tb := table(head, foot)
	var r Retained
	Layout(el(dom.Div, block, tb), geom.V2(300, 600), ahemFonts{}, Retain(&r))
	tn := r.nodes[tb]
	cells, cols := (&tree{}).slots(tn, tn.rows())
	at := map[*dom.Node][4]int{}
	for _, s := range cells {
		at[s.cell.box.Node] = [4]int{s.row, s.col, s.rs, s.cs}
	}
	assert.Equal(t, 3, cols)
	assert.Equal(t, [4]int{0, 0, 2, 1}, at[a])
	assert.Equal(t, [4]int{1, 1, 1, 1}, at[c], "C goes past A, which spans down into its row")
	assert.Equal(t, [4]int{1, 2, 1, 1}, at[d], "a rowspan past its group ends with it")
	assert.Equal(t, [4]int{2, 0, 2, 1}, at[e], "rowspan 0 spans to its group's end")
	assert.Equal(t, [4]int{3, 1, 1, 1}, at[g])
}

func TestASpanningCellIsAsWideAsItsColumnsAndATallOneGrowsItsRows(t *testing.T) {
	wide := spanning(cell("HHHHHHHHHHHHHHHHHHHH"), "colspan", "2")
	a, b := cell("H"), cell("H")
	tall := spanning(cell("H", func(st *style.Computed) { st.Size.Height = style.Px(80) }), "rowspan", "2")
	c, d := cell("H"), cell("H")
	tb := table(el(dom.Tbody, nil, row(wide), row(a, b), row(tall, c), row(d)))
	tr := lay(el(dom.Div, block, tb), 400)
	assert.InDelta(t, tr.Of[a].Size.X+2+tr.Of[b].Size.X,
		tr.Of[wide].Size.X, 0.01, "its two columns and the spacing between")
	assert.Greater(t, tr.Of[a].Size.X, float32(12), "its columns grew to hold it")
	r2, r3 := tr.Of[c.Parent()], tr.Of[d.Parent()]
	assert.InDelta(t, r2.Size.Y+2+r3.Size.Y, tr.Of[tall].Size.Y, 0.01, "the tall cell spans both rows and the spacing")
	assert.InDelta(t, r2.Size.Y, r3.Size.Y, 0.01, "rows as tall as each other grow alike")
	assert.InDelta(t, 39, r2.Size.Y, 0.01, "both grew for it")
}

func TestAKeptSpannedTableMatchesAFreshOne(t *testing.T) {
	wide := spanning(cell("a heading"), "colspan", "2")
	a := cell("H")
	tb := table(el(dom.Tbody, nil, row(wide), row(a, spanning(cell("HH"), "rowspan", "2")), row(cell("HHH"))))
	root := el(dom.Div, block, tb)
	var r Retained
	layKept(&r, root, 300)
	a.Children()[0].SetText("a longer cell")
	kept := boxes(layKept(&r, root, 300))
	assert.Equal(t, boxes(lay(root, 300)), kept)
}

func TestACaptionSitsAboveOrBelowAtTheTablesWidth(t *testing.T) {
	caption := el(dom.Caption, func(st *style.Computed) { st.Display = style.Block }, text("HH"))
	body := el(dom.Tbody, nil, row(cell("HHHH"), cell("HHHH")))
	tb := table(caption, body)
	tr := lay(el(dom.Div, block, tb), 400)
	require.NotNil(t, tr.Of[caption])
	assert.Equal(t, tr.Of[tb].Size.X, tr.Of[caption].Size.X, "as wide as the table")
	assert.InDelta(t, at(tr.Of[caption]).Y+tr.Of[caption].Size.Y, at(tr.Of[body]).Y-2, 0.01, "the rows start under it")
	assert.Equal(t, tr.Of[caption].Size.Y, tr.Of[tb].Wrap.Top, "its room is outside what the table's border frames")
	assert.Zero(t, tr.Of[tb].Wrap.Bottom)

	dom.Data(tb).Style.CaptionSide = style.CaptionBottom
	dom.Data(caption).Style.CaptionSide = style.CaptionBottom
	tr = lay(el(dom.Div, block, tb), 400)
	assert.InDelta(t, at(tr.Of[body]).Y+tr.Of[body].Size.Y+2, at(tr.Of[caption]).Y, 0.01, "below, under the rows")
	assert.Equal(t, tr.Of[caption].Size.Y, tr.Of[tb].Wrap.Bottom)
	assert.Zero(t, tr.Of[tb].Wrap.Top)
}

func TestAFooterWrittenFirstGoesUnderTheBody(t *testing.T) {
	foot := el(dom.Tfoot, nil, row(cell("total")))
	body := el(dom.Tbody, nil, row(cell("Ada")))
	tr := lay(el(dom.Div, block, table(foot, body)), 400)
	assert.Greater(t, at(tr.Of[foot]).Y, at(tr.Of[body]).Y)
}
