package css

import (
	"math"
	"slices"
	"strings"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/ease"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// Transitions runs CSS transitions: Compute starts them where a style
// change is covered by the element's transition list, and Step advances
// them. Paint reads dom.Data(n).Shown, which is Style itself while nothing
// runs on n.
type Transitions struct {
	nodes map[*dom.Node]*nodeAnim
	root  *dom.Node
	// given is the sheets the last Compute had, all the same behind the
	// user agent's, kept so a frame like the last allocates nothing.
	given, all []*Sheet
	changed    func(n *dom.Node, old, next *style.Computed)
	m, pm      matcher
	treeStates dom.State
	// computed counts the elements restyled, for tests.
	computed int
}

// nodeAnim is one element's running transitions and the style it shows.
type nodeAnim struct {
	shown style.Computed
	runs  map[string]*run
	// follows is set while the element shows an ancestor's colour
	// transition through inheritance.
	follows bool
	touched bool
}

type run struct {
	from, to      value
	reverseStart  value
	factor        float32
	elapsed       float32
	delay, length float32
	easing        ease.Func
}

func NewTransitions() *Transitions {
	return &Transitions{nodes: map[*dom.Node]*nodeAnim{}}
}

// Compute restyles root as css.Compute does, starting a transition for
// each changed property its new style's transitions cover.
// Nothing is done while root and its tree are clean.
func (t *Transitions) Compute(root *dom.Node, sheets []*Sheet) []Warning {
	t.root = root
	dirty := dom.DirtyStyle | dom.DirtyTree | dom.DirtyState | dom.DirtyDescendant
	if d := dom.Data(root); d.Style != nil && d.Dirty&dirty == 0 {
		return nil
	}
	if !slices.Equal(t.given, sheets) || t.all == nil {
		t.given = slices.Clone(sheets)
		t.all = append([]*Sheet{userAgent()}, sheets...)
		t.treeStates = treeStates(t.all)
	}
	if t.changed == nil {
		t.changed = t.restyled
	}
	r := restyler{sheets: t.all, changed: t.changed, m: &t.m, pm: &t.pm, treeStates: t.treeStates,
		computed: &t.computed}
	r.visit(root, nil, 0, false, false)
	t.refresh()
	return r.warns
}

// Step advances every transition by dt seconds and writes their values
// into Shown. It reports whether any still runs.
func (t *Transitions) Step(dt float32) bool {
	for n, a := range t.nodes {
		if !t.attached(n) {
			delete(t.nodes, n)
			continue
		}
		for name, r := range a.runs {
			r.elapsed += dt
			if r.elapsed >= r.delay+r.length {
				delete(a.runs, name)
			}
		}
	}
	t.refresh()
	return len(t.nodes) > 0
}

// Running reports whether n shows a transition.
func (t *Transitions) Running(n *dom.Node) bool { return t.nodes[n] != nil }

func (t *Transitions) attached(n *dom.Node) bool {
	for ; n != nil; n = n.Parent() {
		if n == t.root {
			return true
		}
	}
	return false
}

// restyled starts, retargets or cancels n's transitions after its style
// went from old to new.
func (t *Transitions) restyled(n *dom.Node, old, next *style.Computed) {
	a := t.nodes[n]
	before := old
	if a != nil {
		before = &a.shown
	}
	for _, p := range animatable {
		to := p.get(next)
		var r *run
		if a != nil {
			r = a.runs[p.name]
		}
		tr, ok := transitionFor(next.Transitions, p.name)
		switch {
		case r != nil && r.to.equal(to):
			continue
		case !ok:
			if r != nil {
				delete(a.runs, p.name)
			}
			continue
		case r == nil && p.get(before).equal(to):
			continue
		}
		from := p.get(before)
		next := &run{from: from, to: to, reverseStart: from, factor: 1,
			delay: tr.Delay, length: tr.Duration, easing: easing(tr.Easing)}
		if r != nil && r.reverseStart.equal(to) {
			next.reverseStart = r.to
			next.factor = clamp01f(r.easing(r.progress())*r.factor + 1 - r.factor)
			next.length *= next.factor
			if next.delay < 0 {
				next.delay *= next.factor
			}
		}
		if next.delay+next.length <= 0 {
			if r != nil {
				delete(a.runs, p.name)
			}
			continue
		}
		if a == nil {
			a = &nodeAnim{runs: map[string]*run{}}
			t.nodes[n] = a
		}
		a.runs[p.name] = next
	}
	if t.nodes[n] == nil {
		setShown(n, next)
	}
}

// refresh rebuilds every animated element's Shown, then carries colour
// transitions down to descendants whose colour is inherited.
func (t *Transitions) refresh() {
	for _, a := range t.nodes {
		a.touched = false
	}
	for n, a := range t.nodes {
		if len(a.runs) > 0 {
			t.build(n, a)
		}
	}
	for n, a := range t.nodes {
		if _, ok := a.runs["color"]; ok && !a.follows {
			t.carry(n, a.shown.Color)
		}
	}
	for n, a := range t.nodes {
		if !a.touched && (a.follows || len(a.runs) == 0) {
			delete(t.nodes, n)
			setShown(n, dom.Data(n).Style)
		}
	}
}

func (t *Transitions) build(n *dom.Node, a *nodeAnim) {
	a.shown = *dom.Data(n).Style
	if _, ok := a.runs["color"]; ok {
		a.runs["color"].apply(&a.shown)
		a.shown.FollowColor(a.shown.Color)
	}
	for _, p := range animatable {
		if r, ok := a.runs[p.name]; ok {
			p.set(&a.shown, r.at())
		}
	}
	a.touched = true
	setShown(n, &a.shown)
}

// carry shows c as the colour of n's children that inherit it, and of
// theirs.
func (t *Transitions) carry(n *dom.Node, c color.Color) {
	for _, k := range n.Children() {
		if k.IsText() {
			continue
		}
		s := dom.Data(k).Style
		if s == nil || !s.ColorInherited {
			continue
		}
		a := t.nodes[k]
		if a == nil {
			a = &nodeAnim{runs: map[string]*run{}}
			t.nodes[k] = a
		}
		if _, own := a.runs["color"]; own {
			continue
		}
		if !a.touched {
			t.build(k, a)
		}
		a.follows = len(a.runs) == 0
		a.shown.Color = c
		a.shown.FollowColor(c)
		t.carry(k, c)
	}
}

// setShown points n and its text nodes at s.
func setShown(n *dom.Node, s *style.Computed) {
	dom.Data(n).Shown = s
	for _, k := range n.Children() {
		if k.IsText() {
			dom.Data(k).Shown = s
		}
	}
}

func (r *run) progress() float32 {
	if r.length <= 0 {
		if r.elapsed >= r.delay {
			return 1
		}
		return 0
	}
	return clamp01f((r.elapsed - r.delay) / r.length)
}

// apply writes color's value into s.
func (r *run) apply(s *style.Computed) { s.Color = r.at().c }

func (r *run) at() value {
	if r.elapsed < r.delay {
		return r.from
	}
	return r.from.lerp(r.to, r.easing(r.progress()))
}

func clamp01f(v float32) float32 { return min(max(v, 0), 1) }

// transitionFor finds the transition the list gives property name: the
// last entry naming it, a shorthand of it or all.
func transitionFor(list []style.Transition, name string) (style.Transition, bool) {
	for i := len(list) - 1; i >= 0; i-- {
		p := list[i].Property
		if p == name || p == "all" || p == propBorderColor && strings.HasPrefix(name, "border-") {
			return list[i], list[i].Duration+list[i].Delay > 0
		}
	}
	return style.Transition{}, false
}

// value is one animatable property's value.
type value struct {
	kind    valueKind
	c       color.Color
	f       float32
	ops     []style.TransformOp
	shadows []style.Shadow
}

type valueKind uint8

const (
	colorValue valueKind = iota
	numberValue
	transformValue
	shadowValue
)

type animProp struct {
	name string
	get  func(*style.Computed) value
	set  func(*style.Computed, value)
}

func colorProp(name string, f func(*style.Computed) *color.Color) animProp {
	return animProp{name,
		func(s *style.Computed) value { return value{kind: colorValue, c: *f(s)} },
		func(s *style.Computed, v value) { *f(s) = v.c },
	}
}

//nolint:gochecknoglobals,goconst // the animatable properties' table
var animatable = []animProp{
	colorProp("color", func(s *style.Computed) *color.Color { return &s.Color }),
	colorProp("background-color", func(s *style.Computed) *color.Color { return &s.Background }),
	colorProp("border-top-color", func(s *style.Computed) *color.Color { return &s.BorderColor.Top }),
	colorProp("border-right-color", func(s *style.Computed) *color.Color { return &s.BorderColor.Right }),
	colorProp("border-bottom-color", func(s *style.Computed) *color.Color { return &s.BorderColor.Bottom }),
	colorProp("border-left-color", func(s *style.Computed) *color.Color { return &s.BorderColor.Left }),
	colorProp("outline-color", func(s *style.Computed) *color.Color { return &s.OutlineColor }),
	{"opacity",
		func(s *style.Computed) value { return value{kind: numberValue, f: s.Opacity} },
		func(s *style.Computed, v value) { s.Opacity = v.f }},
	{"transform",
		func(s *style.Computed) value { return value{kind: transformValue, ops: s.Transform} },
		func(s *style.Computed, v value) { s.Transform = v.ops }},
	{"box-shadow",
		func(s *style.Computed) value { return value{kind: shadowValue, shadows: s.Shadows} },
		func(s *style.Computed, v value) { s.Shadows = v.shadows }},
}

func (v value) equal(o value) bool {
	switch v.kind {
	case colorValue:
		return v.c == o.c
	case numberValue:
		return v.f == o.f
	case transformValue:
		if len(v.ops) != len(o.ops) {
			return false
		}
		for i := range v.ops {
			if v.ops[i] != o.ops[i] {
				return false
			}
		}
		return true
	case shadowValue:
		// shadow by shadow, as below
	}
	if len(v.shadows) != len(o.shadows) {
		return false
	}
	for i := range v.shadows {
		if v.shadows[i] != o.shadows[i] {
			return false
		}
	}
	return true
}

func lerpf(a, b, t float32) float32 { return a + (b-a)*t }

func (v value) lerp(o value, t float32) value {
	switch v.kind {
	case colorValue:
		return value{kind: colorValue, c: lerpColor(v.c, o.c, t)}
	case numberValue:
		return value{kind: numberValue, f: lerpf(v.f, o.f, t)}
	case transformValue:
		return value{kind: transformValue, ops: lerpTransform(v.ops, o.ops, t)}
	case shadowValue:
		// shadow by shadow, as below
	}
	return value{kind: shadowValue, shadows: lerpShadows(v.shadows, o.shadows, t)}
}

// lerpColor interpolates in premultiplied sRGB, as Chrome does for
// colours given as hex, rgb() or names.
func lerpColor(a, b color.Color, t float32) color.Color {
	sa, sb := a.SRGB(), b.SRGB()
	alpha := lerpf(a.A, b.A, t)
	var out [3]float32
	if alpha > 0 {
		for i := range out {
			out[i] = clamp01f(lerpf(sa[i]*a.A, sb[i]*b.A, t) / alpha)
		}
	}
	c := color.FromSRGB(out)
	c.A = alpha
	return c
}

func lerpShadows(a, b []style.Shadow, t float32) []style.Shadow {
	n := max(len(a), len(b))
	out := make([]style.Shadow, n)
	for i := range n {
		var x, y style.Shadow
		switch {
		case i < len(a) && i < len(b):
			x, y = a[i], b[i]
		case i < len(a):
			x, y = a[i], style.Shadow{Inset: a[i].Inset}
		default:
			x, y = style.Shadow{Inset: b[i].Inset}, b[i]
		}
		if x.Inset != y.Inset {
			if t < 1 {
				return a
			}
			return b
		}
		out[i] = style.Shadow{
			X: lerpf(x.X, y.X, t), Y: lerpf(x.Y, y.Y, t),
			Blur: lerpf(x.Blur, y.Blur, t), Spread: lerpf(x.Spread, y.Spread, t),
			Color: lerpColor(x.Color, y.Color, t), Inset: x.Inset,
		}
	}
	return out
}

func lerpLength(a, b style.Length, t float32) style.Length {
	return style.Length{Kind: style.Fixed, Px: lerpf(a.Px, b.Px, t),
		Percent: lerpf(a.Percent, b.Percent, t), IsPercent: a.IsPercent || b.IsPercent}
}

// identity is the neutral function of op's kind.
func identity(op style.TransformOp) style.TransformOp {
	switch op.Kind {
	case style.Translate:
		return style.TransformOp{Kind: style.Translate, X: style.Px(0), Y: style.Px(0)}
	case style.ScaleOp:
		return style.TransformOp{Kind: style.ScaleOp, A: [6]float32{1, 1}}
	case style.Matrix:
		return style.TransformOp{Kind: style.Matrix, A: [6]float32{1, 0, 0, 1, 0, 0}}
	case style.Rotate, style.Skew:
		// zero angles, as below
	}
	return style.TransformOp{Kind: op.Kind}
}

// lerpTransform interpolates function by function when both lists have
// the same functions (none stands for the other's identities), and
// otherwise as decomposed matrices. A list it can't make a matrix of
// (a percentage translation) switches at the end.
func lerpTransform(a, b []style.TransformOp, t float32) []style.TransformOp {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	if len(a) == 0 {
		a = identities(b)
	}
	if len(b) == 0 {
		b = identities(a)
	}
	if sameKinds(a, b) {
		out := make([]style.TransformOp, len(a))
		for i := range a {
			out[i] = lerpOp(a[i], b[i], t)
		}
		return out
	}
	ma, okA := toMatrix(a)
	mb, okB := toMatrix(b)
	if !okA || !okB {
		if t < 1 {
			return a
		}
		return b
	}
	m := recompose(decompose(ma).lerp(decompose(mb), t))
	return []style.TransformOp{{Kind: style.Matrix, A: m}}
}

func identities(ops []style.TransformOp) []style.TransformOp {
	out := make([]style.TransformOp, len(ops))
	for i, op := range ops {
		out[i] = identity(op)
	}
	return out
}

func sameKinds(a, b []style.TransformOp) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Kind != b[i].Kind {
			return false
		}
	}
	return true
}

