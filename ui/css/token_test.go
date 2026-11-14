package css_test

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/css"
)

func kinds(toks []css.Token) string {
	var parts []string
	for _, t := range toks {
		if t.Kind == css.EOF {
			break
		}
		parts = append(parts, t.Kind.String()+"<"+t.String()+">")
	}
	return strings.Join(parts, " ")
}

func TestTokenize(t *testing.T) {
	cases := []struct{ src, want string }{
		{".props > * { min-width: 0; }",
			"delim<.> ident<props> whitespace< > delim<>> whitespace< > delim<*> whitespace< > {<{> whitespace< > " +
				"ident<min-width> :<:> whitespace< > number<0> ;<;> whitespace< > }<}>"},
		{"#a1 #1a", "hash<#a1> whitespace< > hash<#1a>"},
		{"12px 50% -.5em 1e3 +3", "dimension<12px> whitespace< > percentage<50%> whitespace< > dimension<-.5em> " +
			"whitespace< > number<1e3> whitespace< > number<+3>"},
		{"calc(100% - 2rem)", "function<calc(> percentage<100%> whitespace< > delim<-> whitespace< > dimension<2rem> )<)>"},
		{"var(--gap, 4px)", "function<var(> ident<--gap> ,<,> whitespace< > dimension<4px> )<)>"},
		{"a/* note */b", "ident<a> ident<b>"},
		{`"a\"b" 'c'`, `string<"a\"b"> whitespace< > string<"c">`},
		{"url(img/panel.png) url( 'x.png' )",
			"url<url(img/panel.png)> whitespace< > function<url(> whitespace< > string<\"x.png\"> whitespace< > )<)>"},
		{"!important", "delim<!> ident<important>"},
		{"@media", "at-keyword<@media>"},
		{`\31 a`, "ident<1a>"},
		{"<!-- -->", "<!--<<!--> whitespace< > --><-->>"},
		{"--x", "ident<--x>"},
		{"1.5fr", "dimension<1.5fr>"},
	}
	for _, c := range cases {
		if got := kinds(css.Tokenize(c.src)); got != c.want {
			t.Errorf("Tokenize(%q)\n got %s\nwant %s", c.src, got, c.want)
		}
	}
}

func TestTokenizeNumbers(t *testing.T) {
	toks := css.Tokenize("3 3.0 -2.5e1px")
	if !toks[0].Int || toks[0].Num != 3 {
		t.Errorf("3: %+v", toks[0])
	}
	if toks[2].Int || toks[2].Num != 3 {
		t.Errorf("3.0: %+v", toks[2])
	}
	if toks[4].Kind != css.Dimension || toks[4].Num != -25 || toks[4].Unit != "px" {
		t.Errorf("-2.5e1px: %+v", toks[4])
	}
}

func TestTokenizeHashID(t *testing.T) {
	toks := css.Tokenize("#main #123")
	if !toks[0].HasID || toks[2].HasID {
		t.Errorf("HasID: %v %v", toks[0].HasID, toks[2].HasID)
	}
}

func TestTokenizePositions(t *testing.T) {
	toks := css.Tokenize("a {\r\n  color: red;\n}")
	var color css.Token
	for _, tk := range toks {
		if tk.Kind == css.Ident && tk.Text == "color" {
			color = tk
		}
	}
	if color.Pos != (css.Pos{Line: 2, Col: 3}) {
		t.Errorf("color at %v, want 2:3", color.Pos)
	}
}

func TestTokenizeBadString(t *testing.T) {
	toks := css.Tokenize("'abc\nd")
	if toks[0].Kind != css.BadString {
		t.Errorf("got %v", toks[0].Kind)
	}
}
