package css //nolint:testpackage // holds the unexported test helpers (sheet, page) and reads computed styles off dom

import (
	"strings"
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

func sheet(t *testing.T, src string) *Sheet {
	t.Helper()
	s, warns := Parse("t.css", src)
	if len(warns) > 0 {
		t.Fatalf("warnings: %s", warnings(warns))
	}
	return s
}

func styleOf(n *dom.Node) *style.Computed { return dom.Data(n).Style }

func hex(s string) color.Color { return linear(value2(s)) }

func value2(src string) Component { return items(componentsOf(src))[0] }

func page(kids ...*dom.Node) *dom.Node {
	return with(dom.NewElement(dom.HTML), with(dom.NewElement(dom.Body), kids...))
}

func TestDefaultParses(t *testing.T) {
	if _, warns := Parse("ua", Default); len(warns) > 0 {
		t.Fatal(warnings(warns))
	}
}

func TestCascadeOrder(t *testing.T) {
	d := el(dom.Div, "x", "a", "b")
	root := page(d)
	Compute(root, []*Sheet{sheet(t, `
#x { color: #00f; }
div.a { color: #0f0; }
.a.b { color: #f00; }
div { background-color: #111 !important; }
#x { background-color: #222; }
.b { opacity: 0.2; }
.a { opacity: 0.4; }
`)})
	s := styleOf(d)
	if s.Color != hex("#00f") {
		t.Errorf("id beats classes: %v", s.Color)
	}
	if s.Background != hex("#111") {
		t.Errorf("!important beats specificity: %v", s.Background)
	}
	if s.Opacity != 0.4 {
		t.Errorf("later rule wins a tie: %v", s.Opacity)
	}
}

func TestInlineAndImportant(t *testing.T) {
	d := el(dom.Div, "x")
	d.SetStyle("color: #0f0; opacity: 0.5 !important; width: 10px")
	root := page(d)
	Compute(root, []*Sheet{sheet(t, `#x { color: #f00; opacity: 0.1 !important; width: 20px !important }`)})
	s := styleOf(d)
	if s.Color != hex("#0f0") || s.Opacity != 0.5 || s.Size.Width != style.Px(20) {
		t.Errorf("color %v opacity %v width %v", s.Color, s.Opacity, s.Size.Width)
	}
}

func TestAgentImportantWinsAndAgentLoses(t *testing.T) {
	b := dom.NewElement(dom.Button)
	root := page(b)
	Compute(root, []*Sheet{sheet(t, `button { display: block; padding: 4px }`)})
	s := styleOf(b)
	if s.Display != style.Block || s.Padding.Top != style.Px(4) {
		t.Errorf("author beats agent: %v %v", s.Display, s.Padding.Top)
	}
}

func TestInheritance(t *testing.T) {
	sp := dom.NewElement(dom.Span)
	d := with(el(dom.Div, "", "box"), sp)
	root := page(d)
	Compute(root, []*Sheet{sheet(t, `
.box { color: #f7efdd; font-size: 20px; padding: 3px; line-height: 1.5; border: 2px solid; }
span { border-color: inherit; padding: inherit; }
`)})
	s := styleOf(sp)
	if s.Color != hex("#f7efdd") || s.Font.Size != 20 || s.LineHeight.Factor != 1.5 {
		t.Errorf("inherited text: %v %v %v", s.Color, s.Font.Size, s.LineHeight)
	}
	if s.Padding.Left != style.Px(3) {
		t.Errorf("explicit inherit: %v", s.Padding.Left)
	}
	if s.BorderColor.Top != hex("#f7efdd") {
		t.Errorf("currentcolor border, inherited: %v", s.BorderColor.Top)
	}
	if s.Border.Top != 0 {
		t.Errorf("span has no border style: %v", s.Border.Top)
	}
	if b := styleOf(d).Border; b.Top != 2 || b.Left != 2 {
		t.Errorf("box border: %+v", b)
	}
	if styleOf(sp).Font.Family[0] != "Go" {
		t.Errorf("family from the agent's html: %v", s.Font.Family)
	}
}

func TestEmRemPercentAndCalc(t *testing.T) {
	inner := el(dom.Div, "in")
	outer := with(el(dom.Div, "out"), inner)
	root := page(outer)
	Compute(root, []*Sheet{sheet(t, `
html { font-size: 10px; }
#out { font-size: 2em; width: calc(100% - 2rem); }
#in { font-size: 150%; padding-left: 1em; margin-top: calc(50% + 1em - 4px); letter-spacing: 0.1em }
`)})
	if s := styleOf(outer); s.Font.Size != 20 ||
		s.Size.Width != (style.Length{Kind: style.Fixed, Px: -20, Percent: 100, IsPercent: true}) {
		t.Errorf("outer: %v %v", s.Font.Size, s.Size.Width)
	}
	s := styleOf(inner)
	if s.Font.Size != 30 || s.Padding.Left != style.Px(30) {
		t.Errorf("inner: %v %v", s.Font.Size, s.Padding.Left)
	}
	if s.Margin.Top != (style.Length{Kind: style.Fixed, Px: 26, Percent: 50, IsPercent: true}) {
		t.Errorf("calc: %v", s.Margin.Top)
	}
	if s.LetterSpacing != 3 {
		t.Errorf("letter-spacing: %v", s.LetterSpacing)
	}
}

func TestVars(t *testing.T) {
	a := el(dom.Div, "a")
	b := el(dom.Div, "b")
	c := el(dom.Div, "c")
	root := page(with(el(dom.Div, "", "theme"), a, b, c))
	Compute(root, []*Sheet{sheet(t, `
:root { --ink: #f00; --pad: 4px; --both: var(--pad) var(--pad2, 8px); }
.theme { --ink: #0f0; }
#a { color: var(--ink); padding: var(--both); }
#b { --x: var(--y); --y: var(--x); color: var(--x, #00f); width: var(--missing); }
#c { --pad: 6px; margin: var(--pad) 0 0; border: var(--pad) solid var(--ink) }
`)})
	if s := styleOf(a); s.Color != hex("#0f0") || s.Padding.Top != style.Px(4) || s.Padding.Right != style.Px(8) {
		t.Errorf("a: %v %v %v", s.Color, s.Padding.Top, s.Padding.Right)
	}
	if s := styleOf(b); s.Color != hex("#00f") || s.Size.Width.Kind != style.Auto {
		t.Errorf("b, cycle and missing: %v %v", s.Color, s.Size.Width)
	}
	if s := styleOf(c); s.Margin.Top != style.Px(6) || s.Border.Left != 6 || s.BorderColor.Left != hex("#0f0") {
		t.Errorf("c, shorthand with var: %v %v %v", s.Margin.Top, s.Border.Left, s.BorderColor.Left)
	}
	if v := styleOf(a).Vars["--both"]; v != "4px 8px" {
		t.Errorf("--both is %q", v)
	}
}

func TestInvalidVarIsUnset(t *testing.T) {
	sp := dom.NewElement(dom.Span)
	root := page(with(el(dom.Div, "p"), sp))
	Compute(root, []*Sheet{sheet(t, `
#p { color: #0f0; --w: red; --c: 12px }
span { color: var(--c); width: var(--w); }
`)})
	if s := styleOf(sp); s.Color != hex("#0f0") || s.Size.Width.Kind != style.Auto {
		t.Errorf("unset: %v %v", s.Color, s.Size.Width)
	}
}

func TestStatesAndStructure(t *testing.T) {
	b1, b2 := dom.NewElement(dom.Button), dom.NewElement(dom.Button)
	root := page(with(el(dom.Nav, ""), b1, b2))
	sheets := []*Sheet{sheet(t, `
button { opacity: 0.5 }
button:hover:not(:disabled) { opacity: 1 }
button:first-child { order: 1 }
button:last-child { order: 2 }
`)}
	Compute(root, sheets)
	if styleOf(b1).Order != 1 || styleOf(b2).Order != 2 {
		t.Errorf("first/last: %v %v", styleOf(b1).Order, styleOf(b2).Order)
	}
	before := styleOf(b2)
	b1.SetState(dom.Hover, true)
	Compute(root, sheets)
	if styleOf(b1).Opacity != 1 {
		t.Errorf("hover: %v", styleOf(b1).Opacity)
	}
	if styleOf(b2) != before {
		t.Error("b2 restyled though nothing changed for it")
	}
	b1.SetState(dom.Disabled, true)
	Compute(root, sheets)
	if styleOf(b1).Opacity != 0.5 {
		t.Errorf("hover but disabled: %v", styleOf(b1).Opacity)
	}
}

func TestRestyleOnlyWhatChanged(t *testing.T) {
	a, b := el(dom.Div, "a"), el(dom.Div, "b")
	kid := dom.NewElement(dom.Span)
	txt := dom.NewText("hi")
	a.Append(kid)
	kid.Append(txt)
	root := page(a, b)
	sheets := []*Sheet{sheet(t, `.big { font-size: 30px } .big span { color: #f00 }`)}
	Compute(root, sheets)
	if dom.Data(txt).Style != styleOf(kid) {
		t.Error("text takes its parent's style")
	}
	root.Walk(func(n *dom.Node) bool { dom.Data(n).Dirty = 0; return true })
	sb, skid := styleOf(b), styleOf(kid)

	a.AddClass("big")
	Compute(root, sheets)
	if styleOf(b) != sb {
		t.Error("sibling restyled")
	}
	if styleOf(kid) == skid || styleOf(kid).Font.Size != 30 || styleOf(kid).Color != hex("#f00") {
		t.Errorf("child not restyled: %v", styleOf(kid).Font.Size)
	}
	if dom.Data(txt).Style != styleOf(kid) {
		t.Error("text keeps its parent's style")
	}
	if dom.Data(kid).Dirty&dom.DirtyLayout == 0 || dom.Data(b).Dirty&dom.DirtyLayout != 0 {
		t.Error("DirtyLayout on what changed only")
	}
	root.Walk(func(n *dom.Node) bool {
		if dom.Data(n).Dirty&(dom.DirtyStyle|dom.DirtyTree) != 0 {
			t.Errorf("%s still dirty", describe(n))
		}
		return true
	})
}

func TestBlockification(t *testing.T) {
	sp, abs := dom.NewElement(dom.Span), dom.NewElement(dom.Span)
	root := page(with(el(dom.Div, "f"), sp), abs)
	Compute(root,
		[]*Sheet{sheet(t, `#f { display: flex } #f > span { display: inline-flex } span { position: absolute }`)})
	if styleOf(sp).Display != style.FlexDisplay || styleOf(abs).Display != style.Block {
		t.Errorf("%v %v", styleOf(sp).Display, styleOf(abs).Display)
	}
	if styleOf(root).Display != style.Block {
		t.Errorf("root: %v", styleOf(root).Display)
	}
}

func TestComputedValues(t *testing.T) {
	d := el(dom.Div, "d")
	root := page(d)
	Compute(root, []*Sheet{sheet(t, `#d {
  display: grid; grid-template-columns: 10px minmax(0, 1fr) repeat(auto-fill, 20px);
  grid-auto-rows: 2em; grid-column: 2 / span 3; grid-auto-flow: column dense;
  gap: 6px 12px; flex: 2; overflow: hidden visible; aspect-ratio: 16 / 9;
  font-weight: bolder; font-style: italic; text-overflow: ellipsis; white-space: nowrap;
  box-shadow: 1px 2px 3px #000, inset 0 0 4px 1px; transform: translate(10px, 50%) rotate(0.5turn) scale(2);
  transition: opacity 200ms, color 1s ease-in 0.5s; z-index: 3; cursor: pointer; visibility: hidden;
  border-radius: 4px 50%; outline: 2px dashed #fff; border-image: url(p.png) 8 4;
  font-family: "My Face", monospace;
}`)})
	s := styleOf(d)
	if len(s.Columns) != 3 || s.Columns[0].Tracks[0].Min != style.Px(10) ||
		s.Columns[1].Tracks[0] != (style.Track{Min: style.Px(0), Max: style.Length{Kind: style.Fr, Px: 1}}) ||
		s.Columns[2].Count != style.AutoFill {
		t.Errorf("columns: %+v", s.Columns)
	}
	if s.AutoRows[0].Min != style.Px(32) || s.Column != [2]style.GridLine{{Line: 2}, {Span: 3}} ||
		s.Flow != style.FlowColumnDense {
		t.Errorf("grid: %v %v %v", s.AutoRows, s.Column, s.Flow)
	}
	if s.Gap.Height != style.Px(6) || s.Gap.Width != style.Px(12) || s.Grow != 2 || s.Shrink != 1 || s.Basis != pct(0) {
		t.Errorf("flex: %v %v %v %v", s.Gap, s.Grow, s.Shrink, s.Basis)
	}
	if s.Overflow != (style.Size[style.Overflow]{Width: style.Hidden, Height: style.OverflowAuto}) {
		t.Errorf("overflow: %v", s.Overflow)
	}
	if s.AspectRatio != 16.0/9 || s.Font.Weight != 700 || !s.Font.Italic || !s.Ellipsis ||
		s.WhiteSpace != style.NoWrapSpace {
		t.Errorf("misc: %v %v %v %v %v", s.AspectRatio, s.Font.Weight, s.Font.Italic, s.Ellipsis, s.WhiteSpace)
	}
	lookOf(t, s)
}

// lookOf checks the rest of TestComputedValues' style: shadows,
// transforms, transitions and the box's look.
func lookOf(t *testing.T, s *style.Computed) {
	t.Helper()
	if len(s.Shadows) != 2 || s.Shadows[0].Blur != 3 || !s.Shadows[1].Inset || s.Shadows[1].Spread != 1 ||
		s.Shadows[1].Color != s.Color {
		t.Errorf("shadows: %+v", s.Shadows)
	}
	if len(s.Transform) != 3 || s.Transform[0].Y != pct(50) || s.Transform[1].A[0] < 3.14 || s.Transform[2].A[1] != 2 {
		t.Errorf("transform: %+v", s.Transform)
	}
	want := []style.Transition{{Property: "opacity", Duration: 0.2, Easing: "ease"},
		{Property: "color", Duration: 1, Delay: 0.5, Easing: "ease-in"}}
	if len(s.Transitions) != 2 || s.Transitions[0] != want[0] || s.Transitions[1] != want[1] {
		t.Errorf("transitions: %+v", s.Transitions)
	}
	if s.ZAuto || s.ZIndex != 3 || s.Cursor != "pointer" || s.Visible {
		t.Errorf("z %v %v cursor %q visible %v", s.ZAuto, s.ZIndex, s.Cursor, s.Visible)
	}
	if s.Radius.TopRight.Width != pct(50) || s.OutlineWidth != 2 || s.OutlineStyle != style.BorderDashed {
		t.Errorf("look: %v %v %v", s.Radius, s.OutlineWidth, s.OutlineStyle)
	}
	if s.BorderImage.Source != "p.png" ||
		s.BorderImage.Slice != (style.Sides[float32]{Top: 8, Right: 4, Bottom: 8, Left: 4}) {
		t.Errorf("border-image: %+v", s.BorderImage)
	}
	if len(s.Font.Family) != 2 || s.Font.Family[0] != "My Face" || s.Font.Family[1] != "monospace" {
		t.Errorf("family: %v", s.Font.Family)
	}
}

func TestInlineStyleWarnings(t *testing.T) {
	d := el(dom.Div, "d", "x")
	d.SetStyle("float: left")
	warns := Compute(page(d), nil)
	if got := warnings(warns); got != "<div#d.x> style:1:1: floats are not supported; use flex or grid" {
		t.Errorf("got %s", got)
	}
}

func TestFontFaces(t *testing.T) {
	s, warns := Parse("ui/tool.css", `
@font-face { font-family: "Silk Screen"; src: url(fonts/Silk.ttf); font-weight: bold; font-style: italic }
@font-face { font-family: Pixel; src: url("Pixel.ttf") }
@font-face { src: url(x.ttf) }
`)
	if got := warnings(warns); got != "ui/tool.css:4:1: @font-face needs font-family and src; dropped" {
		t.Errorf("warnings: %s", got)
	}
	want := []FontFace{{"Silk Screen", "ui/fonts/Silk.ttf", 700, true}, {"Pixel", "ui/Pixel.ttf", 400, false}}
	got := s.FontFaces()
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("faces: %+v", got)
	}
}

