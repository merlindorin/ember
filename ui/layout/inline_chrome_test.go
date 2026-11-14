//go:build chrome

package layout_test

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"

	"golang.org/x/text/cases"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/fonts"
	"github.com/lantervale/ember/ui/layout"
)

// textLines is, for every text node under body with more than spaces,
// in document order, its
// lines in Chrome: the text of each, spaces at either end trimmed, and the
// rectangle of its other characters.
const textLines = `
const out = [];
const walk = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
const range = document.createRange();
for (let n = walk.nextNode(); n; n = walk.nextNode()) {
	if (n.data.trim() === "") continue;
	const lines = [];
	let cur = null;
	const chars = Array.from(n.data);
	let off = 0;
	for (const ch of chars) {
		range.setStart(n, off); range.setEnd(n, off + ch.length); off += ch.length;
		const rects = range.getClientRects();
		if (rects.length === 0) continue;
		const r = rects[rects.length - 1];
		if (r.width === 0 && r.height === 0) continue;
		if (!cur || Math.abs(r.top - cur.top) > 0.5) {
			cur = {text: "", top: r.top, x0: Infinity, y0: Infinity, x1: -Infinity, y1: -Infinity};
			lines.push(cur);
		}
		cur.text += ch;
		if (ch.trim() !== "") {
			cur.x0 = Math.min(cur.x0, r.left); cur.x1 = Math.max(cur.x1, r.right);
			cur.y0 = Math.min(cur.y0, r.top); cur.y1 = Math.max(cur.y1, r.bottom);
		}
	}
	const kept = lines.filter(l => l.text.trim() !== "");
	out.push(kept.map(l => ({text: l.text.trim(), rect: [l.x0, l.y0, l.x1 - l.x0, l.y1 - l.y0]})));
}
return out;`

type chromeLine struct {
	Text string     `json:"text"`
	Rect [4]float32 `json:"rect"`
}

// ourLines is every text node's fragments in the layout of d, in the
// root's space, the same way: ellipses and edge spaces aside.
func ourLines(d *ui.Document, size geom.Vec2) ([][]chromeLine, []*dom.Node) {
	d.Layout(size)
	fs := fonts.Of(d)
	tree := layout.Layout(d.HTML(), size, fs)
	runs := map[*dom.Node]*box.Box{}
	var walk func(b *box.Box)
	walk = func(b *box.Box) {
		if b.Kind == box.TextRun {
			runs[b.Node] = b
		}
		for _, c := range b.Children {
			walk(c)
		}
	}
	walk(tree.Root)
	var out [][]chromeLine
	var nodes []*dom.Node
	d.Body().Walk(func(n *dom.Node) bool {
		if !n.IsText() || strings.TrimSpace(n.Text()) == "" {
			return true
		}
		nodes = append(nodes, n)
		var ls []chromeLine
		if b := runs[n]; b != nil {
			o := b.Rect()
			font := b.Style.Font
			ascent := fs.Metrics(font).Ascent
			for _, f := range b.Lines {
				trimmed := strings.TrimSpace(f.Text)
				if trimmed == "" {
					continue
				}
				lead := strings.TrimSuffix(f.Text, strings.TrimLeft(f.Text, " "))
				trail := strings.TrimPrefix(f.Text, strings.TrimRight(f.Text, " "))
				spacing := b.Style.LetterSpacing
				lw := fs.Advance(font, lead) + spacing*float32(len(lead))
				tw := fs.Advance(font, trail) + spacing*float32(len(trail))
				r := [4]float32{o.X + f.Rect.X + lw, o.Y + f.Baseline - ascent, f.Rect.W - lw - tw, 0}
				if n := len(ls); n > 0 && ls[n-1].Rect[1] == r[1] {
					// a justified line: one fragment a word
					prev := &ls[n-1]
					prev.Text += " " + trimmed
					prev.Rect[2] = r[0] + r[2] - prev.Rect[0]
					continue
				}
				ls = append(ls, chromeLine{Text: trimmed, Rect: r})
			}
		}
		out = append(out, ls)
		return true
	})
	return out, nodes
}