func lerpOp(a, b style.TransformOp, t float32) style.TransformOp {
	switch a.Kind {
	case style.Translate:
		return style.TransformOp{Kind: style.Translate, X: lerpLength(a.X, b.X, t), Y: lerpLength(a.Y, b.Y, t)}
	case style.Matrix:
		return style.TransformOp{Kind: style.Matrix, A: recompose(decompose(a.A).lerp(decompose(b.A), t))}
	case style.ScaleOp, style.Rotate, style.Skew:
		// argument by argument, as below
	}
	out := style.TransformOp{Kind: a.Kind}
	for i := range out.A {
		out.A[i] = lerpf(a.A[i], b.A[i], t)
	}
	return out
}

// toMatrix multiplies a list into one matrix(a, b, c, d, e, f), false
// when a translation has a percentage.
func toMatrix(ops []style.TransformOp) ([6]float32, bool) {
	m := [6]float64{1, 0, 0, 1, 0, 0}
	for _, op := range ops {
		a := func(i int) float64 { return float64(op.A[i]) }
		var n [6]float64
		switch op.Kind {
		case style.Translate:
			if op.X.Percent != 0 || op.Y.Percent != 0 {
				return [6]float32{}, false
			}
			n = [6]float64{1, 0, 0, 1, float64(op.X.Px), float64(op.Y.Px)}
		case style.ScaleOp:
			n = [6]float64{a(0), 0, 0, a(1), 0, 0}
		case style.Rotate:
			n = [6]float64{math.Cos(a(0)), math.Sin(a(0)), -math.Sin(a(0)), math.Cos(a(0)), 0, 0}
		case style.Skew:
			n = [6]float64{1, math.Tan(a(1)), math.Tan(a(0)), 1, 0, 0}
		case style.Matrix:
			n = [6]float64{a(0), a(1), a(2), a(3), a(4), a(5)}
		}
		m = [6]float64{
			m[0]*n[0] + m[2]*n[1], m[1]*n[0] + m[3]*n[1],
			m[0]*n[2] + m[2]*n[3], m[1]*n[2] + m[3]*n[3],
			m[0]*n[4] + m[2]*n[5] + m[4], m[1]*n[4] + m[3]*n[5] + m[5],
		}
	}
	var out [6]float32
	for i, v := range m {
		out[i] = float32(v)
	}
	return out, true
}

