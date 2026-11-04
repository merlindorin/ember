package layout

import (
	"cmp"
	"math"
	"slices"

	"github.com/merlindorin/ember/ui/internal/style"
)

// flexDir is a flex direction with its axes.
type flexDir style.FlexDirection

func (d flexDir) isRow() bool    { return d == flexDir(style.Row) || d == flexDir(style.RowReverse) }
func (d flexDir) isColumn() bool { return !d.isRow() }
func (d flexDir) isReverse() bool {
	return d == flexDir(style.RowReverse) || d == flexDir(style.ColumnReverse)
}

func (d flexDir) main() axis {
	if d.isRow() {
		return horizontal
	}
	return vertical
}

func (d flexDir) cross() axis { return d.main().other() }

func (d flexDir) mainStart(r rect) float32 {
	if d.isRow() {
		return r.l
	}
	return r.t
}

func (d flexDir) mainEnd(r rect) float32 {
	if d.isRow() {
		return r.r
	}
	return r.b
}

func (d flexDir) crossStart(r rect) float32 {
	if d.isRow() {
		return r.t
	}
	return r.l
}

func (d flexDir) crossEnd(r rect) float32 {
	if d.isRow() {
		return r.b
	}
	return r.r
}

func (d flexDir) mainSum(r rect) float32  { return d.mainStart(r) + d.mainEnd(r) }
func (d flexDir) crossSum(r rect) float32 { return d.crossStart(r) + d.crossEnd(r) }

func (d flexDir) autoMainStart(a autoEdges) bool {
	if d.isRow() {
		return a.l
	}
	return a.t
}

func (d flexDir) autoMainEnd(a autoEdges) bool {
	if d.isRow() {
		return a.r
	}
	return a.b
}

func (d flexDir) autoCrossStart(a autoEdges) bool {
	if d.isRow() {
		return a.t
	}
	return a.l
}

func (d flexDir) autoCrossEnd(a autoEdges) bool {
	if d.isRow() {
		return a.b
	}
	return a.r
}

func (d flexDir) mainLength(s style.Size[style.Length]) style.Length {
	if d.isRow() {
		return s.Width
	}
	return s.Height
}

func (d flexDir) crossLength(s style.Size[style.Length]) style.Length {
	if d.isRow() {
		return s.Height
	}
	return s.Width
}

type flexItem struct {
	node                    *node
	order                   uint32
	size                    size
	sizeStyle               style.Size[style.Length]
	minSize, maxSize        size
	aspectRatio             float32
	alignSelf               style.Align
	overflow                style.Size[style.Overflow]
	scrollbarWidth          float32
	shrink, grow            float32
	basisIsDefinite         bool
	resolvedMinimumMainSize float32
	relativeInset           size
	margin                  rect
	marginIsAuto            autoEdges
	padding, border         rect
	flexBasis               float32
	innerFlexBasis          float32
	violation               float32
	frozen                  bool
	contentFlexFraction     float32
	hypotheticalInnerSize   size
	hypotheticalOuterSize   size
	targetSize              size
	outerTargetSize         size
	baseline                float32
	offsetMain, offsetCross float32
	oof                     []oofCandidate
}

func (it *flexItem) boxSizingAdjustment() size {
	if contentBox(it.node.st) {
		return it.padding.add(it.border).sum()
	}
	return size{}
}

func (it *flexItem) isScrollContainer() bool {
	return scrollContainer(it.overflow.Width) || scrollContainer(it.overflow.Height)
}

func (it *flexItem) isStretched(d flexDir) bool {
	return !d.autoCrossStart(it.marginIsAuto) && !d.autoCrossEnd(it.marginIsAuto) &&
		it.alignSelf == style.AlignStretch && d.crossLength(it.sizeStyle).Kind == style.Auto
}

func (it *flexItem) baselineAligned(d flexDir) bool {
	return it.alignSelf == style.AlignBaseline && !d.autoCrossStart(it.marginIsAuto) && !d.autoCrossEnd(it.marginIsAuto)
}

type flexLine struct {
	items       []*flexItem
	crossSize   float32
	offsetCross float32
}

type flexConstants struct {
	dir                     flexDir
	isRow, isColumn         bool
	isWrap, isWrapReverse   bool
	minSize, maxSize        size
	border                  rect
	contentBoxInset         rect
	paddingBoxInset         rect
	padding                 rect
	scrollbarGutter         point
	isScrollContainer       bool
	gap                     size
	alignItems              style.Align
	alignContent            alignment
	justifyContent          alignment
	nodeOuterSize           size
	nodeInnerSize           size
	knownMainSizeIsDefinite bool
	hasDefiniteMainSize     bool
	hasDefiniteCrossSize    bool
	crossAvailIsDefinite    bool
	containerSize           size
	innerContainerSize      size
}

func (c *flexConstants) scrollOriginReversed() [2]bool {
	main, cross := c.dir.isReverse(), c.isWrapReverse
	if c.isColumn {
		return [2]bool{cross, main}
	}
	return [2]bool{main, cross}
}

func normalToStretch(a style.Align) style.Align {
	if a == style.AlignNormal {
		return style.AlignStretch
	}
	return a
}

// flex lays out a flex container, CSS Flexbox §9, Taffy's compute_flexbox_layout.
func (t *tree) flex(n *node, in input) output {
	st := n.st
	ratio := aspectRatio(st)
	padding := edges(st.Padding, in.parent.w)
	border := borders(st)
	pbSum := padding.sum().add(border.sum())
	adj := boxSizingAdjustment(st, pbSum)
	minSize := resolveSize(st.MinSize, in.parent).withAspectRatio(ratio).mAdd(adj)
	maxSize := resolveMax(st.MaxSize, in.parent).withAspectRatio(ratio).mAdd(adj)
	clamped := noSize
	if in.sizing == inherentSize {
		clamped = resolveSize(st.Size, in.parent).withAspectRatio(ratio).mAdd(adj).mClamp(minSize, maxSize)
	}
	minMax := size{definiteMinMax(minSize.w, maxSize.w), definiteMinMax(minSize.h, maxSize.h)}
	styled := in.known.or(minMax.or(clamped).mMax(pbSum))
	if in.run == computeSize {
		if styled.both() {
			return sizeOutput(styled)
		}
		if in.axis == axisHorizontal && some(styled.w) {
			return sizeOutput(size{styled.w, 0})
		}
	}
	knownIsDefinite := bools{
		w: in.knownIsDefinite.w || !some(in.known.w),
		h: in.knownIsDefinite.h || !some(in.known.h),
	}
	sub := in
	sub.known = styled
	sub.knownIsDefinite = knownIsDefinite
	return t.flexPreliminary(n, sub)
}

