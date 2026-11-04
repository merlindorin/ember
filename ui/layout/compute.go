package layout

import (
	"math"
	"slices"

	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/style"
)

type runMode uint8

const (
	performLayout runMode = iota
	computeSize
	performHiddenLayout
)

type sizingMode uint8

const (
	contentSize sizingMode = iota
	inherentSize
)

type requestedAxis uint8

const (
	axisHorizontal requestedAxis = iota
	axisVertical
	axisBoth
)

func requested(a axis) requestedAxis {
	if a == horizontal {
		return axisHorizontal
	}
	return axisVertical
}

type bools struct{ w, h bool }

func (b bools) get(a axis) bool {
	if a == horizontal {
		return b.w
	}
	return b.h
}

func (b bools) with(a axis, v bool) bools {
	if a == horizontal {
		b.w = v
	} else {
		b.h = v
	}
	return b
}

// input is Taffy's LayoutInput.
type input struct {
	run             runMode
	sizing          sizingMode
	axis            requestedAxis
	known           size
	knownIsDefinite bools
	parent          size
	avail           spaces
	// collapsible is whether the top and bottom margins may collapse through.
	collapsible [2]bool
	// probe is a layout run only to read the node's first baseline, as a
	// baseline-aligned grid row does, cached apart from its final layout.
	probe bool
}

// marginSet is Taffy's CollapsibleMarginSet.
type marginSet struct{ positive, negative float32 }

func marginFrom(m float32) marginSet {
	if m >= 0 {
		return marginSet{positive: m}
	}
	return marginSet{negative: m}
}

func (s marginSet) collapseWithMargin(m float32) marginSet {
	if m >= 0 {
		s.positive = fmax(s.positive, m)
	} else {
		s.negative = fmin(s.negative, m)
	}
	return s
}

func (s marginSet) collapseWithSet(o marginSet) marginSet {
	s.positive = fmax(s.positive, o.positive)
	s.negative = fmin(s.negative, o.negative)
	return s
}

func (s marginSet) resolve() float32 { return s.positive + s.negative }

// overflowRect is Taffy's ScrollableOverflow: the scrollable overflow rect relative to the
// padding box, and whether each axis scrolls from its end.
type overflowRect struct {
	rect     rect
	reversed [2]bool
}

func newOverflowRect(r rect, reversed [2]bool) overflowRect {
	return overflowRect{rect: r, reversed: reversed}
}

type staticEdge uint8

const (
	edgeStart staticEdge = iota
	edgeCenter
	edgeEnd
)

type staticAlign struct {
	keyword  staticEdge
	safe     bool
	unsafe   bool
	fallback staticEdge
}

func alignFromEdge(e staticEdge) staticAlign { return staticAlign{keyword: e, fallback: e} }

type staticPosition struct {
	area  span
	align staticAlign
}

func staticFromStart(anchor float32) staticPosition {
	return staticPosition{area: span{anchor, anchor}, align: alignFromEdge(edgeStart)}
}

// oofCandidate is an absolutely or fixed positioned box waiting for its containing block.
type oofCandidate struct {
	node     *node
	order    uint32
	position style.Position
	x, y     staticPosition
	// area is its own containing block, a positioned span's, in the owner's border box;
	// the owner lays it out whatever it claims.
	area *oofArea
}

func translateCandidates(cs []oofCandidate, offset point) []oofCandidate {
	out := make([]oofCandidate, len(cs))
	for i, c := range cs {
		c.x.area.start += offset.x
		c.x.area.end += offset.x
		c.y.area.start += offset.y
		c.y.area.end += offset.y
		out[i] = c
	}
	return out
}

type oofArea struct {
	size   size
	offset point
}

// output is Taffy's LayoutOutput.
type output struct {
	size                    size
	overflow                overflowRect
	firstBaseline           float32
	lastBaseline            float32
	topMargin, bottomMargin marginSet
	collapsesThrough        bool
	oof                     []oofCandidate
	area                    *oofArea
}

