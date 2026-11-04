package layout

import "github.com/merlindorin/ember/ui/internal/style"

// alignment is an align-* or justify-* value with its overflow safety.
type alignment struct {
	keyword style.Align
	safe    bool
	unsafe  bool
}

func align(k style.Align) alignment { return alignment{keyword: k} }

func childFirstBaseline(first, height float32, overflowY style.Overflow) float32 {
	if scrollContainer(overflowY) {
		return fmax(fmin(or(first, height), height), 0)
	}
	return first
}

func resolveSelfAlignmentSafety(a alignment, overflows bool) style.Align {
	if a.safe && overflows {
		return style.AlignStart
	}
	return a.keyword
}

// applyAlignmentFallback is the content-distribution fallback of CSS Box Alignment.
func applyAlignmentFallback(free float32, items int, a alignment) style.Align {
	k := a.keyword
	if k == style.AlignNormal {
		k = style.AlignStretch
	}
	safe := a.safe
	if items <= 1 || free <= 0 {
		switch k {
		case style.AlignStretch:
			k = style.AlignFlexStart
		case style.AlignSpaceBetween:
			k, safe = style.AlignFlexStart, true
		case style.AlignSpaceAround, style.AlignSpaceEvenly:
			k, safe = style.AlignCenter, true
		case style.AlignNormal, style.AlignAuto, style.AlignStart, style.AlignEnd, style.AlignFlexStart, style.AlignFlexEnd,
			style.AlignCenter, style.AlignBaseline: // no fallback needed
		}
	}
	if free <= 0 && safe {
		k = style.AlignStart
	}
	return k
}

func computeAlignmentOffset(free float32, items int, gap float32, k style.Align, reversed, first bool) float32 {
	if first {
		switch k {
		case style.AlignStart:
			return 0
		case style.AlignFlexStart:
			if reversed {
				return free
			}
			return 0
		case style.AlignEnd:
			return free
		case style.AlignFlexEnd:
			if reversed {
				return 0
			}
			return free
		case style.AlignCenter:
			return free / 2
		case style.AlignSpaceAround:
			if free >= 0 {
				return free / float32(items) / 2
			}
			return free / 2
		case style.AlignSpaceEvenly:
			if free >= 0 {
				return free / float32(items+1)
			}
			return free / 2
		case style.AlignNormal, style.AlignAuto, style.AlignStretch, style.AlignBaseline, style.AlignSpaceBetween:
		}
		return 0
	}
	free = fmax(free, 0)
	switch k {
	case style.AlignSpaceBetween:
		return gap + free/float32(items-1)
	case style.AlignSpaceAround:
		return gap + free/float32(items)
	case style.AlignSpaceEvenly:
		return gap + free/float32(items+1)
	case style.AlignNormal, style.AlignAuto, style.AlignStretch, style.AlignStart, style.AlignEnd, style.AlignFlexStart,
		style.AlignFlexEnd, style.AlignCenter, style.AlignBaseline: // only the gap
	}
	return gap
}

// overflowContainer is what a child's scrollable overflow contributes to.
type overflowContainer struct {
	scroll   bool
	size     size
	reversed [2]bool
	padding  rect
}