//nolint:nestif // Taffy's flexbox compute_preliminary, kept in its shape
func (t *tree) flexPreliminary(n *node, in input) output {
	c := t.flexConstants(n, in.known, in.knownIsDefinite, in.parent, in.avail)
	items := t.flexItems(n, c)
	if len(items) == 0 && !hasOutOfFlowOrHidden(n) {
		return t.flexEmpty(in, c)
	}
	avail := flexAvailableSpace(in.known, in.avail, c)
	t.flexBaseSizes(c, avail, items)
	lines := collectFlexLines(c, avail, items)
	d := c.dir
	if inner := c.nodeInnerSize.get(d.main()); some(inner) {
		c.innerContainerSize.set(d.main(), inner)
		c.containerSize.set(d.main(), inner+d.mainSum(c.contentBoxInset))
	} else {
		t.flexContainerMainSize(avail, lines, c)
		c.nodeInnerSize.set(d.main(), c.innerContainerSize.get(d.main()))
		c.nodeOuterSize.set(d.main(), c.containerSize.get(d.main()))
		gapLen := n.st.Gap.Width
		if d.isColumn() {
			gapLen = n.st.Gap.Height
		}
		c.gap.set(d.main(), or(resolve(gapLen, c.innerContainerSize.get(d.main())), 0))
	}
	if in.run == computeSize && mainIsRequested(d, in.axis) {
		return sizeOutput(size{}.with(d.main(), c.containerSize.get(d.main())))
	}
	for _, l := range lines {
		resolveFlexibleLengths(l, c)
	}
	for _, l := range lines {
		t.hypotheticalCrossSize(l, c, in.known, avail)
	}
	t.childrenBaselines(in.known, avail, lines, c)
	calculateCrossSize(lines, in.known, c)
	alignContentStretch(lines, in.known, c)
	determineUsedCrossSize(lines, c)
	distributeRemainingFreeSpace(lines, c)
	resolveCrossAutoMargins(lines, c)
	totalCross := flexContainerCrossSize(lines, in.known, c)
	if in.run == computeSize {
		return sizeOutput(c.containerSize)
	}
	alignLines(lines, c, totalCross)
	inflowOverflow := t.flexFinalPass(lines, c)
	candidates := t.flexOOFCandidates(n, c, lines)
	area := oofArea{
		size:   c.containerSize.sub(c.paddingBoxInset.sum()),
		offset: point{c.paddingBoxInset.l, c.paddingBoxInset.t},
	}
	for i, k := range n.kids {
		if k.st.Display == style.DisplayNone {
			k.unrounded = placed{order: uint32(i)}
			t.performChildLayout(k, noSize, noSize, maxContentSpaces, contentSize)
		}
	}
	first := none
	var firstLine *flexLine
	if len(lines) > 0 {
		firstLine = lines[0]
		if c.isWrapReverse {
			firstLine = lines[len(lines)-1]
		}
	}
	if firstLine != nil && len(firstLine.items) > 0 {
		if c.isColumn {
			it := firstLine.items[0]
			if d.isReverse() {
				it = firstLine.items[len(firstLine.items)-1]
			}
			first = it.baseline
		} else {
			first = firstLine.items[0].baseline
			for _, it := range firstLine.items {
				if it.baselineAligned(d) {
					first = it.baseline
					break
				}
			}
		}
	}
	out := output{
		size: c.containerSize,
		overflow: finalizeOverflow(inflowOverflow,
			c.containerSize.sub(c.border.sum()).sub(size{c.scrollbarGutter.x, c.scrollbarGutter.y}),
			c.scrollOriginReversed(), c.isScrollContainer),
		firstBaseline: first,
		lastBaseline:  none,
		oof:           candidates,
		area:          &area,
	}
	return out
}

func hasOutOfFlowOrHidden(n *node) bool {
	for _, k := range n.kids {
		if outOfFlow(k.st.Position) || k.st.Display == style.DisplayNone {
			return true
		}
	}
	return false
}

func mainIsRequested(d flexDir, a requestedAxis) bool {
	if d.isRow() {
		return a == axisHorizontal
	}
	return a == axisVertical
}

func (t *tree) flexEmpty(in input, c *flexConstants) output {
	d := c.dir
	avail := flexAvailableSpace(in.known, in.avail, c)
	if inner := c.nodeInnerSize.get(d.main()); some(inner) {
		c.innerContainerSize.set(d.main(), inner)
		c.containerSize.set(d.main(), inner+d.mainSum(c.contentBoxInset))
	} else {
		t.flexContainerMainSize(avail, nil, c)
	}
	if in.run == computeSize && mainIsRequested(d, in.axis) {
		return sizeOutput(size{}.with(d.main(), c.containerSize.get(d.main())))
	}
	flexContainerCrossSize(nil, in.known, c)
	out := sizeOutput(c.containerSize)
	out.overflow = finalizeOverflow(rect{},
		c.containerSize.sub(c.border.sum()).sub(size{c.scrollbarGutter.x, c.scrollbarGutter.y}),
		c.scrollOriginReversed(), c.isScrollContainer)
	out.area = &oofArea{}
	return out
}

func (t *tree) flexConstants(n *node, known size, knownIsDefinite bools, parent size, avail spaces) *flexConstants {
	st := n.st
	d := flexDir(st.Direction)
	ratio := aspectRatio(st)
	padding := edges(st.Padding, parent.w)
	border := borders(st)
	adj := boxSizingAdjustment(st, padding.sum().add(border.sum()))
	alignContent := align(st.AlignContent)
	if alignContent.keyword == style.AlignNormal {
		alignContent.keyword = style.AlignStretch
	}
	justify := align(st.JustifyContent)
	if justify.keyword == style.AlignNormal {
		justify.keyword = style.AlignFlexStart
	}
	gutter := n.ownGutter()
	inset := padding.add(border)
	inset.b += gutter.y
	inset.r += gutter.x
	paddingBoxInset := border.add(rect{r: gutter.x, b: gutter.y})
	innerPadding := rect{
		l: inset.l - border.l,
		r: inset.r - border.r - gutter.x,
		t: inset.t - border.t,
		b: inset.b - border.b - gutter.y,
	}
	inner := known.mSub(inset.sum())
	hasMain := knownIsDefinite.get(d.main()) && some(known.get(d.main()))
	hasCross := knownIsDefinite.get(d.cross()) && some(known.get(d.cross()))
	gapBasis := inner.or(size{})
	return &flexConstants{
		dir:                     d,
		isRow:                   d.isRow(),
		isColumn:                d.isColumn(),
		isWrap:                  st.Wrap != style.NoWrap,
		isWrapReverse:           st.Wrap == style.WrapReverse,
		minSize:                 transferred(resolveSize(st.MinSize, parent), ratio, known).mAdd(adj),
		maxSize:                 transferred(resolveMax(st.MaxSize, parent), ratio, known).mAdd(adj),
		border:                  border,
		contentBoxInset:         inset,
		paddingBoxInset:         paddingBoxInset,
		padding:                 innerPadding,
		scrollbarGutter:         gutter,
		isScrollContainer:       isScrollContainer(st),
		gap:                     size{resolveOrZero(st.Gap.Width, gapBasis.w), resolveOrZero(st.Gap.Height, gapBasis.h)},
		alignItems:              normalToStretch(st.AlignItems),
		alignContent:            alignContent,
		justifyContent:          justify,
		nodeOuterSize:           known,
		nodeInnerSize:           inner,
		knownMainSizeIsDefinite: knownIsDefinite.get(d.main()),
		hasDefiniteMainSize:     hasMain,
		hasDefiniteCrossSize:    hasCross,
		crossAvailIsDefinite:    hasCross || avail.get(d.cross()).definite(),
	}
}

