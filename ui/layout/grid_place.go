package layout

import "github.com/lantervale/ember/ui/internal/style"

const (
	minOZLine = -maxGridTracks
	maxOZLine = maxGridTracks
)

type placementKind uint8

const (
	placeAuto placementKind = iota
	placeLine
	placeSpan
)

// ozPlacement is one end of a grid placement with lines counted from zero at the explicit
// grid's start, Taffy's OriginZeroGridPlacement.
type ozPlacement struct {
	kind placementKind
	v    int
}

type ozLines struct{ start, end ozPlacement }

type lineSpan struct{ start, end int }

func (l lineSpan) span() int { return max(l.end-l.start, 0) }

func clampOZ(l int) int { return min(max(l, minOZLine), maxOZLine) }

func toOriginZero(g style.GridLine, explicit int) ozPlacement {
	switch {
	case g.Line > 0:
		return ozPlacement{placeLine, clampOZ(g.Line - 1)}
	case g.Line < 0:
		return ozPlacement{placeLine, clampOZ(g.Line + explicit + 1)}
	case g.Span > 0:
		return ozPlacement{placeSpan, min(max(g.Span, 1), maxGridTracks)}
	}
	return ozPlacement{}
}

func placementOf(lines [2]style.GridLine, explicit int) ozLines {
	return ozLines{toOriginZero(lines[0], explicit), toOriginZero(lines[1], explicit)}
}

func (p ozLines) isDefinite() bool { return p.start.kind == placeLine || p.end.kind == placeLine }

func (p ozLines) indefiniteSpan() int {
	switch {
	case p.start.kind == placeSpan:
		return p.start.v
	case p.end.kind == placeSpan:
		return p.end.v
	}
	return 1
}

func (p ozLines) resolveDefinite() lineSpan {
	s, e := p.start, p.end
	switch {
	case s.kind == placeLine && e.kind == placeLine:
		if s.v == e.v {
			return lineSpan{s.v, s.v + 1}
		}
		return lineSpan{min(s.v, e.v), max(s.v, e.v)}
	case s.kind == placeLine && e.kind == placeSpan:
		return lineSpan{s.v, s.v + e.v}
	case s.kind == placeLine:
		return lineSpan{s.v, s.v + 1}
	case s.kind == placeSpan:
		return lineSpan{e.v - s.v, e.v}
	}
	return lineSpan{e.v - 1, e.v}
}

// resolveAbsolute is an absolutely positioned item's lines, ok false where an end is auto.
func (p ozLines) resolveAbsolute() (int, int, bool, bool) {
	s, e := p.start, p.end
	switch {
	case s.kind == placeLine && e.kind == placeLine:
		if s.v == e.v {
			return s.v, s.v + 1, true, true
		}
		return min(s.v, e.v), max(s.v, e.v), true, true
	case s.kind == placeLine && e.kind == placeSpan:
		return s.v, s.v + e.v, true, true
	case s.kind == placeLine:
		return s.v, 0, true, false
	case s.kind == placeSpan && e.kind == placeLine:
		return e.v - s.v, e.v, true, true
	case s.kind == placeAuto && e.kind == placeLine:
		return 0, e.v, false, true
	}
	return 0, 0, false, false
}

type itemPlacement struct{ h, v ozLines }

func (p itemPlacement) get(a axis) ozLines {
	if a == horizontal {
		return p.h
	}
	return p.v
}

// gridSizeEstimate is the track counts the definite placements need, before auto-placement.
func gridSizeEstimate(explicitCols, explicitRows int, placements []itemPlacement) (trackCounts, trackCounts) {
	colMin, colMax, colSpan, rowMin, rowMax, rowSpan := 0, 0, 0, 0, 0, 0
	for _, p := range placements {
		a, b, s := minMaxSpan(p.h)
		colMin, colMax, colSpan = min(colMin, a), max(colMax, b), max(colSpan, s)
		a, b, s = minMaxSpan(p.v)
		rowMin, rowMax, rowSpan = min(rowMin, a), max(rowMax, b), max(rowSpan, s)
	}
	negCols := max(-colMin, 0)
	posCols := max(colMax-explicitCols, 0)
	negRows := max(-rowMin, 0)
	posRows := max(rowMax-explicitRows, 0)
	if negCols+explicitCols+posCols < colSpan {
		posCols = colSpan - explicitCols - negCols
	}
	if negRows+explicitRows+posRows < rowSpan {
		posRows = rowSpan - explicitRows - negRows
	}
	return trackCounts{negCols, explicitCols, posCols}, trackCounts{negRows, explicitRows, posRows}
}

