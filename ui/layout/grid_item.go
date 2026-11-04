package layout

import "github.com/merlindorin/ember/ui/internal/style"

// gridItem is an in-flow grid item, Taffy's GridItem.
type gridItem struct {
	node                      *node
	st                        *style.Computed
	sourceOrder               int
	row, column               lineSpan
	alignSelf                 style.Align
	justifySelf               style.Align
	baseline                  float32
	baselineShim              float32
	rowIndexes, columnIndexes lineSpan

	crossesFlexibleRow, crossesFlexibleColumn   bool
	crossesIntrinsicRow, crossesIntrinsicColumn bool

	areaCache       *size
	knownCache      *[2]size
	minContentCache size
	minimumCache    size
	maxContentCache size

	yPosition, height float32
	oof               []oofCandidate
}

func newGridItem(n *node, alignItems, justifyItems style.Align, order int) *gridItem {
	st := n.st
	self, justify := st.AlignSelf, st.JustifySelf
	if self == style.AlignAuto {
		self = alignItems
	}
	if justify == style.AlignAuto {
		justify = justifyItems
	}
	return &gridItem{
		node: n, st: st, sourceOrder: order, alignSelf: self, justifySelf: justify,
		baseline: none, minContentCache: noSize, minimumCache: noSize, maxContentCache: noSize,
	}
}

func (it *gridItem) hasAutoBlockMargin() bool {
	return it.st.Margin.Top.Kind == style.Auto || it.st.Margin.Bottom.Kind == style.Auto
}

func (it *gridItem) baselineAligned() bool {
	cyclic := hasPercent(it.st.Size.Height) && (it.crossesIntrinsicRow || it.crossesFlexibleRow)
	return it.alignSelf == style.AlignBaseline && !it.hasAutoBlockMargin() && !cyclic
}

func (it *gridItem) placement(a axis) lineSpan {
	if a == horizontal {
		return it.column
	}
	return it.row
}

func (it *gridItem) indexes(a axis) lineSpan {
	if a == horizontal {
		return it.columnIndexes
	}
	return it.rowIndexes
}

// trackRange is the indexes of the tracks the item spans, gutters between them included.
func (it *gridItem) trackRange(a axis) (int, int) {
	ix := it.indexes(a)
	return ix.start + 1, ix.end
}

func (it *gridItem) spansTrack(a axis, tracks []gridTrack, pred func(*gridTrack) bool) bool {
	from, to := it.trackRange(a)
	for i := from; i < to; i++ {
		if pred(&tracks[i]) {
			return true
		}
	}
	return false
}

func (it *gridItem) span(a axis) int { return it.placement(a).span() }

func (it *gridItem) crossesFlexible(a axis) bool {
	if a == horizontal {
		return it.crossesFlexibleColumn
	}
	return it.crossesFlexibleRow
}

func (it *gridItem) crossesIntrinsic(a axis) bool {
	if a == horizontal {
		return it.crossesIntrinsicColumn
	}
	return it.crossesIntrinsicRow
}

func (it *gridItem) spannedLimit(a axis, tracks []gridTrack,
	parent float32, f func(style.Length, float32) float32) float32 {
	from, to := it.trackRange(a)
	sum := float32(0)
	for i := from; i < to; i++ {
		v := f(tracks[i].max, parent)
		if !some(v) {
			return none
		}
		sum += v
	}
	return sum
}

func (it *gridItem) paddingBorderSize(basis float32) size {
	return edges(it.st.Padding, basis).add(borders(it.st)).sum()
}

func isSelfStretch(a style.Align) bool { return a == style.AlignStretch || a == style.AlignNormal }

func (it *gridItem) knownDimensions(area size) size {
	if it.knownCache != nil && sameSize(it.knownCache[0], area) {
		return it.knownCache[1]
	}
	k := it.computeKnownDimensions(area)
	it.knownCache = &[2]size{area, k}
	return k
}

func sameSize(a, b size) bool {
	eq := func(x, y float32) bool { return x == y || (!some(x) && !some(y)) }
	return eq(a.w, b.w) && eq(a.h, b.h)
}

