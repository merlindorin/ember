package layout

import (
	"slices"
	"sort"

	"github.com/lantervale/ember/ui/internal/style"
)

type trackEstimate func(t *gridTrack, parent float32) float32

// sizer measures items' contributions to one axis, Taffy's IntrinsicSizeMeasurer.
type sizer struct {
	t        *tree
	other    []gridTrack
	estimate trackEstimate
	axis     axis
	inner    size
}

func (s *sizer) area(it *gridItem, tracks []gridTrack) size {
	return it.gridAreaSizeCached(s.axis, tracks, s.other, s.inner, s.estimate)
}

func (s *sizer) minContent(it *gridItem, tracks []gridTrack) float32 {
	area := s.area(it, tracks)
	avail := area.with(s.axis, none)
	return s.t.minContentContributionCached(it, s.axis, area, avail) + it.marginSums(avail.w).get(s.axis)
}

func (s *sizer) maxContent(it *gridItem, tracks []gridTrack) float32 {
	area := s.area(it, tracks)
	avail := area.with(s.axis, none)
	return s.t.maxContentContributionCached(it, s.axis, area, avail) + it.marginSums(avail.w).get(s.axis)
}

func (s *sizer) minimum(it *gridItem, tracks []gridTrack) float32 {
	area := s.area(it, tracks)
	avail := area.with(s.axis, none)
	return s.t.minimumContributionCached(it, s.axis, tracks, area, s.inner) + it.marginSums(avail.w).get(s.axis)
}

func alignmentGutterAdjustment(a alignment, inner float32, estimate trackEstimate, tracks []gridTrack) float32 {
	if len(tracks) <= 1 {
		return 0
	}
	outer, innerWeight := 0, 0
	switch a.keyword {
	case style.AlignStart, style.AlignFlexStart, style.AlignEnd, style.AlignFlexEnd, style.AlignCenter:
		outer = 1
	case style.AlignSpaceBetween:
		innerWeight = 1
	case style.AlignSpaceAround:
		outer, innerWeight = 1, 2
	case style.AlignSpaceEvenly:
		outer, innerWeight = 1, 1
	case style.AlignNormal, style.AlignAuto, style.AlignStretch, style.AlignBaseline:
	}
	if innerWeight == 0 || !some(inner) {
		return 0
	}
	sum := float32(0)
	for i := range tracks {
		v := estimate(&tracks[i], inner)
		if !some(v) {
			sum = none
			break
		}
		sum += v
	}
	free := float32(0)
	if some(sum) {
		free = fmax(0, inner-sum)
	}
	weighted := ((len(tracks)-3)/2)*innerWeight + 2*outer
	return free / float32(weighted) * float32(innerWeight)
}

func resolveItemTrackIndexes(items []*gridItem, cols, rows trackCounts) {
	for _, it := range items {
		cs, _ := cols.trackVecIndex(it.column.start)
		ce, _ := cols.trackVecIndex(it.column.end)
		rs, _ := rows.trackVecIndex(it.row.start)
		re, _ := rows.trackVecIndex(it.row.end)
		it.columnIndexes = lineSpan{cs, ce}
		it.rowIndexes = lineSpan{rs, re}
	}
}

func markCrossings(items []*gridItem, cols, rows []gridTrack) {
	for _, it := range items {
		it.crossesFlexibleColumn = it.spansTrack(horizontal, cols, (*gridTrack).isFlexible)
		it.crossesIntrinsicColumn = it.spansTrack(horizontal, cols, (*gridTrack).hasIntrinsicSizingFunction)
		it.crossesFlexibleRow = it.spansTrack(vertical, rows, (*gridTrack).isFlexible)
		it.crossesIntrinsicRow = it.spansTrack(vertical, rows, (*gridTrack).hasIntrinsicSizingFunction)
	}
}

