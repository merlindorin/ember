package layout

import (
	"sort"

	"github.com/merlindorin/ember/ui/internal/style"
)

func gridAlignContent(a style.Align) alignment {
	if a == style.AlignNormal {
		return align(style.AlignStretch)
	}
	return align(a)
}

// grid lays out a grid container, CSS Grid §12, Taffy's compute_grid_layout.
//
//nolint:cyclop,funlen,gocognit,gocyclo,nestif // Taffy's compute_grid_layout, kept in its shape
func (t *tree) grid(n *node, in input) output {
	st := n.st
	ratio := aspectRatio(st)
	padding := edges(st.Padding, in.parent.w)
	border := borders(st)
	pbSize := padding.add(border).sum()
	adj := boxSizingAdjustment(st, pbSize)
	minSize := resolveSize(st.MinSize, in.parent).withAspectRatio(ratio).mAdd(adj)
	maxSize := resolveMax(st.MaxSize, in.parent).withAspectRatio(ratio).mAdd(adj)
	preferred := noSize
	if in.sizing == inherentSize {
		preferred = resolveSize(st.Size, in.parent).withAspectRatio(ratio).mAdd(adj)
	}
	g := n.ownGutter()
	inset := padding.add(border)
	inset.b += g.y
	inset.r += g.x
	alignContent := gridAlignContent(st.AlignContent)
	justifyContent := gridAlignContent(st.JustifyContent)
	alignItems, justifyItems := st.AlignItems, st.JustifyItems

	known := in.known.or(preferred)
	constrained := spaces{spaceOr(known.w, in.avail.w), spaceOr(known.h, in.avail.h)}
	constrained = spaces{constrained.w.mClamp(minSize.w, maxSize.w).mMax(pbSize.w),
		constrained.h.mClamp(minSize.h, maxSize.h).mMax(pbSize.h)}
	availGrid := spaces{
		w: constrained.w.mapDefinite(func(v float32) float32 { return v - inset.horizontal() }),
		h: constrained.h.mapDefinite(func(v float32) float32 { return v - inset.vertical() }),
	}
	outer := known.mClamp(minSize, maxSize).mMax(pbSize)
	innerMin := minSize.mSub(inset.sum())
	innerMax := maxSize.mSub(inset.sum())
	inner := size{mSub(outer.w, inset.horizontal()), mSub(outer.h, inset.vertical())}

	if in.run == computeSize {
		if outer.both() {
			return sizeOutput(outer)
		}
		if in.axis == axisHorizontal && some(outer.w) {
			return sizeOutput(size{outer.w, 0})
		}
	}

	autoFit := outer.or(maxSize).or(minSize).mClamp(minSize, maxSize).mMax(pbSize).mSub(inset.sum())
	strategy := func(v float32) autoRepeatStrategy {
		if some(v) {
			return maxRepetitionsThatDoNotOverflow
		}
		return minRepetitionsThatDoOverflow
	}
	strategies := outer.or(maxSize)
	colReps, colCount := explicitGridSize(st, autoFit.w, strategy(strategies.w), horizontal)
	rowReps, rowCount := explicitGridSize(st, autoFit.h, strategy(strategies.h), vertical)

	var items []*gridItem
	var placements []itemPlacement
	for i, k := range n.kids {
		if k.st.Display == style.DisplayNone || outOfFlow(k.st.Position) {
			continue
		}
		placements = append(placements, itemPlacement{
			h: placementOf(k.st.Column, colCount),
			v: placementOf(k.st.Row, rowCount),
		})
		items = append(items, newGridItem(k, alignItems, justifyItems, i))
	}
	estCols, estRows := gridSizeEstimate(colCount, rowCount, placements)
	occupancy := newCellOccupancy(estCols, estRows)
	byOrder := make([]int, len(items))
	for i := range byOrder {
		byOrder[i] = i
	}
	sort.SliceStable(byOrder, func(i, j int) bool { return items[byOrder[i]].st.Order < items[byOrder[j]].st.Order })
	orderedItems := make([]*gridItem, len(items))
	orderedPlacements := make([]itemPlacement, len(items))
	for i, j := range byOrder {
		orderedItems[i], orderedPlacements[i] = items[j], placements[j]
	}
	placeGridItems(occupancy, orderedItems, orderedPlacements, st.Flow)
	finalCols, finalRows := occupancy.columns, occupancy.rows
	columns := initializeGridTracks(finalCols, st, horizontal, colReps,
		func(i int) bool { return occupancy.occupied(horizontal, i) })
	rows := initializeGridTracks(finalRows, st, vertical, rowReps,
		func(i int) bool { return occupancy.occupied(vertical, i) })

	resolveItemTrackIndexes(items, finalCols, finalRows)
	markCrossings(items, columns, rows)
	hasBaseline := false
	for _, it := range items {
		if it.baselineAligned() {
			hasBaseline = true
		}
	}
	byMax := func(tr *gridTrack, parent float32) float32 { return definiteValue(tr.max, parent) }
	byBase := func(tr *gridTrack, _ float32) float32 { return tr.baseSize }
	t.trackSizing(horizontal, innerMin.w, innerMax.w, justifyContent,
		alignContent, availGrid, inner, columns, rows, items, byMax, hasBaseline)
	colSum := sumBaseSizes(columns)
	inner.w = or(inner.w, colSum)
	if in.run == computeSize && in.axis == axisHorizontal {
		w := fmax(mClamp(colSum+inset.horizontal(), minSize.w, maxSize.w), pbSize.w)
		return sizeOutput(size{w, 0})
	}
	for _, it := range items {
		it.areaCache = nil
	}
	t.trackSizing(vertical, innerMin.h, innerMax.h, alignContent,
		justifyContent, availGrid, inner, rows, columns, items, byBase, false)
	rowSum := sumBaseSizes(rows)
	borderBox := size{
		w: fmax(mClamp(or(known.w, colSum+inset.horizontal()), minSize.w, maxSize.w), pbSize.w),
		h: fmax(mClamp(or(known.h, rowSum+inset.vertical()), minSize.h, maxSize.h), pbSize.h),
	}
	contentBox := size{fmax(0, borderBox.w-inset.horizontal()), fmax(0, borderBox.h-inset.vertical())}
	if in.run == computeSize {
		return sizeOutput(borderBox)
	}
	inner = contentBox

	hasPercentColumn, hasPercentRow := false, false
	for i := range columns {
		hasPercentColumn = hasPercentColumn || columns[i].usesPercentage()
	}
	for i := range rows {
		hasPercentRow = hasPercentRow || rows[i].usesPercentage()
	}
	rerunColumns := !some(outer.w) && hasPercentColumn
	columnChanged := false
	if !rerunColumns {
		for _, it := range items {
			if !it.crossesIntrinsicColumn {
				continue
			}
			area := it.gridAreaSize(horizontal, columns, rows, inner, byBase)
			v := t.minContentContribution(it, horizontal, area, area.with(horizontal, none))
			if !some(it.minContentCache.w) || v != it.minContentCache.w {
				columnChanged = true
			}
			it.areaCache = &area
			it.minContentCache.w = v
			it.maxContentCache.w = none
			it.minimumCache.w = none
		}
		rerunColumns = columnChanged
	} else {
		for _, it := range items {
			it.areaCache = nil
			it.minContentCache.w, it.maxContentCache.w, it.minimumCache.w = none, none, none
		}
	}
	rerunRows := !some(outer.h) && hasPercentRow
	rowChanged := false
	if rerunRows {
		for _, it := range items {
			it.areaCache = nil
			it.minContentCache.h, it.maxContentCache.h, it.minimumCache.h = none, none, none
		}
	}
	if rerunColumns {
		t.trackSizing(horizontal, innerMin.w, innerMax.w, justifyContent,
			alignContent, availGrid, inner, columns, rows, items, byBase, hasBaseline)
		if !rerunRows {
			for _, it := range items {
				if !it.crossesIntrinsicRow {
					continue
				}
				area := it.gridAreaSize(vertical, rows, columns, inner, byBase)
				v := t.minContentContribution(it, vertical, area, area.with(vertical, none))
				if !some(it.minContentCache.h) || v != it.minContentCache.h {
					rowChanged = true
				}
				it.areaCache = &area
				it.minContentCache.h = v
				it.maxContentCache.h = none
				it.minimumCache.h = none
			}
			rerunRows = rowChanged
		}
	}
	if rerunRows {
		t.trackSizing(vertical, innerMin.h, innerMax.h, alignContent,
			justifyContent, availGrid, inner, rows, columns, items, byBase, false)
	}
	if columnChanged && !hasPercentColumn {
		borderBox.w = fmax(mClamp(or(known.w, sumBaseSizes(columns)+inset.horizontal()), minSize.w, maxSize.w), pbSize.w)
		contentBox.w = fmax(0, borderBox.w-inset.horizontal())
	}
	if rowChanged && !hasPercentRow {
		borderBox.h = fmax(mClamp(or(known.h, sumBaseSizes(rows)+inset.vertical()), minSize.h, maxSize.h), pbSize.h)
		contentBox.h = fmax(0, borderBox.h-inset.vertical())
	}

	inlineWithoutScrollbar := fmax(borderBox.w-pbSize.w, 0)
	inlineGutter := fmin(g.x, inlineWithoutScrollbar)
	alignTracks(contentBox.w, span{padding.l, padding.r + inlineGutter}, span{border.l, border.r}, columns, justifyContent)
	alignTracks(contentBox.h, span{padding.t, padding.b}, span{border.t, border.b}, rows, alignContent)

	paddingBoxInset := border.add(rect{r: g.x, b: g.y})
	container := overflowContainer{
		scroll:  isScrollContainer(st),
		size:    size{borderBox.w - paddingBoxInset.horizontal(), 0},
		padding: padding,
	}
	var overflow rect
	for i, it := range items {
		area := rect{
			t: rows[it.rowIndexes.start+1].offset,
			b: rows[it.rowIndexes.end].offset,
			l: columns[it.columnIndexes.start+1].offset,
			r: columns[it.columnIndexes.end].offset,
		}
		contribution, y, h, baseline := t.alignAndPositionItem(it, uint32(i), area,
			size{area.r - area.l, area.b - area.t}, alignItems, justifyItems, paddingBoxInset, container)
		it.yPosition, it.height, it.baseline = y, h, baseline
		overflow = overflow.union(contribution)
	}

	var candidates []oofCandidate
	order := uint32(len(items)) //nolint:gosec // a grid's item count never nears 2^32
	next := 0
	claimsHere := containingBlockClaims(st)
	for i, k := range n.kids {
		if next < len(items) && items[next].sourceOrder == i {
			candidates = append(candidates, items[next].oof...)
			next++
			continue
		}
		if k.st.Display == style.DisplayNone {
			k.unrounded = placed{order: order}
			t.performChildLayout(k, noSize, noSize, maxContentSpaces, inherentSize)
			order++
			continue
		}
		if !outOfFlow(k.st.Position) {
			continue
		}
		justify := k.st.JustifySelf
		if justify == style.AlignAuto {
			justify = justifyItems
		}
		self := k.st.AlignSelf
		if self == style.AlignAuto {
			self = alignItems
		}
		var area rect
		if claimsHere.take(k.st.Position) {
			area = staticGridArea(k.st, finalRows, finalCols, rows, columns, border, g, borderBox)
		} else {
			area = rect{
				l: border.l + padding.l,
				r: borderBox.w - border.r - padding.r - g.x,
				t: border.t + padding.t,
				b: borderBox.h - border.b - padding.b - g.y,
			}
		}
		candidates = append(candidates, oofCandidate{
			node:     k,
			order:    order,
			position: k.st.Position,
			x:        staticFromAlignment(align(justify), span{area.l, area.r}),
			y:        staticFromAlignment(align(self), span{area.t, area.b}),
		})
		order++
	}
	n.grid = newGridInfo(finalCols, columns, finalRows, rows)
	area := oofArea{size: borderBox.sub(paddingBoxInset.sum()), offset: point{paddingBoxInset.l, paddingBoxInset.t}}
	if isScrollContainer(st) {
		overflow = overflow.union(rect{r: padding.r, b: padding.b})
	}
	port := borderBox.sub(border.sum()).sub(size{g.x, g.y})
	out := output{
		size:          borderBox,
		overflow:      finalizeOverflow(overflow, port, [2]bool{}, isScrollContainer(st)),
		firstBaseline: none,
		lastBaseline:  none,
		oof:           candidates,
		area:          &area,
	}
	if len(items) > 0 {
		firstRow := items[0].rowIndexes.start
		for _, it := range items {
			firstRow = min(firstRow, it.rowIndexes.start)
		}
		var first *gridItem
		for _, it := range items {
			if it.rowIndexes.start != firstRow {
				continue
			}
			if first == nil {
				first = it
				if it.baselineAligned() {
					break
				}
				continue
			}
			if it.baselineAligned() {
				first = it
				break
			}
		}
		out.firstBaseline = first.yPosition + or(first.baseline, first.height)
	}
	return out
}