func minMaxSpan(p ozLines) (int, int, int) {
	s, e := p.start, p.end
	var lo, hi int
	switch {
	case s.kind == placeLine && e.kind == placeLine:
		if s.v == e.v {
			lo, hi = s.v, s.v+1
		} else {
			lo, hi = min(s.v, e.v), max(s.v, e.v)
		}
	case s.kind == placeLine && e.kind == placeAuto:
		lo, hi = s.v, s.v+1
	case s.kind == placeLine:
		lo, hi = s.v, s.v+e.v
	case s.kind == placeAuto && e.kind == placeLine:
		lo, hi = e.v-1, e.v
	case s.kind == placeSpan && e.kind == placeLine:
		lo, hi = e.v-s.v, e.v
	}
	span := 1
	if s.kind != placeLine && e.kind != placeLine {
		span = p.indefiniteSpan()
	}
	return max(lo, minOZLine), min(hi, maxOZLine), span
}

type cellState uint8

const (
	unoccupied cellState = iota
	definitelyPlaced
	autoPlaced
)

type interval struct {
	start, end int
	state      cellState
}

// trackIntervals is the occupied ranges along one track.
type trackIntervals struct {
	intervals []interval
	cursor    int
	hasCursor bool
}

func (t *trackIntervals) empty() bool { return len(t.intervals) == 0 }

func (t *trackIntervals) paint(start, end int, state cellState) {
	if start >= end {
		return
	}
	if n := len(t.intervals); n == 0 {
		t.intervals = append(t.intervals, interval{start, end, state})
		return
	} else if last := &t.intervals[n-1]; last.end <= start {
		if last.state == state && last.end == start {
			last.end = end
		} else {
			t.intervals = append(t.intervals, interval{start, end, state})
		}
		return
	}
	var result []interval
	for _, iv := range t.intervals {
		if iv.end <= start {
			result = append(result, iv)
		} else if iv.start < start {
			result = append(result, interval{iv.start, start, iv.state})
		}
	}
	if n := len(result); n > 0 && result[n-1].state == state && result[n-1].end == start {
		result[n-1].end = end
	} else {
		result = append(result, interval{start, end, state})
	}
	for _, iv := range t.intervals {
		var trimmed interval
		switch {
		case iv.start >= end:
			trimmed = iv
		case iv.end > end:
			trimmed = interval{end, iv.end, iv.state}
		default:
			continue
		}
		if n := len(result); n > 0 && result[n-1].state == trimmed.state && result[n-1].end == trimmed.start {
			result[n-1].end = trimmed.end
		} else {
			result = append(result, trimmed)
		}
	}
	t.intervals = result
}

func (t *trackIntervals) collisionExtent(start, end int) (int, bool) {
	for i := len(t.intervals) - 1; i >= 0; i-- {
		iv := t.intervals[i]
		if iv.end <= start {
			break
		}
		if iv.start < end && iv.end > start {
			return iv.end - 1, true
		}
	}
	return 0, false
}

// cellOccupancy is which grid cells items cover, kept as intervals per row and column.
type cellOccupancy struct {
	columns, rows trackCounts
	rowTracks     []trackIntervals
	columnTracks  []trackIntervals
}

func newCellOccupancy(columns, rows trackCounts) *cellOccupancy {
	return &cellOccupancy{
		columns:      columns,
		rows:         rows,
		rowTracks:    make([]trackIntervals, rows.len()),
		columnTracks: make([]trackIntervals, columns.len()),
	}
}

func (m *cellOccupancy) tracks(a axis) []trackIntervals {
	if a == horizontal {
		return m.columnTracks
	}
	return m.rowTracks
}

func (m *cellOccupancy) counts(a axis) trackCounts {
	if a == horizontal {
		return m.columns
	}
	return m.rows
}

