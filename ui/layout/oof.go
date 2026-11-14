package layout

import (
	"math"

	"github.com/lantervale/ember/ui/internal/style"
)

// claims is which out-of-flow boxes a containing block lays out.
type claims struct{ absolute, fixed bool }

func (c claims) take(p style.Position) bool {
	switch p {
	case style.Absolute:
		return c.absolute
	case style.FixedPosition:
		return c.fixed
	case style.Static, style.Relative, style.Sticky:
	}
	return false
}

func containingBlockClaims(st *style.Computed) claims {
	return claims{absolute: st.Position != style.Static}
}

func alignInLine(container span, s float32, e staticEdge) float32 {
	switch e {
	case edgeEnd:
		return container.end - s
	case edgeCenter:
		return (container.start + container.end - s) / 2
	case edgeStart:
	}
	return container.start
}

type oofAlignment uint8

const (
	oofNormal oofAlignment = iota
	oofStretch
	oofStart
	oofCenter
	oofEnd
)

func oofAlignmentOf(a style.Align) oofAlignment {
	switch a {
	case style.AlignNormal, style.AlignAuto:
		return oofNormal
	case style.AlignStretch:
		return oofStretch
	case style.AlignCenter:
		return oofCenter
	case style.AlignEnd, style.AlignFlexEnd:
		return oofEnd
	case style.AlignStart, style.AlignFlexStart, style.AlignBaseline, style.AlignSpaceBetween, style.AlignSpaceAround,
		style.AlignSpaceEvenly: // at the start
	}
	return oofStart
}

func (a oofAlignment) physicalEdge(reversed bool) staticEdge {
	switch {
	case a == oofCenter:
		return edgeCenter
	case a == oofEnd && !reversed, a != oofEnd && reversed:
		return edgeEnd
	}
	return edgeStart
}

// oofAxis is one axis of an absolutely positioned box, CSS Positioned Layout §4.
type oofAxis struct {
	cbSize     float32
	inset      span // none where auto
	margin     span // none where auto
	static     staticPosition
	alignment  oofAlignment
	safe       bool
	unsafe     bool
	isInline   bool
	reversed   bool
	scrollAxis bool
}

func (a *oofAxis) startEdge() staticEdge {
	if a.reversed {
		return edgeEnd
	}
	return edgeStart
}

func (a *oofAxis) endEdge() staticEdge {
	if a.reversed {
		return edgeStart
	}
	return edgeEnd
}

func (a *oofAxis) imcb() span {
	var r span
	var weaker staticEdge
	switch {
	case some(a.inset.start) && some(a.inset.end):
		r, weaker = span{a.inset.start, a.cbSize - a.inset.end}, a.endEdge()
	case some(a.inset.start):
		r, weaker = span{a.inset.start, a.cbSize}, edgeEnd
	case some(a.inset.end):
		r, weaker = span{0, a.cbSize - a.inset.end}, edgeStart
	default:
		sp := a.static
		switch sp.align.keyword {
		case edgeStart:
			r = span{sp.area.start, a.cbSize}
		case edgeEnd:
			r = span{0, sp.area.end}
		case edgeCenter:
			center := (sp.area.start + sp.area.end) / 2
			startDistance, endDistance := center, a.cbSize-center
			if abs(startDistance) <= abs(endDistance) {
				r = span{0, 2 * startDistance}
			} else {
				r = span{a.cbSize - 2*endDistance, a.cbSize}
			}
		}
		weaker = a.endEdge()
	}
	if r.end < r.start {
		if weaker == edgeStart {
			r.start = r.end
		} else {
			r.end = r.start
		}
	}
	return r
}

func abs(v float32) float32 { return float32(math.Abs(float64(v))) }

func (a *oofAxis) stretchFitSize() float32 {
	r := a.imcb()
	return fmax(r.end-r.start-or(a.margin.start, 0)-or(a.margin.end, 0), 0)
}

func (a *oofAxis) isStretchSized() bool {
	if !some(a.inset.start) || !some(a.inset.end) {
		return false
	}
	return a.alignment == oofStretch || a.alignment == oofNormal
}

func (a *oofAxis) resolveMargins(s float32) span {
	both := some(a.inset.start) && some(a.inset.end)
	r := a.imcb()
	free := r.end - r.start - s
	switch {
	case some(a.margin.start) && some(a.margin.end):
		return a.margin
	case !both:
		return span{or(a.margin.start, 0), or(a.margin.end, 0)}
	case some(a.margin.start):
		return span{a.margin.start, free - a.margin.start}
	case some(a.margin.end):
		return span{free - a.margin.end, a.margin.end}
	}
	if free < 0 && a.isInline {
		if a.reversed {
			return span{free, 0}
		}
		return span{0, free}
	}
	return span{free / 2, free / 2}
}