// decomposed is a 2D matrix as Chrome decomposes it: translate · rotate
// (angle) · skewX (k, as a tangent) · scale(sx, sy).
type decomposed struct {
	tx, ty, angle, sx, sy, k float64
}

func decompose(m [6]float32) decomposed {
	a, b, c, d := float64(m[0]), float64(m[1]), float64(m[2]), float64(m[3])
	sx := math.Hypot(a, b)
	angle := math.Atan2(b, a)
	cs, sn := math.Cos(angle), math.Sin(angle)
	shear, sy := cs*c+sn*d, -sn*c+cs*d
	k := 0.0
	if sy != 0 {
		k = shear / sy
	}
	return decomposed{tx: float64(m[4]), ty: float64(m[5]), angle: angle, sx: sx, sy: sy, k: k}
}

func (x decomposed) lerp(y decomposed, t float32) decomposed {
	if d := y.angle - x.angle; d > math.Pi {
		x.angle += 2 * math.Pi
	} else if d < -math.Pi {
		y.angle += 2 * math.Pi
	}
	f := func(a, b float64) float64 { return a + (b-a)*float64(t) }
	return decomposed{f(x.tx, y.tx), f(x.ty, y.ty), f(x.angle, y.angle), f(x.sx, y.sx), f(x.sy, y.sy), f(x.k, y.k)}
}