func (m *cellOccupancy) expandToFit(rowSpan, colSpan lineSpan) {
	negRows := max(-m.rows.negative-rowSpan.start, 0)
	posRows := max(rowSpan.end-m.rows.implicitEnd(), 0)
	negCols := max(-m.columns.negative-colSpan.start, 0)
	posCols := max(colSpan.end-m.columns.implicitEnd(), 0)
	if negRows > 0 {
		m.rowTracks = append(make([]trackIntervals, negRows), m.rowTracks...)
	}
	if posRows > 0 {
		m.rowTracks = append(m.rowTracks, make([]trackIntervals, posRows)...)
	}
	if negCols > 0 {
		m.columnTracks = append(make([]trackIntervals, negCols), m.columnTracks...)
	}
	if posCols > 0 {
		m.columnTracks = append(m.columnTracks, make([]trackIntervals, posCols)...)
	}
	m.rows.negative += negRows
	m.rows.positive += posRows
	m.columns.negative += negCols
	m.columns.positive += posCols
}

func (m *cellOccupancy) markArea(primary axis, primarySpan, secondarySpan lineSpan, state cellState) {
	rowSpan, colSpan := primarySpan, secondarySpan
	if primary == horizontal {
		rowSpan, colSpan = secondarySpan, primarySpan
	}
	m.expandToFit(rowSpan, colSpan)
	if state == autoPlaced && primarySpan.start < primarySpan.end && secondarySpan.start < secondarySpan.end {
		m.advanceCursor(primary, secondarySpan.start, primarySpan.end)
	}
	for r := m.rows.ozToTrack(rowSpan.start); r < m.rows.ozToTrack(rowSpan.end); r++ {
		m.rowTracks[r].paint(colSpan.start, colSpan.end, state)
	}
	for c := m.columns.ozToTrack(colSpan.start); c < m.columns.ozToTrack(colSpan.end); c++ {
		m.columnTracks[c].paint(rowSpan.start, rowSpan.end, state)
	}
}

// collisionJump is the line past the last cell an area along primary collides with.
func (m *cellOccupancy) collisionJump(primary axis, primarySpan, secondarySpan lineSpan) (int, bool) {
	lists := m.tracks(primary.other())
	counts := m.counts(primary.other())
	start := max(counts.ozToTrack(secondarySpan.start), 0)
	end := min(counts.ozToTrack(secondarySpan.end), len(lists))
	extent, found := 0, false
	for i := start; i < end; i++ {
		if cell, ok := lists[i].collisionExtent(primarySpan.start, primarySpan.end); ok {
			if !found || cell > extent {
				extent = cell
			}
			found = true
		}
	}
	return extent + 1, found
}

func (m *cellOccupancy) occupiedTrackJump(a axis, s lineSpan) (int, bool) {
	counts := m.counts(a)
	lists := m.tracks(a)
	start := max(counts.ozToTrack(s.start), 0)
	end := min(counts.ozToTrack(s.end), len(lists))
	for i := end - 1; i >= start; i-- {
		if !lists[i].empty() {
			return counts.trackToOz(i) + 1, true
		}
	}
	return 0, false
}

func (m *cellOccupancy) occupied(a axis, i int) bool {
	lists := m.tracks(a)
	return i < len(lists) && !lists[i].empty()
}

func (m *cellOccupancy) cursor(primary axis, startAt int) (int, bool) {
	counts := m.counts(primary.other())
	lists := m.tracks(primary.other())
	i := counts.ozToTrack(startAt)
	if i < 0 || i >= len(lists) || !lists[i].hasCursor {
		return 0, false
	}
	return lists[i].cursor, true
}

func (m *cellOccupancy) advanceCursor(primary axis, secondaryStart, primaryEnd int) {
	i := m.counts(primary.other()).ozToTrack(secondaryStart)
	lists := m.tracks(primary.other())
	t := &lists[i]
	if !t.hasCursor || primaryEnd > t.cursor {
		t.cursor, t.hasCursor = primaryEnd, true
	}
}

func flowPrimary(f style.GridFlow) axis {
	if f == style.FlowColumn || f == style.FlowColumnDense {
		return vertical
	}
	return horizontal
}

func flowDense(f style.GridFlow) bool { return f == style.FlowRowDense || f == style.FlowColumnDense }

func indefiniteSpan(position, s int) lineSpan { return lineSpan{position, position + s} }