func (a *oofAxis) position(s float32, margin span) float32 {
	r := a.imcb()
	marginBox := s + margin.start + margin.end
	switch {
	case !some(a.inset.start) && !some(a.inset.end):
		sp := a.static
		var off float32
		if sp.align.safe && marginBox > r.end-r.start {
			off = alignInLine(r, marginBox, a.startEdge())
		} else {
			off = alignInLine(sp.area, marginBox, sp.align.keyword)
		}
		return off + margin.start
	case !some(a.inset.end):
		return r.start + margin.start
	case !some(a.inset.start):
		return r.end - s - margin.end
	}
	if !some(a.margin.start) || !some(a.margin.end) {
		return r.start + margin.start
	}
	size := r.end - r.start
	edge := a.alignment.physicalEdge(a.reversed)
	var start float32
	switch {
	case a.unsafe:
		start = alignInLine(r, marginBox, edge)
	case a.safe:
		if marginBox > size {
			edge = a.startEdge()
		}
		start = alignInLine(r, marginBox, edge)
	default:
		if marginBox <= size || a.alignment == oofNormal {
			start = alignInLine(r, marginBox, edge)
			break
		}
		limit := span{fmin(r.start, 0), fmax(r.end, a.cbSize)}
		if a.scrollAxis {
			if a.reversed {
				limit.start = float32(math.Inf(-1))
			} else {
				limit.end = float32(math.Inf(1))
			}
		}
		if marginBox <= limit.end-limit.start {
			start = fmax(fmin(alignInLine(r, marginBox, edge), limit.end-marginBox), limit.start)
		} else {
			start = alignInLine(limit, marginBox, a.startEdge())
		}
	}
	return start + margin.start
}

// oofLayout lays out the out-of-flow boxes a node's layout surfaced, when the node is
// their containing block, Taffy's compute_oof_layout.
func (t *tree) oofLayout(n *node, out *output) {
	if out.area == nil {
		return
	}
	if len(out.oof) == 0 {
		n.hoisted = nil
		return
	}
	area := *out.area
	hoisted, unclaimed, r := t.performOOF(n, out.oof, area.size,
		area.offset, containingBlockClaims(n.st), n.st.Overflow, [2]bool{})
	scroll := isScrollContainer(n.st)
	r = finalizeOverflow(r, area.size, [2]bool{}, scroll).rect
	n.hoisted = hoisted
	out.oof = unclaimed
	out.overflow = finalizeOverflow(out.overflow.rect.union(r), area.size, out.overflow.reversed, scroll)
}

type oofBoxLayout struct {
	layout   placed
	surfaced []oofCandidate
	overflow style.Size[style.Overflow]
}

