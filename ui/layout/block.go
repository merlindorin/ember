package layout

import "github.com/lantervale/ember/ui/internal/style"

// blockContext marks a block laid out inside its parent's block formatting context, so its
// margins may collapse with its children's; nil is the root of a formatting context.
type blockContext struct{}

type blockItem struct {
	node                   *node
	order                  uint32
	inSameBFC              bool
	sizeStyle              style.Size[style.Length]
	size, minSize, maxSize size
	overflow               style.Size[style.Overflow]
	scrollbarWidth         float32
	position               style.Position
	alignSelf              style.Align
	justifySelf            style.Align
	padding, border        rect
	pbSum                  size
	staticPosition         point
	collapsesThrough       bool
	final                  *placed
	oof                    []oofCandidate
}

func blockEstablishesBFC(st *style.Computed) bool {
	return isScrollContainer(st) || st.AlignContent != style.AlignNormal
}

// block lays out a block container whose children are blocks, Taffy's compute_block_layout.
func (t *tree) block(n *node, in input, ctx *blockContext) output {
	st := n.st
	ratio := aspectRatio(st)
	padding := edges(st.Padding, in.parent.w)
	border := borders(st)
	pbSize := padding.add(border).sum()
	adj := boxSizingAdjustment(st, pbSize)
	minSize := resolveSize(st.MinSize, in.parent).withAspectRatio(ratio).mAdd(adj)
	maxSize := resolveMax(st.MaxSize, in.parent).withAspectRatio(ratio).mAdd(adj)
	clamped := noSize
	if in.sizing == inherentSize {
		clamped = resolveSize(st.Size, in.parent).withAspectRatio(ratio).mAdd(adj).mClamp(minSize, maxSize)
	}
	minMax := size{definiteMinMax(minSize.w, maxSize.w), definiteMinMax(minSize.h, maxSize.h)}
	styled := in.known.or(minMax).or(clamped).mMax(pbSize)
	if in.run == computeSize {
		if styled.both() {
			return sizeOutput(styled)
		}
		if in.axis == axisHorizontal && some(styled.w) {
			return sizeOutput(size{styled.w, 0})
		}
	}
	sub := in
	sub.known = styled
	if blockEstablishesBFC(st) {
		ctx = nil
	}
	return t.blockInner(n, sub, ctx)
}