func sizeOutput(s size) output {
	return output{size: s, firstBaseline: none, lastBaseline: none}
}

// placed is a node's layout, Taffy's Layout.
type placed struct {
	order     uint32
	location  point
	size      size
	overflow  overflowRect
	scrollbar size
	border    rect
	padding   rect
	margin    rect
}

func (t *tree) measureChildSize(n *node, known, parent size,
	avail spaces, mode sizingMode, a axis, collapsible [2]bool) float32 {
	out := t.computeChild(n, input{
		run:             computeSize,
		sizing:          mode,
		axis:            requested(a),
		known:           known,
		knownIsDefinite: bools{true, true},
		parent:          parent,
		avail:           avail,
		collapsible:     collapsible,
	})
	return out.size.get(a)
}

func (t *tree) measureChildSizeBoth(n *node, known, parent size,
	avail spaces, mode sizingMode, collapsible [2]bool) size {
	return t.computeChild(n, input{
		run:             computeSize,
		sizing:          mode,
		axis:            axisBoth,
		known:           known,
		knownIsDefinite: bools{true, true},
		parent:          parent,
		avail:           avail,
		collapsible:     collapsible,
	}).size
}

func (t *tree) performChildLayout(n *node, known, parent size, avail spaces, mode sizingMode) output {
	return t.computeChild(n, input{
		run:             performLayout,
		sizing:          mode,
		axis:            axisBoth,
		known:           known,
		knownIsDefinite: bools{true, true},
		parent:          parent,
		avail:           avail,
	})
}

// probeChildLayout is performChildLayout run only for n's first baseline,
// cached in n's probe slot so its final layout keeps its own.
func (t *tree) probeChildLayout(n *node, known, parent size, avail spaces, mode sizingMode) output {
	return t.computeChild(n, input{
		run:             performLayout,
		sizing:          mode,
		axis:            axisBoth,
		known:           known,
		knownIsDefinite: bools{true, true},
		parent:          parent,
		avail:           avail,
		probe:           true,
	})
}

// computeChild lays a node out by its formatting context, through its cache.
func (t *tree) computeChild(n *node, in input) output { return t.computeChildIn(n, in, nil) }

// computeChildIn is computeChild for a block laid out in its parent's formatting context.
func (t *tree) computeChildIn(n *node, in input, ctx *blockContext) output {
	if in.run == performHiddenLayout {
		return t.hidden(n)
	}
	key := in
	if !n.readsParent {
		key.parent = noSize
	}
	if t.lookups != nil {
		*t.lookups++
	}
	if out, ok := n.cache.get(key); ok {
		return out
	}
	if in.run == performLayout {
		n.performed = t.gen
	}
	out := t.dispatch(n, in, ctx)
	for range 2 {
		if in.run != performLayout || !n.fitGutter(out) {
			break
		}
		t.regutterOf(n)
		n.cache.clear()
		out = t.dispatch(n, in, ctx)
	}
	if in.run == performLayout {
		t.oofLayout(n, &out)
	}
	n.cache.store(key, out)
	return out
}

func (t *tree) dispatch(n *node, in input, ctx *blockContext) output {
	switch n.context() {
	case contextNone:
		return t.hidden(n)
	case contextFlex:
		return t.flex(n, in)
	case contextGrid:
		return t.grid(n, in)
	case contextTable:
		return t.table(n, in)
	case contextBlock:
		return t.block(n, in, ctx)
	case contextLeaf:
	}
	return t.leaf(n, in)
}

// ownGutter is the scrollbar room n's own layout leaves: overflow: scroll's, and overflow
// auto's once its content overflows that way.
func (n *node) ownGutter() point {
	g := scrollbarGutter(n.st)
	return point{g.x + n.gutter.w, g.y + n.gutter.h}
}

