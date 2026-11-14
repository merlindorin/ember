//go:build chrome

package ui_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/export"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestAThreeElementPageMatchesChrome(t *testing.T) {
	d := ui.NewDocument()
	d.Body().Append(
		ui.Div(ui.ID("card"), ui.Style("padding: 12px; border: 2px solid #335"),
			ui.P(ui.Text("Hello, ember")),
			ui.Button(ui.Style("padding: 4px; border: 1px solid #888"), ui.Text("Play")),
		),
	)
	r := chrome.Match(t, d, geom.V2(800, 600))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestChromeTakesAScreenshot(t *testing.T) {
	d := ui.NewDocument()
	d.Body().Append(ui.Div(ui.Style("height: 40px; background-color: #335")))
	var page bytes.Buffer
	require.NoError(t, export.Write(&page, d))
	png := filepath.Join(t.TempDir(), "page.png")
	require.NoError(t, chrome.Screenshot(page.Bytes(), geom.V2(320, 200), png))
	f, err := os.Open(png)
	require.NoError(t, err)
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	require.NoError(t, err)
	assert.Equal(t, 320, cfg.Width)
	assert.Equal(t, 200, cfg.Height)
}

func TestAScrollingBoxReservesChromesGutter(t *testing.T) {
	d := ui.NewDocument()
	d.Body().Append(ui.Div(ui.Style("width: 100px; height: 50px; overflow-y: scroll"),
		ui.Div(ui.Style("height: 10px"))))
	chrome.Match(t, d, geom.V2(640, 400))
}

func TestLinesMatchChromeOnARetinaDisplay(t *testing.T) {
	for _, size := range []string{"13px", "15px", "17px"} {
		d := ui.NewDocument()
		d.Body().Append(ui.P(ui.Style("width: 120px; font-size: "+size),
			ui.Text("The quick brown fox jumps over the lazy dog, then naps in the sun.")))
		r := chrome.MatchScaled(t, d, geom.V2(640, 400), 2)
		t.Logf("%s: %d boxes, max delta %.3f pt", size, r.Boxes, r.MaxDelta)
	}
}