func pct(v float32) style.Length { return style.Length{Kind: style.Fixed, Percent: v, IsPercent: true} }

func TestScrollbarWidth(t *testing.T) {
	a, b, c := el(dom.Div, "a"), el(dom.Div, "b"), el(dom.Div, "c")
	Compute(page(a, b, c), []*Sheet{sheet(t, `#a { overflow: auto } #b { overflow-y: scroll } #c { overflow: hidden }`)})
	if styleOf(a).ScrollbarWidth != 15 || styleOf(b).ScrollbarWidth != 15 || styleOf(c).ScrollbarWidth != 0 {
		t.Errorf("%v %v %v", styleOf(a).ScrollbarWidth, styleOf(b).ScrollbarWidth, styleOf(c).ScrollbarWidth)
	}
}

func TestZeroPercentIsAPercent(t *testing.T) {
	d := el(dom.Div, "d")
	Compute(page(d), []*Sheet{sheet(t, `#d { flex-basis: 0%; width: 0px; margin-left: calc(0% + 4px) }`)})
	s := styleOf(d)
	if !s.Basis.IsPercent || s.Size.Width.IsPercent || !s.Margin.Left.IsPercent {
		t.Errorf("basis %+v width %+v margin %+v", s.Basis, s.Size.Width, s.Margin.Left)
	}
}