// transferred is a min or max size with what the aspect ratio transfers from the other
// axis, except into an axis whose size is already known: a stretched item keeps its size.
func transferred(limit size, ratio float32, known size) size {
	t := limit.withAspectRatio(ratio)
	if some(known.w) && !some(limit.w) {
		t.w = none
	}
	if some(known.h) && !some(limit.h) {
		t.h = none
	}
	return t
}

func relativeInset(inset rect, d flexDir) size {
	mainV := or(d.mainStart(inset), or(-d.mainEnd(inset), 0))
	crossV := or(d.crossStart(inset), or(-d.crossEnd(inset), 0))
	return size{}.with(d.main(), mainV).with(d.cross(), crossV)
}

func (t *tree) flexItems(n *node, c *flexConstants) []*flexItem {
	percentBasis := c.nodeInnerSize
	if !c.knownMainSizeIsDefinite {
		percentBasis = percentBasis.with(c.dir.main(), none)
	}
	items := make([]*flexItem, 0, len(n.kids))
	all := make([]flexItem, len(n.kids))
	for i, k := range n.kids {
		st := k.st
		if outOfFlow(st.Position) || st.Display == style.DisplayNone {
			continue
		}
		ratio := aspectRatio(st)
		padding := edges(st.Padding, c.nodeInnerSize.w)
		border := borders(st)
		adj := boxSizingAdjustment(st, padding.add(border).sum())
		it := &all[i]
		*it = flexItem{
			node:           k,
			order:          uint32(i),
			size:           resolveSize(st.Size, percentBasis).withAspectRatio(ratio).mAdd(adj),
			sizeStyle:      st.Size,
			minSize:        resolveSize(st.MinSize, percentBasis).mAdd(adj),
			maxSize:        resolveMax(st.MaxSize, percentBasis).mAdd(adj),
			aspectRatio:    ratio,
			margin:         edges(st.Margin, c.nodeInnerSize.w),
			marginIsAuto:   autoMargins(st),
			padding:        padding,
			border:         border,
			overflow:       st.Overflow,
			scrollbarWidth: st.ScrollbarWidth,
			grow:           st.Grow,
			shrink:         st.Shrink,
		}
		self := st.AlignSelf
		if self == style.AlignAuto {
			self = c.alignItems
		}
		it.alignSelf = normalToStretch(self)
		if st.Position == style.Relative {
			inset := rect{
				l: resolve(st.Inset.Left, c.nodeInnerSize.w), r: resolve(st.Inset.Right, c.nodeInnerSize.w),
				t: resolve(st.Inset.Top, c.nodeInnerSize.h), b: resolve(st.Inset.Bottom, c.nodeInnerSize.h),
			}
			it.relativeInset = relativeInset(inset, c.dir)
		}
		items = append(items, it)
	}
	slices.SortStableFunc(items, func(a, b *flexItem) int { return cmp.Compare(a.node.st.Order, b.node.st.Order) })
	return items
}

func flexAvailableSpace(known size, outer spaces, c *flexConstants) spaces {
	w := outer.w.mSub(c.contentBoxInset.horizontal()).mMax(0)
	if some(known.w) {
		w = def(fmax(known.w-c.contentBoxInset.horizontal(), 0))
	}
	h := outer.h.mSub(c.contentBoxInset.vertical()).mMax(0)
	if some(known.h) {
		h = def(fmax(known.h-c.contentBoxInset.vertical(), 0))
	}
	return spaces{w, h}
}

func automaticMinSize(o style.Size[style.Overflow]) size {
	s := noSize
	if scrollContainer(o.Width) {
		s.w = 0
	}
	if scrollContainer(o.Height) {
		s.h = 0
	}
	return s
}

//nolint:gocognit,nestif // Taffy's determine_flex_base_size, kept in its shape
func (t *tree) flexBaseSizes(c *flexConstants, avail spaces, items []*flexItem) {
	d := c.dir
	for _, it := range items {
		st := it.node.st
		crossParent := c.nodeInnerSize.get(d.cross())
		childParent := noSize.with(d.cross(), crossParent)
		tMin := it.minSize.withAspectRatio(it.aspectRatio)
		tMax := it.maxSize.withAspectRatio(it.aspectRatio)
		minCross, maxCross := tMin.get(d.cross()), tMax.get(d.cross())
		var crossAvail space
		switch a := avail.get(d.cross()); a.kind {
		case definite:
			crossAvail = def(mClamp(fmax(or(crossParent, a.v)-d.crossSum(it.margin), 0), minCross, maxCross))
		case minContent:
			crossAvail = spaceOr(minCross, minContentSpace)
		case maxContent:
			crossAvail = spaceOr(maxCross, maxContentSpace)
		}
		crossIsDefinite := some(it.size.get(d.cross()))
		ckd := it.size.with(d.main(), none)
		ckd.set(d.cross(), mClamp(ckd.get(d.cross()), tMin.get(d.cross()), tMax.get(d.cross())))
		if it.alignSelf == style.AlignStretch && !d.autoCrossStart(it.marginIsAuto) &&
			!d.autoCrossEnd(it.marginIsAuto) && !some(ckd.get(d.cross())) {
			ckd.set(d.cross(), crossAvail.value())
			crossIsDefinite = !c.isWrap && c.hasDefiniteCrossSize && some(crossParent)
		}
		adjMain := it.boxSizingAdjustment().get(d.main())
		percentMain := none
		if c.knownMainSizeIsDefinite {
			percentMain = c.nodeInnerSize.get(d.main())
		}
		basisStyle := st.Basis
		basis := mAdd(resolve(basisStyle, percentMain), adjMain)
		it.flexBasis = func() float32 {
			mainSize := it.size.get(d.main())
			stretch := fmax(mSub(percentMain, d.mainSum(it.margin)), 0)
			if !some(percentMain) {
				stretch = none
			}
			keywordAvail := space{kind: 255}
			if isSizingKeyword(basisStyle) {
				if k, s := resolveSizingKeyword(basisStyle, stretch, percentMain); k == keywordMeasure {
					keywordAvail = s
				}
			} else {
				b := basis
				if !some(b) && basisStyle.Kind == style.Auto {
					b = mainSize
				}
				if some(b) {
					it.basisIsDefinite = true
					return b
				}
				if basisStyle.Kind == style.Auto {
					if k, s := resolveSizingKeyword(d.mainLength(it.sizeStyle), stretch, percentMain); k == keywordMeasure {
						keywordAvail = s
					}
				}
			}
			if crossIsDefinite && some(it.aspectRatio) && some(ckd.get(d.cross())) {
				it.basisIsDefinite = true
				cross := ckd.get(d.cross())
				if d.isRow() {
					return cross * it.aspectRatio
				}
				return cross / it.aspectRatio
			}
			mainAvail := maxContentSpace
			if avail.get(d.main()).kind == minContent {
				mainAvail = minContentSpace
			}
			if keywordAvail.kind != 255 {
				mainAvail = keywordAvail
			}
			childAvail := maxContentSpaces.with(d.main(), mainAvail).with(d.cross(), crossAvail)
			return t.measureChildSize(it.node, ckd, childParent, childAvail, contentSize, d.main(), [2]bool{})
		}()
		pbMain := d.mainSum(it.padding) + d.mainSum(it.border)
		it.flexBasis = fmax(it.flexBasis, pbMain)
		it.innerFlexBasis = it.flexBasis - pbMain
		pbSums := it.padding.add(it.border).sum()
		styleMinMain := it.minSize.or(automaticMinSize(it.overflow)).get(d.main())
		if some(styleMinMain) {
			it.resolvedMinimumMainSize = styleMinMain
		} else {
			childAvail := spaces{minContentSpace, minContentSpace}.with(d.cross(), crossAvail)
			minContentMain := t.measureChildSize(it.node, ckd, childParent, childAvail, contentSize, d.main(), [2]bool{})
			clampedMin := mMin(mMin(minContentMain, it.size.get(d.main())), tMax.get(d.main()))
			it.resolvedMinimumMainSize = mMax(clampedMin, pbSums.get(d.main()))
		}
		hypMin := mMax(mMax(it.resolvedMinimumMainSize, tMin.get(d.main())), pbSums.get(d.main()))
		hypInner := mClamp(it.flexBasis, hypMin, tMax.get(d.main()))
		it.hypotheticalInnerSize.set(d.main(), hypInner)
		it.hypotheticalOuterSize.set(d.main(), hypInner+d.mainSum(it.margin))
	}
}

