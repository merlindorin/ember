package layout

import (
	"math"

	"github.com/lantervale/ember/ui/internal/style"
)

const maxGridTracks = 10000

// trackCounts is how many tracks lie before the explicit grid, in it, and after it.
type trackCounts struct{ negative, explicit, positive int }

func (c trackCounts) len() int           { return c.negative + c.explicit + c.positive }
func (c trackCounts) implicitStart() int { return -c.negative }
func (c trackCounts) implicitEnd() int   { return c.explicit + c.positive }
func (c trackCounts) ozToTrack(l int) int {
	return l + c.negative
}
func (c trackCounts) trackToOz(i int) int { return i - c.negative }

// trackVecIndex is the index of an origin-zero line in a track list that alternates
// gutters and tracks, false outside the grid.
func (c trackCounts) trackVecIndex(l int) (int, bool) {
	if l < -c.negative || l > c.explicit+c.positive {
		return 0, false
	}
	return 2 * (l + c.negative), true
}

type trackKind uint8

const (
	trackTrack trackKind = iota
	trackGutter
)

// gridTrack is a track or a gutter, Taffy's GridTrack.
type gridTrack struct {
	kind                       trackKind
	collapsed                  bool
	min, max                   style.Length
	offset                     float32
	baseSize                   float32
	growthLimit                float32
	contentAlignmentAdjustment float32
	itemIncurredIncrease       float32
	baseSizePlannedIncrease    float32
	growthLimitPlannedIncrease float32
	infinitelyGrowable         bool
	fitContentLimit            float32
}

var inf = float32(math.Inf(1)) //nolint:gochecknoglobals // +Inf is never a constant

func newTrack(t style.Track) gridTrack {
	lo := t.Min
	if lo.Kind == style.Fr || lo.Kind == style.FitContent || lo.Kind == style.None {
		lo = style.Length{}
	}
	hi := t.Max
	if hi.Kind == style.None {
		hi = style.Length{}
	}
	return gridTrack{kind: trackTrack, min: lo, max: hi, fitContentLimit: inf}
}

func gutterTrack(gap style.Length) gridTrack {
	if gap.Kind != style.Fixed {
		gap = style.Px(0)
	}
	return gridTrack{kind: trackGutter, min: gap, max: gap, fitContentLimit: inf}
}

func (t *gridTrack) collapse() {
	t.collapsed = true
	t.min, t.max = style.Px(0), style.Px(0)
	t.fitContentLimit = inf
}

func (t *gridTrack) isFlexible() bool { return t.max.Kind == style.Fr }

func (t *gridTrack) usesPercentage() bool { return usesPercentage(t.min) || usesPercentage(t.max) }

func (t *gridTrack) hasIntrinsicSizingFunction() bool {
	return isIntrinsic(t.min) || isIntrinsic(t.max)
}

func (t *gridTrack) resolveFitContentLimit(parent float32) {
	t.fitContentLimit = or(fitContentLimit(t.max, parent), inf)
}

func (t *gridTrack) fitContentLimitedGrowthLimit() float32 {
	return fmin(t.growthLimit, t.fitContentLimit)
}

func (t *gridTrack) flexFactor() float32 {
	if t.max.Kind == style.Fr {
		return t.max.Px
	}
	return 0
}

// Track sizing functions, as Taffy's Min/MaxTrackSizingFunction read them.

func definiteValue(l style.Length, parent float32) float32 {
	if l.Kind != style.Fixed {
		return none
	}
	return resolve(l, parent)
}

func hasDefiniteValue(l style.Length, parent float32) bool { return some(definiteValue(l, parent)) }

func fitContentLimit(l style.Length, parent float32) float32 {
	if l.Kind != style.FitContent {
		return none
	}
	if !hasPercent(l) {
		return l.Px
	}
	if !some(parent) {
		return none
	}
	return l.Px + l.Percent*parent/100
}

func definiteLimit(l style.Length, parent float32) float32 {
	if l.Kind == style.FitContent {
		return fitContentLimit(l, parent)
	}
	return definiteValue(l, parent)
}

func isIntrinsic(l style.Length) bool {
	switch l.Kind {
	case style.Auto, style.MinContent, style.MaxContent, style.FitContent:
		return true
	case style.Fixed, style.Fr, style.None:
	}
	return false
}

func isMaxContentAlike(l style.Length) bool {
	return l.Kind == style.Auto || l.Kind == style.MaxContent || l.Kind == style.FitContent
}

func isMaxOrFitContent(l style.Length) bool {
	return l.Kind == style.MaxContent || l.Kind == style.FitContent
}

func isMinOrMaxContent(l style.Length) bool {
	return l.Kind == style.MinContent || l.Kind == style.MaxContent
}

func usesPercentage(l style.Length) bool {
	return (l.Kind == style.Fixed || l.Kind == style.FitContent) && hasPercent(l)
}

func behavesAsAuto(l style.Length, parent float32) bool {
	return l.Kind == style.Auto || (!some(parent) && usesPercentage(l))
}

type autoRepeatStrategy uint8

const (
	maxRepetitionsThatDoNotOverflow autoRepeatStrategy = iota
	minRepetitionsThatDoOverflow
)

func templateOf(st *style.Computed, a axis) ([]style.TrackRepeat, []style.Track, style.Length) {
	if a == horizontal {
		return st.Columns, st.AutoColumns, st.Gap.Width
	}
	return st.Rows, st.AutoRows, st.Gap.Height
}

func isAutoRepeat(r style.TrackRepeat) bool {
	return r.Count == style.AutoFill || r.Count == style.AutoFit
}