// regutterOf notes n's element, or its nearest one, for layoutIn to build
// again with its new gutter, in a retained layout.
func (t *tree) regutterOf(n *node) {
	if t.keep == nil {
		return
	}
	for ; n != nil; n = n.parent {
		if e := n.box.Node; e != nil && n.box.Kind == box.Element {
			if !slices.Contains(t.regutter, e) {
				t.regutter = append(t.regutter, e)
			}
			return
		}
	}
}

// fitGutter reports whether n's overflow: auto scrollbars no longer suit out, and fixes
// them, as Chrome lays out again with or without the bar: room for a bar once the content
// overflows that way, none once it fits. A node starts from the bars its last layout chose.
func (n *node) fitGutter(out output) bool {
	st := n.st
	if !isScrollContainer(st) || st.ScrollbarWidth == 0 || n.control == textareaControl {
		return false
	}
	g := n.ownGutter()
	port := out.size.sub(borders(st).sum()).sub(size{g.x, g.y})
	r := out.overflow.rect
	const slack = 0.01
	changed := false
	fit := func(auto bool, room *float32, overflows bool) {
		switch {
		case !auto:
		case *room == 0 && overflows:
			*room = st.ScrollbarWidth
			changed = true
		case *room != 0 && !overflows:
			*room = 0
			changed = true
		}
	}
	fit(st.Overflow.Height == style.OverflowAuto, &n.gutter.w, r.b > port.h+slack)
	fit(st.Overflow.Width == style.OverflowAuto, &n.gutter.h, r.r > port.w+slack)
	return changed
}

func (t *tree) hidden(n *node) output {
	n.cache.clear()
	n.unrounded = placed{}
	n.hoisted = nil
	in := input{
		run:             performHiddenLayout,
		sizing:          inherentSize,
		axis:            axisBoth,
		known:           noSize,
		knownIsDefinite: bools{true, true},
		parent:          noSize,
		avail:           maxContentSpaces,
	}
	for _, c := range n.kids {
		t.computeChild(c, in)
	}
	return sizeOutput(size{})
}

// root lays out the root node in the viewport, Taffy's compute_root_layout.
func (t *tree) root(n *node, avail spaces) {
	st := n.st
	icb := avail.values()
	var layout placed
	var candidates []oofCandidate
	if outOfFlow(st.Position) && icb.both() {
		c := oofCandidate{
			node:     n,
			position: st.Position,
			x:        staticPosition{area: span{0, 0}, align: alignFromEdge(edgeStart)},
			y:        staticFromStart(0),
		}
		lb := t.layoutOOFBox(c, icb, point{}, style.Size[style.Overflow]{}, nil)
		layout, candidates = lb.layout, lb.surfaced
	} else {
		layout, candidates = t.inFlowRoot(n, avail)
	}
	if len(candidates) == 0 {
		return
	}
	inset := layout.border.add(rect{r: layout.scrollbar.w, b: layout.scrollbar.h})
	paddingBox := layout.size.sub(inset.sum())
	areaW, areaX := paddingBox.w, inset.l
	if some(icb.w) {
		areaW, areaX = fmax(icb.w-layout.scrollbar.w, 0), -layout.location.x
	}
	areaH, areaY := paddingBox.h, inset.t
	if some(icb.h) {
		areaH, areaY = fmax(icb.h-layout.scrollbar.h, 0), -layout.location.y
	}
	hoisted, _, _ := t.performOOF(n, candidates, size{areaW, areaH},
		point{areaX, areaY}, claims{true, true}, st.Overflow, [2]bool{})
	n.hoisted = append(n.hoisted, hoisted...)
}