// trackSizing is CSS Grid §12.3's track sizing algorithm for one axis, Taffy's
// track_sizing_algorithm.
func (t *tree) trackSizing(a axis, axisMin, axisMax float32,
	axisAlign, otherAlign alignment, availGrid spaces, inner size,
	tracks, other []gridTrack, items []*gridItem, estimate trackEstimate, hasBaselineItem bool) {
	percentBasis := or(inner.get(a), axisMin)
	if !some(inner.get(a)) && !some(axisMin) {
		percentBasis = none
	}
	for i := range tracks {
		tr := &tracks[i]
		tr.baseSize = or(definiteValue(tr.min, percentBasis), 0)
		tr.growthLimit = or(definiteValue(tr.max, percentBasis), inf)
		if tr.growthLimit < tr.baseSize {
			tr.growthLimit = tr.baseSize
		}
	}
	for i := range tracks {
		tracks[i].resolveFitContentLimit(inner.get(a))
	}
	if hasBaselineItem {
		t.resolveItemBaselines(a, items, inner)
	}
	allFixed := true
	for i := range tracks {
		if tracks[i].baseSize != tracks[i].growthLimit || !hasDefiniteValue(tracks[i].min, percentBasis) {
			allFixed = false
			break
		}
	}
	if allFixed {
		return
	}
	adjust := alignmentGutterAdjustment(otherAlign, inner.get(a.other()), estimate, other)
	if len(other) > 3 {
		for i := 2; i < len(other); i += 2 {
			other[i].contentAlignmentAdjustment = adjust
		}
	}
	t.resolveIntrinsicTrackSizes(a, tracks, other, items, availGrid.get(a), inner, estimate)
	maximiseTracks(tracks, availGrid.get(a))
	expansion := maxContentSpace
	switch {
	case some(inner.get(a)):
		expansion = def(inner.get(a))
	case availGrid.get(a).kind == minContent && a == horizontal:
		expansion = minContentSpace
	}
	t.expandFlexibleTracks(a, tracks, other, items, axisMin, axisMax, expansion, inner, estimate)
	if axisAlign.keyword == style.AlignStretch && !axisAlign.safe && !axisAlign.unsafe {
		stretchAutoTracks(tracks, axisMin, expansion)
	}
}

func (t *tree) resolveItemBaselines(a axis, items []*gridItem, inner size) {
	other := a.other()
	sorted := append([]*gridItem(nil), items...)
	sort.SliceStable(sorted,
		func(i, j int) bool { return sorted[i].placement(other).start < sorted[j].placement(other).start })
	for len(sorted) > 0 {
		row := sorted[0].placement(other).start
		n := 1
		for n < len(sorted) && sorted[n].placement(other).start == row {
			n++
		}
		rowItems := sorted[:n]
		sorted = sorted[n:]
		count := 0
		for _, it := range rowItems {
			if it.baselineAligned() {
				count++
			}
		}
		if count <= 1 {
			continue
		}
		for _, it := range rowItems {
			if !it.baselineAligned() {
				continue
			}
			out := t.probeChildLayout(it.node, noSize, inner, spaces{minContentSpace, minContentSpace}, inherentSize)
			h := out.size.h
			it.baseline = or(childFirstBaseline(out.firstBaseline, h, it.st.Overflow.Height), h)
		}
		marginBox := func(it *gridItem) float32 {
			return or(it.baseline, 0) + resolveOrZero(it.st.Margin.Top, inner.w)
		}
		maxBaseline := float32(0)
		first := true
		for _, it := range rowItems {
			if it.baselineAligned() {
				if v := marginBox(it); first || v > maxBaseline {
					maxBaseline = v
				}
				first = false
			}
		}
		for _, it := range rowItems {
			if it.baselineAligned() {
				it.baselineShim = maxBaseline - marginBox(it)
				it.knownCache = nil
			}
		}
	}
}

func flushBaseIncreases(tracks []gridTrack) {
	for i := range tracks {
		tracks[i].baseSize += tracks[i].baseSizePlannedIncrease
		tracks[i].baseSizePlannedIncrease = 0
	}
}