func TestOverflowWrap(t *testing.T) {
	a, b := el(dom.Div, "a"), el(dom.Div, "b")
	Compute(page(a, b), []*Sheet{sheet(t, `#a { overflow-wrap: break-word } #b { overflow-wrap: anywhere }`)})
	if s := styleOf(a); !s.BreakWord || s.BreakAnywhere {
		t.Errorf("break-word: %v %v", s.BreakWord, s.BreakAnywhere)
	}
	if s := styleOf(b); !s.BreakWord || !s.BreakAnywhere {
		t.Errorf("anywhere: %v %v", s.BreakWord, s.BreakAnywhere)
	}
}

func TestSwitchLook(t *testing.T) {
	sw, box := el(dom.Input, "sw"), el(dom.Input, "box")
	sw.SetAttr("type", "checkbox")
	sw.SetAttr("switch", "")
	box.SetAttr("type", "checkbox")
	root := page(sw, box)
	Compute(root, nil)
	if s := styleOf(sw); s.Size.Width != style.Px(32) || s.Background != hex("#efefef") || s.Border.Top != 1 ||
		s.Color != hex("#767676") {
		t.Errorf("switch off: %v %v %v %v", s.Size.Width, s.Background, s.Border.Top, s.Color)
	}
	sw.SetState(dom.Checked, true)
	box.SetState(dom.Checked, true)
	Compute(root, nil)
	if s := styleOf(sw); s.Background != hex("#0075ff") || s.Color != hex("#ffffff") {
		t.Errorf("switch on: %v %v", s.Background, s.Color)
	}
	if s := styleOf(box); s.Size.Width != style.Px(13) || s.Background != hex("#0075ff") {
		t.Errorf("checkbox on: %v %v", s.Size.Width, s.Background)
	}
	sw.SetAttr("disabled", "")
	box.SetAttr("disabled", "")
	sw.SetState(dom.Disabled, true)
	box.SetState(dom.Disabled, true)
	Compute(root, nil)
	if s := styleOf(sw); s.Background != hex("#d1d1d1") || s.Color != hex("#f8f8f8") {
		t.Errorf("switch on, disabled: %v %v", s.Background, s.Color)
	}
	if s := styleOf(box); s.Background != hex("#d1d1d1") || s.Color != hex("#ededed") {
		t.Errorf("checkbox on, disabled (native Chrome's): %v %v", s.Background, s.Color)
	}
}