func collectFlexLines(c *flexConstants, avail spaces, items []*flexItem) []*flexLine {
	d := c.dir
	if !c.isWrap || !c.knownMainSizeIsDefinite {
		return []*flexLine{{items: items}}
	}
	var mainAvail space
	switch maxMain := c.maxSize.get(d.main()); {
	case some(maxMain):
		a := or(avail.get(d.main()).value(), maxMain)
		if !c.hasDefiniteMainSize {
			a = fmin(a, maxMain)
		}
		mainAvail = def(mMax(a, c.minSize.get(d.main())))
	case !d.isRow() && !c.hasDefiniteMainSize && avail.get(d.main()).definite():
		mainAvail = maxContentSpace
	default:
		mainAvail = avail.get(d.main())
	}
	switch mainAvail.kind {
	case maxContent:
		return []*flexLine{{items: items}}
	case minContent:
		lines := make([]*flexLine, 0, len(items))
		for _, it := range items {
			lines = append(lines, &flexLine{items: []*flexItem{it}})
		}
		return lines
	case definite:
	}
	var lines []*flexLine
	gap := c.gap.get(d.main())
	rest := items
	for len(rest) > 0 {
		length := float32(0)
		index := len(rest)
		for i, it := range rest {
			g := gap
			if i == 0 {
				g = 0
			}
			length += it.hypotheticalOuterSize.get(d.main()) + g
			if length > mainAvail.v && i != 0 {
				index = i
				break
			}
		}
		lines = append(lines, &flexLine{items: rest[:index]})
		rest = rest[index:]
	}
	return lines
}

func itemKnownDefiniteness(c *flexConstants, it *flexItem) bools {
	d := c.dir
	mainIsDefinite := c.hasDefiniteMainSize || it.basisIsDefinite
	crossAuto := d.autoCrossStart(it.marginIsAuto) || d.autoCrossEnd(it.marginIsAuto)
	stretched := !crossAuto && it.alignSelf == style.AlignStretch && d.crossLength(it.sizeStyle).Kind == style.Auto
	crossIsDefinite := stretched || some(it.size.get(d.cross())) || (!d.isRow() && c.crossAvailIsDefinite)
	return bools{true, true}.with(d.main(), mainIsDefinite).with(d.cross(), crossIsDefinite)
}

// minContentMainSize is a single-line container's min-content main size: what its items
// can shrink to, side by side, or the widest of them when they may wrap.
func minContentMainSize(lines []*flexLine, c *flexConstants) float32 {
	d := c.dir
	total, widest := float32(0), float32(0)
	for _, l := range lines {
		for _, it := range l.items {
			v := it.resolvedMinimumMainSize
			if it.shrink == 0 {
				v = fmax(v, it.flexBasis)
			}
			v = mMax(v, it.minSize.get(d.main())) + d.mainSum(it.margin)
			total += v
			widest = fmax(widest, v)
		}
		total += sumAxisGaps(c.gap.get(d.main()), len(l.items))
	}
	if c.isWrap {
		return widest
	}
	return total
}

func sumAxisGaps(gap float32, items int) float32 {
	if items <= 1 {
		return 0
	}
	return gap * float32(items-1)
}

func (t *tree) flexContainerMainSize(avail spaces, lines []*flexLine, c *flexConstants) {
	d := c.dir
	inset := d.mainSum(c.contentBoxInset)
	itemMainLength := func(it *flexItem) float32 {
		pb := d.mainSum(it.padding.add(it.border))
		return fmax(mMax(it.flexBasis, it.minSize.get(d.main()))+d.mainSum(it.margin), pb)
	}
	longestLine := func() float32 {
		longest := float32(0)
		for i, l := range lines {
			total := sumAxisGaps(c.gap.get(d.main()), len(l.items))
			for _, it := range l.items {
				total += itemMainLength(it)
			}
			if i == 0 || total > longest {
				longest = total
			}
		}
		return longest
	}
	outer := c.nodeOuterSize.get(d.main())
	if !some(outer) {
		switch a := avail.get(d.main()); {
		case a.kind == definite:
			s := longestLine() + inset
			if len(lines) > 1 {
				s = fmax(s, a.v)
			} else if c.isRow {
				s = fmin(s, fmax(a.v+inset, minContentMainSize(lines, c)+inset))
			}
			outer = s
		case a.kind == minContent && c.isWrap:
			outer = longestLine() + inset
		default:
			outer = t.flexIntrinsicMainSize(avail, lines, c) + inset
		}
	}
	outer = fmax(mClamp(outer, c.minSize.get(d.main()), c.maxSize.get(d.main())), inset-c.scrollbarGutter.get(d.main()))
	inner := fmax(outer-inset, 0)
	c.containerSize.set(d.main(), outer)
	c.innerContainerSize.set(d.main(), inner)
	c.nodeInnerSize.set(d.main(), inner)
}