func flushGrowthIncreases(tracks []gridTrack, infinitelyGrowable bool) {
	for i := range tracks {
		tr := &tracks[i]
		if tr.growthLimitPlannedIncrease > 0 {
			if tr.growthLimit == inf {
				tr.growthLimit = tr.baseSize + tr.growthLimitPlannedIncrease
			} else {
				tr.growthLimit += tr.growthLimitPlannedIncrease
			}
			tr.infinitelyGrowable = infinitelyGrowable
		} else {
			tr.infinitelyGrowable = false
		}
		tr.growthLimitPlannedIncrease = 0
	}
}

func crossedFlexFactorSum(tracks []gridTrack) float32 {
	sum := float32(0)
	for i := range tracks {
		if tracks[i].isFlexible() {
			sum += tracks[i].flexFactor()
		}
	}
	return sum
}

//nolint:cyclop,funlen,gocognit,gocyclo,nestif // Taffy's resolve_intrinsic_track_sizes, kept in its shape
func (t *tree) resolveIntrinsicTrackSizes(a axis, tracks, other []gridTrack, items []*gridItem, availGrid space,
	inner size, estimate trackEstimate) {
	var front, back []*gridItem
	needsSort := false
	for _, it := range items {
		if it.crossesFlexible(a) {
			back = append(back, it)
		} else {
			needsSort = needsSort || it.span(a) > 1
			front = append(front, it)
		}
	}
	if needsSort {
		sort.SliceStable(front, func(i, j int) bool { return front[i].span(a) < front[j].span(a) })
	}
	ordered := slices.Concat(front, back)
	innerA := inner.get(a)
	s := &sizer{t: t, other: other, estimate: estimate, axis: a, inner: inner}
	idx := 0
	isFlex := false
	for !isFlex && idx < len(ordered) {
		spanLen := ordered[idx].span(a)
		isFlex = ordered[idx].crossesFlexible(a)
		next := len(ordered)
		if !isFlex {
			for j := idx; j < len(ordered); j++ {
				if ordered[j].crossesFlexible(a) || ordered[j].span(a) > spanLen {
					next = j
					break
				}
			}
		}
		batch := ordered[idx:next]
		idx = next
		batchSpan := batch[0].placement(a).span()
		scrollable := func(it *gridItem) bool {
			if a == horizontal {
				return scrollContainer(it.st.Overflow.Width)
			}
			return scrollContainer(it.st.Overflow.Height)
		}
		if !isFlex && batchSpan == 1 {
			for _, it := range batch {
				ti := it.indexes(a).start + 1
				tr := &tracks[ti]
				newBase := tr.baseSize
				switch {
				case tr.min.Kind == style.MinContent:
					newBase = fmax(tr.baseSize, s.minContent(it, tracks))
				case tr.min.Kind == style.MaxContent:
					newBase = fmax(tr.baseSize, s.maxContent(it, tracks))
				case behavesAsAuto(tr.min, innerA):
					var v float32
					if availGrid.kind != definite {
						minimum := s.minimum(it, tracks)
						minContent := s.minContent(it, tracks)
						v = fmax(mMin(minContent, definiteLimit(tr.max, innerA)), minimum)
					} else {
						v = s.minimum(it, tracks)
					}
					newBase = fmax(tr.baseSize, v)
				}
				tracks[ti].baseSize = newBase
				tr = &tracks[ti]
				minContent := none
				if tr.hasIntrinsicSizingFunction() {
					minContent = s.minContent(it, tracks)
				}
				planned := none
				switch {
				case tr.max.Kind == style.FitContent:
					maxContent := fmin(s.maxContent(it, tracks), tr.fitContentLimit)
					if some(minContent) && !scrollable(it) {
						planned = fmax(minContent, maxContent)
					} else {
						planned = maxContent
					}
				case isMaxContentAlike(tr.max) || (usesPercentage(tr.max) && !some(innerA)):
					planned = s.maxContent(it, tracks)
				case isIntrinsic(tr.max):
					planned = minContent
				}
				if some(planned) {
					tracks[ti].growthLimitPlannedIncrease = fmax(tracks[ti].growthLimitPlannedIncrease, planned)
				}
			}
			for i := range tracks {
				tr := &tracks[i]
				if tr.growthLimitPlannedIncrease > 0 {
					if tr.growthLimit == inf {
						tr.growthLimit = tr.growthLimitPlannedIncrease
					} else {
						tr.growthLimit = fmax(tr.growthLimit, tr.growthLimitPlannedIncrease)
					}
				}
				tr.infinitelyGrowable = false
				tr.growthLimitPlannedIncrease = 0
				if tr.growthLimit < tr.baseSize {
					tr.growthLimit = tr.baseSize
				}
			}
			continue
		}
		hasIntrinsicMin := func(tr *gridTrack) bool { return !some(definiteValue(tr.min, innerA)) }
		for _, it := range batch {
			if !it.crossesIntrinsic(a) {
				continue
			}
			var space float32
			if availGrid.kind != definite {
				minimum := s.minimum(it, tracks)
				minContent := s.minContent(it, tracks)
				limit := it.spannedLimit(a, tracks, innerA, definiteLimit)
				limited := fmax(mMin(minContent, limit), minimum)
				if isFlex {
					from, to := it.trackRange(a)
					spanned := tracks[from:to]
					inflexible := float32(0)
					for i := range spanned {
						if !spanned[i].isFlexible() {
							inflexible += spanned[i].baseSize
						}
					}
					scale := fmin(crossedFlexFactorSum(spanned), 1)
					excess := fmax(limited-inflexible, 0)
					space = fmax(minimum, inflexible+excess*scale)
				} else {
					space = limited
				}
			} else {
				space = s.minimum(it, tracks)
			}
			from, to := it.trackRange(a)
			if space > 0 {
				if scrollable(it) {
					distributeToBaseSize(isFlex, space, tracks[from:to],
						hasIntrinsicMin, (*gridTrack).fitContentLimitedGrowthLimit, true)
				} else {
					distributeToBaseSize(isFlex, space, tracks[from:to], hasIntrinsicMin,
						func(tr *gridTrack) float32 { return tr.growthLimit }, true)
				}
			}
		}
		flushBaseIncreases(tracks)
		minOrMax := func(tr *gridTrack) bool { return isMinOrMaxContent(tr.min) }
		for _, it := range batch {
			if !it.spansTrack(a, tracks, minOrMax) {
				continue
			}
			space := s.minContent(it, tracks)
			from, to := it.trackRange(a)
			if space > 0 {
				if scrollable(it) {
					distributeToBaseSize(isFlex, space, tracks[from:to], minOrMax, (*gridTrack).fitContentLimitedGrowthLimit, true)
				} else {
					distributeToBaseSize(isFlex, space, tracks[from:to], minOrMax,
						func(tr *gridTrack) float32 { return tr.growthLimit }, true)
				}
			}
		}
		flushBaseIncreases(tracks)
		maxContentMin := func(tr *gridTrack) bool { return tr.min.Kind == style.MaxContent }
		if availGrid.kind == maxContent {
			autoMin := func(tr *gridTrack) bool { return tr.min.Kind == style.Auto && tr.max.Kind != style.MinContent }
			for _, it := range batch {
				if !it.spansTrack(a, tracks, func(tr *gridTrack) bool { return autoMin(tr) || maxContentMin(tr) }) {
					continue
				}
				maxContentSize := s.maxContent(it, tracks)
				limit := it.spannedLimit(a, tracks, innerA, definiteLimit)
				space := mMin(maxContentSize, limit)
				from, to := it.trackRange(a)
				if isFlex {
					spanned := tracks[from:to]
					inflexible := float32(0)
					for i := range spanned {
						if !spanned[i].isFlexible() {
							inflexible += spanned[i].baseSize
						}
					}
					scale := fmin(crossedFlexFactorSum(spanned), 1)
					space = inflexible + fmax(space-inflexible, 0)*scale
				}
				if space > 0 {
					anyMaxContent := false
					for i := from; i < to; i++ {
						if maxContentMin(&tracks[i]) {
							anyMaxContent = true
						}
					}
					if anyMaxContent {
						distributeToBaseSize(isFlex, space, tracks[from:to],
							maxContentMin, func(*gridTrack) float32 { return inf }, false)
					} else {
						distributeToBaseSize(isFlex, space, tracks[from:to], autoMin, (*gridTrack).fitContentLimitedGrowthLimit, false)
					}
				}
			}
			flushBaseIncreases(tracks)
		}
		for _, it := range batch {
			if !it.spansTrack(a, tracks, maxContentMin) {
				continue
			}
			space := s.maxContent(it, tracks)
			from, to := it.trackRange(a)
			if space > 0 {
				distributeToBaseSize(isFlex, space, tracks[from:to], maxContentMin,
					func(tr *gridTrack) float32 { return tr.growthLimit }, false)
			}
		}
		flushBaseIncreases(tracks)
		for i := range tracks {
			if tracks[i].growthLimit < tracks[i].baseSize {
				tracks[i].growthLimit = tracks[i].baseSize
			}
		}
		if !isFlex {
			intrinsicMax := func(tr *gridTrack) bool { return !hasDefiniteValue(tr.max, innerA) }
			for _, it := range batch {
				if !it.spansTrack(a, tracks, intrinsicMax) {
					continue
				}
				space := s.minContent(it, tracks)
				from, to := it.trackRange(a)
				if space > 0 {
					distributeToGrowthLimit(space, tracks[from:to], intrinsicMax)
				}
			}
			flushGrowthIncreases(tracks, true)
			maxContentMax := func(tr *gridTrack) bool {
				return isMaxContentAlike(tr.max) || (usesPercentage(tr.max) && !some(innerA))
			}
			for _, it := range batch {
				if !it.spansTrack(a, tracks, maxContentMax) {
					continue
				}
				space := s.maxContent(it, tracks)
				from, to := it.trackRange(a)
				if space > 0 {
					distributeToGrowthLimit(space, tracks[from:to], maxContentMax)
				}
			}
			flushGrowthIncreases(tracks, false)
		}
	}
	for i := range tracks {
		if tracks[i].growthLimit == inf {
			tracks[i].growthLimit = tracks[i].baseSize
		}
	}
}

