package css

import (
	"fmt"
	"strings"

	"github.com/merlindorin/ember/ui/internal/style"
)

// gradientKinds are the gradient functions in the subset, by name.
//
//nolint:gochecknoglobals // lookup table
var gradientKinds = map[string]style.GradientKind{
	"linear-gradient": style.LinearGradient,
	"conic-gradient":  style.ConicGradient,
	"radial-gradient": style.RadialGradient,
}

// gradientKind is the kind of gradient c is, false when it is none.
func gradientKind(c Component) (style.GradientKind, bool) {
	if c.Kind != Function {
		return 0, false
	}
	k, ok := gradientKinds[strings.ToLower(c.Text)]
	return k, ok
}

// gradientTerm checks a linear-, conic- or radial-gradient(): its own
// first part when it has one (an angle or "to" a side or corner; from an
// angle and at a position; a shape, a size keyword and at a position),
// then two colour stops or more, each a colour with up to two positions,
// lengths or percentages, a conic one's angles or percentages.
// Transition hints and repeating gradients aren't in the subset.
//
//nolint:gochecknoglobals // a grammar term, built once
var gradientTerm = term(func(c Component) error {
	kind, ok := gradientKind(c)
	if !ok {
		return fmt.Errorf("expected a linear, conic or radial gradient, found %q", c.String())
	}
	parts := splitOn(items(c.Args), func(c Component) bool { return c.Kind == Comma })
	if len(parts) > 0 && prelude(kind, parts[0]) == nil {
		parts = parts[1:]
	}
	if len(parts) < 2 {
		return fmt.Errorf("%s() needs two colour stops, found %d", c.Text, len(parts))
	}
	for _, p := range parts {
		if len(p) == 0 || len(p) > 3 {
			return fmt.Errorf("bad colour stop in %q", c.String())
		}
		if err := colorTerm(p[0]); err != nil {
			return err
		}
		for _, at := range p[1:] {
			if err := stopPosition(kind, at); err != nil {
				return err
			}
		}
	}
	return nil
})

// stopPosition checks a stop's position: a conic gradient's angle or
// percentage, another's length or percentage.
func stopPosition(kind style.GradientKind, c Component) error {
	if kind == style.ConicGradient {
		if _, err := angle(c); err == nil {
			return nil
		}
		if c.Kind == Percentage {
			return nil
		}
		return fmt.Errorf("expected an angle or a percentage, found %q", c.String())
	}
	return lengthPercent(c)
}

// prelude checks a gradient's first part, before its stops.
func prelude(kind style.GradientKind, v []Component) error {
	switch kind {
	case style.ConicGradient:
		return conicPrelude(v)
	case style.RadialGradient:
		return radialPrelude(v)
	case style.LinearGradient:
		// a direction, as below
	}
	return direction(v)
}

// conicPrelude checks "from <angle>" and "at <position>", either or both.
func conicPrelude(v []Component) error {
	if len(v) == 0 {
		return fmt.Errorf("empty")
	}
	if isWord(v[0], "from") {
		if len(v) < 2 {
			return fmt.Errorf("from needs an angle")
		}
		if _, err := angle(v[1]); err != nil {
			return err
		}
		v = v[2:]
	}
	return atPosition(v)
}

// radialPrelude checks a shape, circle or ellipse, a size keyword and
// "at <position>", in that order, any of them.
func radialPrelude(v []Component) error {
	if len(v) == 0 {
		return fmt.Errorf("empty")
	}
	shape, size := false, false
	for len(v) > 0 && !isWord(v[0], "at") {
		switch {
		case !shape && isWord(v[0], "circle", "ellipse"):
			shape = true
		case !size && isWord(v[0], "closest-side", "closest-corner", "farthest-side", "farthest-corner"):
			size = true
		default:
			return fmt.Errorf("expected a shape or a size keyword, found %q", v[0].String())
		}
		v = v[1:]
	}
	return atPosition(v)
}

// atPosition checks what may end a prelude: nothing, or "at" and a
// position.
func atPosition(v []Component) error {
	if len(v) == 0 {
		return nil
	}
	if !isWord(v[0], "at") || len(v) < 2 {
		return fmt.Errorf("expected at and a position")
	}
	return checkPosition(v[1:])
}

// imageTerm is a url() or a gradient.
//
//nolint:gochecknoglobals // a grammar term, built once
var imageTerm = either(urlTerm, gradientTerm)