func sumBaseSizes(tracks []gridTrack) float32 {
	s := float32(0)
	for i := range tracks {
		s += tracks[i].baseSize
	}
	return s
}

func alignTracks(contentBox float32, padding, border span, tracks []gridTrack, a alignment) {
	used := sumBaseSizes(tracks)
	free := contentBox - used
	origin := padding.start + border.start
	count := 0
	for i := 1; i < len(tracks); i += 2 {
		if !tracks[i].collapsed {
			count++
		}
	}
	k := applyAlignmentFallback(free, count, a)
	emptyOffset := float32(0)
	if count == 0 {
		emptyOffset = computeAlignmentOffset(free, count, 0, k, false, true)
	}
	total := origin + emptyOffset
	seen := false
	for i := range tracks {
		tr := &tracks[i]
		gutter := i%2 == 0
		nonCollapsed := !gutter && !tr.collapsed
		offset := float32(0)
		if nonCollapsed {
			offset = computeAlignmentOffset(free, count, 0, k, false, !seen)
			seen = true
		}
		tr.offset = total + offset
		total += offset + tr.baseSize
	}
}

// alignAndPositionItem sizes a grid item in its area and aligns it, Taffy's
// align_and_position_item.
//
//nolint:cyclop,funlen,gocyclo // Taffy's align_and_position_item, kept in its shape
func (t *tree) alignAndPositionItem(it *gridItem, order uint32, area rect, cb size, _, _ style.Align,
	paddingBoxInset rect, container overflowContainer) (rect, float32, float32, float32) {
	n := it.node
	st := n.st
	areaSize := size{area.r - area.l, area.b - area.t}
	ratio := aspectRatio(st)
	justify, self := it.justifySelf, it.alignSelf
	pos := st.Position
	insetH := span{resolve(st.Inset.Left, cb.w), resolve(st.Inset.Right, cb.w)}
	insetV := span{resolve(st.Inset.Top, cb.h), resolve(st.Inset.Bottom, cb.h)}
	padding := edges(st.Padding, cb.w)
	border := borders(st)
	pbSize := padding.add(border).sum()
	adj := boxSizingAdjustment(st, pbSize)
	inherent := resolveSize(st.Size, cb).withAspectRatio(ratio).mAdd(adj)
	minSize := resolveSize(st.MinSize, cb).mAdd(adj).or(pbSize).mMax(pbSize).withAspectRatio(ratio)
	maxSize := resolveMax(st.MaxSize, cb).withAspectRatio(ratio).mAdd(adj)
	hAlign := justify
	if hAlign == style.AlignNormal {
		if some(inherent.w) || isSizingKeyword(st.Size.Width) {
			hAlign = style.AlignStart
		} else {
			hAlign = style.AlignStretch
		}
	}
	vAlign := self
	if vAlign == style.AlignNormal {
		if some(inherent.h) || isSizingKeyword(st.Size.Height) || some(ratio) {
			vAlign = style.AlignStart
		} else {
			vAlign = style.AlignStretch
		}
	}
	margin := optionalEdges(st.Margin, cb.w)
	inner := size{
		w: fmax(mSub(mSub(areaSize.w, margin.l), margin.r), 0),
		h: fmax(mSub(mSub(areaSize.h, margin.t), margin.b)-it.baselineShim, 0),
	}
	kw, kwSpace := keywordNone, space{}
	if !some(inherent.w) {
		kw, kwSpace = resolveSizingKeyword(st.Size.Width, inner.w, cb.w)
	}
	kh, khSpace := keywordNone, space{}
	if !some(inherent.h) {
		kh, khSpace = resolveSizingKeyword(st.Size.Height, inner.h, cb.h)
	}
	keywordMeasured := noSize
	if kw == keywordMeasure && kh == keywordMeasure && !outOfFlow(pos) {
		keywordMeasured = t.measureChildSizeBoth(n, noSize, cb, spaces{kwSpace, khSpace}, inherentSize, [2]bool{})
	}
	w := inherent.w
	if !some(w) {
		switch {
		case outOfFlow(pos) && some(insetH.start) && some(insetH.end):
			w = fmax(inner.w-insetH.start-insetH.end, 0)
		case kw == keywordMeasure:
			w = keywordMeasured.w
			if !some(w) {
				w = t.measureChildSize(n, noSize, cb, spaces{kwSpace, def(inner.h)}, inherentSize, horizontal, [2]bool{})
			}
		case some(margin.l) && some(margin.r) && hAlign == style.AlignStretch && !outOfFlow(pos):
			w = inner.w
		}
	}
	s := size{w, inherent.h}.withAspectRatio(ratio)
	if !some(s.h) {
		switch {
		case outOfFlow(pos) && some(insetV.start) && some(insetV.end):
			s.h = fmax(inner.h-insetV.start-insetV.end, 0)
		case kh == keywordMeasure:
			s.h = keywordMeasured.h
			if !some(s.h) {
				wSpace := def(inner.w)
				if some(s.w) {
					wSpace = def(s.w)
				}
				s.h = t.measureChildSize(n, size{s.w, none}, cb, spaces{wSpace, khSpace}, inherentSize, vertical, [2]bool{})
			}
		case some(margin.t) && some(margin.b) && vAlign == style.AlignStretch && !outOfFlow(pos):
			s.h = inner.h
		}
	}
	s = s.withAspectRatio(ratio).mClamp(minSize, maxSize)
	innerSpaces := spaces{def(inner.w), def(inner.h)}
	if outOfFlow(pos) && !s.both() {
		s = t.measureChildSizeBoth(n, s, cb, innerSpaces, inherentSize, [2]bool{})
	}
	out := t.performChildLayout(n, s, cb, innerSpaces, inherentSize)
	final := s.or(out.size).mClamp(minSize, maxSize)
	x, xMargin := alignItemWithinArea(span{area.l, area.r},
		align(hAlign), final.w, pos, insetH, span{margin.l, margin.r}, 0)
	y, yMargin := alignItemWithinArea(span{area.t, area.b}, align(vAlign),
		final.h, pos, insetV, span{margin.t, margin.b}, it.baselineShim)
	x += gridRelativeOffset(pos, insetH)
	yWithout := y
	y += gridRelativeOffset(pos, insetV)
	gutter := scrollbarGutter(st)
	resolvedMargin := rect{l: xMargin.start, r: xMargin.end, t: yMargin.start, b: yMargin.end}
	it.oof = nil
	if len(out.oof) > 0 {
		it.oof = translateCandidates(out.oof, point{x, y})
	}
	n.unrounded = placed{
		order:     order,
		location:  point{x, y},
		size:      final,
		overflow:  out.overflow,
		scrollbar: size{gutter.x, gutter.y},
		padding:   padding,
		border:    border,
		margin:    resolvedMargin,
	}
	contribution := overflowContribution(point{x - paddingBoxInset.l, y - paddingBoxInset.t},
		final, out.overflow.rect, border, st.Overflow, container, resolvedMargin)
	return contribution, yWithout, final.h, childFirstBaseline(out.firstBaseline, final.h, st.Overflow.Height)
}