// distributeToBaseSize distributes an item's extra space over its tracks' base sizes; minimum
// says whether it is a minimum contribution (else a maximum one).
func distributeToBaseSize(isFlex bool, space float32, tracks []gridTrack,
	affected func(*gridTrack) bool, limit func(*gridTrack) float32, minimum bool) {
	proportion := func(*gridTrack) float32 { return 1 }
	if isFlex {
		inner := affected
		affected = func(tr *gridTrack) bool { return tr.isFlexible() && inner(tr) }
		sum := float32(0)
		for i := range tracks {
			if affected(&tracks[i]) {
				sum += tracks[i].flexFactor()
			}
		}
		if sum > 0 {
			proportion = (*gridTrack).flexFactor
		}
	}
	hit := false
	for i := range tracks {
		if affected(&tracks[i]) {
			hit = true
		}
	}
	if space == 0 || !hit {
		return
	}
	base := func(tr *gridTrack) float32 { return tr.baseSize }
	sizes := float32(0)
	for i := range tracks {
		sizes += tracks[i].baseSize
	}
	extra := fmax(0, space-sizes)
	extra = distributeUpToLimits(extra, tracks, affected, proportion, base, limit)
	if extra > 0.000001 {
		filter := func(tr *gridTrack) bool { return isIntrinsic(tr.max) }
		if !minimum {
			filter = func(tr *gridTrack) bool { return isMaxOrFitContent(tr.max) }
		}
		count := 0
		for i := range tracks {
			if affected(&tracks[i]) && filter(&tracks[i]) {
				count++
			}
		}
		if count == 0 {
			filter = func(*gridTrack) bool { return true }
		}
		distributeUpToLimits(extra, tracks, func(tr *gridTrack) bool { return affected(tr) && filter(tr) }, proportion, base,
			func(tr *gridTrack) float32 { return tr.fitContentLimit })
	}
	for i := range tracks {
		tr := &tracks[i]
		if tr.itemIncurredIncrease > tr.baseSizePlannedIncrease {
			tr.baseSizePlannedIncrease = tr.itemIncurredIncrease
		}
		tr.itemIncurredIncrease = 0
	}
}