func (it *gridItem) computeKnownDimensions(area size) size {
	st := it.st
	margins := it.marginSums(area.w)
	ratio := aspectRatio(st)
	pbSize := it.paddingBorderSize(area.w)
	adj := boxSizingAdjustment(st, pbSize)
	inherent := resolveSize(st.Size, area).withAspectRatio(ratio).mAdd(adj)
	minSize := resolveSize(st.MinSize, area).withAspectRatio(ratio).mAdd(adj).or(pbSize).mMax(pbSize)
	maxSize := resolveMax(st.MaxSize, area).withAspectRatio(ratio).mAdd(adj)
	inner := area.mSub(margins).mMax(size{})
	w := inherent.w
	if !some(w) {
		switch {
		case isSizingKeyword(st.Size.Width):
		case st.Margin.Left.Kind != style.Auto && st.Margin.Right.Kind != style.Auto && isSelfStretch(it.justifySelf):
			w = inner.w
		}
	}
	s := size{w, inherent.h}.withAspectRatio(ratio)
	if !some(s.h) {
		switch {
		case isSizingKeyword(st.Size.Height):
		case st.Margin.Top.Kind != style.Auto && st.Margin.Bottom.Kind != style.Auto && isSelfStretch(it.alignSelf):
			s.h = inner.h
		}
	}
	return s.withAspectRatio(ratio).mClamp(minSize, maxSize)
}

// gridAreaSize is the size of the item's grid area where its tracks are known.
func (it *gridItem) gridAreaSize(a axis, tracks, other []gridTrack,
	avail size, estimate func(*gridTrack, float32) float32) size {
	s := noSize
	from, to := it.trackRange(a)
	sum := float32(0)
	for i := from; i < to; i++ {
		t := &tracks[i]
		lo, hi := definiteValue(t.min, avail.get(a)), definiteValue(t.max, avail.get(a))
		if !some(lo) || !some(hi) || lo != hi {
			sum = none
			break
		}
		sum += t.baseSize
	}
	s.set(a, sum)
	from, to = it.trackRange(a.other())
	sum = 0
	for i := from; i < to; i++ {
		v := estimate(&other[i], avail.get(a.other()))
		if !some(v) {
			sum = none
			break
		}
		sum += v + other[i].contentAlignmentAdjustment
	}
	s.set(a.other(), sum)
	return s
}

func (it *gridItem) gridAreaSizeCached(a axis, tracks, other []gridTrack,
	avail size, estimate func(*gridTrack, float32) float32) size {
	if it.areaCache != nil {
		return *it.areaCache
	}
	s := it.gridAreaSize(a, tracks, other, avail, estimate)
	it.areaCache = &s
	return s
}

func (it *gridItem) marginSums(innerWidth float32) size {
	m := it.st.Margin
	return size{
		w: resolveOrZero(m.Left, 0) + resolveOrZero(m.Right, 0),
		h: resolveOrZero(m.Top, innerWidth) + it.baselineShim + resolveOrZero(m.Bottom, innerWidth),
	}
}

func (it *gridItem) availMinusMargins(area, avail size) size {
	if !some(avail.w) && !some(avail.h) {
		return avail
	}
	margin := edges(it.st.Margin, area.w)
	return size{
		w: mMax(mSub(avail.w, margin.horizontal()), 0),
		h: mMax(mSub(mSub(avail.h, margin.vertical()), it.baselineShim), 0),
	}
}

func (it *gridItem) keywordAdjustedSpace(area size, avail spaces) spaces {
	st := it.st
	if !isSizingKeyword(st.Size.Width) && !isSizingKeyword(st.Size.Height) {
		return avail
	}
	margins := it.marginSums(area.w)
	for _, a := range []axis{horizontal, vertical} {
		l := st.Size.Width
		if a == vertical {
			l = st.Size.Height
		}
		if !isSizingKeyword(l) {
			continue
		}
		stretch := mMax(mSub(area.get(a), margins.get(a)), 0)
		if k, s := resolveSizingKeyword(l, stretch, area.get(a)); k == keywordMeasure {
			avail = avail.with(a, s)
		}
	}
	return avail
}