//nolint:funlen,gocognit // Taffy's determine_container_main_size, kept in its shape
func (t *tree) flexIntrinsicMainSize(avail spaces, lines []*flexLine, c *flexConstants) float32 {
	d := c.dir
	mainSize := float32(0)
	for _, l := range lines {
		for _, it := range l.items {
			styleMin := it.minSize.get(d.main())
			marginSum := d.mainSum(it.margin)
			stylePreferred := it.size.get(d.main())
			styleMax := it.maxSize.get(d.main())
			clampingBasis := it.flexBasis
			if c.isRow {
				clampingBasis = mMax(it.flexBasis, stylePreferred)
			}
			basisMin, basisMax := none, none
			if it.shrink == 0 {
				basisMin = clampingBasis
			}
			if it.grow == 0 {
				basisMax = clampingBasis
			}
			minMain := or(or(mMax(styleMin, basisMin), basisMin), it.resolvedMinimumMainSize)
			minMain = fmax(minMain, it.resolvedMinimumMainSize)
			maxMain := or(or(mMin(styleMax, basisMax), basisMax), float32(math.Inf(1)))
			var contribution float32
			switch {
			case some(stylePreferred) && (maxMain <= minMain || maxMain <= stylePreferred):
				contribution = fmax(fmin(stylePreferred, maxMain), minMain) + marginSum
			case maxMain <= minMain:
				contribution = minMain + marginSum
			case it.isScrollContainer():
				contribution = it.flexBasis + marginSum
			case some(stylePreferred):
				pb := d.mainSum(it.padding) + d.mainSum(it.border)
				innerMain := fmax(stylePreferred, pb)
				if c.isRow {
					contribution = mClamp(innerMain+marginSum, styleMin, styleMax)
				} else {
					contribution = mClamp(fmax(innerMain, it.flexBasis)+marginSum, styleMin, styleMax)
				}
			default:
				crossParent := c.nodeInnerSize.get(d.cross())
				crossAvail := avail.get(d.cross()).mapDefinite(func(v float32) float32 {
					return fmax(or(crossParent, v)-d.crossSum(it.margin), 0)
				}).mClamp(it.minSize.get(d.cross()), it.maxSize.get(d.cross()))
				childAvail := avail.with(d.cross(), crossAvail)
				ckd := it.size.with(d.main(), none)
				tMin := it.minSize.withAspectRatio(it.aspectRatio)
				tMax := it.maxSize.withAspectRatio(it.aspectRatio)
				ckd.set(d.cross(), mClamp(ckd.get(d.cross()), tMin.get(d.cross()), tMax.get(d.cross())))
				if it.alignSelf == style.AlignStretch && !some(ckd.get(d.cross())) {
					ckd.set(d.cross(), crossAvail.value())
				}
				measured := t.measureChildSize(it.node, ckd, c.nodeInnerSize, childAvail, contentSize, d.main(), [2]bool{})
				transferred := none
				if some(it.aspectRatio) && some(ckd.get(d.cross())) {
					if c.isRow {
						transferred = ckd.get(d.cross()) * it.aspectRatio
					} else {
						transferred = ckd.get(d.cross()) / it.aspectRatio
					}
				}
				innerMain := mMax(measured, transferred)
				if c.isRow {
					contribution = mClamp(innerMain+marginSum, styleMin, styleMax)
				} else {
					contribution = mClamp(fmax(innerMain, it.flexBasis)+marginSum, styleMin, styleMax)
				}
			}
			diff := contribution - marginSum - it.flexBasis
			switch {
			case diff > 0:
				it.contentFlexFraction = diff / fmax(1, it.grow)
			case diff < 0:
				it.contentFlexFraction = diff / (fmax(1, it.shrink) * it.innerFlexBasis)
			default:
				it.contentFlexFraction = 0
			}
		}
		floor := c.isWrap && c.isRow
		largest := float32(0)
		sum := float32(0)
		for _, it := range l.items {
			var contribution float32
			switch {
			case it.contentFlexFraction > 0:
				contribution = fmax(1, it.grow) * it.contentFlexFraction
			case it.contentFlexFraction < 0:
				scaled := fmax(1, it.shrink) * it.innerFlexBasis
				if scaled != 0 {
					contribution = scaled * it.contentFlexFraction
				}
			}
			s := it.flexBasis + contribution + d.mainSum(it.margin)
			if floor {
				largest = fmax(largest, s)
			}
			sum += s
		}
		mainSize = fmax(mainSize, fmax(largest, sum+sumAxisGaps(c.gap.get(d.main()), len(l.items))))
	}
	return mainSize
}