// matchText checks that every text node breaks into the same lines as in
// Chrome, each line's glyphs within tol points horizontally, and its
// glyphs' top (baseline minus ascent) within tol vertically.
func matchText(t *testing.T, d *ui.Document, size geom.Vec2, tol float32) {
	t.Helper()
	raw, err := chrome.Eval(d, size, textLines)
	if err != nil {
		t.Fatal(err)
	}
	var want [][]chromeLine
	if err = json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	got, nodes := ourLines(d, size)
	if len(got) != len(want) {
		t.Fatalf("ember has %d text nodes, Chrome %d", len(got), len(want))
	}
	for i := range want {
		g, w := got[i], want[i]
		if len(g) != len(w) {
			t.Errorf("%q: ember lines %s, Chrome %s", nodes[i].Text(), texts(g), texts(w))
			continue
		}
		for j := range w {
			gt := strings.TrimSuffix(g[j].Text, "…")
			if fold := cases.Fold(); fold.String(gt) != fold.String(w[j].Text) && !strings.HasPrefix(w[j].Text, gt) {
				t.Errorf("%q line %d: ember %q, Chrome %q", nodes[i].Text(), j, g[j].Text, w[j].Text)
				continue
			}
			if strings.HasSuffix(g[j].Text, "…") {
				continue
			}
			gr, wr := g[j].Rect, w[j].Rect
			dx := max(abs(gr[0]-wr[0]), abs(gr[0]+gr[2]-wr[0]-wr[2]))
			dy := abs(gr[1] - wr[1])
			if dx > tol || dy > tol {
				t.Errorf("%q line %d %q: ember %v, Chrome %v", nodes[i].Text(), j, w[j].Text, gr, wr)
			}
		}
	}
}

func texts(ls []chromeLine) string {
	var s []string
	for _, l := range ls {
		s = append(s, fmt.Sprintf("%q", l.Text))
	}
	return "[" + strings.Join(s, " ") + "]"
}

func abs(v float32) float32 { return float32(math.Abs(float64(v))) }

