package css //nolint:testpackage // holds the unexported el helper the other tests use

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/dom"
)

func el(tag dom.Tag, id string, classes ...string) *dom.Node {
	n := dom.NewElement(tag)
	n.SetID(id)
	for _, c := range classes {
		n.AddClass(c)
	}
	return n
}

func with(n *dom.Node, kids ...*dom.Node) *dom.Node {
	n.Append(kids...)
	return n
}

func mustSelector(t *testing.T, src string) Selector {
	t.Helper()
	sels, err := ParseSelectors(src)
	if err != nil {
		t.Fatalf("%q: %v", src, err)
	}
	if len(sels) != 1 {
		t.Fatalf("%q: %d selectors", src, len(sels))
	}
	return sels[0]
}

func TestSpecificity(t *testing.T) {
	cases := []struct {
		src  string
		want Specificity
	}{
		{"*", Specificity{0, 0, 0}},
		{"div", Specificity{0, 0, 1}},
		{"DIV.a.b", Specificity{0, 2, 1}},
		{"#x", Specificity{1, 0, 0}},
		{"nav > div p.item:hover", Specificity{0, 2, 3}},
		{"#a #b .c", Specificity{2, 1, 0}},
		{":not(#a, .b)", Specificity{1, 0, 0}},
		{"button:not(:disabled):focus-visible", Specificity{0, 2, 1}},
		{":root", Specificity{0, 1, 0}},
		{"p:first-child:last-child", Specificity{0, 2, 1}},
		{`input[type="checkbox"][checked]`, Specificity{0, 2, 1}},
	}
	for _, c := range cases {
		if got := mustSelector(t, c.src).Specificity; got != c.want {
			t.Errorf("%q: got %v, want %v", c.src, got, c.want)
		}
	}
}

func TestSelectorString(t *testing.T) {
	for _, src := range []string{`input[type="range"]`, "nav > div p.item:hover", "#a .b", "button:not(:disabled)"} {
		if got := mustSelector(t, src).String(); got != src {
			t.Errorf("got %q, want %q", got, src)
		}
	}
}

func TestMatches(t *testing.T) {
	a1 := el(dom.Span, "", "item")
	a1.SetAttr("title", "tip")
	a2 := el(dom.Span, "", "item", "on")
	a2.SetState(dom.Hover, true)
	a3 := el(dom.Span, "last")
	a3.SetState(dom.Disabled, true)
	ul := with(el(dom.Div, ""),
		with(el(dom.P, ""), dom.NewText("x "), a1, dom.NewText(" y")),
		with(el(dom.P, ""), a2),
		with(el(dom.P, ""), a3),
	)
	nav := with(el(dom.Nav, "main"), ul)
	body := with(el(dom.Body, ""), nav)

	cases := []struct {
		src  string
		e    *dom.Node
		want bool
	}{
		{"span", a1, true},
		{"SPAN", a1, true},
		{"*", a1, true},
		{"nav span", a1, true},
		{"nav > span", a1, false},
		{"nav > div > p > span", a1, true},
		{"body p > span.item", a2, true},
		{"#main span#last", a3, true},
		{".item.on", a1, false},
		{".item.on", a2, true},
		{"span:hover", a2, true},
		{"span:hover", a1, false},
		{"span:disabled", a3, true},
		{"span:not(:disabled)", a3, false},
		{"span:not(.on, #last)", a1, true},
		{"span:not(.on, #last)", a2, false},
		{"p:first-child > span", a1, true},
		{"p:first-child > span", a2, false},
		{"p:last-child span", a3, true},
		{"span:first-child:last-child", a1, true},
		{":root", body, true},
		{":root", nav, false},
		{"body:first-child", body, false},
		{"div nav", ul, false},
		{":not(nav span)", a1, false},
		{"section span", a1, false},
		{"[title]", a1, true},
		{`span[title="tip"]`, a1, true},
		{`span[title=tip]`, a1, true},
		{`span[title="Tip"]`, a1, false},
		{"[title]", a2, false},
		{"[ID=last]", a3, true},
		{`[class="item on"]`, a2, true},
	}
	for _, c := range cases {
		if got := mustSelector(t, c.src).Matches(c.e); got != c.want {
			t.Errorf("%q on <%s>: got %v", c.src, c.e.Tag(), got)
		}
	}
}

func TestParseSelectorsErrors(t *testing.T) {
	for _, src := range []string{"", "p,", ".", "p span > ", "p ~ span", "[x^=y]", "[x=y i]", "[=y]", "[x=]", "p|span",
		":hover()", ":not()", "p::after", "p.b#1x", "blink"} {
		if _, err := ParseSelectors(src); err == nil {
			t.Errorf("%q: no error", src)
		}
	}
}

func TestNthChild(t *testing.T) {
	var rows []*dom.Node
	body := el(dom.Tbody, "")
	for range 6 {
		r := el(dom.Tr, "")
		body.Append(dom.NewText(" "), r)
		rows = append(rows, r)
	}
	for src, want := range map[string][]bool{
		"tr:nth-child(odd)":    {true, false, true, false, true, false},
		"tr:nth-child(even)":   {false, true, false, true, false, true},
		"tr:nth-child(3)":      {false, false, true, false, false, false},
		"tr:nth-child(3n)":     {false, false, true, false, false, true},
		"tr:nth-child(2n + 1)": {true, false, true, false, true, false},
		"tr:nth-child(-n + 2)": {true, true, false, false, false, false},
		"tr:nth-child(n+5)":    {false, false, false, false, true, true},
		"tr:nth-child(+3n-1)":  {false, true, false, false, true, false},
	} {
		s := mustSelector(t, src)
		for i, r := range rows {
			if got := s.Matches(r); got != want[i] {
				t.Errorf("%s on row %d: %v", src, i+1, got)
			}
		}
		if s.Specificity != (Specificity{0, 1, 1}) {
			t.Errorf("%s: specificity %v", src, s.Specificity)
		}
	}
	if _, err := ParseSelectors("tr:nth-child(x)"); err == nil {
		t.Error("tr:nth-child(x) parsed")
	}
}
