package layout

import "github.com/lantervale/ember/ui/internal/style"

// resolve is a length against a basis, as Taffy's maybe_resolve: none for auto and sizing
// keywords, and for a percentage of a basis that is none.
func resolve(l style.Length, basis float32) float32 {
	if l.Kind != style.Fixed {
		return none
	}
	if !hasPercent(l) {
		return l.Px
	}
	if !some(basis) {
		return none
	}
	return l.Px + l.Percent*basis/100
}

// hasPercent reports a length with a percentage part, 0% included.
func hasPercent(l style.Length) bool { return l.IsPercent || l.Percent != 0 }

// readsParent reports a style whose own lengths resolve against the parent's size: its
// layout depends on that size only then.
func readsParent(st *style.Computed) bool {
	for _, l := range []style.Length{
		st.Size.Width, st.Size.Height, st.MinSize.Width, st.MinSize.Height, st.MaxSize.Width, st.MaxSize.Height,
		st.Padding.Left, st.Padding.Right, st.Padding.Top, st.Padding.Bottom,
		st.Margin.Left, st.Margin.Right, st.Margin.Top, st.Margin.Bottom,
	} {
		if hasPercent(l) {
			return true
		}
	}
	return false
}

func resolveOrZero(l style.Length, basis float32) float32 {
	return or(resolve(l, basis), 0)
}

func resolveSize(s style.Size[style.Length], basis size) size {
	return size{resolve(s.Width, basis.w), resolve(s.Height, basis.h)}
}

// resolveMax is max-width/height, where none is no limit.
func resolveMax(s style.Size[style.Length], basis size) size {
	return resolveSize(s, basis)
}

func edges(s style.Sides[style.Length], basis float32) rect {
	return rect{
		l: resolveOrZero(s.Left, basis),
		r: resolveOrZero(s.Right, basis),
		t: resolveOrZero(s.Top, basis),
		b: resolveOrZero(s.Bottom, basis),
	}
}

func borders(s *style.Computed) rect {
	return rect{l: s.Border.Left, r: s.Border.Right, t: s.Border.Top, b: s.Border.Bottom}
}

// optionalEdges resolves each side to none where it is auto.
func optionalEdges(s style.Sides[style.Length], basis float32) rect {
	return rect{
		l: resolve(s.Left, basis),
		r: resolve(s.Right, basis),
		t: resolve(s.Top, basis),
		b: resolve(s.Bottom, basis),
	}
}

type autoEdges struct{ l, r, t, b bool }

func autoMargins(s *style.Computed) autoEdges {
	m := s.Margin
	return autoEdges{
		l: m.Left.Kind == style.Auto,
		r: m.Right.Kind == style.Auto,
		t: m.Top.Kind == style.Auto,
		b: m.Bottom.Kind == style.Auto,
	}
}

func aspectRatio(s *style.Computed) float32 {
	if s.AspectRatio > 0 {
		return s.AspectRatio
	}
	return none
}

func contentBox(s *style.Computed) bool { return s.BoxSizing == style.ContentBox }

func boxSizingAdjustment(s *style.Computed, paddingBorder size) size {
	if contentBox(s) {
		return paddingBorder
	}
	return size{}
}

func scrollContainer(o style.Overflow) bool {
	return o == style.Hidden || o == style.Scroll || o == style.OverflowAuto
}

func isScrollContainer(s *style.Computed) bool {
	return scrollContainer(s.Overflow.Width) || scrollContainer(s.Overflow.Height)
}

// scrollbarGutter is the room overflow: scroll reserves, transposed: a box that scrolls
// vertically reserves a gutter's width.
func scrollbarGutter(s *style.Computed) point {
	var g point
	if s.Overflow.Height == style.Scroll {
		g.x = s.ScrollbarWidth
	}
	if s.Overflow.Width == style.Scroll {
		g.y = s.ScrollbarWidth
	}
	return g
}

func outOfFlow(p style.Position) bool { return p == style.Absolute || p == style.FixedPosition }

func isSizingKeyword(l style.Length) bool {
	switch l.Kind {
	case style.MinContent, style.MaxContent, style.FitContent:
		return true
	case style.Auto, style.Fixed, style.Fr, style.None:
	}
	return false
}