func TestInlineTextMatchesChrome(t *testing.T) {
	cases := map[string]*ui.Document{
		"a paragraph": page(ui.P(ui.Style("width: 140px"), ui.Text(sentence))),
		"hyphens and slashes": page(ui.P(ui.Style("width: 90px"),
			ui.Text("State-of-the-art anti-aliasing, read/write and/or well-known."))),
		"a hyphen across spans": page(ui.P(ui.Style("width: 60px"), ui.Text("tile"),
			ui.Span(ui.Style("font-weight: bold"), ui.Text("-corner-north")), ui.Text("-east"))),
		"padded spans": page(ui.P(ui.Style("width: 220px"),
			ui.Text("Press "),
			ui.Span(ui.Style("padding: 2px 6px; border: 1px solid #888; margin: 0 4px"), ui.Text("Enter")),
			ui.Text(" to start, or "),
			ui.Span(ui.Style("padding-left: 10px; border-right: 3px solid"),
				ui.Text("a long span that breaks across lines"), ui.Span(ui.Style("padding: 0 5px"), ui.Text("nested"))),
			ui.Text(" end."))),
		"spans in tall lines": page(ui.P(ui.Style("width: 160px; line-height: 2"), ui.Text("Tall "),
			ui.Span(ui.Style("font-size: 24px; padding: 3px"), ui.Text("big words")), ui.Text(" and small."))),
		"an empty span": page(ui.P(ui.Text("before"),
			ui.Span(ui.Style("padding: 0 8px; border: 1px solid")), ui.Text("after"))),
		"overflow-wrap": page(
			ui.P(ui.Style("width: 100px; overflow-wrap: break-word"),
				ui.Text("Open assets/tilesets/dungeon_walls_north_east.tsj now")),
			ui.P(ui.Style("width: 100px; overflow-wrap: anywhere"),
				ui.Text("Open assets/tilesets/dungeon_walls_north_east.tsj now")),
			ui.Div(ui.Style("display: flex; width: 50px"),
				ui.Div(ui.Style("overflow-wrap: anywhere"), ui.Text("supercalifragilistic"))),
			ui.Div(ui.Style("display: flex; width: 50px"),
				ui.Div(ui.Style("overflow-wrap: break-word"), ui.Text("supercalifragilistic"))),
		),
		"justified": page(
			ui.P(ui.Style("width: 170px; text-align: justify"), ui.Text(sentence+" "),
				ui.Span(ui.Style("font-weight: bold; padding: 0 3px"), ui.Text("Bold words")),
				ui.Text(" in the middle of it all, then the end.")),
			ui.P(ui.Style("width: 170px; text-align: justify; white-space: pre-line"),
				ui.Text("A short line\nthen a much longer line that wraps around\nend")),
		),
		"text-transform": page(
			ui.P(ui.Style("width: 160px; text-transform: uppercase"), ui.Text("Die Straße der Gießerei, weiß und groß")),
			ui.P(ui.Style("width: 160px; text-transform: lowercase"), ui.Text("ÉCOLE DES BEAUX-ARTS ET MÉTIERS")),
			ui.P(ui.Style("width: 160px; text-transform: capitalize"), ui.Text("(hello) o'neil's well-known "),
				ui.Span(ui.Style("font-weight: bold"), ui.Text("tile")), ui.Text("set, 3rd row \"quoted\" words")),
			ui.P(ui.Style("width: 160px; text-transform: uppercase"), ui.Text("none "),
				ui.Span(ui.Style("text-transform: none"), ui.Text("kept as typed")), ui.Text(" again")),
		),
		"letter-spacing": page(
			ui.P(ui.Style("width: 180px; letter-spacing: 2px"), ui.Text(sentence)),
			ui.P(ui.Style("width: 180px"), ui.Text("Normal then "),
				ui.Span(ui.Style("letter-spacing: 0.3em"), ui.Text("SPACED OUT")), ui.Text(" and normal again.")),
		),
		"ellipses": page(
			ui.Div(ui.Style("width: 120px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; letter-spacing: 1px"),
				ui.Text(sentence)),
			ui.Div(ui.Style("width: 120px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis"),
				ui.Text("Item: "), ui.Span(ui.Style("font-weight: bold"), ui.Text("grass_corner_north_east_long"))),
		),
		"fallback glyphs stretch the line": page(
			ui.P(ui.Span(ui.Text("▸ tile 0 ✓ ▾ done → next"))),
			ui.P(ui.Style("line-height: 1.2"), ui.Text("plain, then "), ui.Span(ui.Text("✓ done")), ui.Text(" and on")),
			ui.P(ui.Style("width: 120px"), ui.Text("wrapping text with a ✓ on the second line maybe")),
		),
		"pre and spans": page(ui.Div(ui.Style("width: 200px; white-space: pre-wrap"), ui.Text("one  two\nthree "),
			ui.Span(ui.Style("font-size: 22px"), ui.Text("four five")), ui.Text(" six"))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			matchText(t, d, geom.V2(800, 600), 0.5)
			if name != "ellipses" {
				// Chrome splits a span its ellipsis cuts into several rects, an artefact
				// of how it draws the ellipsis, not a box.
				matchSpanFragments(t, d, geom.V2(800, 600), 0.5)
			}
		})
	}
}

// spanRects is every span's getClientRects, in document order.
const spanRects = `return Array.from(document.body.querySelectorAll("span"))
	.map(e => Array.from(e.getClientRects()).map(r => [r.x, r.y, r.width, r.height]));`

// matchSpanFragments checks every span's per-line border boxes against Chrome's
// getClientRects.
func matchSpanFragments(t *testing.T, d *ui.Document, size geom.Vec2, tol float32) {
	t.Helper()
	raw, err := chrome.Eval(d, size, spanRects)
	if err != nil {
		t.Fatal(err)
	}
	var want [][][4]float32
	if err = json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	d.Layout(size)
	tree := layout.Layout(d.HTML(), size, fonts.Of(d))
	var spans []*dom.Node
	d.Body().Walk(func(n *dom.Node) bool {
		if !n.IsText() && n.Tag() == dom.Span {
			spans = append(spans, n)
		}
		return true
	})
	if len(spans) != len(want) {
		t.Fatalf("ember has %d spans, Chrome %d", len(spans), len(want))
	}
	for i, n := range spans {
		b := tree.Of[n]
		o := b.Rect()
		if len(b.Fragments) != len(want[i]) {
			t.Errorf("span %d %q: ember %d fragments, Chrome %d", i, n.Text(), len(b.Fragments), len(want[i]))
			continue
		}
		for j, f := range b.Fragments {
			w := want[i][j]
			g := [4]float32{o.X + f.X, o.Y + f.Y, f.W, f.H}
			for k := range g {
				if abs(g[k]-w[k]) > tol {
					t.Errorf("span %d %q fragment %d: ember %v, Chrome %v", i, n.Text(), j, g, w)
					break
				}
			}
		}
	}
}