func TestSummaryDisclosure(t *testing.T) {
	closed, open, plain := el(dom.Summary, "closed"), el(dom.Summary, "open"), el(dom.Summary, "plain")
	d1, d2, d3 := with(el(dom.Details, ""), closed), with(el(dom.Details, ""), open), with(el(dom.Details, ""), plain)
	d2.SetAttr("open", "")
	Compute(page(d1, d2, d3), []*Sheet{sheet(t, `#plain { list-style: none }`)})
	if s := styleOf(closed); s.ListStyle != style.ListDisclosureClosed || s.Display != style.Block {
		t.Errorf("closed: %v %v", s.ListStyle, s.Display)
	}
	if styleOf(open).ListStyle != style.ListDisclosureOpen || styleOf(plain).ListStyle != style.ListNone {
		t.Errorf("open %v plain %v", styleOf(open).ListStyle, styleOf(plain).ListStyle)
	}
}

func TestPlaceholder(t *testing.T) {
	in, area, div := el(dom.Input, "in"), el(dom.Textarea, "area"), el(dom.Div, "d")
	in.SetAttr("placeholder", "Name")
	area.SetAttr("placeholder", "Notes")
	Compute(page(in, area, div), []*Sheet{sheet(t, `
#area { color: #123456; --ph: #abcdef; }
textarea::placeholder { color: var(--ph); }
.form ::placeholder { color: currentColor; }
`)})
	if s := styleOf(in); !s.HasPlaceholder || s.PlaceholderColor != hex("#757575") {
		t.Errorf("UA placeholder: %v %v", s.HasPlaceholder, s.PlaceholderColor)
	}
	if s := styleOf(area); s.PlaceholderColor != hex("#abcdef") {
		t.Errorf("author placeholder with var: %v", s.PlaceholderColor)
	}
	if styleOf(in).Color == styleOf(in).PlaceholderColor {
		t.Error("the element's own colour is untouched")
	}
	if styleOf(div).HasPlaceholder {
		t.Error("an element without a placeholder attribute shows none to style")
	}
}