// direction checks a gradient's direction: an angle, or "to" with a side
// or a corner.
func direction(v []Component) error {
	if len(v) == 1 {
		_, err := angle(v[0])
		return err
	}
	if len(v) < 2 || len(v) > 3 || !isWord(v[0], "to") {
		return fmt.Errorf("expected an angle or to a side")
	}
	x, y := 0, 0
	for _, c := range v[1:] {
		switch {
		case isWord(c, kwLeft, kwRight) && x == 0:
			x = 1
		case isWord(c, kwTop, kwBottom) && y == 0:
			y = 1
		default:
			return fmt.Errorf("expected to a side or a corner, found %q", c.String())
		}
	}
	return nil
}

// gradient is the gradient c: a linear one's direction, by default to
// bottom; a conic one's from angle and centre, by default 0 and the box's
// middle; a radial one's shape, size and centre, by default an ellipse to
// the farthest corner from the middle; and its stops, a stop with two
// positions counted as two, a conic one's angles made percentages of a
// turn.
func (x *ctx) gradient(c Component) *style.Gradient {
	kind, _ := gradientKind(c)
	middle := style.BackgroundPosition{X: style.Length{Kind: style.Fixed, Percent: 50, IsPercent: true},
		Y: style.Length{Kind: style.Fixed, Percent: 50, IsPercent: true}}
	g := &style.Gradient{Kind: kind, At: middle}
	if kind == style.LinearGradient {
		g.Angle = 180
	}
	parts := splitOn(items(c.Args), func(c Component) bool { return c.Kind == Comma })
	if len(parts) > 0 && prelude(kind, parts[0]) == nil {
		x.prelude(g, parts[0])
		parts = parts[1:]
	}
	for _, p := range parts {
		col := x.color(p[0])
		if len(p) == 1 {
			g.Stops = append(g.Stops, style.Stop{Color: col})
		}
		for _, at := range p[1:] {
			g.Stops = append(g.Stops, style.Stop{Color: col, At: x.stopAt(kind, at), Placed: true})
		}
	}
	return g
}

// prelude reads a gradient's first part into g.
func (x *ctx) prelude(g *style.Gradient, d []Component) {
	switch g.Kind {
	case style.LinearGradient:
		if len(d) == 1 {
			a, _ := angle(d[0])
			g.Angle = float32(a)
		} else {
			g.Corner, g.Angle = toward(d[1:])
		}
		return
	case style.ConicGradient:
		if isWord(d[0], "from") {
			a, _ := angle(d[1])
			g.Angle = float32(a)
			d = d[2:]
		}
	case style.RadialGradient:
		for len(d) > 0 && !isWord(d[0], "at") {
			switch strings.ToLower(d[0].Text) {
			case "circle":
				g.Circle = true
			case "closest-side":
				g.Size = style.ClosestSide
			case "closest-corner":
				g.Size = style.ClosestCorner
			case "farthest-side":
				g.Size = style.FarthestSide
			}
			d = d[1:]
		}
	}
	if len(d) > 1 && isWord(d[0], "at") {
		g.At = x.position(d[1:])
	}
}

// stopAt is a stop's position: a conic gradient's angle as a percentage
// of a turn.
func (x *ctx) stopAt(kind style.GradientKind, c Component) style.Length {
	if kind == style.ConicGradient {
		if a, err := angle(c); err == nil {
			return style.Length{Kind: style.Fixed, Percent: float32(a / 3.6), IsPercent: true}
		}
	}
	return x.length(c)
}

// toward is "to" sides as an angle, or a corner, whose angle the box's
// shape sets.
func toward(v []Component) ([2]int8, float32) {
	var c [2]int8
	for _, w := range v {
		switch strings.ToLower(w.Text) {
		case kwLeft:
			c[0] = -1
		case kwRight:
			c[0] = 1
		case kwTop:
			c[1] = -1
		case kwBottom:
			c[1] = 1
		}
	}
	if c[0] != 0 && c[1] != 0 {
		return c, 0
	}
	switch {
	case c[0] > 0:
		return [2]int8{}, 90
	case c[0] < 0:
		return [2]int8{}, 270
	case c[1] < 0:
		return [2]int8{}, 0
	}
	return [2]int8{}, 180
}

// backgroundImage is background-image's url() or gradient.
func backgroundImage(x *ctx, v []Component) style.Image {
	if len(v) > 0 {
		if _, ok := gradientKind(v[0]); ok {
			return style.Image{Gradient: x.gradient(v[0])}
		}
	}
	return style.Image{Source: imageSource(x, v)}
}