//nolint:cyclop,funlen,gocognit,gocyclo,nestif // Taffy's block compute_inner, kept in its shape
func (t *tree) blockInner(n *node, in input, ctx *blockContext) output {
	st := n.st
	ratio := aspectRatio(st)
	padding := edges(st.Padding, in.parent.w)
	border := borders(st)
	g := n.ownGutter()
	gutter := rect{r: g.x, b: g.y}
	pb := padding.add(border)
	pbSize := pb.sum()
	inset := pb.add(gutter)
	adj := boxSizingAdjustment(st, pbSize)
	sz := resolveSize(st.Size, in.parent).withAspectRatio(ratio).mAdd(adj)
	minSize := resolveSize(st.MinSize, in.parent).withAspectRatio(ratio).mAdd(adj)
	maxSize := resolveMax(st.MaxSize, in.parent).withAspectRatio(ratio).mAdd(adj)
	derived := in.known.withAspectRatio(ratio).mClamp(minSize, maxSize)
	known := in.known.or(derived)
	percentBasis := size{known.w, none}
	if in.knownIsDefinite.h {
		percentBasis.h = known.h
	}
	contentBoxSize := percentBasis.mSub(inset.sum())
	newBFC := blockEstablishesBFC(st)
	oof := outOfFlow(st.Position)
	adjoin := [2]bool{
		in.collapsible[0] && !newBFC && !oof && padding.t == 0 && border.t == 0,
		in.collapsible[1] && !newBFC && !oof && padding.b == 0 && border.b == 0,
	}
	ownCollapse := [2]bool{adjoin[0], adjoin[1] && !some(sz.h)}
	preventsCollapse := st.Display != style.Block || ctx == nil || newBFC || oof ||
		padding.t > 0 || padding.b > 0 || border.t > 0 || border.b > 0

	items := t.blockItems(n, contentBoxSize)
	outerW := known.w
	if !some(outerW) {
		availW := in.avail.w.mSub(inset.horizontal())
		intrinsic := t.blockContentWidth(items, availW) + inset.horizontal()
		outerW = mMax(mClamp(intrinsic, minSize.w, maxSize.w), pbSize.w)
	}
	if in.run == computeSize && some(known.h) {
		return sizeOutput(size{outerW, known.h})
	}
	if in.run == computeSize && in.axis == axisHorizontal {
		return sizeOutput(size{outerW, 0})
	}
	percentH := percentBasis.h
	if !some(percentH) && in.sizing == inherentSize {
		percentH = mMax(sz.h, minSize.h)
	}
	percentW := or(in.parent.w, outerW)
	rPadding := edges(st.Padding, percentW)
	rBorder := border
	rInset := rPadding.add(rBorder).add(gutter)

	res := t.blockChildren(in.run, items, outerW, percentH, rInset,
		rBorder, rPadding, st, ownCollapse, isScrollContainer(st))
	outerH := mMax(or(known.h, mClamp(res.height, minSize.h, maxSize.h)), pbSize.h)
	final := size{outerW, outerH}
	constrainedByMin := some(minSize.h) && minSize.h > 0 && minSize.h >= outerH
	ownBottomCollapse := ownCollapse[1] && !constrainedByMin
	alignContent := st.AlignContent
	if alignContent == style.AlignNormal && centresContent(n) {
		alignContent = style.AlignCenter
	}
	if alignContent != style.AlignNormal {
		innerH := outerH - rInset.vertical()
		contentH := res.height - rInset.vertical()
		free := innerH - contentH
		k := applyAlignmentFallback(free, 1, align(alignContent))
		offset := computeAlignmentOffset(free, 1, 0, k, false, true)
		if some(res.firstBaseline) {
			res.firstBaseline += offset
		}
		anyInFlow := false
		for _, it := range items {
			if it.final != nil {
				it.final.location.y += offset
				anyInFlow = true
			}
			if outOfFlow(it.position) {
				it.staticPosition.y += offset
			}
		}
		if anyInFlow {
			res.overflow = rect{}
			container := overflowContainer{
				scroll:  isScrollContainer(st),
				size:    final.sub(rBorder.sum()).sub(gutter.sum()),
				padding: rPadding,
			}
			for _, it := range items {
				if it.final == nil {
					continue
				}
				l := it.final
				loc := point{l.location.x - rBorder.l - gutter.l, l.location.y - rBorder.t}
				res.overflow = res.overflow.union(overflowContribution(loc, l.size,
					l.overflow.rect, l.border, it.overflow, container, negativeEdges(l.margin)))
			}
		}
	}
	allCollapse := true
	for _, it := range items {
		if !outOfFlow(it.position) && !it.collapsesThrough {
			allCollapse = false
		}
	}
	out := output{
		size:             final,
		firstBaseline:    res.firstBaseline,
		lastBaseline:     none,
		collapsesThrough: !preventsCollapse && outerH == 0 && allCollapse,
	}
	if ownCollapse[0] {
		out.topMargin = res.firstTop
	} else {
		out.topMargin = marginFrom(resolveOrZero(st.Margin.Top, in.parent.w))
	}
	if ownBottomCollapse {
		out.bottomMargin = res.lastBottom
	} else {
		out.bottomMargin = marginFrom(resolveOrZero(st.Margin.Bottom, in.parent.w))
	}
	if in.run == computeSize {
		return out
	}
	for _, it := range items {
		if it.final != nil {
			it.node.unrounded = *it.final
		}
	}
	var candidates []oofCandidate
	for _, it := range items {
		if outOfFlow(it.position) {
			inlineArea := span{rInset.l, outerW - rInset.r}
			blockArea := span{it.staticPosition.y, it.staticPosition.y}
			justify := it.justifySelf
			if justify == style.AlignAuto {
				justify = st.JustifyItems
			}
			self := it.alignSelf
			if self == style.AlignAuto {
				self = style.AlignNormal
			}
			candidates = append(candidates, oofCandidate{
				node:     it.node,
				order:    it.order,
				position: it.position,
				x:        staticFromAlignment(align(justify), inlineArea),
				y:        staticFromAlignment(align(self), blockArea),
			})
		} else if len(it.oof) > 0 && it.final != nil {
			candidates = append(candidates, translateCandidates(it.oof, it.final.location)...)
		}
	}
	areaInset := rBorder.add(gutter)
	out.oof = candidates
	out.area = &oofArea{size: final.sub(areaInset.sum()), offset: point{areaInset.l, areaInset.t}}
	if isScrollContainer(st) {
		res.overflow = res.overflow.union(rect{r: rPadding.r, b: rPadding.b})
	}
	out.overflow = finalizeOverflow(res.overflow,
		final.sub(rBorder.sum()).sub(gutter.sum()), [2]bool{}, isScrollContainer(st))
	for i, k := range n.kids {
		if k.st.Display == style.DisplayNone {
			k.unrounded = placed{order: uint32(i)}
			t.performChildLayout(k, noSize, noSize, maxContentSpaces, inherentSize)
		}
	}
	return out
}