func alignItemWithinArea(area span, a alignment, s float32,
	pos style.Position, inset, margin span, shim float32) (float32, span) {
	nonAuto := span{or(margin.start, 0) + shim, or(margin.end, 0)}
	areaSize := fmax(area.end-area.start, 0)
	free := fmax(areaSize-s-nonAuto.start-nonAuto.end, 0)
	autos := 0
	if !some(margin.start) {
		autos++
	}
	if !some(margin.end) {
		autos++
	}
	autoSize := float32(0)
	if autos > 0 {
		autoSize = free / float32(autos)
	}
	resolved := span{or(margin.start, autoSize) + shim, or(margin.end, autoSize)}
	overflows := s+nonAuto.start+nonAuto.end > areaSize
	var offset float32
	switch resolveSelfAlignmentSafety(a, overflows) {
	case style.AlignEnd, style.AlignFlexEnd:
		offset = areaSize - s - resolved.end
	case style.AlignCenter:
		offset = (areaSize - s + resolved.start - resolved.end) / 2
	case style.AlignNormal, style.AlignAuto, style.AlignStretch, style.AlignStart, style.AlignFlexStart,
		style.AlignBaseline, style.AlignSpaceBetween, style.AlignSpaceAround, style.AlignSpaceEvenly:
		offset = resolved.start
	}
	if outOfFlow(pos) {
		switch {
		case some(inset.start):
			offset = inset.start + nonAuto.start
		case some(inset.end):
			offset = areaSize - inset.end - s - nonAuto.end
		}
	}
	return area.start + offset, resolved
}