func TestATablesBareRowsMatchChrome(t *testing.T) {
	d := ui.NewDocument()
	d.Body().Append(ui.Table(
		ui.Tr(ui.Th(ui.Text("Item")), ui.Th(ui.Text("Qty"))),
		ui.Tr(ui.Td(ui.Text("Torch")), ui.Td(ui.Text("3"))),
	))
	r := chrome.Match(t, d, geom.V2(640, 400))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestNarrowAndShortPagesMatchChrome(t *testing.T) {
	for _, size := range []geom.Vec2{geom.V2(420, 170), geom.V2(600, 170)} {
		d := ui.NewDocument()
		d.Body().Append(ui.Div(ui.Style("padding: 12px; display: flex; gap: 8px; background-color: #222"),
			ui.Div(ui.Style("width: 80px; height: 44px; background-color: #555")),
			ui.Div(ui.Style("flex: 1; height: 44px; background-color: #555"))))
		r := chrome.MatchScaled(t, d, size, 2)
		t.Logf("%v: %d boxes, max delta %.3f pt", size, r.Boxes, r.MaxDelta)
	}
}

func TestAModalDialogSitsWhereChromesTopLayerPutsIt(t *testing.T) {
	d := ui.NewDocument()
	modal := ui.Dialog(ui.Style("width: 240px; padding: 16px"), ui.P(ui.Text("Delete the tileset?")))
	d.Body().Append(ui.P(ui.Text("page")), modal)
	d.ShowModal(modal)
	r := chrome.Match(t, d, geom.V2(640, 400))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestAPathCutAtItsStartMatchesChrome(t *testing.T) {
	d := ui.NewDocument()
	d.Body().Append(ui.P(ui.Class("truncate-start"),
		ui.Style("width: 140px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis"),
		ui.Text("/Users/heron/assets/crypt/tiled/walls.tsj")))
	r := chrome.Match(t, d, geom.V2(640, 400))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestATextareaScrollsAsChromes(t *testing.T) {
	const flood = "The crypt's west wing floods every spring, " +
		"and the lower vaults stay wet until the summer heat dries the stones."
	empty := ui.Textarea(ui.Placeholder("Notes"), ui.Attr("rows", "3"), ui.Style("display: block; width: 180px"))
	full := ui.Textarea(ui.Attr("rows", "3"), ui.Style("display: block; width: 180px"), ui.Text(flood))
	scrolled := ui.Textarea(
		ui.Attr("rows", "3"),
		ui.Style("display: block; width: 180px; overflow-y: scroll"),
		ui.Text(flood),
	)
	d := ui.NewDocument()
	d.Body().Append(empty, full, scrolled)
	out, err := chrome.Eval(
		d,
		geom.V2(400, 400),
		`return [...document.querySelectorAll("textarea")]
			.map(e => [e.scrollWidth, e.scrollHeight, e.clientWidth, e.clientHeight]);`,
	)
	require.NoError(t, err)
	var want [][4]float32
	require.NoError(t, json.Unmarshal(out, &want))
	d.Layout(geom.V2(400, 400))
	for i, n := range []*ui.Node{empty, full, scrolled} {
		got := ui.ScrollSize(d, n)
		assert.InDelta(t, want[i][0], got.X, 1, "textarea %d: scrollWidth (Chrome %v)", i, want[i])
		assert.InDelta(t, want[i][1], got.Y, 1, "textarea %d: scrollHeight (Chrome %v)", i, want[i])
	}
}

func TestAPositionedChildOutsideItsParentsPaddingBoxIsHitAsInChrome(t *testing.T) {
	// Three frames, each with a × at right: -14px; top: -14px: over a
	// solid 14px border, over a 14px border with no style (so none, and
	// the × wholly outside the frame), and under a frame that clips.
	frames := []string{
		"border: 14px solid #444",
		"border-width: 14px",
		"border: 14px solid #444; overflow: hidden",
	}
	page := ui.Div(ui.ID("page"), ui.Style("position: absolute; inset: 0"))
	for i, css := range frames {
		page.Append(ui.Div(ui.Style(fmt.Sprintf("position: absolute; left: %dpx; top: 30px", 20+i*100)),
			ui.Div(ui.ID(fmt.Sprint("frame", i)), ui.Style("position: relative; width: 48px; height: 48px; "+css),
				ui.Button(ui.ID(fmt.Sprint("close", i)),
					ui.Style("position: absolute; right: -14px; top: -14px; width: 16px; height: 16px; padding: 0")))))
	}
	d := ui.NewDocument()
	d.Body().Append(page)
	// Chrome says where each × is drawn, then what it hits at its centre
	// and one pixel in from two corners.
	out, err := chrome.Eval(d, geom.V2(360, 120), `return [0, 1, 2].flatMap(i => {
		const r = document.getElementById("close" + i).getBoundingClientRect();
		return [[r.x + r.width / 2, r.y + r.height / 2], [r.x + 1, r.y + 1], [r.right - 1, r.bottom - 1]];
	}).map(([x, y]) => ({x, y, id: document.elementFromPoint(x, y).id}));`)
	require.NoError(t, err)
	var got []chromeHit
	require.NoError(t, json.Unmarshal(out, &got))
	require.Len(t, got, 9)
	assert.Equal(t, "close0", got[0].ID, "Chrome hits the × over its frame's border")
	assert.Equal(t, "close1", got[3].ID, "and the × outside a borderless frame")
	assert.NotEqual(t, "close2", got[6].ID, "but not the × a clipping frame cuts away")
	d.Layout(geom.V2(360, 120))
	for _, g := range got {
		assert.Equal(t, g.ID, ui.HitAt(d, geom.V2(g.X, g.Y)).ID(), "at %v, %v", g.X, g.Y)
	}
}

func TestANeighbourOverlappingAPositionedChildCoversItAsInChrome(t *testing.T) {
	// Two portraits overlapping by 8 points, as a strip's do, each with a
	// × at right: -14px; top: -14px: the second, later in tree order,
	// paints over the first's ×, so the second takes the pointer there,
	// unless the × stacks above it.
	strip := func(id, cross string) *ui.Node {
		row := ui.Div(ui.ID(id), ui.Style("display: flex"))
		for i := range 2 {
			margin := "0"
			if i > 0 {
				margin = "-8px"
			}
			portrait := "position: relative; width: 48px; height: 48px; border: 14px solid #444; flex: none; "
			row.Append(ui.Div(ui.ID(fmt.Sprint(id, "-p", i)), ui.Style(portrait+"margin-left: "+margin),
				ui.Button(ui.ID(fmt.Sprint(id, "-x", i)),
					ui.Style("position: absolute; right: -14px; top: -14px; width: 16px; height: 16px; padding: 0; "+cross))))
		}
		return row
	}
	page := ui.Div(ui.ID("page"),
		ui.Style("position: absolute; inset: 0; display: flex; flex-direction: column; gap: 20px; padding: 20px"),
		strip("plain", ""), strip("raised", "z-index: 1"))
	d := ui.NewDocument()
	d.Body().Append(page)
	out, err := chrome.Eval(d, geom.V2(240, 200), `return ["plain", "raised"].map(id => {
		const r = document.getElementById(id + "-x0").getBoundingClientRect();
		const x = r.x + r.width / 2, y = r.y + r.height / 2;
		return {x, y, id: document.elementFromPoint(x, y).id};
	});`)
	require.NoError(t, err)
	var got []chromeHit
	require.NoError(t, json.Unmarshal(out, &got))
	require.Len(t, got, 2)
	assert.Equal(t, "plain-p1", got[0].ID, "Chrome: the next portrait covers the first's ×")
	assert.Equal(t, "raised-x0", got[1].ID, "and a × stacked above it is hit")
	d.Layout(geom.V2(240, 200))
	for _, g := range got {
		assert.Equal(t, g.ID, ui.HitAt(d, geom.V2(g.X, g.Y)).ID(), "at %v, %v", g.X, g.Y)
	}
}

// chromeHit is a point and the id of the element Chrome finds there.
type chromeHit struct {
	X  float32 `json:"x"`
	Y  float32 `json:"y"`
	ID string  `json:"id"`
}

func TestABoxsRightAndBottomEdgesAreOutsideAsInChrome(t *testing.T) {
	cell := ui.Div(ui.ID("cell"), ui.Style("position: absolute; left: 100px; top: 10px; width: 40px; height: 40px"))
	clip := ui.Div(
		ui.ID("clip"),
		ui.Style("position: absolute; left: 200px; top: 10px; width: 40px; height: 40px; overflow: hidden"),
		ui.Div(ui.ID("inner"), ui.Style("width: 80px; height: 80px")),
	)
	d := ui.NewDocument()
	d.Body().Append(ui.Div(ui.ID("page"), ui.Style("position: absolute; inset: 0"), cell, clip))
	points := [][2]float32{{140, 30}, {120, 50}, {139, 49}, {100, 10}, {240, 30}, {220, 50}, {239, 49}}
	js, err := json.Marshal(points)
	require.NoError(t, err)
	out, err := chrome.Eval(
		d,
		geom.V2(400, 300),
		`return `+string(js)+`.map(([x, y]) => document.elementFromPoint(x, y).id);`,
	)
	require.NoError(t, err)
	var want []string
	require.NoError(t, json.Unmarshal(out, &want))
	require.Equal(
		t,
		[]string{"page", "page", "cell", "cell", "page", "page", "inner"},
		want,
		"Chrome: the right and bottom edges out, a pixel inside in, the top-left corner in, a clip's edges out",
	)
	d.Layout(geom.V2(400, 300))
	for i, p := range points {
		assert.Equal(t, want[i], ui.HitAt(d, geom.V2(p[0], p[1])).ID(), "at %v", p)
	}
}

func TestAPointerEventsNoneOverlayLetsClicksThroughAsInChrome(t *testing.T) {
	under := ui.Button(
		ui.ID("under"),
		ui.Style("position: absolute; left: 10px; top: 10px; width: 100px; height: 40px"),
		ui.Text("Under"),
	)
	glass := ui.Div(
		ui.ID("glass"),
		ui.Style("position: absolute; inset: 0; pointer-events: none"),
		ui.Div(
			ui.ID("back"),
			ui.Style("position: absolute; left: 10px; top: 100px; width: 100px; height: 40px; pointer-events: auto"),
		),
	)
	solid := ui.Div(ui.ID("solid"), ui.Style("position: absolute; left: 200px; top: 0; width: 100px; height: 100px"))
	d := ui.NewDocument()
	d.Body().Append(under, glass, solid)
	points := []geom.Vec2{{X: 60, Y: 30}, {X: 60, Y: 120}, {X: 250, Y: 50}}
	out, err := chrome.Eval(
		d,
		geom.V2(400, 300),
		`return [[60, 30], [60, 120], [250, 50]].map(([x, y]) => document.elementFromPoint(x, y).id);`,
	)
	require.NoError(t, err)
	var want []string
	require.NoError(t, json.Unmarshal(out, &want))
	require.Equal(
		t,
		[]string{"under", "back", "solid"},
		want,
		"Chrome: through the glass, its auto child, an opaque box",
	)
	d.Layout(geom.V2(400, 300))
	for i, p := range points {
		assert.Equal(t, want[i], ui.HitAt(d, p).ID(), "at %v", p)
	}
}

func TestARightClickReachesItsElementThenItsAncestorsAsInChrome(t *testing.T) {
	var got []string
	record := func(id string) ui.Arg { return ui.OnContextMenu(func() { got = append(got, id) }) }
	label := ui.Span(ui.ID("label"), record("label"), ui.Text("Bow"))
	cell := ui.Div(
		ui.ID("cell"),
		record("cell"),
		ui.Style("position: absolute; left: 10px; top: 10px; width: 120px; height: 40px; padding: 8px"),
		label,
	)
	other := ui.Div(
		ui.ID("other"),
		record("other"),
		ui.Style("position: absolute; left: 200px; top: 10px; width: 100px; height: 60px"),
	)
	veil := ui.Div(
		ui.ID("veil"),
		record("veil"),
		ui.Style("position: absolute; left: 190px; top: 0; width: 140px; height: 90px; pointer-events: none"),
	)
	grid := ui.Div(ui.ID("grid"), record("grid"), ui.Style("position: absolute; inset: 0"), cell, other, veil)
	d := ui.NewDocument()
	d.Body().Append(grid)
	points := [][2]float32{{22, 24}, {120, 45}, {250, 40}}
	out, err := chrome.Eval(d, geom.V2(640, 360), `
const points = [[22, 24], [120, 45], [250, 40]];
return points.map(([x, y]) => {
	const order = [];
	const els = [...document.querySelectorAll("[id]")];
	const note = e => order.push(e.currentTarget.id);
	els.forEach(el => el.addEventListener("contextmenu", note));
	document.elementFromPoint(x, y).dispatchEvent(new MouseEvent("contextmenu",
		{bubbles: true, cancelable: true, clientX: x, clientY: y, button: 2}));
	els.forEach(el => el.removeEventListener("contextmenu", note));
	return order;
});`)
	require.NoError(t, err)
	var want [][]string
	require.NoError(t, json.Unmarshal(out, &want))
	require.Equal(t, []string{"label", "cell", "grid"}, want[0], "Chrome: the label, then up")

	l := &live{doc: d, win: platform.NewWindow(platform.Size(640, 360))}
	l.win.Add(d)
	l.auto = automation.New(automation.Recording())
	l.w = ember.New(ember.Automation(l.auto)).AddEntities(l.win)
	l.frames(2)
	var h heard
	for i, p := range points {
		got = nil
		h.press(l, input.MouseRight, geom.V2(p[0], p[1]))
		h.release(l, input.MouseRight)
		assert.Equal(t, want[i], got, "at %v", p)
	}
}