func distributeToGrowthLimit(space float32, tracks []gridTrack, affected func(*gridTrack) bool) {
	hit := false
	for i := range tracks {
		if affected(&tracks[i]) {
			hit = true
		}
	}
	if space == 0 || !hit {
		return
	}
	limitOrBase := func(tr *gridTrack) float32 {
		if tr.growthLimit == inf {
			return tr.baseSize
		}
		return tr.growthLimit
	}
	sizes := float32(0)
	for i := range tracks {
		sizes += limitOrBase(&tracks[i])
	}
	extra := fmax(0, space-sizes)
	fit := func(tr *gridTrack) float32 { return tr.fitContentLimit }
	one := func(*gridTrack) float32 { return 1 }
	extra = distributeUpToLimits(extra, tracks, func(tr *gridTrack) bool {
		return affected(tr) && (tr.infinitelyGrowable || tr.growthLimit == inf)
	}, one, limitOrBase, fit)
	if extra > 0 {
		distributeUpToLimits(extra, tracks, affected, one, limitOrBase, fit)
	}
	for i := range tracks {
		tr := &tracks[i]
		if tr.itemIncurredIncrease > tr.growthLimitPlannedIncrease {
			tr.growthLimitPlannedIncrease = tr.itemIncurredIncrease
		}
		tr.itemIncurredIncrease = 0
	}
}