// explicitGridSize is how many times an auto-fill or auto-fit repeat() repeats, and the
// explicit grid's track count, in one axis.
//
//nolint:funlen,gocognit,nestif // Taffy's compute_explicit_grid_size_in_axis, kept in its shape
func explicitGridSize(st *style.Computed, containerSize float32, strategy autoRepeatStrategy,
	a axis) (int, int) {
	template, _, gap := templateOf(st, a)
	if len(template) == 0 {
		return 0, 0
	}
	nonAuto, autos := 0, 0
	allFixed := true
	for _, r := range template {
		if len(r.Tracks) == 0 {
			return 0, 0
		}
		if isAutoRepeat(r) {
			autos++
		} else {
			nonAuto += r.Count * len(r.Tracks)
		}
		for _, tr := range r.Tracks {
			if tr.Min.Kind != style.Fixed && tr.Max.Kind != style.Fixed {
				allFixed = false
			}
		}
	}
	nonAuto = min(nonAuto, maxGridTracks)
	if autos > 1 || (autos == 1 && !allFixed) {
		return 0, 0
	}
	if autos == 0 {
		return 0, nonAuto
	}
	insertion := 0
	var rep style.TrackRepeat
	for _, r := range template {
		if isAutoRepeat(r) {
			rep = r
			break
		}
		insertion += r.Count * len(r.Tracks)
	}
	repTracks := len(rep.Tracks)
	num := 1
	if some(containerSize) {
		trackValue := func(tr style.Track) float32 {
			maxV := definiteValue(tr.Max, containerSize)
			minV := definiteValue(tr.Min, containerSize)
			if some(maxV) {
				return mMax(maxV, minV)
			}
			return minV
		}
		nonRepeating := float32(0)
		for _, r := range template {
			if isAutoRepeat(r) {
				continue
			}
			sum := float32(0)
			for _, tr := range r.Tracks {
				sum += trackValue(tr)
			}
			nonRepeating += sum * float32(r.Count)
		}
		gapSize := resolveOrZero(gap, containerSize)
		perRep := float32(0)
		for _, tr := range rep.Tracks {
			perRep += trackValue(tr)
		}
		first := nonRepeating + perRep + float32(max(nonAuto+repTracks-1, 0))*gapSize
		if first <= containerSize {
			perRepUsed := perRep + float32(repTracks)*gapSize
			fit := float64((containerSize - first) / perRepUsed)
			if strategy == maxRepetitionsThatDoNotOverflow {
				num = int(math.Floor(fit)) + 1
			} else {
				num = int(math.Ceil(fit)) + 1
			}
		}
	}
	remaining := max(maxGridTracks-insertion, 0)
	if remaining == 0 {
		num = 0
	} else {
		maxReps := (remaining + repTracks - 1) / repTracks
		num = min(max(num, 1), maxReps)
	}
	return num, min(nonAuto+repTracks*num, maxGridTracks)
}

// initializeGridTracks lays out the track list of one axis: a gutter, then each track
// followed by a gutter, with auto-fit tracks that hold no item collapsed.
//
//nolint:nestif // Taffy's initialize_grid_tracks, kept in its shape
func initializeGridTracks(counts trackCounts, st *style.Computed,
	a axis, autoRepetitions int, hasItems func(int) bool) []gridTrack {
	template, autoTracks, gap := templateOf(st, a)
	tracks := make([]gridTrack, 0, counts.len()*2+1)
	tracks = append(tracks, gutterTrack(gap))
	implicit := func(n int, next func(i int) style.Track) {
		for i := 0; i < n; i++ {
			tracks = append(tracks, newTrack(next(i)), gutterTrack(gap))
		}
	}
	auto := func(int) style.Track { return style.Track{} }
	if counts.negative > 0 {
		if len(autoTracks) == 0 {
			implicit(counts.negative, auto)
		} else {
			offset := len(autoTracks) - counts.negative%len(autoTracks)
			implicit(counts.negative, func(i int) style.Track { return autoTracks[(offset+i)%len(autoTracks)] })
		}
	}
	current := counts.negative
	limit := counts.negative + counts.explicit
	if counts.explicit > 0 {
		for _, r := range template {
			if !isAutoRepeat(r) {
				n := min(len(r.Tracks)*r.Count, limit-current)
				for i := 0; i < n; i++ {
					tracks = append(tracks, newTrack(r.Tracks[i%len(r.Tracks)]), gutterTrack(gap))
					current++
				}
				continue
			}
			n := min(len(r.Tracks)*autoRepetitions, limit-current)
			for i := 0; i < n; i++ {
				track := newTrack(r.Tracks[i%len(r.Tracks)])
				gutter := gutterTrack(gap)
				if r.Count == style.AutoFit && !hasItems(current) {
					track.collapse()
					gutter.collapse()
				}
				tracks = append(tracks, track, gutter)
				current++
			}
			if r.Count == style.AutoFit && current == counts.len() {
				for i := len(tracks) - 1; i >= 0; i-- {
					if tracks[i].kind == trackTrack && !tracks[i].collapsed {
						break
					}
					tracks[i].collapse()
				}
			}
		}
	}
	areaTracks := counts.negative + counts.explicit - current
	if len(autoTracks) == 0 {
		implicit(counts.positive+areaTracks, auto)
	} else {
		implicit(counts.positive+areaTracks, func(i int) style.Track { return autoTracks[i%len(autoTracks)] })
	}
	tracks[0].collapse()
	tracks[len(tracks)-1].collapse()
	return tracks
}