//nolint:cyclop,funlen,gocognit,gocyclo,nestif // Taffy's resolve_flexible_lengths, kept in its shape
func resolveFlexibleLengths(l *flexLine, c *flexConstants) {
	d := c.dir
	m := d.main()
	totalGap := sumAxisGaps(c.gap.get(m), len(l.items))
	hypOuter := float32(0)
	for _, it := range l.items {
		hypOuter += it.hypotheticalOuterSize.get(m)
	}
	usedFlexFactor := totalGap + hypOuter
	inner := c.nodeInnerSize.get(m)
	growing := usedFlexFactor < or(inner, 0)
	shrinking := usedFlexFactor > or(inner, 0)
	exact := !growing && !shrinking
	for _, it := range l.items {
		target := it.hypotheticalInnerSize.get(m)
		it.targetSize.set(m, target)
		if exact || (it.grow == 0 && it.shrink == 0) ||
			(growing && it.flexBasis > it.hypotheticalInnerSize.get(m)) ||
			(shrinking && it.flexBasis < it.hypotheticalInnerSize.get(m)) {
			it.frozen = true
			it.outerTargetSize.set(m, target+d.mainSum(it.margin))
		}
	}
	if exact {
		return
	}
	used := totalGap
	for _, it := range l.items {
		if it.frozen {
			used += it.outerTargetSize.get(m)
		} else {
			used += it.flexBasis + d.mainSum(it.margin)
		}
	}
	initialFree := or(mSub(inner, used), 0)
	for {
		allFrozen := true
		itemsSpace, sumGrow, sumShrink := float32(0), float32(0), float32(0)
		for _, it := range l.items {
			if it.frozen {
				itemsSpace += it.outerTargetSize.get(m)
			} else {
				allFrozen = false
				itemsSpace += it.flexBasis + d.mainSum(it.margin)
				sumGrow += it.grow
				sumShrink += it.shrink
			}
		}
		if allFrozen {
			break
		}
		usedSpace := totalGap + itemsSpace
		var free float32
		switch {
		case growing && sumGrow < 1:
			free = mMin(initialFree*sumGrow-totalGap, mSub(inner, usedSpace))
		case shrinking && sumShrink < 1:
			free = mMax(initialFree*sumShrink-totalGap, mSub(inner, usedSpace))
		default:
			free = or(mSub(inner, usedSpace), usedFlexFactor-usedSpace)
		}
		if isNormal(free) {
			if growing && sumGrow > 0 {
				for _, it := range l.items {
					if !it.frozen {
						it.targetSize.set(m, it.flexBasis+free*(it.grow/sumGrow))
					}
				}
			} else if shrinking && sumShrink > 0 {
				sumScaled := float32(0)
				for _, it := range l.items {
					if !it.frozen {
						sumScaled += it.innerFlexBasis * it.shrink
					}
				}
				if sumScaled > 0 {
					for _, it := range l.items {
						if !it.frozen {
							scaled := it.innerFlexBasis * it.shrink
							it.targetSize.set(m, it.flexBasis+free*(scaled/sumScaled))
						}
					}
				}
			}
		}
		totalViolation := float32(0)
		for _, it := range l.items {
			if it.frozen {
				continue
			}
			clamped := fmax(mClamp(it.targetSize.get(m), it.resolvedMinimumMainSize, it.maxSize.get(m)), 0)
			it.violation = clamped - it.targetSize.get(m)
			it.targetSize.set(m, clamped)
			it.outerTargetSize.set(m, clamped+d.mainSum(it.margin))
			totalViolation += it.violation
		}
		for _, it := range l.items {
			if it.frozen {
				continue
			}
			switch {
			case totalViolation > 0:
				it.frozen = it.violation > 0
			case totalViolation < 0:
				it.frozen = it.violation < 0
			default:
				it.frozen = true
			}
		}
	}
}

func (t *tree) hypotheticalCrossSize(l *flexLine, c *flexConstants, nodeSize size, avail spaces) {
	d := c.dir
	crossKnown := !c.isWrap && some(nodeSize.get(d.cross()))
	for _, it := range l.items {
		pb := d.crossSum(it.padding.add(it.border))
		if crossKnown && it.isStretched(d) && !it.baselineAligned(d) {
			continue
		}
		availMain := def(fmax(c.containerSize.get(d.main())-d.mainSum(it.margin), 0))
		tMinCross := it.minSize.withAspectRatio(it.aspectRatio).get(d.cross())
		tMaxCross := it.maxSize.withAspectRatio(it.aspectRatio).get(d.cross())
		childCross := mMax(mClamp(it.size.get(d.cross()), tMinCross, tMaxCross), pb)
		if some(it.aspectRatio) && d.crossLength(it.sizeStyle).Kind == style.Auto {
			// Its cross size follows its flexed main size through the ratio, as
			// Chrome's does, not the main size it had before flexing.
			adj := it.boxSizingAdjustment()
			flexed := noSize.with(d.main(), it.targetSize.get(d.main())-adj.get(d.main())).withAspectRatio(it.aspectRatio)
			childCross = mMax(mClamp(flexed.get(d.cross())+adj.get(d.cross()), tMinCross, tMaxCross), pb)
		}
		availCross := avail.get(d.cross()).mapDefinite(func(v float32) float32 {
			return fmax(v-d.crossSum(it.margin), 0)
		}).mClamp(tMinCross, tMaxCross).mMax(pb)
		stretch := fmax(mSub(c.nodeInnerSize.get(d.cross()), d.crossSum(it.margin)), 0)
		if !some(c.nodeInnerSize.get(d.cross())) {
			stretch = none
		}
		if k, s := resolveSizingKeyword(d.crossLength(it.sizeStyle),
			stretch, c.nodeInnerSize.get(d.cross())); k == keywordMeasure {
			availCross = s
		}
		innerCross := childCross
		if !some(innerCross) {
			var known size
			var av spaces
			if c.isRow {
				known = size{it.targetSize.w, childCross}
				av = spaces{availMain, availCross}
			} else {
				known = size{childCross, it.targetSize.h}
				av = spaces{availCross, availMain}
			}
			out := t.computeChild(it.node, input{
				run:             computeSize,
				sizing:          contentSize,
				axis:            requested(d.cross()),
				known:           known,
				knownIsDefinite: itemKnownDefiniteness(c, it),
				parent:          c.nodeInnerSize,
				avail:           av,
			})
			innerCross = fmax(mClamp(out.size.get(d.cross()), tMinCross, tMaxCross), pb)
		}
		it.hypotheticalInnerSize.set(d.cross(), innerCross)
		it.hypotheticalOuterSize.set(d.cross(), innerCross+d.crossSum(it.margin))
	}
}

func (t *tree) childrenBaselines(nodeSize size, avail spaces, lines []*flexLine, c *flexConstants) {
	if !c.isRow {
		return
	}
	for _, l := range lines {
		count := 0
		for _, it := range l.items {
			if it.baselineAligned(c.dir) {
				count++
			}
		}
		if count <= 1 {
			continue
		}
		for _, it := range l.items {
			if !it.baselineAligned(c.dir) {
				continue
			}
			av := spaces{def(c.containerSize.w), spaceOr(nodeSize.h, avail.h)}.mSub(it.margin.sum()).mMax(size{})
			out := t.computeChild(it.node, input{
				run:             performLayout,
				sizing:          contentSize,
				axis:            axisBoth,
				known:           size{it.targetSize.w, it.hypotheticalInnerSize.h},
				knownIsDefinite: itemKnownDefiniteness(c, it),
				parent:          c.nodeInnerSize,
				avail:           av,
			})
			h := out.size.h
			b := or(childFirstBaseline(out.firstBaseline, h, it.overflow.Height), h)
			it.baseline = b + it.margin.t
		}
	}
}

func calculateCrossSize(lines []*flexLine, nodeSize size, c *flexConstants) {
	d := c.dir
	if !c.isWrap && some(nodeSize.get(d.cross())) {
		pb := d.crossSum(c.contentBoxInset)
		minCross, maxCross := c.minSize.get(d.cross()), c.maxSize.get(d.cross())
		lines[0].crossSize = or(mMax(mSub(mClamp(nodeSize.get(d.cross()), minCross, maxCross), pb), 0), 0)
		return
	}
	for _, l := range lines {
		maxBaseline := float32(0)
		for _, it := range l.items {
			maxBaseline = fmax(maxBaseline, it.baseline)
		}
		l.crossSize = 0
		for _, it := range l.items {
			v := it.hypotheticalOuterSize.get(d.cross())
			if it.baselineAligned(d) {
				v = maxBaseline - it.baseline + v
			}
			l.crossSize = fmax(l.crossSize, v)
		}
	}
	if !c.isWrap {
		pb := d.crossSum(c.contentBoxInset)
		lines[0].crossSize = mClamp(lines[0].crossSize,
			mSub(c.minSize.get(d.cross()), pb), mSub(c.maxSize.get(d.cross()), pb))
	}
}