func maximiseTracks(tracks []gridTrack, availGrid space) {
	used := float32(0)
	for i := range tracks {
		used += tracks[i].baseSize
	}
	free := availGrid.freeSpace(used)
	if free == inf {
		for i := range tracks {
			tracks[i].baseSize = tracks[i].growthLimit
		}
		return
	}
	if free > 0 {
		distributeUpToLimits(free, tracks, func(*gridTrack) bool { return true }, func(*gridTrack) float32 { return 1 },
			func(tr *gridTrack) float32 { return tr.baseSize }, (*gridTrack).fitContentLimitedGrowthLimit)
		for i := range tracks {
			tracks[i].baseSize += tracks[i].itemIncurredIncrease
			tracks[i].itemIncurredIncrease = 0
		}
	}
}

func (t *tree) expandFlexibleTracks(a axis, tracks, other []gridTrack, items []*gridItem,
	axisMin, axisMax float32, expansion space, inner size, estimate trackEstimate) {
	s := &sizer{t: t, other: other, estimate: estimate, axis: a, inner: inner}
	var fraction float32
	switch expansion.kind {
	case definite:
		used := float32(0)
		for i := range tracks {
			used += tracks[i].baseSize
		}
		if expansion.v-used > 0 {
			fraction = findSizeOfFr(tracks, expansion.v)
		}
	case maxContent:
		fromTracks := float32(0)
		for i := range tracks {
			if tracks[i].max.Kind != style.Fr {
				continue
			}
			f := tracks[i].flexFactor()
			v := tracks[i].baseSize
			if f > 1 {
				v = tracks[i].baseSize / f
			}
			fromTracks = fmax(fromTracks, v)
		}
		fromItems := float32(0)
		for _, it := range items {
			if !it.crossesFlexible(a) {
				continue
			}
			contribution := s.maxContent(it, tracks)
			from, to := it.trackRange(a)
			fromItems = fmax(fromItems, findSizeOfFr(tracks[from:to], contribution))
		}
		fraction = fmax(fromTracks, fromItems)
		hypothetical := float32(0)
		for i := range tracks {
			if tracks[i].max.Kind == style.Fr {
				hypothetical += fmax(tracks[i].baseSize, tracks[i].flexFactor()*fraction)
			} else {
				hypothetical += tracks[i].baseSize
			}
		}
		lo, hi := or(axisMin, 0), or(axisMax, inf)
		if hypothetical < lo {
			fraction = findSizeOfFr(tracks, lo)
		} else if hypothetical > hi {
			fraction = findSizeOfFr(tracks, hi)
		}
	case minContent:
	}
	for i := range tracks {
		if tracks[i].max.Kind == style.Fr {
			tracks[i].baseSize = fmax(tracks[i].baseSize, tracks[i].flexFactor()*fraction)
		}
	}
}