func recompose(x decomposed) [6]float32 {
	cs, sn := math.Cos(x.angle), math.Sin(x.angle)
	shear := x.k * x.sy
	return [6]float32{
		float32(cs * x.sx), float32(sn * x.sx),
		float32(cs*shear - sn*x.sy), float32(sn*shear + cs*x.sy),
		float32(x.tx), float32(x.ty),
	}
}

//nolint:gochecknoglobals // CSS's named easing curves, fixed
var (
	easeCurve      = cubicBezier(0.25, 0.1, 0.25, 1)
	easeInCurve    = cubicBezier(0.42, 0, 1, 1)
	easeOutCurve   = cubicBezier(0, 0, 0.58, 1)
	easeInOutCurve = cubicBezier(0.42, 0, 0.58, 1)
)

// easing reads a transition-timing-function as written in the computed
// style.
func easing(src string) ease.Func {
	cs := items(componentsOf(src))
	if len(cs) != 1 {
		return easeCurve
	}
	c := cs[0]
	switch {
	case isWord(c, "linear"):
		return ease.Linear
	case isWord(c, "ease-in"):
		return easeInCurve
	case isWord(c, "ease-out"):
		return easeOutCurve
	case isWord(c, "ease-in-out"):
		return easeInOutCurve
	case isWord(c, "step-start"):
		return steps(1, kwStart)
	case isWord(c, "step-end"):
		return steps(1, kwEnd)
	case c.Kind == Function:
		args := splitOn(items(c.Args), func(c Component) bool { return c.Kind == Comma })
		switch strings.ToLower(c.Text) {
		case "cubic-bezier":
			if len(args) == 4 {
				return cubicBezier(args[0][0].Num, args[1][0].Num, args[2][0].Num, args[3][0].Num)
			}
		case "steps":
			pos := kwEnd
			if len(args) == 2 {
				pos = strings.ToLower(args[1][0].Text)
			}
			return steps(int(args[0][0].Num), pos)
		}
	}
	return easeCurve
}

