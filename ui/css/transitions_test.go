package css //nolint:testpackage // tests the unexported matrix decomposition, with the package's test helpers

import (
	"math"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func shown(n *dom.Node) *style.Computed { return dom.Data(n).Shown }

func nearf(a, b float32) bool { return math.Abs(float64(a-b)) < 0.002 }

func srgbNear(t *testing.T, what string, got color.Color, want [4]float32) {
	t.Helper()
	s := got.SRGB()
	if !nearf(s[0], want[0]) || !nearf(s[1], want[1]) || !nearf(s[2], want[2]) || !nearf(got.A, want[3]) {
		t.Errorf("%s: got sRGB %v a %v, want %v", what, s, got.A, want)
	}
}

func transitionPage(t *testing.T, css string) (*Transitions, *dom.Node, *dom.Node, []*Sheet) {
	t.Helper()
	b := el(dom.Button, "b")
	root := page(b)
	sheets := []*Sheet{sheet(t, css)}
	tr := NewTransitions()
	tr.Compute(root, sheets)
	return tr, root, b, sheets
}

func TestIdleShownIsStyle(t *testing.T) {
	tr, root, b, sheets := transitionPage(t,
		`button { color: #000; transition: color 1s linear } button.on { color: #fff }`)
	if shown(b) != styleOf(b) {
		t.Fatal("idle Shown is Style")
	}
	if tr.Step(0.1) {
		t.Error("nothing runs")
	}
	b.AddClass("x")
	tr.Compute(root, sheets)
	if shown(b) != styleOf(b) {
		t.Error("a change no transition covers shows at once")
	}
}

func TestColorTransitionLinear(t *testing.T) {
	tr, root, b, sheets := transitionPage(t,
		`button { color: #000; background-color: #f00; transition: color 1s linear } `+
			`button.on { color: #fff; background-color: #00f }`)
	txt := dom.NewText("go")
	b.Append(txt)
	tr.Compute(root, sheets)
	b.AddClass("on")
	tr.Compute(root, sheets)
	srgbNear(t, "start", shown(b).Color, [4]float32{0, 0, 0, 1})
	if shown(b).Background != styleOf(b).Background {
		t.Error("background has no transition: it switches at once")
	}
	if !tr.Step(0.25) {
		t.Fatal("running")
	}
	srgbNear(t, "quarter", shown(b).Color, [4]float32{0.25, 0.25, 0.25, 1})
	if dom.Data(txt).Shown != shown(b) {
		t.Error("text shows its element's transition")
	}
	if styleOf(b).Color != hex("#fff") {
		t.Error("Style stays the target")
	}
	tr.Step(0.8)
	if shown(b) != styleOf(b) || dom.Data(txt).Shown != styleOf(b) {
		t.Error("finished: Shown is Style again")
	}
}

func TestDelayEasingAndPremultiplied(t *testing.T) {
	tr, root, b, sheets := transitionPage(t,
		`button { background-color: rgba(255, 0, 0, 0); transition: background-color 2s ease-in 1s } `+
			`.on { background-color: #00f }`)
	b.AddClass("on")
	tr.Compute(root, sheets)
	tr.Step(0.5)
	srgbNear(t, "in the delay", shown(b).Background, [4]float32{1, 0, 0, 0})
	tr.Step(1.5)
	e := easeInCurve(0.5)
	srgbNear(t, "eased half", shown(b).Background, [4]float32{0, 0, 1, e})
}

func TestOpacityShadowAndBorderColors(t *testing.T) {
	tr, root, b, sheets := transitionPage(t, `
button { opacity: 1; border: 1px solid #000; box-shadow: 0 0 0 #000; transition: opacity 1s linear,
	border-color 1s linear, box-shadow 1s linear }
.on { opacity: 0; border-color: #fff; box-shadow: 10px 4px 2px #fff, inset 0 0 6px #f00 }`)
	b.AddClass("on")
	tr.Compute(root, sheets)
	tr.Step(0.5)
	s := shown(b)
	if !nearf(s.Opacity, 0.5) {
		t.Errorf("opacity %v", s.Opacity)
	}
	srgbNear(t, "border", s.BorderColor.Left, [4]float32{0.5, 0.5, 0.5, 1})
	if len(s.Shadows) != 2 || !nearf(s.Shadows[0].X, 5) || !nearf(s.Shadows[0].Blur, 1) || !s.Shadows[1].Inset ||
		!nearf(s.Shadows[1].Blur, 3) || !nearf(s.Shadows[1].Color.A, 0.5) {
		t.Errorf("shadows %+v", s.Shadows)
	}
}

func TestTransformTransitions(t *testing.T) {
	tr, root, b, sheets := transitionPage(t, `
button { transition: transform 1s linear }
.a { transform: translate(10px, 50%) rotate(90deg) }
.b { transform: rotate(45deg) scale(2) }`)
	b.AddClass("a")
	tr.Compute(root, sheets)
	tr.Step(0.5)
	ops := shown(b).Transform
	if len(ops) != 2 || !nearf(ops[0].X.Px, 5) || !nearf(ops[0].Y.Percent, 25) || !nearf(ops[1].A[0], math.Pi/4) {
		t.Errorf("none to a list, function by function: %+v", ops)
	}
	tr.Step(1)
	b.RemoveClass("a")
	b.AddClass("b")
	tr.Compute(root, sheets)
	tr.Step(0.5)
	if ops = shown(b).Transform; len(ops) != 2 || ops[0].Kind != style.Translate {
		t.Errorf("a list with %% can't be a matrix: switches at the end: %+v", ops)
	}
	tr.Step(1)
	b.SetStyle("transform: scale(1) rotate(0deg)")
	tr.Compute(root, sheets)
	tr.Step(0.5)
	ops = shown(b).Transform
	if len(ops) != 1 || ops[0].Kind != style.Matrix {
		t.Fatalf("mismatched lists interpolate as a matrix: %+v", ops)
	}
	d := decompose(ops[0].A)
	if !nearf(float32(d.angle), float32(math.Pi/8)) || !nearf(float32(d.sx), 1.5) || !nearf(float32(d.sy), 1.5) {
		t.Errorf("halfway matrix: %+v", d)
	}
}

func TestMatrixRoundTrip(t *testing.T) {
	for _, m := range [][6]float32{{1, 0, 0, 1, 0, 0}, {2, 1, -0.5, 3, 7, -8}, {-1, 0, 0, 1, 0, 0},
		{0.866, 0.5, -1, 1.732, 3, 4}} {
		got := recompose(decompose(m))
		for i := range m {
			if !nearf(got[i], m[i]) {
				t.Errorf("%v came back %v", m, got)
				break
			}
		}
	}
}

func TestReversalIsShortened(t *testing.T) {
	tr, root, b, sheets := transitionPage(t, `button { opacity: 0; transition: opacity 1s linear } .on { opacity: 1 }`)
	b.AddClass("on")
	tr.Compute(root, sheets)
	tr.Step(0.25)
	b.RemoveClass("on")
	tr.Compute(root, sheets)
	if !nearf(shown(b).Opacity, 0.25) {
		t.Fatalf("reverses from the shown value: %v", shown(b).Opacity)
	}
	tr.Step(0.125)
	if !nearf(shown(b).Opacity, 0.125) {
		t.Errorf("reversal takes 0.25 s, not 1 s: %v", shown(b).Opacity)
	}
	if tr.Step(0.13) {
		t.Error("reversal over after 0.25 s")
	}
}

func TestRetargetAndCancel(t *testing.T) {
	tr, root, b, sheets := transitionPage(t, `button { opacity: 0; transition: opacity 1s linear } `+
		`.half { opacity: 0.5 } .on { opacity: 1 } .snap { transition: none; opacity: 1 }`)
	b.AddClass("on")
	tr.Compute(root, sheets)
	tr.Step(0.5)
	b.RemoveClass("on")
	b.AddClass("half")
	tr.Compute(root, sheets)
	if !nearf(shown(b).Opacity, 0.5) {
		t.Errorf("retarget starts at the shown value: %v", shown(b).Opacity)
	}
	b.AddClass("snap")
	tr.Compute(root, sheets)
	if shown(b) != styleOf(b) || tr.Running(b) {
		t.Error("transition: none cancels")
	}
}

func TestInheritedColorFollows(t *testing.T) {
	icon, label, own := el(dom.Span, "icon"), el(dom.Span, "label"), el(dom.Span, "own")
	deep := el(dom.Strong, "deep")
	label.Append(deep)
	b := with(el(dom.Button, "b"), icon, label, own)
	root := page(b)
	sheets := []*Sheet{sheet(t, `button { color: #000; transition: color 1s linear } button.on { color: #fff } `+
		`#own { color: #f00 } #label { color: inherit }`)}
	tr := NewTransitions()
	tr.Compute(root, sheets)
	b.AddClass("on")
	tr.Compute(root, sheets)
	tr.Step(0.5)
	for _, n := range []*dom.Node{icon, label, deep} {
		srgbNear(t, describe(n), shown(n).Color, [4]float32{0.5, 0.5, 0.5, 1})
	}
	if shown(own) != styleOf(own) {
		t.Error("a span with its own colour doesn't follow")
	}
	tr.Step(0.6)
	for _, n := range []*dom.Node{icon, label, deep} {
		if shown(n) != styleOf(n) {
			t.Errorf("%s: Shown is Style once the parent is done", describe(n))
		}
	}
}

func TestRemovedNodesAreDropped(t *testing.T) {
	tr, root, b, sheets := transitionPage(t, `button { opacity: 0; transition: opacity 1s } .on { opacity: 1 }`)
	b.AddClass("on")
	tr.Compute(root, sheets)
	b.Parent().Remove(b)
	if tr.Step(0.1) {
		t.Error("a removed node's transition stops")
	}
}

func TestEasings(t *testing.T) {
	cases := []struct {
		src  string
		t    float32
		want float32
	}{
		{"linear", 0.3, 0.3},
		{"ease", 0.5, 0.8024},
		{"ease-in", 0.5, 0.3153},
		{"ease-out", 0.5, 0.6847},
		{"ease-in-out", 0.5, 0.5},
		{"cubic-bezier(0, 0, 1, 1)", 0.7, 0.7},
		{"steps(4)", 0.3, 0.25},
		{"steps(4, start)", 0.3, 0.5},
		{"steps(4, jump-both)", 0.3, 0.4},
		{"steps(4, jump-none)", 0.5, 0.6667},
		{"step-start", 0.1, 1},
		{"step-end", 0.9, 0},
		{"ease", 1, 1},
	}
	for _, c := range cases {
		if got := easing(c.src)(c.t); !nearf(got, c.want) {
			t.Errorf("%s(%v) = %v, want %v", c.src, c.t, got, c.want)
		}
	}
}

func TestCurrentColorFollowsTheTransition(t *testing.T) {
	tr, root, b, sheets := transitionPage(t, `button { color: #000; border: 1px solid; border-left-color: #f00; `+
		`outline: 1px solid currentColor; transition: color 1s linear } .on { color: #fff }`)
	b.AddClass("on")
	tr.Compute(root, sheets)
	tr.Step(0.5)
	s := shown(b)
	srgbNear(t, "border top", s.BorderColor.Top, [4]float32{0.5, 0.5, 0.5, 1})
	srgbNear(t, "outline", s.OutlineColor, [4]float32{0.5, 0.5, 0.5, 1})
	srgbNear(t, "border left keeps its own colour", s.BorderColor.Left, [4]float32{1, 0, 0, 1})
	marks := style.BorderTopCurrent | style.BorderRightCurrent | style.BorderBottomCurrent | style.OutlineCurrent
	if styleOf(b).CurrentColor != marks {
		t.Errorf("marks: %b", styleOf(b).CurrentColor)
	}
}

func TestAStaticFrameRestylesWithoutAllocating(t *testing.T) {
	root := page(el(dom.Div, "a", "x"), el(dom.P, "", "y"))
	sheets := []*Sheet{sheet(t, ".x { color: #f00; } .y { margin: 4px; }")}
	tr := NewTransitions()
	tr.Compute(root, sheets)
	root.Walk(func(n *dom.Node) bool { dom.Data(n).Dirty = 0; return true })
	if allocs := testing.AllocsPerRun(100, func() { tr.Compute(root, sheets) }); allocs != 0 {
		t.Errorf("a static frame allocates %v times", allocs)
	}
	root.Children()[0].AddClass("y")
	tr.Compute(root, sheets)
	if dom.Data(root.Children()[0]).Style.Margin.Top.Px == 0 {
		t.Error("a change after static frames still restyles")
	}
}

func TestAStateChangeRestylesDescendantsOnlyWhenASelectorReadsIt(t *testing.T) {
	label := el(dom.Span, "", "label")
	item := with(el(dom.Div, "", "item"), label, el(dom.Span, "", "key"))
	root := page(item)
	for _, c := range []struct {
		src      string
		restyled int
	}{
		{".item:hover { background-color: #111; } .label { color: #fff; }", 1},
		{".item:hover > .label { color: #f00; }", 3},
		{".item:focus > .label { color: #f00; } .item:hover { color: #222; }", 3},
		{".item:focus > .label { color: #f00; } .item:hover { background-color: #222; }", 1},
	} {
		tr := NewTransitions()
		sheets := []*Sheet{sheet(t, c.src)}
		tr.Compute(root, sheets)
		item.SetState(dom.Hover, true)
		before := tr.computed
		tr.Compute(root, sheets)
		if got := tr.computed - before; got != c.restyled {
			t.Errorf("%s: hovering .item restyled %d elements, want %d", c.src, got, c.restyled)
		}
		item.SetState(dom.Hover, false)
		tr.Compute(root, sheets)
	}
}

func TestAPaintOnlyChangeLeavesLayoutClean(t *testing.T) {
	item := el(dom.Div, "", "item")
	root := page(item)
	sheets := []*Sheet{sheet(t,
		".item:hover { color: #f00; background-color: #111; box-shadow: 0 0 4px #000; opacity: 0.5; } "+
			".item:focus { padding: 4px; }")}
	tr := NewTransitions()
	tr.Compute(root, sheets)
	root.Walk(func(n *dom.Node) bool { dom.Data(n).Dirty = 0; return true })

	item.SetState(dom.Hover, true)
	tr.Compute(root, sheets)
	if dom.Data(item).Dirty&dom.DirtyLayout != 0 {
		t.Error("a hover that only recolours flags layout")
	}
	if dom.Data(root).Dirty&dom.DirtyDescendant != 0 {
		t.Error("nor tells the root something below needs layout")
	}
	if dom.Data(item).Style.Background == (color.Color{}) {
		t.Error("the hover style still applies")
	}

	item.SetState(dom.Focus, true)
	tr.Compute(root, sheets)
	if dom.Data(item).Dirty&dom.DirtyLayout == 0 || dom.Data(root).Dirty&dom.DirtyDescendant == 0 {
		t.Error("a focus that pads doesn't flag layout on its path")
	}
	root.Walk(func(n *dom.Node) bool { dom.Data(n).Dirty = 0; return true })
	item.Append(dom.NewText("x"))
	item.Children()[0].SetText("y")
	tr.Compute(root, sheets)
	if dom.Data(root).Dirty&dom.DirtyDescendant == 0 {
		t.Error("a text change below still needs layout")
	}
}

func TestAChangeRestylesTheSubtreeOnlyWhenASelectorReadsItAbove(t *testing.T) {
	deep := el(dom.Span, "", "deep")
	panel := with(el(dom.Div, "", "panel"), with(el(dom.Div, "", "row"), deep))
	root := page(panel)
	sheets := []*Sheet{sheet(t, `[data-device="pad"] .row .deep { color: #f00; } .open { background-color: #0f0; }`)}
	count := func(change func()) int {
		tr := NewTransitions()
		tr.Compute(root, sheets)
		change()
		before := tr.computed
		tr.Compute(root, sheets)
		return tr.computed - before
	}
	if got := count(func() { root.SetAttr("data-device", "pad") }); got < 4 {
		t.Errorf("an attribute read above: %d restyled, want the subtree", got)
	}
	if dom.Data(deep).Style.Color != color.Hex("#f00") {
		t.Error("two levels down, the span restyles")
	}
	if got := count(func() { panel.SetStyle("left: 10px") }); got != 1 {
		t.Errorf("a style attribute: %d restyled, want the panel alone", got)
	}
	if got := count(func() { panel.AddClass("open") }); got != 1 {
		t.Errorf("a class no selector reads above: %d restyled, want the panel alone", got)
	}
	if got := count(func() { panel.SetAttr("data-side", "left") }); got != 1 {
		t.Errorf("an attribute no selector reads: %d restyled, want the panel alone", got)
	}
}