// layoutOOFBox sizes and places one absolutely positioned box in its containing block.
func (t *tree) layoutOOFBox(c oofCandidate, area size, offset point,
	containerOverflow style.Size[style.Overflow], gridOwner *node) oofBoxLayout {
	n := c.node
	st := n.st
	if gridOwner != nil && gridOwner.grid != nil {
		r := gridOwner.grid.absoluteArea(st, rect{l: offset.x, r: offset.x + area.w, t: offset.y, b: offset.y + area.h})
		area, offset = size{r.r - r.l, r.b - r.t}, point{r.l, r.t}
	}
	ratio := aspectRatio(st)
	margin := optionalEdges(st.Margin, area.w)
	padding := edges(st.Padding, area.w)
	border := borders(st)
	pbSum := padding.add(border).sum()
	adj := boxSizingAdjustment(st, pbSum)
	inset := rect{
		l: resolve(st.Inset.Left, area.w), r: resolve(st.Inset.Right, area.w),
		t: resolve(st.Inset.Top, area.h), b: resolve(st.Inset.Bottom, area.h),
	}
	styleSize := resolveSize(st.Size, area).withAspectRatio(ratio).mAdd(adj)
	minSize := resolveSize(st.MinSize, area).withAspectRatio(ratio).mAdd(adj).or(pbSum).mMax(pbSum)
	maxSize := resolveMax(st.MaxSize, area).withAspectRatio(ratio).mAdd(adj)
	known := styleSize.mClamp(minSize, maxSize)

	justifySelf, alignSelf := st.JustifySelf, st.AlignSelf
	h := oofAxis{
		cbSize: area.w, inset: span{inset.l, inset.r}, margin: span{margin.l, margin.r},
		static:    staticPosition{area: span{c.x.area.start - offset.x, c.x.area.end - offset.x}, align: c.x.align},
		alignment: oofAlignmentOf(justifySelf), isInline: true,
		scrollAxis: scrollContainer(containerOverflow.Width),
	}
	v := oofAxis{
		cbSize: area.h, inset: span{inset.t, inset.b}, margin: span{margin.t, margin.b},
		static:     staticPosition{area: span{c.y.area.start - offset.y, c.y.area.end - offset.y}, align: c.y.align},
		alignment:  oofAlignmentOf(alignSelf),
		scrollAxis: scrollContainer(containerOverflow.Height),
	}
	stretchFit := size{h.stretchFitSize(), v.stretchFitSize()}
	avail := stretchFit.mClamp(minSize, maxSize)
	availSpace := spaces{def(avail.w), def(avail.h)}

	if isSizingKeyword(st.Size.Width) || isSizingKeyword(st.Size.Height) {
		t.resolveAbsoluteSizingKeywords(n, &known, st.Size, area, inset, margin, minSize, maxSize, contentSize)
		known = known.withAspectRatio(ratio).mClamp(minSize, maxSize)
	}
	stretchW := !some(known.w) && h.isStretchSized()
	stretchH := !some(known.h) && v.isStretchSized()
	stretchRatio := ratio
	if stretchW && stretchH && h.alignment == oofStretch && v.alignment == oofStretch {
		stretchRatio = none
	}
	if stretchW {
		known.w = stretchFit.w
		known = known.withAspectRatio(stretchRatio).mClamp(minSize, maxSize)
	}
	if !some(known.h) && stretchH {
		known.h = stretchFit.h
		known = known.withAspectRatio(stretchRatio).mClamp(minSize, maxSize)
	}
	final := known
	if !some(final.w) && !some(final.h) {
		final.w = mClamp(t.measureChildSize(n, final, area, availSpace, contentSize, horizontal, [2]bool{}),
			minSize.w, maxSize.w)
	}
	if !final.both() {
		measured := t.measureChildSizeBoth(n, final, area, availSpace, contentSize, [2]bool{})
		final = final.or(measured)
	}
	final = final.mClamp(minSize, maxSize)

	out := t.computeChild(n, input{
		run:             performLayout,
		sizing:          contentSize,
		axis:            axisBoth,
		known:           final,
		knownIsDefinite: bools{true, true},
		parent:          area,
		avail:           availSpace,
	})
	hm := h.resolveMargins(final.w)
	vm := v.resolveMargins(final.h)
	gutter := scrollbarGutter(st)
	layout := placed{
		order:     c.order,
		size:      final,
		overflow:  out.overflow,
		scrollbar: size{gutter.x, gutter.y},
		location:  point{h.position(final.w, hm) + offset.x, v.position(final.h, vm) + offset.y},
		padding:   padding,
		border:    border,
		margin:    rect{l: hm.start, r: hm.end, t: vm.start, b: vm.end},
	}
	n.unrounded = layout
	return oofBoxLayout{layout: layout, surfaced: out.oof, overflow: st.Overflow}
}

// performOOF lays out the candidates this containing block claims, and the ones they
// surface in turn; it returns the boxes it hoisted, the ones it left for an ancestor, and
// their scrollable overflow.
func (t *tree) performOOF(n *node, candidates []oofCandidate, area size, offset point, cl claims,
	containerOverflow style.Size[style.Overflow], reversed [2]bool) ([]*node, []oofCandidate, rect) {
	var overflow rect
	var hoisted []*node
	var unclaimed []oofCandidate
	if len(candidates) == 0 {
		return nil, nil, overflow
	}
	container := overflowContainer{
		scroll:   scrollContainer(containerOverflow.Width) || scrollContainer(containerOverflow.Height),
		size:     area,
		reversed: reversed,
	}
	var work []oofCandidate
	for _, c := range candidates {
		if c.area != nil || cl.take(c.position) {
			work = append(work, c)
		} else {
			unclaimed = append(unclaimed, c)
		}
	}
	gridOwner := n
	if n.grid == nil {
		gridOwner = nil
	}
	for i := 0; i < len(work); i++ {
		c := work[i]
		hoisted = append(hoisted, c.node)
		cbSize, cbAt, owner := area, offset, gridOwner
		if c.area != nil {
			cbSize, cbAt, owner = c.area.size, c.area.offset, nil
		}
		lb := t.layoutOOFBox(c, cbSize, cbAt, containerOverflow, owner)
		for _, s := range translateCandidates(lb.surfaced, lb.layout.location) {
			if cl.take(s.position) {
				work = append(work, s)
			} else {
				unclaimed = append(unclaimed, s)
			}
		}
		loc := point{lb.layout.location.x - offset.x, lb.layout.location.y - offset.y}
		overflow = overflow.union(overflowContribution(loc, lb.layout.size,
			lb.layout.overflow.rect, lb.layout.border, lb.overflow, container, rect{}))
	}
	return hoisted, unclaimed, overflow
}