func negativeEdges(r rect) rect {
	return rect{l: fmin(r.l, 0), r: fmin(r.r, 0), t: fmin(r.t, 0), b: fmin(r.b, 0)}
}

func staticFromAlignment(a alignment, area span) staticPosition {
	edge := func(k style.Align) staticEdge {
		switch k {
		case style.AlignCenter:
			return edgeCenter
		case style.AlignEnd, style.AlignFlexEnd:
			return edgeEnd
		// they sit at the start
		case style.AlignNormal, style.AlignAuto, style.AlignStretch, style.AlignStart, style.AlignFlexStart,
			style.AlignBaseline, style.AlignSpaceBetween, style.AlignSpaceAround, style.AlignSpaceEvenly:
		}
		return edgeStart
	}
	fallback := a.keyword
	if a.safe {
		fallback = style.AlignStart
	}
	return staticPosition{area: area,
		align: staticAlign{keyword: edge(a.keyword), safe: a.safe, unsafe: a.unsafe, fallback: edge(fallback)}}
}

func (t *tree) blockItems(n *node, inner size) []*blockItem {
	var items []*blockItem
	order := uint32(0)
	for _, k := range n.kids {
		st := k.st
		if st.Display == style.DisplayNone {
			continue
		}
		ratio := aspectRatio(st)
		padding := edges(st.Padding, inner.w)
		border := borders(st)
		pbSum := padding.add(border).sum()
		adj := boxSizingAdjustment(st, pbSum)
		same := st.Display == style.Block && !outOfFlow(st.Position) &&
			!isScrollContainer(st) && st.AlignContent == style.AlignNormal
		items = append(items, &blockItem{
			node:           k,
			order:          order,
			inSameBFC:      same,
			sizeStyle:      st.Size,
			size:           resolveSize(st.Size, inner).withAspectRatio(ratio).mAdd(adj),
			minSize:        resolveSize(st.MinSize, inner).withAspectRatio(ratio).mAdd(adj),
			maxSize:        resolveMax(st.MaxSize, inner).withAspectRatio(ratio).mAdd(adj),
			overflow:       st.Overflow,
			scrollbarWidth: st.ScrollbarWidth,
			position:       st.Position,
			alignSelf:      st.AlignSelf,
			justifySelf:    st.JustifySelf,
			padding:        padding,
			border:         border,
			pbSum:          pbSum,
		})
		order++
	}
	return items
}

func relativeOffset(it *blockItem, parent size) point {
	if it.position != style.Relative {
		return point{}
	}
	st := it.node.st
	l, r := resolve(st.Inset.Left, parent.w), resolve(st.Inset.Right, parent.w)
	tp, b := resolve(st.Inset.Top, parent.h), resolve(st.Inset.Bottom, parent.h)
	return point{or(l, or(-r, 0)), or(tp, or(-b, 0))}
}