func alignContentStretch(lines []*flexLine, nodeSize size, c *flexConstants) {
	if c.alignContent.keyword != style.AlignStretch || c.alignContent.safe || c.alignContent.unsafe {
		return
	}
	d := c.dir
	pb := d.crossSum(c.contentBoxInset)
	minCross, maxCross := c.minSize.get(d.cross()), c.maxSize.get(d.cross())
	containerMinInner := or(mMax(mSub(mClamp(or(nodeSize.get(d.cross()), minCross), minCross, maxCross), pb), 0), 0)
	total := sumAxisGaps(c.gap.get(d.cross()), len(lines))
	for _, l := range lines {
		total += l.crossSize
	}
	if total < containerMinInner {
		add := (containerMinInner - total) / float32(len(lines))
		for _, l := range lines {
			l.crossSize += add
		}
	}
}

func determineUsedCrossSize(lines []*flexLine, c *flexConstants) {
	d := c.dir
	for _, l := range lines {
		for _, it := range l.items {
			if it.isStretched(d) {
				maxIgnoringRatio := resolveMax(it.node.st.MaxSize, c.nodeInnerSize).mAdd(it.boxSizingAdjustment())
				it.targetSize.set(d.cross(),
					mClamp(fmax(l.crossSize-d.crossSum(it.margin), 0), it.minSize.get(d.cross()), maxIgnoringRatio.get(d.cross())))
			} else {
				it.targetSize.set(d.cross(), it.hypotheticalInnerSize.get(d.cross()))
			}
			it.outerTargetSize.set(d.cross(), it.targetSize.get(d.cross())+d.crossSum(it.margin))
		}
	}
}

//nolint:gocognit,nestif // Taffy's distribute_remaining_free_space, kept in its shape
func distributeRemainingFreeSpace(lines []*flexLine, c *flexConstants) {
	d := c.dir
	m := d.main()
	for _, l := range lines {
		used := sumAxisGaps(c.gap.get(m), len(l.items))
		for _, it := range l.items {
			used += it.outerTargetSize.get(m)
		}
		free := c.innerContainerSize.get(m) - used
		autos := 0
		for _, it := range l.items {
			if d.autoMainStart(it.marginIsAuto) {
				autos++
			}
			if d.autoMainEnd(it.marginIsAuto) {
				autos++
			}
		}
		if free > 0 && autos > 0 {
			margin := free / float32(autos)
			for _, it := range l.items {
				if d.autoMainStart(it.marginIsAuto) {
					if c.isRow {
						it.margin.l = margin
					} else {
						it.margin.t = margin
					}
				}
				if d.autoMainEnd(it.marginIsAuto) {
					if c.isRow {
						it.margin.r = margin
					} else {
						it.margin.b = margin
					}
				}
			}
			free = 0
		}
		count := len(l.items)
		reverse := d.isReverse()
		mode := applyAlignmentFallback(free, count, c.justifyContent)
		for i := range l.items {
			it := l.items[i]
			idx := i
			if reverse {
				it = l.items[count-1-i]
			}
			it.offsetMain = computeAlignmentOffset(free, count, c.gap.get(m), mode, reverse, idx == 0)
		}
	}
}

func resolveCrossAutoMargins(lines []*flexLine, c *flexConstants) {
	d := c.dir
	for _, l := range lines {
		maxBaseline, maxBelow := float32(0), float32(0)
		for _, it := range l.items {
			maxBaseline = fmax(maxBaseline, it.baseline)
			if it.baselineAligned(d) {
				maxBelow = fmax(maxBelow, it.outerTargetSize.get(d.cross())-it.baseline)
			}
		}
		for _, it := range l.items {
			free := l.crossSize - it.outerTargetSize.get(d.cross())
			start, end := d.autoCrossStart(it.marginIsAuto), d.autoCrossEnd(it.marginIsAuto)
			switch {
			case start && end:
				if c.isRow {
					it.margin.t, it.margin.b = free/2, free/2
				} else {
					it.margin.l, it.margin.r = free/2, free/2
				}
			case start:
				if c.isRow {
					it.margin.t = free
				} else {
					it.margin.l = free
				}
			case end:
				if c.isRow {
					it.margin.b = free
				} else {
					it.margin.r = free
				}
			default:
				it.offsetCross = alignFlexItemCross(it, free, maxBaseline, maxBelow, c)
			}
		}
	}
}

func alignFlexItemCross(it *flexItem, free, maxBaseline, maxBelow float32, c *flexConstants) float32 {
	switch it.alignSelf {
	case style.AlignStart:
		return 0
	case style.AlignFlexStart, style.AlignStretch:
		if c.isWrapReverse {
			return free
		}
		return 0
	case style.AlignEnd:
		return free
	case style.AlignFlexEnd:
		if c.isWrapReverse {
			return 0
		}
		return free
	case style.AlignCenter:
		return free / 2
	case style.AlignBaseline:
		if c.isRow {
			if c.isWrapReverse {
				lineCross := free + it.outerTargetSize.get(c.dir.cross())
				return lineCross - maxBelow - it.baseline
			}
			return maxBaseline - it.baseline
		}
		if c.isWrapReverse {
			return free
		}
		return 0
	case style.AlignNormal, style.AlignAuto, style.AlignSpaceBetween, style.AlignSpaceAround, style.AlignSpaceEvenly:
	}
	return 0
}

func flexContainerCrossSize(lines []*flexLine, nodeSize size, c *flexConstants) float32 {
	d := c.dir
	totalGap := sumAxisGaps(c.gap.get(d.cross()), len(lines))
	total := float32(0)
	for _, l := range lines {
		total += l.crossSize
	}
	pb := d.crossSum(c.contentBoxInset)
	outer := fmax(mClamp(or(nodeSize.get(d.cross()), total+totalGap+pb),
		c.minSize.get(d.cross()), c.maxSize.get(d.cross())), pb-c.scrollbarGutter.get(d.cross()))
	c.containerSize.set(d.cross(), outer)
	c.innerContainerSize.set(d.cross(), fmax(outer-pb, 0))
	return total
}

func alignLines(lines []*flexLine, c *flexConstants, totalCross float32) {
	n := len(lines)
	gap := c.gap.get(c.dir.cross())
	free := c.innerContainerSize.get(c.dir.cross()) - totalCross - sumAxisGaps(gap, n)
	mode := applyAlignmentFallback(free, n, c.alignContent)
	for i := range lines {
		l := lines[i]
		if c.isWrapReverse {
			l = lines[n-1-i]
		}
		l.offsetCross = computeAlignmentOffset(free, n, gap, mode, c.isWrapReverse, i == 0)
	}
}

