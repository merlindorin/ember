package layout

import "math"

// An optional length is a float32 where NaN means none, so Taffy's Option<f32> arithmetic
// keeps its shape: none on the left stays none, none on the right is ignored.
var none = float32(math.NaN()) //nolint:gochecknoglobals // NaN is never a constant

func some(v float32) bool { return v == v }

func or(v, alt float32) float32 {
	if some(v) {
		return v
	}
	return alt
}

func fmin(a, b float32) float32 {
	if !some(a) {
		return b
	}
	if !some(b) || a < b {
		return a
	}
	return b
}

func fmax(a, b float32) float32 {
	if !some(a) {
		return b
	}
	if !some(b) || a > b {
		return a
	}
	return b
}

func mMin(a, b float32) float32 {
	if !some(a) || !some(b) {
		return a
	}
	return fmin(a, b)
}

func mMax(a, b float32) float32 {
	if !some(a) || !some(b) {
		return a
	}
	return fmax(a, b)
}

func mClamp(v, lo, hi float32) float32 {
	return mMax(mMin(v, hi), lo)
}

func mAdd(a, b float32) float32 {
	if !some(a) || !some(b) {
		return a
	}
	return a + b
}

func mSub(a, b float32) float32 {
	if !some(a) || !some(b) {
		return a
	}
	return a - b
}

func isNormal(v float32) bool {
	if !some(v) || math.IsInf(float64(v), 0) || v == 0 {
		return false
	}
	return math.Abs(float64(v)) >= 1.1754943508222875e-38
}

func round(v float32) float32 {
	return float32(math.Round(float64(v)))
}

type axis uint8

const (
	horizontal axis = iota
	vertical
)

func (a axis) other() axis { return 1 - a }

type size struct{ w, h float32 }

var noSize = size{none, none} //nolint:gochecknoglobals // a read-only sentinel; Go has no struct constants

func (s size) get(a axis) float32 {
	if a == horizontal {
		return s.w
	}
	return s.h
}

func (s *size) set(a axis, v float32) {
	if a == horizontal {
		s.w = v
	} else {
		s.h = v
	}
}

func (s size) with(a axis, v float32) size { s.set(a, v); return s }
func (s size) add(o size) size             { return size{s.w + o.w, s.h + o.h} }
func (s size) sub(o size) size             { return size{s.w - o.w, s.h - o.h} }
func (s size) mMax(o size) size            { return size{mMax(s.w, o.w), mMax(s.h, o.h)} }
func (s size) mAdd(o size) size            { return size{mAdd(s.w, o.w), mAdd(s.h, o.h)} }
func (s size) mSub(o size) size            { return size{mSub(s.w, o.w), mSub(s.h, o.h)} }
func (s size) mClamp(lo, hi size) size {
	return size{mClamp(s.w, lo.w, hi.w), mClamp(s.h, lo.h, hi.h)}
}
func (s size) or(o size) size { return size{or(s.w, o.w), or(s.h, o.h)} }
func (s size) both() bool     { return some(s.w) && some(s.h) }

func (s size) withAspectRatio(ratio float32) size {
	if !some(ratio) {
		return s
	}
	switch {
	case some(s.w) && !some(s.h):
		return size{s.w, s.w / ratio}
	case !some(s.w) && some(s.h):
		return size{s.h * ratio, s.h}
	}
	return s
}

type rect struct{ l, r, t, b float32 }

func (r rect) add(o rect) rect     { return rect{r.l + o.l, r.r + o.r, r.t + o.t, r.b + o.b} }
func (r rect) horizontal() float32 { return r.l + r.r }
func (r rect) vertical() float32   { return r.t + r.b }
func (r rect) sum() size           { return size{r.horizontal(), r.vertical()} }

func (r rect) union(o rect) rect {
	return rect{min(r.l, o.l), max(r.r, o.r), min(r.t, o.t), max(r.b, o.b)}
}

type point struct{ x, y float32 }

func (p point) get(a axis) float32 {
	if a == horizontal {
		return p.x
	}
	return p.y
}

type span struct{ start, end float32 }

type spaceKind uint8

const (
	definite spaceKind = iota
	minContent
	maxContent
)

// space is Taffy's AvailableSpace.
type space struct {
	kind spaceKind
	v    float32
}

//nolint:gochecknoglobals // read-only sentinels; Go has no struct constants
var (
	minContentSpace = space{kind: minContent}
	maxContentSpace = space{kind: maxContent}
)

func def(v float32) space { return space{v: v} }

// spaceOr is a definite space when v is some, otherwise fallback.
func spaceOr(v float32, fallback space) space {
	if some(v) {
		return def(v)
	}
	return fallback
}

func (s space) definite() bool { return s.kind == definite }

func (s space) value() float32 {
	if s.kind == definite {
		return s.v
	}
	return none
}

func (s space) mapDefinite(f func(float32) float32) space {
	if s.kind == definite {
		return def(f(s.v))
	}
	return s
}

func (s space) mMax(v float32) space {
	if s.kind == definite && some(v) {
		return def(fmax(s.v, v))
	}
	return s
}

func (s space) mClamp(lo, hi float32) space {
	if s.kind == definite {
		return def(mClamp(s.v, lo, hi))
	}
	return s
}

func (s space) mSub(v float32) space {
	if s.kind == definite && some(v) {
		return def(s.v - v)
	}
	return s
}

func (s space) freeSpace(used float32) float32 {
	switch s.kind {
	case maxContent:
		return float32(math.Inf(1))
	case minContent:
		return 0
	case definite:
	}
	return s.v - used
}

type spaces struct{ w, h space }

//nolint:gochecknoglobals // a read-only sentinel; Go has no struct constants
var maxContentSpaces = spaces{maxContentSpace, maxContentSpace}

func (s spaces) get(a axis) space {
	if a == horizontal {
		return s.w
	}
	return s.h
}

func (s spaces) with(a axis, v space) spaces {
	if a == horizontal {
		s.w = v
	} else {
		s.h = v
	}
	return s
}

func (s spaces) values() size { return size{s.w.value(), s.h.value()} }

func (s spaces) mSub(o size) spaces { return spaces{s.w.mSub(o.w), s.h.mSub(o.h)} }
func (s spaces) mMax(o size) spaces { return spaces{s.w.mMax(o.w), s.h.mMax(o.h)} }