func gridRelativeOffset(pos style.Position, inset span) float32 {
	if pos != style.Relative {
		return 0
	}
	return or(inset.start, or(-inset.end, 0))
}

func staticGridArea(st *style.Computed, rowCounts, colCounts trackCounts,
	rows, columns []gridTrack, border rect, gutter point, borderBox size) rect {
	axisIndexes := func(lines [2]style.GridLine, counts trackCounts) (int, int, bool, bool) {
		s, e, okS, okE := placementOf(lines, counts.explicit).resolveAbsolute()
		var si, ei int
		if okS {
			si, okS = counts.trackVecIndex(s)
		}
		if okE {
			ei, okE = counts.trackVecIndex(e)
		}
		return si, ei, okS, okE
	}
	startEdge := func(tracks []gridTrack, i int) float32 {
		if i+1 < len(tracks) {
			return tracks[i+1].offset
		}
		return tracks[i].offset
	}
	endEdge := func(tracks []gridTrack, i int) float32 {
		if i == 0 {
			if len(tracks) > 1 {
				return tracks[1].offset
			}
			return tracks[0].offset
		}
		return tracks[i].offset
	}
	cs, ce, okCS, okCE := axisIndexes(st.Column, colCounts)
	rs, re, okRS, okRE := axisIndexes(st.Row, rowCounts)
	r := rect{l: border.l, r: borderBox.w - border.r - gutter.x, t: border.t, b: borderBox.h - border.b - gutter.y}
	if okCS {
		r.l = startEdge(columns, cs)
	}
	if okCE {
		r.r = endEdge(columns, ce)
	}
	if okRS {
		r.t = startEdge(rows, rs)
	}
	if okRE {
		r.b = endEdge(rows, re)
	}
	return r
}