func (t *tree) gridContribution(it *gridItem, a axis, area, avail size, fallback space) float32 {
	known := it.knownDimensions(area)
	if v := known.get(a); some(v) {
		return v
	}
	am := it.availMinusMargins(area, avail)
	space := spaces{spaceOr(am.w, fallback), spaceOr(am.h, fallback)}
	return t.measureChildSize(it.node, known, area, it.keywordAdjustedSpace(area, space), inherentSize, a, [2]bool{})
}

func (t *tree) minContentContribution(it *gridItem, a axis, area, avail size) float32 {
	return t.gridContribution(it, a, area, avail, minContentSpace)
}

func (t *tree) maxContentContribution(it *gridItem, a axis, area, avail size) float32 {
	return t.gridContribution(it, a, area, avail, maxContentSpace)
}

func (t *tree) minContentContributionCached(it *gridItem, a axis, area, avail size) float32 {
	if v := it.minContentCache.get(a); some(v) {
		return v
	}
	v := t.minContentContribution(it, a, area, avail)
	it.minContentCache.set(a, v)
	return v
}

func (t *tree) maxContentContributionCached(it *gridItem, a axis, area, avail size) float32 {
	if v := it.maxContentCache.get(a); some(v) {
		return v
	}
	v := t.maxContentContribution(it, a, area, avail)
	it.maxContentCache.set(a, v)
	return v
}

// minimumContribution is the item's automatic minimum size in a, Taffy's
// minimum_contribution.
func (t *tree) minimumContribution(it *gridItem, a axis, tracks []gridTrack, area, inner size) float32 {
	st := it.st
	ratio := aspectRatio(st)
	pbSize := it.paddingBorderSize(area.w)
	adj := boxSizingAdjustment(st, pbSize)
	sizeStyle, minStyle, maxStyle := st.Size.Width, st.MinSize.Width, st.MaxSize.Width
	if a == vertical {
		sizeStyle, minStyle, maxStyle = st.Size.Height, st.MinSize.Height, st.MaxSize.Height
	}
	if v := resolveSize(st.Size, area).withAspectRatio(ratio).mAdd(adj).get(a); some(v) {
		if minStyle.Kind == style.Auto && (maxStyle.Kind == style.Auto || maxStyle.Kind == style.None) {
			return v
		}
		basis := area.get(a)
		lo := mAdd(resolve(minStyle, basis), adj.get(a))
		hi := mAdd(resolve(maxStyle, basis), adj.get(a))
		return mClamp(v, lo, hi)
	}
	if isSizingKeyword(sizeStyle) {
		return t.minContentContributionCached(it, a, area, area)
	}
	if v := resolveSize(st.MinSize, area.or(size{})).withAspectRatio(ratio).mAdd(adj).get(a); some(v) {
		return v
	}
	if v := automaticMinSize(st.Overflow).get(a); some(v) {
		return v
	}
	from, to := it.trackRange(a)
	spansAuto, anyFr := false, false
	for i := from; i < to; i++ {
		if behavesAsAuto(tracks[i].min, inner.get(a)) {
			spansAuto = true
		}
		if tracks[i].max.Kind == style.Fr {
			anyFr = true
		}
	}
	if !spansAuto || (to-from != 1 && anyFr) {
		return 0
	}
	contribution := t.minContentContributionCached(it, a, area, area)
	limit := it.spannedLimit(a, tracks, inner.get(a), definiteValue)
	margins := it.marginSums(area.w)
	if some(limit) {
		contribution = fmin(contribution, fmax(limit-margins.get(a), pbSize.get(a)))
	}
	return contribution
}

func (t *tree) minimumContributionCached(it *gridItem, a axis, tracks []gridTrack, area, inner size) float32 {
	if v := it.minimumCache.get(a); some(v) {
		return v
	}
	v := t.minimumContribution(it, a, tracks, area, inner)
	it.minimumCache.set(a, v)
	return v
}
