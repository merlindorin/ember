// Package align checks that a laid-out Document's controls line up as a
// player reads them: the controls sharing a flex row are one height, its
// separators stand in the same space on each side, a
// control's label sits in the middle of it, the text of a flex row's
// children, or of a grid row's cells (a label beside its field), shares
// one baseline, nothing with text or a control is cut by the window's
// edge, and no text or control runs past the box holding it. Games run it in tests over their screens,
// as kit's tests run it over every kit page:
//
//	doc.Layout(size)
//	for _, f := range align.Check(doc) {
//		t.Error(f)
//	}
package align

import (
	"fmt"
	"slices"
	"strings"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/assets"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// Tolerance is how far, in points, things may stray and still line up.
const Tolerance = 0.5

// GridTolerance is how far a grid row's baselines may stray: a label
// centred beside a field in a larger font sits a point off its baseline,
// as Chrome draws it, where one left at the row's top is visibly off.
const GridTolerance = 1.5

// OwnHeight is the class a control, or a row of them inside a row, carries
// to keep a height of its own: a taller one on purpose, or a window's
// lights.
const OwnHeight = "own-height"

// OverflowOK is the attribute that marks a spill as meant: on the box
// running past its fixed-size ancestor, or on any box between them, such
// as a badge straddling a frame's edge.
const OverflowOK = "data-overflow-ok"

// Fault is one misalignment: the Rule broken ("height", "centre",
// "baseline", "outside", "separator", "overflow" or "spill"), the automation Path of what breaks it (else its tag and
// classes in the nearest named ancestor), Got and Want in points (the
// height and the row's, the label's offset from the middle and 0, the
// stray baseline and the row's first, how far past the window's edge and
// 0, a separator's space and the row's first one's, how far ink runs past
// a box and 0), and Detail saying it in words.
type Fault struct {
	Rule   string  `json:"Rule"`
	Path   string  `json:"Path"`
	Got    float32 `json:"Got"`
	Want   float32 `json:"Want"`
	Detail string  `json:"Detail"`
}

// String is the fault on one line: its rule, path and detail.
func (f Fault) String() string { return f.Rule + " " + f.Path + ": " + f.Detail }

// Check returns the faults of d's last layout; it lays nothing out. What
// doesn't show (display: none, or under it) has nothing to line up.
func Check(d *ui.Document) []Fault {
	t := assets.Tree(d)
	if t == nil {
		return nil
	}
	c := checker{d: d, t: t}
	d.HTML().Walk(func(n *dom.Node) bool {
		if n.IsText() {
			return false
		}
		b := t.Of[n]
		if b == nil {
			return false
		}
		if b.Style != nil && b.Style.Display == style.Contents {
			return true
		}
		if b.Parent == nil && b != t.Root || b.Style == nil || b.Style.Display == style.DisplayNone {
			return false
		}
		if c.outside(n, b) {
			return false
		}
		if isRow(b) {
			c.row(n)
		}
		if isGrid(b) {
			c.grid(n)
		}
		if isControl(n) && !isWrapper(n) && !stacked(b) {
			c.centred(n, b)
		}
		c.spills(n, b)
		return true
	})
	overflowing := c.overflows(t.Root)
	for _, s := range c.spilt {
		if !overflowing[s.over] {
			c.faults = append(c.faults, s.fault)
		}
	}
	return c.faults
}

type checker struct {
	d      *ui.Document
	t      *box.Tree
	faults []Fault
	// spilt are the spills found, one a container and axis, kept until
	// the overflow rule has run: a container whose ink it reports isn't
	// reported again.
	spilt []spill
}

type spill struct {
	over  *box.Box
	axis  int
	fault Fault
}

func (c *checker) faultf(rule string, n *dom.Node, got, want float32, format string, args ...any) {
	c.faults = append(c.faults, Fault{
		Rule: rule, Path: c.name(n), Got: got, Want: want,
		Detail: fmt.Sprintf(format, args...),
	})
}

// name is n's automation path, else its tag and classes under the nearest
// ancestor automation names.
func (c *checker) name(n *dom.Node) string {
	if p := assets.Path(c.d, n); p != "" {
		return p
	}
	s := n.Tag().String()
	for _, cl := range n.Classes() {
		s += "." + cl
	}
	for a := n.Parent(); a != nil; a = a.Parent() {
		if p := assets.Path(c.d, a); p != "" {
			return s + " in " + p
		}
	}
	return s
}

// isRow reports whether b lays its children out in a flex row.
func isRow(b *box.Box) bool {
	st := b.Style
	return st != nil && (st.Display == style.FlexDisplay || st.Display == style.InlineFlex) &&
		(st.Direction == style.Row || st.Direction == style.RowReverse)
}

// isControl reports whether n is a control whose height its row shares:
// a button, a select, a text field, a switch, or a kit field wrapping one.
func isControl(n *dom.Node) bool {
	if t := n.Tag(); t == dom.Button || t == dom.Select {
		return true
	}
	if n.Tag() == dom.Input {
		typ, _ := n.Attr("type")
		switch typ {
		case "", "text", "search", "number", "password":
			return true
		case "checkbox":
			_, sw := n.Attr("switch")
			return sw
		}
	}
	return isWrapper(n)
}

// isWrapper reports whether n is a kit field wrapping a control, which
// counts for its row's height but is centred through the control.
func isWrapper(n *dom.Node) bool {
	return n.HasClass("kit-search-field") || n.HasClass("kit-shortcode-field")
}

// outOfFlow reports whether b is placed apart from its parent's flow, as
// a menu, a tooltip or a suggestion list hanging off a control.
func outOfFlow(b *box.Box) bool {
	return b.Style != nil && (b.Style.Position == style.Absolute || b.Style.Position == style.FixedPosition)
}

// inRow reports whether the row n sits in a row around it as one of its
// children, whose controls that row checks with its own.
func (c *checker) inRow(n *dom.Node) bool {
	p := n.Parent()
	if p == nil || isControl(n) {
		return false
	}
	b := c.t.Of[p]
	return b != nil && isRow(b)
}

// row checks the children of the flex row n: the heights of its controls,
// those of the rows in it counting as its own, and its baselines.
func (c *checker) row(n *dom.Node) {
	var kids []child
	for _, k := range n.Children() {
		if k.IsText() {
			continue
		}
		if b := c.t.Of[k]; b != nil && b.Size.Y > 0 && !outOfFlow(b) {
			kids = append(kids, child{k, b})
		}
	}
	var controls []child
	var gather func(n *dom.Node)
	gather = func(n *dom.Node) {
		for _, k := range n.Children() {
			if k.IsText() {
				continue
			}
			b := c.t.Of[k]
			switch {
			case b == nil || b.Size.Y <= 0 || outOfFlow(b):
			case isControl(k):
				if !k.HasClass(OwnHeight) {
					controls = append(controls, child{k, b})
				}
			case isRow(b) && !k.HasClass(OwnHeight):
				gather(k)
			}
		}
	}
	gather(n)
	if c.inRow(n) {
		controls = nil
	}
	var heights []float32
	for _, k := range controls {
		heights = append(heights, k.b.Size.Y)
	}
	if ref, ok := mode(heights); ok {
		for _, k := range controls {
			if abs(k.b.Size.Y-ref) > Tolerance {
				c.faultf("height", k.n, k.b.Size.Y, ref, "%.2fpt tall in a row of %.2fpt controls", k.b.Size.Y, ref)
			}
		}
	}
	c.baselines(kids, Tolerance, false)
	c.separators(kids)
}

type child struct {
	n *dom.Node
	b *box.Box
}

// baselines checks that the one-line texts and controls of kids, side by
// side, share a baseline within tolerance, or with firstLine, as a grid
// row's cells do, the first line of whatever text they hold; an icon, a
// picture centred on its line, has none to share.
func (c *checker) baselines(kids []child, tolerance float32, firstLine bool) {
	var first *dom.Node
	var lines []string
	base, has := float32(0), false
	worst, gap := float32(0), tolerance
	for _, k := range kids {
		y, ok := baseline(k.b)
		if !ok || k.n.HasClass("kit-icon") || !firstLine && !isControl(k.n) && !oneLine(k.b) {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %.2f", c.name(k.n), y))
		if !has {
			first, base, has = k.n, y, true
			continue
		}
		if abs(y-base) > gap {
			worst, gap = y, abs(y-base)
		}
	}
	if gap > tolerance {
		c.faultf("baseline", first, worst, base, "baselines in its row: %s", strings.Join(lines, ", "))
	}
}

// centred checks that a control's text sits in the middle of its padding
// box: vertically, and across for a button unless it aligns its text left,
// right or to its end.
func (c *checker) centred(n *dom.Node, b *box.Box) {
	text, ok := textRect(b)
	if !ok {
		return
	}
	pad := paddingBox(b)
	if pad.W <= 0 || pad.H <= 0 {
		return
	}
	if dy := (text.Y + text.H/2) - (pad.Y + pad.H/2); abs(dy) > Tolerance {
		c.faultf("centre", n, dy, 0, "label %.2fpt below the middle of its %.2fpt box (%.2fpt off)", dy, pad.H, abs(dy))
		return
	}
	if a := b.Style.TextAlign; n.Tag() == dom.Button && a != style.Left && a != style.Right && a != style.End {
		content := inFlowRect(b, text)
		if dx := (content.X + content.W/2) - (pad.X + pad.W/2); abs(dx) > Tolerance {
			c.faultf("centre", n, dx, 0, "label %.2fpt right of the middle of its %.2fpt box (%.2fpt off)", dx, pad.W, abs(dx))
		}
	}
}

// stacked reports whether b stacks more than one in-flow block, as a
// flex column or in block flow: a card of parts, not a control with one
// label to centre.
func stacked(b *box.Box) bool {
	st := b.Style
	if st == nil {
		return false
	}
	column := (st.Display == style.FlexDisplay || st.Display == style.InlineFlex) &&
		(st.Direction == style.Column || st.Direction == style.ColumnReverse)
	flow := st.Display == style.Block || st.Display == style.InlineBlock
	blocks := 0
	for _, c := range b.Children {
		if outOfFlow(c) || c.Kind == box.TextRun {
			continue
		}
		if column || flow && (c.Kind == box.AnonymousBlock || c.Style != nil && blockLevel(c.Style.Display)) {
			blocks++
		}
	}
	return blocks > 1
}

// blockLevel reports whether d stands on a line of its own in block flow.
func blockLevel(d style.Display) bool {
	return d == style.Block || d == style.FlexDisplay || d == style.GridDisplay
}

// lineRect is what line f of a text run at at shows: nothing when blank
// or cut away, the clip's width when cut with an ellipsis.
func lineRect(at geom.Rect, f box.Fragment, clip geom.Rect, clipped bool) (geom.Rect, bool) {
	if strings.TrimSpace(f.Text) == "" {
		return geom.Rect{}, false
	}
	l := geom.Rect{X: at.X + f.Rect.X, Y: at.Y + f.Rect.Y, W: f.Rect.W, H: f.Rect.H}
	switch {
	case clipped && f.Ellipsis:
		l.X, l.W = clip.X, clip.W
	case clipped:
		x0, x1 := max(l.X, clip.X), min(l.X+l.W, clip.X+clip.W)
		if x1 <= x0 {
			return geom.Rect{}, false
		}
		l.X, l.W = x0, x1-x0
	}
	return l, true
}

// paddingBox is b's padding box in the root's space: its border box less
// its borders.
func paddingBox(b *box.Box) geom.Rect {
	r := b.Rect()
	return geom.Rect{
		X: r.X + b.Border.Left, Y: r.Y + b.Border.Top,
		W: r.W - b.Border.Horizontal(), H: r.H - b.Border.Vertical(),
	}
}

// union is the rect around r and l, or l alone when r is none yet.
func union(r, l geom.Rect, some bool) geom.Rect {
	if !some {
		return l
	}
	x0, y0 := min(r.X, l.X), min(r.Y, l.Y)
	return geom.Rect{X: x0, Y: y0, W: max(r.X+r.W, l.X+l.W) - x0, H: max(r.Y+r.H, l.Y+l.H) - y0}
}

// textRect is what b's text covers, its line boxes in the root's space,
// cut to the boxes it is clipped by; a line cut with an ellipsis fills the
// box that cuts it.
func textRect(b *box.Box) (geom.Rect, bool) {
	var r geom.Rect
	found := false
	var walk func(b *box.Box, clip geom.Rect, clipped bool)
	walk = func(b *box.Box, clip geom.Rect, clipped bool) {
		for _, c := range b.Children {
			if outOfFlow(c) {
				continue
			}
			if c.Kind == box.TextRun {
				at := c.Rect()
				for _, f := range c.Lines {
					if l, ok := lineRect(at, f, clip, clipped); ok {
						r, found = union(r, l, found), true
					}
				}
				continue
			}
			cc, cl := clip, clipped
			if c.Style != nil && c.Style.Overflow.Width != style.Visible {
				p := c.Rect()
				p = geom.Rect{X: p.X + c.Border.Left, Y: p.Y, W: p.W - c.Border.Horizontal(), H: p.H}
				if cl {
					x0, x1 := max(p.X, cc.X), min(p.X+p.W, cc.X+cc.W)
					p.X, p.W = x0, max(x1-x0, 0)
				}
				cc, cl = p, true
			}
			walk(c, cc, cl)
		}
	}
	walk(b, geom.Rect{}, false)
	return r, found
}

// baseline is the first baseline of b's text in the root's space.
func baseline(b *box.Box) (float32, bool) {
	y, found := float32(0), false
	walkText(b, func(t *box.Box) {
		if !found && len(t.Lines) > 0 {
			y, found = t.Rect().Y+t.Lines[0].Baseline, true
		}
	})
	return y, found
}

// walkText visits the text runs under b with lines, in tree order, past
// what hangs out of flow.
func walkText(b *box.Box, visit func(*box.Box)) {
	for _, c := range b.Children {
		if outOfFlow(c) {
			continue
		}
		if c.Kind == box.TextRun && len(c.Lines) > 0 && strings.TrimSpace(c.Lines[0].Text) != "" {
			visit(c)
		}
		walkText(c, visit)
	}
}

// inFlowRect is r grown over what b draws besides text, its icons and
// glyphs, in the root's space: boxes holding no text, not the blocks that
// wrap its lines.
func inFlowRect(b *box.Box, r geom.Rect) geom.Rect {
	for _, c := range b.Children {
		if outOfFlow(c) || c.Kind == box.TextRun || c.Size.X <= 0 {
			continue
		}
		if holdsText(c) {
			r = inFlowRect(c, r)
			continue
		}
		k := c.Rect()
		x0, y0 := min(r.X, k.X), min(r.Y, k.Y)
		r = geom.Rect{X: x0, Y: y0, W: max(r.X+r.W, k.X+k.W) - x0, H: max(r.Y+r.H, k.Y+k.H) - y0}
	}
	return r
}

// holdsText reports whether b has a text run under it, shown or cut away.
func holdsText(b *box.Box) bool {
	for _, c := range b.Children {
		if c.Kind == box.TextRun || holdsText(c) {
			return true
		}
	}
	return false
}

// oneLine reports whether b's text is a single line, as a label beside
// controls is; a column or a pane is not.
func oneLine(b *box.Box) bool {
	top, first, ok := float32(0), true, true
	walkText(b, func(t *box.Box) {
		for _, f := range t.Lines {
			y := t.Rect().Y + f.Rect.Y
			if first {
				top, first = y, false
			} else if abs(y-top) > f.Rect.H/2 {
				ok = false
			}
		}
	})
	return ok
}

// mode is the height most of hs share, the first among the most common;
// false when fewer than two.
func mode(hs []float32) (float32, bool) {
	if len(hs) < 2 {
		return 0, false
	}
	best, count := hs[0], 0
	for _, h := range hs {
		n := len(slices.DeleteFunc(slices.Clone(hs), func(o float32) bool { return abs(o-h) > Tolerance }))
		if n > count {
			best, count = h, n
		}
	}
	return best, true
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func isGrid(b *box.Box) bool {
	return b.Style != nil && (b.Style.Display == style.GridDisplay || b.Style.Display == style.InlineGrid)
}

// grid checks the baselines of each row of the grid n: in the order the
// grid fills them, its in-flow cells each across from every cell of the
// row so far, so a cell spanning rows joins only the first.
func (c *checker) grid(n *dom.Node) {
	var rowCells []child
	for _, k := range n.Children() {
		if k.IsText() {
			continue
		}
		b := c.t.Of[k]
		if b == nil || b.Size.Y <= 0 || outOfFlow(b) {
			continue
		}
		r := b.Rect()
		for _, o := range rowCells {
			if !across(r, o.b.Rect()) && !across(o.b.Rect(), r) {
				c.baselines(rowCells, GridTolerance, true)
				rowCells = nil
				break
			}
		}
		rowCells = append(rowCells, child{k, b})
	}
	c.baselines(rowCells, GridTolerance, true)
}

// outside reports, and reports as a fault, a control or an element with
// text of its own that the window cuts: past any of its edges, unless a
// box around it scrolls or clips it, which shows it as it means to.
func (c *checker) outside(n *dom.Node, b *box.Box) bool {
	if !isControl(n) && !ownsText(n) || clipped(b) {
		return false
	}
	view := assets.Viewport(c.d)
	r := b.Rect()
	past := max(-r.X, -r.Y, r.X+r.W-view.X, r.Y+r.H-view.Y, 0)
	if past <= Tolerance {
		return false
	}
	c.faultf("outside", n, past, 0, "%.2fpt past the window's edge", past)
	return true
}

// ownsText reports whether n holds text other than white space itself.
func ownsText(n *dom.Node) bool {
	for _, k := range n.Children() {
		if k.IsText() && strings.TrimSpace(k.Text()) != "" {
			return true
		}
	}
	return false
}

// clipped reports whether a box around b scrolls or clips what overflows it.
func clipped(b *box.Box) bool {
	for a := b.Parent; a != nil; a = a.Parent {
		if st := a.Style; st != nil && (st.Overflow.Width != style.Visible || st.Overflow.Height != style.Visible) {
			return true
		}
	}
	return false
}

// across reports whether a's middle is level with some of b.
func across(a, b geom.Rect) bool {
	mid := a.Y + a.H/2
	return mid >= b.Y-Tolerance && mid <= b.Y+b.H+Tolerance
}

// separators checks the vertical separators (hr elements) of a flex row:
// the space between each and the ink beside it, before and after, is the
// same for all of them, the first one's.
func (c *checker) separators(kids []child) {
	want, has := float32(0), false
	for i, k := range kids {
		if k.n.Tag() != dom.Hr || i == 0 || i == len(kids)-1 {
			continue
		}
		r := k.b.Rect()
		for _, gap := range []float32{r.X - inkOf(kids[i-1].b).X - inkOf(kids[i-1].b).W, inkOf(kids[i+1].b).X - r.X - r.W} {
			if !has {
				want, has = gap, true
				continue
			}
			if abs(gap-want) > 1 {
				c.faultf("separator", k.n, gap, want,
					"%.2fpt from the ink beside it where the row's first separator has %.2fpt", gap, want)
				return
			}
		}
	}
}

// inkOf is what shows of b across: its text, else its box.
func inkOf(b *box.Box) geom.Rect {
	if r, ok := textRect(b); ok {
		return r
	}
	return b.Rect()
}

// overflows reports each box that text or a control under it runs past,
// once: the first box up from the ink that doesn't hold it, unless a box
// around the ink clips or scrolls, the line is cut with an ellipsis, or the
// ink hangs out of flow, as a tooltip or a menu does.
func (c *checker) overflows(root *box.Box) map[*box.Box]bool {
	seen := map[*box.Box]bool{}
	var ink func(b *box.Box)
	ink = func(b *box.Box) {
		if outOfFlow(b) && b.Parent != nil {
			return
		}
		if b.Kind == box.TextRun {
			at := b.Rect()
			for _, f := range b.Lines {
				if strings.TrimSpace(f.Text) != "" && !f.Ellipsis {
					c.holds(b, geom.Rect{X: at.X + f.Rect.X, Y: at.Y + f.Rect.Y, W: f.Rect.W, H: f.Rect.H}, seen)
				}
			}
		}
		if b.Node != nil && b.Kind == box.Element && isControl(b.Node) {
			c.holds(b, b.Rect(), seen)
		}
		for _, k := range b.Children {
			ink(k)
		}
	}
	ink(root)
	return seen
}

// spills reports b when its border box runs past the padding box of its
// nearest ancestor sized on that axis (a definite width or height, not
// auto or a content size), unless a scroll container comes first, which
// holds it, or OverflowOK marks it or a box between them meant. Only the
// outermost box that spills is reported, so one spill is one fault.
func (c *checker) spills(n *dom.Node, b *box.Box) {
	if b.Kind != box.Element || b.Style == nil || !b.Style.Visible || marked(n) {
		return
	}
	for axis := range 2 {
		a, past := spillPast(b, axis)
		if a == nil || past <= Tolerance {
			continue
		}
		if p := b.Parent; p != a && p != nil && p.Kind == box.Element {
			if pa, pp := spillPast(p, axis); pa == a && pp > Tolerance {
				continue
			}
		}
		if slices.ContainsFunc(c.spilt, func(s spill) bool { return s.over == a && s.axis == axis }) {
			continue
		}
		edge := map[int]string{0: "side", 1: "bottom"}[axis]
		c.spilt = append(c.spilt, spill{a, axis, Fault{Rule: "spill", Path: c.name(n), Got: past,
			Detail: fmt.Sprintf("%.2fpt past the %s of %s, whose size is fixed", past, edge, c.name(a.Node))}})
	}
}

// marked reports whether n carries OverflowOK.
func marked(n *dom.Node) bool { _, ok := n.Attr(OverflowOK); return ok }

// spillPast is the nearest ancestor of b sized on axis (0 across, 1
// down) that would hold it, and how far b runs past its padding box
// there; nil when a scroll container or OverflowOK comes first.
func spillPast(b *box.Box, axis int) (*box.Box, float32) {
	for a := b.Parent; a != nil; a = a.Parent {
		if a.Node != nil && marked(a.Node) {
			return nil, 0
		}
		st := a.Style
		if a.Kind != box.Element || st == nil || a.Node == nil {
			continue
		}
		if st.Overflow.Width != style.Visible || st.Overflow.Height != style.Visible {
			return nil, 0
		}
		size := st.Size.Width
		if axis == 1 {
			size = st.Size.Height
		}
		if size.Kind != style.Fixed {
			continue
		}
		r, pad := b.Rect(), paddingBox(a)
		if axis == 0 {
			return a, max(pad.X-r.X, r.X+r.W-pad.X-pad.W)
		}
		return a, max(pad.Y-r.Y, r.Y+r.H-pad.Y-pad.H)
	}
	return nil, 0
}

// holds walks up from b, whose ink is r, to the first block box that
// doesn't hold r, and reports it.
func (c *checker) holds(b *box.Box, r geom.Rect, seen map[*box.Box]bool) {
	if clipped(b) {
		return
	}
	for k, a := b, b.Parent; a != nil; k, a = a, a.Parent {
		if outOfFlow(k) {
			return
		}
		st := a.Style
		if a.Node == nil || a.Kind != box.Element || st == nil || st.Display == style.Inline {
			continue
		}
		box := a.Rect()
		past := max(box.X-r.X, box.Y-r.Y, r.X+r.W-box.X-box.W, r.Y+r.H-box.Y-box.H)
		if past <= Tolerance {
			continue
		}
		if !seen[a] {
			seen[a] = true
			c.faultf("overflow", a.Node, past, 0, "%.2fpt of what it holds runs past it", past)
		}
		return
	}
}