func overflowContribution(loc point, s size, childOverflow rect, border rect,
	overflow style.Size[style.Overflow], c overflowContainer, margin rect) rect {
	childOverflow = rect{
		l: childOverflow.l + border.l, r: childOverflow.r + border.l,
		t: childOverflow.t + border.t, b: childOverflow.b + border.t,
	}
	scroll := scrollContainer(overflow.Width) || scrollContainer(overflow.Height)
	propX := !scroll && overflow.Width == style.Visible
	propY := !scroll && overflow.Height == style.Visible
	endW, endH := s.w, s.h
	if propX {
		endW = fmax(s.w, childOverflow.r)
	}
	if propY {
		endH = fmax(s.h, childOverflow.b)
	}
	if endW <= 0 || endH <= 0 {
		return rect{}
	}
	startX, startY := float32(0), float32(0)
	if propX {
		startX = fmin(0, childOverflow.l)
	}
	if propY {
		startY = fmin(0, childOverflow.t)
	}
	r := rect{l: loc.x + startX, r: loc.x + endW, t: loc.y + startY, b: loc.y + endH}
	unreachableX := r.r <= 0
	if c.reversed[0] {
		unreachableX = r.l >= c.size.w
	}
	unreachableY := r.b <= 0
	if c.reversed[1] {
		unreachableY = r.t >= c.size.h
	}
	if c.scroll && (unreachableX || unreachableY) {
		return rect{}
	}
	if !c.scroll {
		return r
	}
	if c.reversed[0] {
		r.l = fmin(r.l, loc.x-fmax(0, c.padding.l+margin.l))
	} else {
		r.r = fmax(r.r, loc.x+s.w+fmax(0, c.padding.r+margin.r))
	}
	if c.reversed[1] {
		r.t = fmin(r.t, loc.y-fmax(0, c.padding.t+margin.t))
	} else {
		r.b = fmax(r.b, loc.y+s.h+fmax(0, c.padding.b+margin.b))
	}
	return r
}

func finalizeOverflow(r rect, s size, reversed [2]bool, scroll bool) overflowRect {
	s = size{fmax(s.w, 0), fmax(s.h, 0)}
	if !scroll {
		return newOverflowRect(r, [2]bool{})
	}
	if reversed[0] {
		r.r = fmin(r.r, s.w)
	} else {
		r.l = 0
	}
	if reversed[1] {
		r.b = fmin(r.b, s.h)
	} else {
		r.t = 0
	}
	return newOverflowRect(r, reversed)
}

type keywordKind uint8

const (
	keywordNone keywordKind = iota
	keywordMeasure
	keywordExact
)

// resolveSizingKeyword is what a min-content, max-content or fit-content size means: a
// space to measure in.
func resolveSizingKeyword(l style.Length, stretch, basis float32) (keywordKind, space) {
	switch l.Kind {
	case style.MinContent:
		return keywordMeasure, minContentSpace
	case style.MaxContent:
		return keywordMeasure, maxContentSpace
	case style.FitContent:
		if l.Px == 0 && !hasPercent(l) {
			if some(stretch) {
				return keywordMeasure, def(stretch)
			}
			return keywordNone, space{}
		}
		if !hasPercent(l) {
			return keywordMeasure, def(l.Px)
		}
		if some(basis) {
			return keywordMeasure, def(l.Px + l.Percent*basis/100)
		}
	case style.Auto, style.Fixed, style.Fr, style.None:
	}
	return keywordNone, space{}
}

// resolveAbsoluteSizingKeywords sizes an absolutely positioned box whose width or height
// is a sizing keyword.
func (t *tree) resolveAbsoluteSizingKeywords(n *node, known *size, sizeStyle style.Size[style.Length],
	area size, inset, margin rect, minSize, maxSize size, mode sizingMode) {
	stretch := size{
		w: fmax(area.w-or(inset.l, 0)-or(inset.r, 0)-or(margin.l, 0)-or(margin.r, 0), 0),
		h: fmax(area.h-or(inset.t, 0)-or(inset.b, 0)-or(margin.t, 0)-or(margin.b, 0), 0),
	}
	kw, kwSpace := keywordNone, space{}
	if !some(known.w) {
		kw, kwSpace = resolveSizingKeyword(sizeStyle.Width, stretch.w, area.w)
	}
	kh, khSpace := keywordNone, space{}
	if !some(known.h) {
		kh, khSpace = resolveSizingKeyword(sizeStyle.Height, stretch.h, area.h)
	}
	if kw == keywordMeasure {
		h := def(stretch.h)
		if kh == keywordMeasure {
			h = khSpace
		}
		known.w = mClamp(t.measureChildSize(n, *known, area, spaces{kwSpace, h}, mode, horizontal, [2]bool{}),
			minSize.w, maxSize.w)
	}
	if kh == keywordMeasure {
		w := def(mClamp(stretch.w, minSize.w, maxSize.w))
		if some(known.w) {
			w = def(known.w)
		}
		known.h = t.measureChildSize(n, *known, area, spaces{w, khSpace}, mode, vertical, [2]bool{})
	}
}
