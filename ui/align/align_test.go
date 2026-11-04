package align_test

import (
	"strings"
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/align"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func laid(nodes ...*ui.Node) *ui.Document {
	d := ui.NewDocument()
	d.Body().Append(nodes...)
	d.Layout(geom.V2(600, 400))
	return d
}

func rules(fs []align.Fault) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Rule+" "+f.Path)
	}
	return out
}

func TestAnAlignedRowPasses(t *testing.T) {
	row := ui.Div(ui.Style("display: flex; align-items: center; gap: 8px"),
		ui.Button(ui.Style("height: 26px; padding: 0 10px"), ui.Text("Save")),
		ui.Button(ui.Style("height: 26px; padding: 0 10px"), ui.Text("Cancel")),
	)
	assert.Empty(t, align.Check(laid(row)))
}

func TestControlsOfARowShareTheirHeight(t *testing.T) {
	row := ui.Div(ui.Style("display: flex; align-items: center; gap: 8px"),
		ui.Button(ui.Style("height: 26px; padding: 0 10px"), ui.Text("Save")),
		ui.Input(ui.Type("text"), ui.Placeholder("Name"), ui.Style("height: 30px")),
		ui.Button(ui.Class("own-height"), ui.Style("height: 40px; padding: 0 10px"), ui.Text("Big")),
	)
	fs := align.Check(laid(row))
	assert.Equal(t, []string{"height name"}, rules(fs), "the input differs; own-height opts out")
	assert.Contains(t, fs[0].Detail, "30")
}

func TestAControlsLabelIsCentred(t *testing.T) {
	low := ui.Button(ui.Style("height: 26px; padding: 6px 10px 0"), ui.Text("Low"))
	aside := ui.Button(ui.Style("height: 26px; width: 120px; padding: 0 0 0 30px"), ui.Text("Aside"))
	left := ui.Button(ui.Style("height: 26px; width: 120px; padding: 0 10px; text-align: left"), ui.Text("Left"))
	menu := ui.Button(ui.Style("height: 26px; padding: 0 10px; position: relative"), ui.Text("Menu"),
		ui.Div(ui.Style("position: absolute; top: 100%; left: 0"), ui.Attr(align.OverflowOK, ""), ui.Text("an open list")))
	fs := align.Check(laid(low, aside, left, menu))
	assert.ElementsMatch(t, []string{"centre low", "centre aside"}, rules(fs),
		"a left-aligned label, or text hanging out of flow, is no fault")
	assert.True(t, strings.Contains(fs[0].Detail, "pt"))
}

func TestACardOfStackedBlocksHasNoLabelToCentre(t *testing.T) {
	part := func(text string) *ui.Node { return ui.Div(ui.Text(text)) }
	column := ui.Button(ui.Style("display: flex; flex-direction: column; width: 120px; height: 120px"),
		part("Volley"), part("Lv.1"), part("A ring of arrows"))
	flow := ui.Button(ui.Style("display: block; width: 120px; height: 120px"), part("Longbow"), part("A heavy arrow"))
	pushed := ui.Button(ui.Style("display: flex; flex-direction: column; height: 50px; padding-top: 24px"), part("Pushed"))
	row := ui.Button(ui.Style("display: flex; height: 50px; padding-top: 24px"),
		ui.Span(ui.Text("★")), ui.Span(ui.Text("Starred")))
	assert.ElementsMatch(t, []string{"centre pushed", "centre starred"},
		rules(align.Check(laid(column, flow, pushed, row))),
		"stacked blocks are a card; one label, or a row of parts, is still centred")
}

func TestBaselinesOfARowLineUp(t *testing.T) {
	row := ui.Div(ui.Style("display: flex; align-items: flex-start; gap: 8px"),
		ui.Span(ui.ID("small"), ui.Style("font-size: 12px"), ui.Text("small")),
		ui.Span(ui.ID("big"), ui.Style("font-size: 24px"), ui.Text("big")),
	)
	assert.Equal(t, []string{"baseline small"}, rules(align.Check(laid(row))), "reported at the row's first child")
}

func TestAWideButtonsLabelIsMeasuredNotItsLineBox(t *testing.T) {
	wide := ui.Button(ui.Style("height: 26px; width: 160px; padding: 0 0 0 40px; display: block"), ui.Text("Wide"))
	assert.Equal(t, []string{"centre wide"}, rules(align.Check(laid(wide))),
		"the block wrapping the label spans the box; the label doesn't")
}

func TestALabelCutByItsBoxIsMeasuredAsItShows(t *testing.T) {
	cut := ui.Button(ui.Style("height: 26px; width: 120px; padding: 0 10px"),
		ui.Span(ui.Style("display: block; overflow: hidden; white-space: nowrap; text-overflow: ellipsis"),
			ui.Text("a name far too long to show whole")))
	assert.Empty(t, align.Check(laid(cut)), "the text past the clip doesn't count")
}