func findSizeOfFr(tracks []gridTrack, fill float32) float32 {
	if fill == 0 {
		return 0
	}
	hypothetical := inf
	for iter := 0; iter < len(tracks)+1; iter++ {
		used, sum := float32(0), float32(0)
		for i := range tracks {
			tr := &tracks[i]
			if tr.max.Kind == style.Fr && tr.flexFactor()*hypothetical >= tr.baseSize {
				sum += tr.flexFactor()
			} else {
				used += tr.baseSize
			}
		}
		previous := hypothetical
		hypothetical = (fill - used) / fmax(sum, 1)
		valid := true
		for i := range tracks {
			tr := &tracks[i]
			if tr.max.Kind == style.Fr {
				f := tr.flexFactor()
				if !(f*hypothetical >= tr.baseSize || f*previous < tr.baseSize) {
					valid = false
					break
				}
			}
		}
		if valid {
			break
		}
	}
	return hypothetical
}

func stretchAutoTracks(tracks []gridTrack, axisMin float32, expansion space) {
	count := 0
	for i := range tracks {
		if tracks[i].max.Kind == style.Auto {
			count++
		}
	}
	if count == 0 {
		return
	}
	used := float32(0)
	for i := range tracks {
		used += tracks[i].baseSize
	}
	var free float32
	switch {
	case expansion.definite():
		free = expansion.freeSpace(used)
	case some(axisMin):
		free = axisMin - used
	}
	if free > 0 {
		extra := free / float32(count)
		for i := range tracks {
			if tracks[i].max.Kind == style.Auto {
				tracks[i].baseSize += extra
			}
		}
	}
}

func distributeUpToLimits(space float32, tracks []gridTrack, affected func(*gridTrack) bool,
	proportion, property, limit func(*gridTrack) float32) float32 {
	const threshold = 0.01
	for iter := 0; iter < len(tracks)+1; iter++ {
		if space <= threshold {
			break
		}
		sum := float32(0)
		minIncrease := inf
		found := false
		for i := range tracks {
			tr := &tracks[i]
			if property(tr)+tr.itemIncurredIncrease < limit(tr) && affected(tr) {
				sum += proportion(tr)
				v := (limit(tr) - property(tr) - tr.itemIncurredIncrease) / proportion(tr)
				if !found || v < minIncrease {
					minIncrease = v
				}
				found = true
			}
		}
		if sum == 0 {
			break
		}
		step := fmin(minIncrease, space/sum)
		for i := range tracks {
			tr := &tracks[i]
			if !affected(tr) {
				continue
			}
			inc := step * proportion(tr)
			if inc > 0 && property(tr)+tr.itemIncurredIncrease+inc <= limit(tr)+threshold {
				tr.itemIncurredIncrease += inc
				space -= inc
			}
		}
	}
	return space
}
