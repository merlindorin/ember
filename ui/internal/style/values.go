// Package style is the computed style: what css produces for each node and
// layout and paint read. Every value is computed: em, rem, var() and
// keywords are resolved, and only percentages (as a part of a Length) wait
// for layout's containing block. Lengths are in points (CSS px).
package style

// LengthKind says how a Length sizes.
type LengthKind uint8

const (
	// Auto is the zero value, so an unset Length is auto.
	Auto LengthKind = iota
	// Fixed is Px plus Percent of the reference (calc() of px and %).
	Fixed
	MinContent
	MaxContent
	// FitContent clamps max-content to Px+Percent, fit-content(<length>);
	// with both zero it is plain fit-content.
	FitContent
	// Fr is a grid track's share, Px holding the factor.
	Fr
	// None is max-width/max-height's none.
	None
)

// Length is a computed length-percentage or sizing keyword.
type Length struct {
	Kind    LengthKind
	Px      float32
	Percent float32 // 0–100
	// IsPercent is set when the value had a percentage, 0% included:
	// 0% and 0px differ where the reference size is indefinite.
	IsPercent bool
}

func Px(v float32) Length      { return Length{Kind: Fixed, Px: v} }
func Percent(v float32) Length { return Length{Kind: Fixed, Percent: v} }

// Resolve is the length against a reference size, ok false when the
// length is not Fixed.
func (l Length) Resolve(ref float32) (float32, bool) {
	if l.Kind != Fixed {
		return 0, false
	}
	return l.Px + l.Percent*ref/100, true
}

// Sides is one value per side, in CSS order.
type Sides[T any] struct{ Top, Right, Bottom, Left T }

// All is the same value on every side.
func All[T any](v T) Sides[T] { return Sides[T]{v, v, v, v} }

// Corners is one value per corner, in CSS order.
type Corners[T any] struct{ TopLeft, TopRight, BottomRight, BottomLeft T }

// Size is a pair of widths and heights, or columns and rows.
type Size[T any] struct{ Width, Height T }