// gridAxisInfo keeps one axis of a laid-out grid for its absolutely positioned children.
type gridAxisInfo struct {
	counts    trackCounts
	positions []span
	collapsed [][2]int
	emptyLine float32
}

// gridInfo is a grid container's tracks after layout, Taffy's DetailedGridInfo.
type gridInfo struct {
	columns, rows gridAxisInfo
}

func newGridAxisInfo(counts trackCounts, tracks []gridTrack) gridAxisInfo {
	info := gridAxisInfo{counts: counts, emptyLine: none}
	for i := range tracks {
		tr := &tracks[i]
		if tr.kind != trackTrack {
			continue
		}
		if tr.collapsed {
			idx := len(info.positions)
			if n := len(info.collapsed); n > 0 && info.collapsed[n-1][1] == idx {
				info.collapsed[n-1][1]++
			} else {
				info.collapsed = append(info.collapsed, [2]int{idx, idx + 1})
			}
		}
		info.positions = append(info.positions, span{tr.offset, tr.offset + tr.baseSize})
	}
	if len(info.positions) == 0 && len(tracks) > 0 {
		info.emptyLine = tracks[0].offset
	}
	return info
}

func newGridInfo(colCounts trackCounts, columns []gridTrack, rowCounts trackCounts, rows []gridTrack) *gridInfo {
	return &gridInfo{columns: newGridAxisInfo(colCounts, columns), rows: newGridAxisInfo(rowCounts, rows)}
}