func collapsibleIf(same bool) [2]bool { return [2]bool{same, same} }

func (t *tree) blockContentWidth(items []*blockItem, availW space) float32 {
	maxW := float32(0)
	for _, it := range items {
		if outOfFlow(it.position) {
			continue
		}
		known := it.size.mClamp(it.minSize, it.maxSize)
		marginX := edges(it.node.st.Margin, availW.value()).horizontal()
		w := known.w
		if !some(w) {
			itemAvail := availW.mSub(marginX)
			if k, s := resolveSizingKeyword(it.sizeStyle.Width, none, none); k == keywordMeasure {
				itemAvail = s
			}
			w = t.measureChildSize(it.node, known, noSize, spaces{itemAvail, minContentSpace},
				inherentSize, horizontal, collapsibleIf(it.inSameBFC))
		}
		maxW = fmax(maxW, fmax(w, it.pbSum.w)+marginX)
	}
	return maxW
}

type blockResult struct {
	overflow      rect
	height        float32
	firstTop      marginSet
	lastBottom    marginSet
	firstBaseline float32
}

//nolint:cyclop,funlen,gocognit,gocyclo // Taffy's perform_final_layout_on_in_flow_children, kept in its shape
func (t *tree) blockChildren(run runMode, items []*blockItem, outerW, percentH float32, rInset, rBorder, rPadding rect,
	st *style.Computed, ownCollapse [2]bool, scroll bool) blockResult {
	innerW := fmax(outerW-rInset.horizontal(), 0)
	container := overflowContainer{scroll: scroll, size: size{innerW + rPadding.horizontal(), 0}, padding: rPadding}
	percentH = mSub(percentH, rInset.vertical())
	parent := size{innerW, percentH}
	var overflow rect
	committed := rInset.t
	yForAbsolute := rInset.t
	var firstTop, active marginSet
	collapsingWithFirst := true
	firstBaseline := none
	for _, it := range items {
		if outOfFlow(it.position) {
			it.staticPosition = point{rInset.l, yForAbsolute}
			continue
		}
		cst := it.node.st
		margin := optionalEdges(cst.Margin, innerW)
		nonAuto := rect{or(margin.l, 0), or(margin.r, 0), or(margin.t, 0), or(margin.b, 0)}
		var gutter size
		if it.overflow.Height == style.Scroll {
			gutter.w = it.scrollbarWidth
		}
		if it.overflow.Width == style.Scroll {
			gutter.h = it.scrollbarWidth
		}
		yMarginOffset := float32(0)
		stretchW := fmax(innerW-nonAuto.horizontal(), 0)
		avoid := point{}
		if !it.inSameBFC {
			if !collapsingWithFirst || !ownCollapse[0] {
				yMarginOffset = active.collapseWithMargin(nonAuto.t).resolve()
			}
			avoid = point{rInset.l, committed + yMarginOffset}
		}
		justify := it.justifySelf
		if justify == style.AlignAuto {
			justify = st.JustifyItems
		}
		nonStretch := justify != style.AlignNormal && justify != style.AlignStretch
		widthStyle := it.sizeStyle.Width
		if (nonStretch || it.node.role == roleTable) && widthStyle.Kind == style.Auto {
			widthStyle = style.Length{Kind: style.FitContent}
		}
		keywordW := none
		if k, s := resolveSizingKeyword(widthStyle, stretchW, innerW); k == keywordMeasure {
			keywordW = t.measureChildSize(it.node, noSize, parent, spaces{s, maxContentSpace},
				inherentSize, horizontal, collapsibleIf(it.inSameBFC))
		}
		known := it.size
		known.w = mClamp(or(or(known.w, keywordW), stretchW), it.minSize.w, it.maxSize.w)
		known = known.mClamp(it.minSize, it.maxSize)
		in := input{
			run:             run,
			sizing:          inherentSize,
			axis:            axisBoth,
			known:           known,
			knownIsDefinite: bools{true, true},
			parent:          parent,
			avail:           spaces{def(stretchW), maxContentSpace},
			collapsible:     collapsibleIf(it.inSameBFC),
		}
		var lo output
		if it.inSameBFC {
			lo = t.computeChildIn(it.node, in, &blockContext{})
		} else {
			lo = t.computeChild(it.node, in)
		}
		it.oof = lo.oof
		finalSize := lo.size
		topSet := lo.topMargin.collapseWithMargin(or(margin.t, 0))
		bottomSet := lo.bottomMargin.collapseWithMargin(or(margin.b, 0))
		freeX := fmax(0, stretchW-finalSize.w)
		autoCount := 0
		if !some(margin.l) {
			autoCount++
		}
		if !some(margin.r) {
			autoCount++
		}
		autoSize := float32(0)
		if autoCount > 0 {
			autoSize = freeX / float32(autoCount)
		}
		resolved := rect{l: or(margin.l, autoSize), r: or(margin.r, autoSize), t: or(margin.t, 0), b: or(margin.b, 0)}
		insetOffset := relativeOffset(it, parent)
		if it.inSameBFC && (!collapsingWithFirst || !ownCollapse[0]) {
			yMarginOffset = active.collapseWithSet(topSet).resolve()
		}
		it.collapsesThrough = lo.collapsesThrough
		if it.inSameBFC {
			it.staticPosition = point{rInset.l, committed + active.resolve()}
		} else {
			it.staticPosition = avoid
		}
		var loc point
		if it.inSameBFC {
			loc = point{rInset.l + insetOffset.x + resolved.l, committed + yMarginOffset + insetOffset.y}
		} else {
			loc = point{avoid.x + resolved.l + insetOffset.x, avoid.y + insetOffset.y}
		}
		hasAutoX := !some(margin.l) || !some(margin.r)
		outerItemW := finalSize.w + resolved.horizontal()
		if nonStretch && !hasAutoX {
			free := innerW - outerItemW
			off := float32(0)
			switch justify {
			case style.AlignCenter:
				off = free / 2
			case style.AlignEnd, style.AlignFlexEnd:
				off = free
			// they stay at the start
			case style.AlignNormal, style.AlignAuto, style.AlignStretch, style.AlignStart, style.AlignFlexStart,
				style.AlignBaseline, style.AlignSpaceBetween, style.AlignSpaceAround, style.AlignSpaceEvenly:
			}
			loc.x = rInset.l + resolved.l + insetOffset.x + off
		}
		if !some(firstBaseline) {
			if b := childFirstBaseline(lo.firstBaseline, finalSize.h, it.overflow.Height); some(b) {
				firstBaseline = loc.y - insetOffset.y + b
			}
		}
		it.final = &placed{
			order:     it.order,
			size:      finalSize,
			overflow:  lo.overflow,
			scrollbar: gutter,
			location:  loc,
			padding:   it.padding,
			border:    it.border,
			margin:    resolved,
		}
		contributionLoc := point{loc.x - rInset.l + rPadding.l, loc.y - rBorder.t}
		overflow = overflow.union(overflowContribution(contributionLoc, finalSize,
			lo.overflow.rect, it.border, it.overflow, container, negativeEdges(resolved)))
		if collapsingWithFirst {
			if it.collapsesThrough {
				firstTop = firstTop.collapseWithSet(topSet).collapseWithSet(bottomSet)
			} else {
				firstTop = firstTop.collapseWithSet(topSet)
				collapsingWithFirst = false
			}
		}
		if it.collapsesThrough {
			active = active.collapseWithSet(topSet).collapseWithSet(bottomSet)
			yForAbsolute = committed + finalSize.h + yMarginOffset
		} else {
			committed = loc.y - insetOffset.y + finalSize.h
			active = bottomSet
			yForAbsolute = committed + active.resolve()
		}
	}
	lastBottom := active
	bottomOffset := lastBottom.resolve()
	if ownCollapse[1] {
		bottomOffset = 0
	}
	committed += rInset.b + bottomOffset
	return blockResult{
		overflow:      overflow,
		height:        fmax(0, committed),
		firstTop:      firstTop,
		lastBottom:    lastBottom,
		firstBaseline: firstBaseline,
	}
}