func TestControlsInANestedRowCountForTheRowAround(t *testing.T) {
	bar := ui.Div(ui.Style("display: flex; align-items: center; gap: 8px"),
		ui.Div(ui.Style("display: flex; gap: 2px"),
			ui.Button(ui.Style("height: 24px; padding: 0 8px"), ui.Text("Back")),
			ui.Button(ui.Style("height: 24px; padding: 0 8px"), ui.Text("Forward")),
		),
		ui.Button(ui.Style("height: 30px; padding: 0 10px"), ui.Text("Folder")),
		ui.Input(ui.Type("search"), ui.Placeholder("Search"), ui.Style("height: 24px")),
	)
	assert.Equal(t, []string{"height folder"}, rules(align.Check(laid(bar))),
		"back and forward sit in a row of their own, inside the bar")
}

func TestAStartAlignedButtonIsCheckedAcrossButALeftOneIsNot(t *testing.T) {
	start := ui.Button(ui.Style("height: 26px; width: 120px; padding: 0 10px; text-align: start"), ui.Text("Start"))
	lights := ui.Div(ui.Style("display: flex; align-items: center; gap: 8px"),
		ui.Span(ui.Class("own-height"), ui.Style("display: flex; gap: 4px"),
			ui.Button(ui.Style("width: 12px; height: 12px; padding: 0")),
			ui.Button(ui.Style("width: 12px; height: 12px; padding: 0"))),
		ui.Button(ui.Style("height: 26px; padding: 0 10px"), ui.Text("One")),
		ui.Button(ui.Style("height: 26px; padding: 0 10px"), ui.Text("Two")),
	)
	assert.Equal(t, []string{"centre start"}, rules(align.Check(laid(start, lights))),
		"start is the default; an own-height group keeps its lights out of the row")
}

func TestALabelAndItsFieldOnAGridRowShareABaseline(t *testing.T) {
	form := func(align string) *ui.Node {
		return ui.Div(ui.Style("display: grid; grid-template-columns: max-content 200px; gap: 8px; align-items: "+align),
			ui.Span(ui.Style("font-size: 12px"), ui.Text("Name")),
			ui.Input(ui.Type("text"), ui.ID("name"), ui.Value("Ayla"), ui.Style("height: 30px")),
			ui.Span(ui.Style("font-size: 12px"), ui.Text("Seed")),
			ui.Input(ui.Type("text"), ui.ID("seed"), ui.Value("7"), ui.Style("height: 30px")),
		)
	}
	assert.Empty(t, align.Check(laid(form("baseline"))))
	fs := align.Check(laid(form("start")))
	assert.Len(t, fs, 2, "one fault per row")
	assert.Equal(t, "baseline", fs[0].Rule)
	assert.Greater(t, fs[0].Got-fs[0].Want, float32(1))
}

func TestWhatTheWindowCutsIsOutside(t *testing.T) {
	cut := ui.Button(ui.ID("cut"), ui.Style("position: absolute; left: -20px; top: 10px"), ui.Text("Cut"))
	below := ui.Button(ui.ID("below"), ui.Style("position: absolute; left: 10px; top: 390px"), ui.Text("Below"))
	scrolled := ui.Div(ui.Style("position: absolute; top: 300px; height: 60px; overflow: auto"),
		ui.Button(ui.ID("scrolled"), ui.Style("margin-top: 120px"), ui.Text("Down there")))
	fs := align.Check(laid(cut, below, scrolled))
	assert.ElementsMatch(t, []string{"outside cut", "outside below"}, rules(fs),
		"a scroller's content isn't cut by the window")
	assert.InDelta(t, 20, fs[0].Got, 0.01)
}

func TestACellSpanningRowsJoinsOnlyTheFirstAndIconsHaveNoBaseline(t *testing.T) {
	card := ui.Div(ui.Style("display: grid; grid-template-columns: 48px 1fr; align-items: center"),
		ui.Div(ui.Style("grid-row: span 2; height: 48px")),
		ui.Strong(ui.Style("font-size: 18px"), ui.Text("Ember Survivors")),
		ui.Span(ui.Style("font-size: 13px"), ui.Text("Area 2")),
	)
	row := ui.Div(ui.Style("display: flex; align-items: center"),
		ui.Span(ui.Class("kit-icon"), ui.Style("font-size: 30px"), ui.Text("◆")), ui.Span(ui.Text("Save")))
	assert.Empty(t, align.Check(laid(card, row)))
}

func TestALabelBesideABlockOfLinesMeetsItsFirstLine(t *testing.T) {
	page := func(align string) *ui.Node {
		return ui.Div(ui.Style("display: grid; grid-template-columns: max-content 200px; align-items: "+align),
			ui.Span(ui.Text("2 columns")),
			ui.Div(ui.Style("padding: 6px; border: 1px dashed gray"), ui.P(ui.Text("name")), ui.P(ui.Text("solid"))),
		)
	}
	assert.Empty(t, align.Check(laid(page("baseline"))))
	assert.Equal(t, []string{"baseline span"}, rules(align.Check(laid(page("start")))))
}