func TestPlaceholderRulesTakeOnlyColor(t *testing.T) {
	_, warns := Parse("t.css",
		"input::placeholder { color: #f00; opacity: 0.5 } p, input::placeholder { color: #f00 } p::before { color: #f00 }")
	got := warnings(warns)
	for _, want := range []string{"::placeholder takes only color, not opacity", "::placeholder can't share a rule",
		"pseudo-elements are not supported"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestBackgroundPosition(t *testing.T) {
	cases := map[string]style.BackgroundPosition{
		"right 6px center": {X: style.Px(6), Y: pct(50), FromRight: true},
		"center":           {X: pct(50), Y: pct(50)},
		"top":              {X: pct(50), Y: pct(0)},
		"10px 20%":         {X: style.Px(10), Y: pct(20)},
		"bottom 4px left":  {X: pct(0), Y: style.Px(4), FromBottom: true},
		"left 10% bottom":  {X: pct(10), Y: pct(100)},
	}
	for src, want := range cases {
		d := el(dom.Div, "d")
		Compute(page(d), []*Sheet{sheet(t, "#d { background-position: "+src+" }")})
		if got := styleOf(d).BackgroundPosition; got != want {
			t.Errorf("%s: got %+v, want %+v", src, got, want)
		}
	}
	for _, bad := range []string{"left right", "center 4px left", "1px 2px 3px", "top top"} {
		if err := validate("background-position", componentsOf(bad)); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestSelectArrowAndRelativeURLs(t *testing.T) {
	s, b := el(dom.Select, "s"), el(dom.Div, "b")
	s2, _ := Parse("ui/theme/tool.css", "#b { background: url(img/panel.png) no-repeat }")
	Compute(page(s, b), []*Sheet{s2})
	st := styleOf(s)
	if st.BackgroundImage.Source != "ember:select-arrow" || !st.BackgroundNoRepeat || !st.BackgroundPosition.FromRight {
		t.Errorf("select: %+v %v %+v", st.BackgroundImage, st.BackgroundNoRepeat, st.BackgroundPosition)
	}
	if _, ok := DefaultImages()["ember:select-arrow"]; !ok {
		t.Error("the arrow's PNG")
	}
	if src := styleOf(b).BackgroundImage.Source; src != "ui/theme/img/panel.png" || !styleOf(b).BackgroundNoRepeat {
		t.Errorf("relative to the sheet: %q", src)
	}
}

func TestObjectFitAndPosition(t *testing.T) {
	a, b := el(dom.Img, "a"), el(dom.Img, "b")
	Compute(page(a, b),
		[]*Sheet{sheet(t, `#a { object-fit: cover; object-position: left top } #b { object-fit: scale-down }`)})
	if s := styleOf(a); s.ObjectFit != style.FitCover ||
		s.ObjectPosition != (style.BackgroundPosition{X: pct(0), Y: pct(0)}) {
		t.Errorf("a: %v %+v", s.ObjectFit, s.ObjectPosition)
	}
	if s := styleOf(b); s.ObjectFit != style.FitScaleDown ||
		s.ObjectPosition != (style.BackgroundPosition{X: pct(50), Y: pct(50)}) {
		t.Errorf("b, initial position: %v %+v", s.ObjectFit, s.ObjectPosition)
	}
}

func TestTextShadowRenderingAccentCaretSelection(t *testing.T) {
	a := el(dom.Div, "a")
	kid := el(dom.Span, "kid")
	a.Append(kid)
	sel := el(dom.P, "sel")
	Compute(page(a, sel), []*Sheet{sheet(t, `
#a { color: #f00; text-shadow: 1px 2px 3px #000, 0 0 4px; image-rendering: pixelated; caret-color: #00f; }
#kid { color: #0f0; }
#sel::selection { background-color: rgba(255, 207, 118, 0.4); }
`)})
	s := styleOf(a)
	if len(s.TextShadows) != 2 || s.TextShadows[0].Blur != 3 || s.TextShadows[1].Color != hex("#f00") {
		t.Errorf("text-shadow: %+v", s.TextShadows)
	}
	if s.ImageRendering != style.RenderPixelated || !s.AccentAuto || s.AccentColor != hex("#f00") || s.CaretAuto ||
		s.CaretColor != hex("#00f") {
		t.Errorf("a: %v %v %v %v %v", s.ImageRendering, s.AccentAuto, s.AccentColor, s.CaretAuto, s.CaretColor)
	}
	k := styleOf(kid)
	if k.ImageRendering != style.RenderPixelated || len(k.TextShadows) != 2 {
		t.Errorf("inherited: %v %v", k.ImageRendering, k.TextShadows)
	}
	if !k.AccentAuto || k.AccentColor != hex("#0f0") {
		t.Errorf("an inherited auto follows the child's own color: %v", k.AccentColor)
	}
	if k.CaretColor != hex("#00f") {
		t.Errorf("an inherited colour stays: %v", k.CaretColor)
	}
	if p := styleOf(sel); !p.HasSelection || p.SelectionColor.A < 0.39 || p.SelectionColor.A > 0.41 {
		t.Errorf("::selection: %v %v", p.HasSelection, p.SelectionColor)
	}
	if styleOf(a).HasSelection {
		t.Error("no ::selection rule for #a")
	}
	_, warns := Parse("t.css", "p::selection { color: red } p::selection, p::placeholder { color: red }")
	got := warnings(warns)
	for _, want := range []string{"::selection takes only background-color, not color", "can't share a rule"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestListItemIsBlockWithAMarkerFlag(t *testing.T) {
	a, b := el(dom.Summary, "a"), el(dom.Summary, "b")
	Compute(page(with(el(dom.Details, ""), a), with(el(dom.Details, ""), b)), []*Sheet{sheet(t, `#b { display: block }`)})
	if s := styleOf(a); s.Display != style.Block || !s.ListItem {
		t.Errorf("UA summary: %v %v", s.Display, s.ListItem)
	}
	if s := styleOf(b); s.Display != style.Block || s.ListItem {
		t.Errorf("display: block: %v %v", s.Display, s.ListItem)
	}
}

func TestTruncateStartCutsTheStart(t *testing.T) {
	cut, plain := el(dom.P, "", "truncate-start"), el(dom.P, "")
	Compute(page(cut, plain), nil)
	if !styleOf(cut).EllipsisStart || styleOf(plain).EllipsisStart {
		t.Errorf("class truncate-start: %v, without: %v", styleOf(cut).EllipsisStart, styleOf(plain).EllipsisStart)
	}
}

func TestScrollbarWidthNoneLeavesNoGutter(t *testing.T) {
	none, thin, auto := el(dom.Div, "", "none"), el(dom.Div, "", "thin"), el(dom.Div, "", "auto")
	s, warns := Parse("t.css", `div { overflow: auto; } .none { scrollbar-width: none; } .thin { scrollbar-width: thin; }`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	Compute(page(none, thin, auto), []*Sheet{s})
	if c := styleOf(none); c.ScrollbarWidth != 0 || !c.ScrollbarNone {
		t.Errorf("none: gutter %v, none %v", c.ScrollbarWidth, c.ScrollbarNone)
	}
	if styleOf(thin).ScrollbarWidth != 15 || styleOf(auto).ScrollbarWidth != 15 {
		t.Errorf("thin and auto keep the 15-point gutter: %v %v", styleOf(thin).ScrollbarWidth, styleOf(auto).ScrollbarWidth)
	}
	if !strings.Contains(s.String(), "scrollbar-width: none") {
		t.Errorf("it travels to Chrome: %s", s.String())
	}
}

func TestBorderSpacing(t *testing.T) {
	plain, tight, two := el(dom.Table, ""), el(dom.Table, "", "tight"), el(dom.Table, "", "two")
	cell := el(dom.Td, "")
	tight.Append(cell)
	s, warns := Parse("t.css", `.tight { border-spacing: 0; } .two { border-spacing: 4px 6px; }`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	Compute(page(plain, tight, two), []*Sheet{s})
	for n, want := range map[*dom.Node][2]float32{plain: {2, 2}, tight: {0, 0}, two: {4, 6}, cell: {0, 0}} {
		if got := styleOf(n).BorderSpacing; got != want {
			t.Errorf("%s: %v, want %v", describe(n), got, want)
		}
	}
}

func TestCaptionSideInheritsAndTableLayoutDoesNot(t *testing.T) {
	plain, below := el(dom.Table, ""), el(dom.Table, "", "below")
	caption := el(dom.Caption, "")
	below.Append(caption)
	s, warns := Parse("t.css", `.below { caption-side: bottom; table-layout: fixed; }`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	Compute(page(plain, below), []*Sheet{s})
	if got := styleOf(plain); got.CaptionSide != style.CaptionTop || got.TableLayout != style.TableAuto {
		t.Errorf("top and auto at first: %v %v", got.CaptionSide, got.TableLayout)
	}
	if got := styleOf(below); got.CaptionSide != style.CaptionBottom || got.TableLayout != style.TableFixed {
		t.Errorf("the table's own: %v %v", got.CaptionSide, got.TableLayout)
	}
	if got := styleOf(caption); got.CaptionSide != style.CaptionBottom || got.TableLayout != style.TableAuto {
		t.Errorf("its caption inherits caption-side alone: %v %v", got.CaptionSide, got.TableLayout)
	}
	if got := styleOf(caption).TextAlign; got != style.Center {
		t.Errorf("a caption is centred: %v", got)
	}
}

func TestTextTransform(t *testing.T) {
	up, low, capd := el(dom.P, "", "up"), el(dom.P, "", "low"), el(dom.P, "", "cap")
	none, kid := el(dom.P, ""), el(dom.Span, "")
	up.Append(kid)
	s, warns := Parse("t.css",
		`.up { text-transform: uppercase } .low { text-transform: lowercase } .cap { text-transform: capitalize }`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	Compute(page(up, low, capd, none), []*Sheet{s})
	for n, want := range map[*dom.Node]style.TextTransform{up: style.Uppercase, low: style.Lowercase,
		capd: style.Capitalize, none: style.NoTransform, kid: style.Uppercase} {
		if got := styleOf(n).TextTransform; got != want {
			t.Errorf("%s: %v, want %v", describe(n), got, want)
		}
	}
}

func TestBackgroundSize(t *testing.T) {
	both, one, auto, plain := el(dom.Div, "", "both"), el(dom.Div, "", "one"), el(dom.Div, "", "auto"), el(dom.Div, "")
	s, warns := Parse("t.css",
		`.both { background-size: 8px 4px } .one { background-size: 1em } .auto { background-size: auto }`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	Compute(page(both, one, auto, plain), []*Sheet{s})
	for n, want := range map[*dom.Node][2]float32{both: {8, 4}, one: {16, 0}, auto: {0, 0}, plain: {0, 0}} {
		if got := styleOf(n).BackgroundSize; got != want {
			t.Errorf("%s: %v, want %v", describe(n), got, want)
		}
	}
}

func TestAnAttributeChangeRestylesTheWholeSubtreeUnderIt(t *testing.T) {
	deep := el(dom.Span, "deep")
	mid := el(dom.Div, "mid")
	mid.Append(deep)
	root := page(mid)
	sheets := []*Sheet{sheet(t, `[data-mode="dark"] #deep { color: #ff0000 }`)}
	Compute(root, sheets)
	if styleOf(deep).Color == color.Hex("#ff0000") {
		t.Fatal("red before the attribute")
	}
	root.SetAttr("data-mode", "dark")
	Compute(root, sheets)
	if styleOf(deep).Color != color.Hex("#ff0000") {
		t.Error("two levels under the change, the span restyles")
	}
}

func TestARadioIsChromesRoundThirteen(t *testing.T) {
	radio := el(dom.Input, "r")
	radio.SetAttr("type", "radio")
	root := page(radio)
	Compute(root, nil)
	if s := styleOf(radio); s.Size.Width != style.Px(13) || s.Size.Height != style.Px(13) ||
		s.BorderColor.Top != hex("#767676") {
		t.Errorf("radio: %v %v %v", s.Size.Width, s.Size.Height, s.BorderColor.Top)
	}
	radio.SetState(dom.Checked, true)
	Compute(root, nil)
	if s := styleOf(radio); s.BorderColor.Top != hex("#0075ff") || s.Color != hex("#0075ff") {
		t.Errorf("checked: %v %v", s.BorderColor.Top, s.Color)
	}
	radio.SetAttr("disabled", "")
	radio.SetState(dom.Disabled, true)
	Compute(root, nil)
	if s := styleOf(radio); s.Color != hex("#d1d1d1") || s.Background != hex("#f8f8f8") {
		t.Errorf("checked, disabled (native Chrome's): %v %v", s.Color, s.Background)
	}
}

func TestAStateChangeRestylesEveryDescendant(t *testing.T) {
	for _, c := range []struct {
		name  string
		state dom.State
	}{{"focus", dom.Focus}, {"hover", dom.Hover}} {
		t.Run(c.name, func(t *testing.T) {
			child, grand, deep := el(dom.Span, "", "child"), el(dom.Span, "", "grand"), el(dom.Span, "", "deep")
			lb := with(el(dom.Div, "", "lb"), child, with(el(dom.Div, "", "b"), grand, with(el(dom.Div, ""), deep)))
			root := page(lb)
			sheets := []*Sheet{sheet(t, `
.child, .grand, .deep { display: none }
.lb:`+c.name+` > .child { display: block }
.lb:`+c.name+` > .b > .grand { display: block }
.lb:`+c.name+` .deep { display: block }
`)}
			Compute(root, sheets)
			shown := func() [3]bool {
				return [3]bool{
					styleOf(child).Display == style.Block,
					styleOf(grand).Display == style.Block,
					styleOf(deep).Display == style.Block,
				}
			}
			if got := shown(); got != [3]bool{} {
				t.Fatalf("before: %v", got)
			}
			lb.SetState(c.state, true)
			Compute(root, sheets)
			if got := shown(); got != [3]bool{true, true, true} {
				t.Errorf("on: child, grandchild, descendant shown %v", got)
			}
			lb.SetState(c.state, false)
			Compute(root, sheets)
			if got := shown(); got != [3]bool{} {
				t.Errorf("off: %v", got)
			}
		})
	}
}