// placeGridItems runs the CSS Grid auto-placement algorithm, Taffy's place_grid_items.
func placeGridItems(m *cellOccupancy, items []*gridItem, placements []itemPlacement, flow style.GridFlow) {
	primary := flowPrimary(flow)
	secondary := primary.other()
	dense := flowDense(flow)
	record := func(it *gridItem, primarySpan, secondarySpan lineSpan, state cellState) {
		primarySpan = clampSpan(primarySpan)
		secondarySpan = clampSpan(secondarySpan)
		m.markArea(primary, primarySpan, secondarySpan, state)
		if primary == horizontal {
			it.column, it.row = primarySpan, secondarySpan
		} else {
			it.column, it.row = secondarySpan, primarySpan
		}
	}
	for i, it := range items {
		p := placements[i]
		if !p.h.isDefinite() || !p.v.isDefinite() {
			continue
		}
		record(it, p.get(primary).resolveDefinite(), p.get(secondary).resolveDefinite(), definitelyPlaced)
	}
	for i, it := range items {
		p := placements[i]
		if !p.get(secondary).isDefinite() || p.get(primary).isDefinite() {
			continue
		}
		secondarySpan := p.get(secondary).resolveDefinite()
		position := m.counts(primary).implicitStart()
		if !dense {
			if c, ok := m.cursor(primary, secondarySpan.start); ok {
				position = c
			}
		}
		span := p.get(primary).indefiniteSpan()
		for {
			primarySpan := indefiniteSpan(position, span)
			next, collides := m.collisionJump(primary, primarySpan, secondarySpan)
			if !collides {
				record(it, primarySpan, secondarySpan, autoPlaced)
				break
			}
			position = next
		}
	}
	start := [2]int{m.counts(primary).implicitStart(), m.counts(secondary).implicitStart()}
	pos := start
	for i, it := range items {
		p := placements[i]
		if p.get(secondary).isDefinite() {
			continue
		}
		primarySpan, secondarySpan := placeIndefinite(m, p, flow, pos)
		record(it, primarySpan, secondarySpan, autoPlaced)
		if dense {
			pos = start
		} else {
			pos = [2]int{primarySpan.end, secondarySpan.start}
		}
	}
}

func clampSpan(s lineSpan) lineSpan {
	start := min(max(s.start, minOZLine), maxOZLine-1)
	end := min(max(s.end, start+1), maxOZLine)
	return lineSpan{start, end}
}

func placeIndefinite(m *cellOccupancy, p itemPlacement, flow style.GridFlow, pos [2]int) (lineSpan, lineSpan) {
	primary := flowPrimary(flow)
	secondary := primary.other()
	primaryStyle, secondaryStyle := p.get(primary), p.get(secondary)
	secondarySpanLen := secondaryStyle.indefiniteSpan()
	primaryStart := m.counts(primary).implicitStart()
	primaryEnd := m.counts(primary).implicitEnd()
	secondaryStart := m.counts(secondary).implicitStart()
	primaryIdx, secondaryIdx := pos[0], pos[1]
	if primaryStyle.isDefinite() {
		primarySpan := primaryStyle.resolveDefinite()
		if flowDense(flow) {
			secondaryIdx = secondaryStart
		} else if primarySpan.start < primaryIdx {
			secondaryIdx++
		}
		for {
			secondarySpan := indefiniteSpan(secondaryIdx, secondarySpanLen)
			if next, collides := m.collisionJump(secondary, secondarySpan, primarySpan); collides {
				secondaryIdx = next
				continue
			}
			return primarySpan, secondarySpan
		}
	}
	primarySpanLen := primaryStyle.indefiniteSpan()
	spansAll := primarySpanLen >= m.counts(primary).len()
	for {
		primarySpan := indefiniteSpan(primaryIdx, primarySpanLen)
		secondarySpan := indefiniteSpan(secondaryIdx, secondarySpanLen)
		if primarySpan.end > primaryEnd {
			if primaryIdx == primaryStart {
				return primarySpan, secondarySpan
			}
			secondaryIdx++
			primaryIdx = primaryStart
			continue
		}
		if spansAll {
			if next, ok := m.occupiedTrackJump(secondary, secondarySpan); ok {
				secondaryIdx = next
				primaryIdx = primaryStart
				continue
			}
			return primarySpan, secondarySpan
		}
		if next, collides := m.collisionJump(primary, primarySpan, secondarySpan); collides {
			primaryIdx = next
			continue
		}
		return primarySpan, secondarySpan
	}
}