func (g *gridAxisInfo) trackEdge(index int, isStart bool) float32 {
	track := g.positions[index]
	for _, c := range g.collapsed {
		if index >= c[0] && index < c[1] {
			var adjacent int
			if isStart {
				adjacent = c[1]
			} else {
				adjacent = c[0] - 1
			}
			if adjacent >= 0 && adjacent < len(g.positions) {
				track = g.positions[adjacent]
			}
			break
		}
	}
	if isStart {
		return track.start
	}
	return track.end
}

func (g *gridAxisInfo) resolve(lines [2]style.GridLine, paddingStart, paddingEnd float32) span {
	minLine, maxLine := -g.counts.negative, g.counts.explicit+g.counts.positive
	s, e, okS, okE := placementOf(lines, g.counts.explicit).resolveAbsolute()
	toIndex := func(l int, ok bool) (int, bool) {
		if !ok || l < minLine || l > maxLine {
			return 0, false
		}
		i, ok := g.counts.trackVecIndex(l)
		return i / 2, ok
	}
	si, okS := toIndex(s, okS)
	ei, okE := toIndex(e, okE)
	start := paddingStart
	if okS {
		switch {
		case si < len(g.positions):
			start = g.trackEdge(si, true)
		case len(g.positions) > 0:
			start = g.positions[len(g.positions)-1].end
		case some(g.emptyLine):
			start = g.emptyLine
		}
	}
	end := paddingEnd
	if okE {
		switch {
		case ei >= 1 && ei-1 < len(g.positions):
			end = g.trackEdge(ei-1, false)
		case len(g.positions) > 0:
			end = g.positions[0].start
		case some(g.emptyLine):
			end = g.emptyLine
		}
	}
	return span{start, fmax(start, end)}
}

// absoluteArea is the containing block an absolutely positioned child's grid lines give it
// inside the padding box.
func (g *gridInfo) absoluteArea(st *style.Computed, paddingBox rect) rect {
	c := g.columns.resolve(st.Column, paddingBox.l, paddingBox.r)
	r := g.rows.resolve(st.Row, paddingBox.t, paddingBox.b)
	return rect{l: c.start, r: c.end, t: r.start, b: r.end}
}