func TestTheSeparatorsOfARowStandInTheSameSpace(t *testing.T) {
	bar := func(second string) *ui.Node {
		return ui.Div(ui.Style("display: flex; align-items: center"),
			ui.Span(ui.Style("padding: 0 6px"), ui.Text("Copy")),
			ui.Hr(ui.Style("width: 1px; height: 12px; margin: 0 4px; border: 0")),
			ui.Span(ui.Style("padding: 0 6px"), ui.Text("Paste")),
			ui.Hr(ui.ID("second"), ui.Style("width: 1px; height: 12px; border: 0; margin: "+second)),
			ui.Span(ui.Style("padding: 0 6px"), ui.Text("Undo")),
		)
	}
	assert.Empty(t, align.Check(laid(bar("0 4px"))))
	fs := align.Check(laid(bar("0")))
	assert.Equal(t, []string{"separator second"}, rules(fs))
	assert.InDelta(t, 10, fs[0].Want, 0.01)
	assert.InDelta(t, 6, fs[0].Got, 0.01)
}

func TestInkPastItsBoxIsAnOverflow(t *testing.T) {
	bar := ui.Div(ui.ID("bar"), ui.Style("display: flex; width: 60px; white-space: nowrap"),
		ui.Span(ui.Text("Cut")), ui.Span(ui.Text("Copy")), ui.Span(ui.Text("Paste")))
	clipped := ui.Div(ui.Style("width: 60px; white-space: nowrap; overflow: hidden"),
		ui.Text("a long line that is clipped"))
	cut := ui.Div(ui.Style("width: 60px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis"),
		ui.Text("a long line cut short"))
	tip := ui.Div(ui.Style("position: relative; width: 60px"), ui.Text("Save"),
		ui.Div(ui.Style("position: absolute; left: 100%; top: 0; white-space: nowrap"),
			ui.Attr(align.OverflowOK, ""), ui.Text("a tip hanging out")))
	fs := align.Check(laid(bar, clipped, cut, tip))
	assert.Equal(t, []string{"overflow bar"}, rules(fs))
	assert.Greater(t, fs[0].Got, float32(1))
}

func TestInkOverflowingInsideAScrollerIsTheScrollers(t *testing.T) {
	scroller := ui.Div(ui.Style("width: 80px; overflow: auto"),
		ui.Div(ui.Style("width: 60px; white-space: nowrap"), ui.Text("a line wider than its column")))
	assert.Empty(t, align.Check(laid(scroller)))
}

func TestAHiddenRowBesideAShownOneHasNothingToLineUp(t *testing.T) {
	row := func(id string, style string) *ui.Node {
		return ui.Div(ui.ID(id), ui.Style("display: flex; gap: 8px; "+style),
			ui.Button(ui.Style("height: 26px; padding: 0 10px"), ui.Text("New game")),
			ui.Button(ui.Style("height: 26px; padding: 0 10px"), ui.Text("Quit")))
	}
	shown, hidden := row("shown", ""), row("hidden", "display: none")
	holder := ui.Div(row("inner", ""))
	d := laid(shown, hidden, holder)
	holder.AddClass("gone")
	holder.SetStyle("display: none")
	d.Layout(geom.V2(600, 400))
	assert.Empty(t, align.Check(d), "hidden from the start, and hidden after showing")
}

// card is a 200×60 card holding a row of three 30-point slots under a
// 40-point title, so the row runs 10 points past its fixed height.
func card(cardStyle string, rowArgs ...ui.Arg) *ui.Node {
	row := ui.Div(append([]ui.Arg{ui.ID("slots"), ui.Style("display: flex; gap: 4px")}, rowArgs...)...)
	for range 3 {
		row.Append(ui.Div(ui.Style("width: 30px; height: 30px; background-color: #333")))
	}
	return ui.Div(ui.ID("gear"), ui.Style("width: 200px; "+cardStyle), ui.Div(ui.Style("height: 40px")), row)
}

func TestARowSpillingPastAFixedHeightCardIsAFault(t *testing.T) {
	fs := align.Check(laid(card("height: 60px")))
	require.Equal(t, []string{"spill gear/slots"}, rules(fs), "the row, once, not each slot in it")
	assert.InDelta(t, 10, fs[0].Got, 0.5)
	assert.Contains(t, fs[0].Detail, "bottom of gear")
}

func TestAMarkedSpillIsMeant(t *testing.T) {
	assert.Empty(t, rules(align.Check(laid(card("height: 60px", ui.Attr(align.OverflowOK, ""))))), "on the box")
	marked := card("height: 60px")
	marked.Children()[1].SetAttr(align.OverflowOK, "")
	assert.Empty(t, rules(align.Check(laid(marked))), "or on a box between")
}

func TestAScrollingCardHoldsWhatRunsPastIt(t *testing.T) {
	assert.Empty(t, rules(align.Check(laid(card("height: 60px; overflow: auto")))))
}

func TestACardOfAutoHeightGrowsToFit(t *testing.T) {
	assert.Empty(t, rules(align.Check(laid(card("")))))
}