// cubicBezier is CSS's cubic-bezier(x1, y1, x2, y2): y for the curve's
// point at x = t.
func cubicBezier(x1, y1, x2, y2 float64) ease.Func {
	bez := func(p1, p2, s float64) float64 {
		return 3*(1-s)*(1-s)*s*p1 + 3*(1-s)*s*s*p2 + s*s*s
	}
	slope := func(p1, p2, s float64) float64 {
		return 3*(1-s)*(1-s)*p1 + 6*(1-s)*s*(p2-p1) + 3*s*s*(1-p2)
	}
	return func(t float32) float32 {
		x := float64(t)
		if x <= 0 || x >= 1 {
			return t
		}
		s := x
		for range 8 {
			d := slope(x1, x2, s)
			if math.Abs(d) < 1e-7 {
				break
			}
			s -= (bez(x1, x2, s) - x) / d
		}
		if math.Abs(bez(x1, x2, s)-x) > 1e-6 {
			lo, hi := 0.0, 1.0
			s = x
			for range 40 {
				if bez(x1, x2, s) < x {
					lo = s
				} else {
					hi = s
				}
				s = (lo + hi) / 2
			}
		}
		return float32(bez(y1, y2, s))
	}
}

// steps is CSS's steps(n, position).
func steps(n int, pos string) ease.Func {
	jumps := n
	switch pos {
	case "jump-both":
		jumps = n + 1
	case "jump-none":
		jumps = max(n-1, 1)
	}
	start := pos == kwStart || pos == "jump-start" || pos == "jump-both"
	return func(t float32) float32 {
		step := math.Floor(float64(t) * float64(n))
		if start {
			step++
		}
		if t >= 0 && step < 0 {
			step = 0
		}
		return float32(min(step, float64(jumps)) / float64(jumps))
	}
}