func (t *tree) flexItemLayout(it *flexItem, totalMain *float32, totalCross,
	lineCross float32, overflow *rect, container overflowContainer, c *flexConstants) {
	d := c.dir
	out := t.computeChild(it.node, input{
		run:             performLayout,
		sizing:          contentSize,
		axis:            axisBoth,
		known:           it.targetSize,
		knownIsDefinite: itemKnownDefiniteness(c, it),
		parent:          c.nodeInnerSize,
		avail: spaces{def(fmax(c.containerSize.w-it.margin.horizontal(), 0)),
			def(fmax(c.containerSize.h-it.margin.vertical(), 0))},
	})
	s := out.size
	offsetMain := *totalMain + it.offsetMain + d.mainStart(it.margin) + it.relativeInset.get(d.main())
	offsetCross := totalCross + it.offsetCross + lineCross + d.crossStart(it.margin) + it.relativeInset.get(d.cross())
	inner := or(childFirstBaseline(out.firstBaseline, s.h, it.overflow.Height), s.h)
	if d.isRow() {
		it.baseline = totalCross + it.offsetCross + lineCross + d.crossStart(it.margin) + inner
	} else {
		it.baseline = *totalMain + it.offsetMain + d.mainStart(it.margin) + inner
	}
	loc := point{offsetCross, offsetMain}
	if d.isRow() {
		loc = point{offsetMain, offsetCross}
	}
	gutter := scrollbarGutter(it.node.st)
	it.node.unrounded = placed{
		order:     it.order,
		size:      s,
		overflow:  out.overflow,
		scrollbar: size{gutter.x, gutter.y},
		location:  loc,
		padding:   it.padding,
		border:    it.border,
		margin:    it.margin,
	}
	it.oof = nil
	if len(out.oof) > 0 {
		it.oof = translateCandidates(out.oof, loc)
	}
	*totalMain += it.offsetMain + d.mainSum(it.margin) + s.get(d.main())
	contributionLoc := point{loc.x - c.paddingBoxInset.l, loc.y - c.paddingBoxInset.t}
	*overflow = overflow.union(overflowContribution(contributionLoc, s,
		out.overflow.rect, it.border, it.overflow, container, it.margin))
}

func (t *tree) flexFinalPass(lines []*flexLine, c *flexConstants) rect {
	d := c.dir
	container := overflowContainer{
		scroll:   c.isScrollContainer,
		size:     c.containerSize.sub(c.border.sum()).sub(size{c.scrollbarGutter.x, c.scrollbarGutter.y}),
		reversed: c.scrollOriginReversed(),
		padding:  c.padding,
	}
	totalCross := d.crossStart(c.contentBoxInset)
	var overflow rect
	layoutLine := func(l *flexLine) {
		totalMain := d.mainStart(c.contentBoxInset)
		if d.isReverse() {
			for i := len(l.items) - 1; i >= 0; i-- {
				t.flexItemLayout(l.items[i], &totalMain, totalCross, l.offsetCross, &overflow, container, c)
			}
		} else {
			for _, it := range l.items {
				t.flexItemLayout(it, &totalMain, totalCross, l.offsetCross, &overflow, container, c)
			}
		}
		totalCross += l.offsetCross + l.crossSize
	}
	if c.isWrapReverse {
		for i := len(lines) - 1; i >= 0; i-- {
			layoutLine(lines[i])
		}
	} else {
		for _, l := range lines {
			layoutLine(l)
		}
	}
	if c.isScrollContainer {
		overflow = overflow.union(rect{
			r: c.contentBoxInset.r - c.border.r - c.scrollbarGutter.x,
			b: c.contentBoxInset.b - c.border.b - c.scrollbarGutter.y,
		})
	}
	return overflow
}

//nolint:funlen // Taffy's perform_absolute_layout_on_absolute_children, kept in its shape
func (t *tree) flexOOFCandidates(n *node, c *flexConstants, lines []*flexLine) []oofCandidate {
	var candidates []oofCandidate
	byOrder := map[uint32]*flexItem{}
	for _, l := range lines {
		for _, it := range l.items {
			byOrder[it.order] = it
		}
	}
	d := c.dir
	mainReversed := d.isReverse()
	crossReversed := c.isWrapReverse
	mainArea := span{d.mainStart(c.contentBoxInset), c.containerSize.get(d.main()) - d.mainEnd(c.contentBoxInset)}
	crossArea := span{d.crossStart(c.contentBoxInset), c.containerSize.get(d.cross()) - d.crossEnd(c.contentBoxInset)}
	for i, k := range n.kids {
		st := k.st
		if st.Display == style.DisplayNone {
			continue
		}
		if !outOfFlow(st.Position) {
			if it := byOrder[uint32(i)]; it != nil {
				candidates = append(candidates, it.oof...)
			}
			continue
		}
		self := st.AlignSelf
		if self == style.AlignAuto {
			self = c.alignItems
		}
		self = normalToStretch(self)
		var mainEdge staticEdge
		switch k := c.justifyContent.keyword; {
		case k == style.AlignStart:
			mainEdge = edgeStart
		case k == style.AlignEnd:
			mainEdge = edgeEnd
		case k == style.AlignSpaceEvenly || k == style.AlignSpaceAround || k == style.AlignCenter:
			mainEdge = edgeCenter
		case (k == style.AlignFlexEnd) != mainReversed:
			mainEdge = edgeEnd
		default:
			mainEdge = edgeStart
		}
		crossEdge := func(k style.Align) staticEdge {
			switch k {
			case style.AlignStart, style.AlignBaseline:
				return edgeStart
			case style.AlignEnd:
				return edgeEnd
			case style.AlignCenter:
				return edgeCenter
			case style.AlignFlexEnd:
				if crossReversed {
					return edgeStart
				}
				return edgeEnd
			case style.AlignNormal, style.AlignAuto, style.AlignStretch, style.AlignFlexStart, style.AlignSpaceBetween,
				style.AlignSpaceAround, style.AlignSpaceEvenly: // at the flex start
			}
			if crossReversed {
				return edgeEnd
			}
			return edgeStart
		}
		selfAlign := alignment{keyword: self}
		cross := staticPosition{area: crossArea, align: staticAlign{
			keyword:  crossEdge(self),
			safe:     selfAlign.safe,
			unsafe:   selfAlign.unsafe,
			fallback: crossEdge(resolveSelfAlignmentSafety(selfAlign, true)),
		}}
		main := staticPosition{area: mainArea, align: alignFromEdge(mainEdge)}
		cand := oofCandidate{node: k, order: uint32(i), position: st.Position, x: main, y: cross}
		if !c.isRow {
			cand.x, cand.y = cross, main
		}
		candidates = append(candidates, cand)
	}
	return candidates
}