func (t *tree) inFlowRoot(n *node, avail spaces) (placed, []oofCandidate) {
	st := n.st
	known := noSize
	margin := edges(st.Margin, avail.w.value())
	rootAvail := avail.mSub(margin.sum()).mMax(size{})
	parent := avail.values()
	if n.context() == contextBlock || n.isBlockLeaf() {
		ratio := aspectRatio(st)
		padding := edges(st.Padding, parent.w)
		pb := padding.add(borders(st)).sum()
		adj := boxSizingAdjustment(st, pb)
		minSize := resolveSize(st.MinSize, parent).withAspectRatio(ratio).mAdd(adj)
		maxSize := resolveMax(st.MaxSize, parent).withAspectRatio(ratio).mAdd(adj)
		clamped := resolveSize(st.Size, parent).withAspectRatio(ratio).mAdd(adj).mClamp(minSize, maxSize)
		minMax := size{definiteMinMax(minSize.w, maxSize.w), definiteMinMax(minSize.h, maxSize.h)}
		known = known.or(minMax).or(clamped).or(size{rootAvail.w.value(), none}).mMax(pb)
	}
	out := t.computeChild(n, input{
		run:             performLayout,
		sizing:          inherentSize,
		axis:            axisBoth,
		known:           known,
		knownIsDefinite: bools{true, true},
		parent:          parent,
		avail:           rootAvail,
	})
	padding := edges(st.Padding, parent.w)
	gutter := scrollbarGutter(st)
	location := point{margin.l, margin.t}
	if st.Position == style.Relative {
		inset := optionalEdges(st.Inset, none)
		inset.l, inset.r = resolve(st.Inset.Left, parent.w), resolve(st.Inset.Right, parent.w)
		inset.t, inset.b = resolve(st.Inset.Top, parent.h), resolve(st.Inset.Bottom, parent.h)
		location.x += or(inset.l, or(-inset.r, 0))
		location.y += or(inset.t, or(-inset.b, 0))
	}
	layout := placed{
		location:  location,
		size:      out.size,
		overflow:  out.overflow,
		scrollbar: size{gutter.x, gutter.y},
		padding:   padding,
		border:    borders(st),
		margin:    margin,
	}
	n.unrounded = layout
	return layout, out.oof
}

func definiteMinMax(lo, hi float32) float32 {
	if some(lo) && some(hi) && hi <= lo {
		return lo
	}
	return none
}

// round lays every box on whole points, Taffy's round_layout: edges are rounded where they
// fall in the root's space, so neighbours never gap or overlap.
func roundLayout(n *node, cumulative point) { roundIn(n, cumulative, false) }

// roundIn is roundLayout under a display: none ancestor when hidden, as keepIn.
func roundIn(n *node, cumulative point, hidden bool) {
	u := n.unrounded
	x, y := cumulative.x+u.location.x, cumulative.y+u.location.y
	l := u
	l.location = point{round(x) - round(cumulative.x), round(y) - round(cumulative.y)}
	l.size = size{round(x+u.size.w) - round(x), round(y+u.size.h) - round(y)}
	l.scrollbar = size{round(u.scrollbar.w + n.gutter.w), round(u.scrollbar.h + n.gutter.h)}
	l.border = roundEdges(u.border, x, y, u.size)
	l.padding = roundEdges(u.padding, x, y, u.size)
	r := u.overflow.rect
	l.overflow.rect = rect{
		l: roundEdge(x, r.l), r: roundEdge(x, r.r),
		t: roundEdge(y, r.t), b: roundEdge(y, r.b),
	}
	n.final = l
	n.absolute = point{round(x), round(y)}
	hidden = hidden || n.st.Display == style.DisplayNone
	for _, c := range n.kids {
		if hidden || !outOfFlow(c.st.Position) || c.st.Display == style.DisplayNone {
			roundIn(c, point{x, y}, hidden)
		}
	}
	for _, c := range n.hoisted {
		roundIn(c, point{x, y}, hidden)
	}
}

func roundEdges(e rect, x, y float32, s size) rect {
	return rect{
		l: round(x+e.l) - round(x),
		r: round(x+s.w) - round(x+s.w-e.r),
		t: round(y+e.t) - round(y),
		b: round(y+s.h) - round(y+s.h-e.b),
	}
}

func roundEdge(offset, edge float32) float32 {
	r := round(offset+edge) - round(offset)
	if r == 0 {
		return float32(math.Copysign(0, float64(edge)))
	}
	return r
}
