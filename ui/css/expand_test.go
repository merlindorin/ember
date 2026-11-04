package css //nolint:testpackage // tests the unexported shorthand expansion

import (
	"strings"
	"testing"
)

func expanded(t *testing.T, name, src string) map[string]string {
	t.Helper()
	out, err := expand(name, items(componentsOf(src)))
	if err != nil {
		t.Fatalf("%s: %s: %v", name, src, err)
	}
	m := map[string]string{}
	for k, v := range out {
		var s []string
		for _, c := range v {
			s = append(s, c.String())
		}
		m[k] = strings.Join(s, " ")
	}
	return m
}

func TestExpand(t *testing.T) {
	cases := []struct {
		name, src string
		want      map[string]string
	}{
		{"margin", "1px 2px 3px",
			map[string]string{"margin-top": "1px", "margin-right": "2px", "margin-bottom": "3px", "margin-left": "2px"}},
		{"padding", "4px", map[string]string{"padding-top": "4px", "padding-left": "4px"}},
		{"border-radius", "1px 2px",
			map[string]string{"border-top-left-radius": "1px", "border-top-right-radius": "2px",
				"border-bottom-right-radius": "1px", "border-bottom-left-radius": "2px"}},
		{"gap", "6px 12px", map[string]string{"row-gap": "6px", "column-gap": "12px"}},
		{"gap", "6px", map[string]string{"row-gap": "6px", "column-gap": "6px"}},
		{"overflow", "hidden auto", map[string]string{"overflow-x": "hidden", "overflow-y": "auto"}},
		{"place-items", "center", map[string]string{"align-items": "center", "justify-items": "center"}},
		{"flex", "1", map[string]string{"flex-grow": "1", "flex-shrink": "1", "flex-basis": "0%"}},
		{"flex", "none", map[string]string{"flex-grow": "0", "flex-shrink": "0", "flex-basis": "auto"}},
		{"flex", "auto", map[string]string{"flex-grow": "1", "flex-shrink": "1", "flex-basis": "auto"}},
		{"flex", "2 0 10px", map[string]string{"flex-grow": "2", "flex-shrink": "0", "flex-basis": "10px"}},
		{"flex", "30%", map[string]string{"flex-grow": "1", "flex-shrink": "1", "flex-basis": "30%"}},
		{"flex-flow", "wrap column", map[string]string{"flex-direction": "column", "flex-wrap": "wrap"}},
		{"grid-column", "1 / span 2", map[string]string{"grid-column-start": "1", "grid-column-end": "span 2"}},
		{"grid-row", "3", map[string]string{"grid-row-start": "3", "grid-row-end": "auto"}},
		{"border", "1px solid #fff",
			map[string]string{"border-top-width": "1px", "border-left-style": "solid", "border-bottom-color": "#fff",
				"border-image-source": "none"}},
		{"border", "0",
			map[string]string{"border-top-width": "0", "border-right-style": "none", "border-left-color": "currentcolor"}},
		{"border-left", "dashed",
			map[string]string{"border-left-style": "dashed", "border-left-width": "medium",
				"border-left-color": "currentcolor"}},
		{"outline", "2px solid rgb(1 2 3)",
			map[string]string{"outline-width": "2px", "outline-style": "solid", "outline-color": "rgb(1 2 3)"}},
		{"background", "url(a.png) #000", map[string]string{"background-image": "url(a.png)", "background-color": "#000"}},
		{"border-image", "url(p.png) 8 fill / 8px round",
			map[string]string{"border-image-source": "url(p.png)", "border-image-slice": "8 fill", "border-image-width": "8px",
				"border-image-repeat": "round", "border-image-outset": "0"}},
		{"transition", "opacity 0.2s, color 100ms ease-in 1s", map[string]string{
			"transition-property": "opacity , color", "transition-duration": "0.2s , 100ms",
			"transition-timing-function": "ease , ease-in", "transition-delay": "0s , 1s"}},
		{"margin", "inherit", map[string]string{"margin-top": "inherit", "margin-left": "inherit"}},
	}
	for _, c := range cases {
		got := expanded(t, c.name, c.src)
		for k, want := range c.want {
			if got[k] != want {
				t.Errorf("%s: %s: %s = %q, want %q", c.name, c.src, k, got[k], want)
			}
		}
		if len(got) != len(shorthands[c.name]) {
			t.Errorf("%s: %s: %d longhands, want %d", c.name, c.src, len(got), len(shorthands[c.name]))
		}
	}
}

func TestValidate(t *testing.T) {
	good := map[string][]string{
		"appearance": {"none", "auto", "textfield"},
		"resize":     {"none", "both", "horizontal", "vertical"},
		"display":    {"grid", "inline-flex", "contents", "inherit"},
		"width":      {"50%", "calc(100% - 2rem)", "max-content", "auto", "0"},
		"grid-template-columns": {"max-content 1fr", "repeat(3, minmax(0, 1fr))", "repeat(auto-fill, 120px) 20%",
			"repeat(auto-fit, minmax(100px, 1fr))", "fit-content(40%) auto"},
		"box-shadow":                 {"none", "0 2px 4px #0008", "inset 1px 1px #000, 0 0 8px 2px rgb(0 0 0 / 50%)"},
		"transform":                  {"translate(10px, 50%) rotate(45deg) scale(1.5)", "none", "matrix(1, 0, 0, 1, 0, 0)"},
		"font-family":                {`"Go Mono", monospace`, "Go"},
		"aspect-ratio":               {"16 / 9", "1", "auto"},
		"line-height":                {"1.5", "20px", "normal", "120%"},
		"opacity":                    {"0.5", "50%"},
		"z-index":                    {"-1", "auto"},
		"transition-timing-function": {"cubic-bezier(0.1, 0.7, 1, 0.1)", "steps(4, jump-end)"},
	}
	for name, vals := range good {
		for _, v := range vals {
			if err := validate(name, componentsOf(v)); err != nil {
				t.Errorf("%s: %s: %v", name, v, err)
			}
		}
	}
	bad := map[string][]string{
		"resize":  {"auto", "12px"},
		"display": {"table", "flex grid", "-webkit-box"},
		"width":   {"-1px", "1fr", "12", "red"},
		"padding": {"-2px", "1px 2px 3px 4px 5px"},
		"margin":  {"1px / 2px"},
		"grid-template-columns": {"[a] 1fr", "repeat(auto-fill, 20px) 1fr", "max-content repeat(auto-fill, 20px)",
			"repeat(auto-fill, 1fr)", "repeat(auto-fit, 9px) repeat(auto-fill, 9px)", "minmax(1fr, 2fr)", "repeat(0, 1fr)",
			"repeat(2, repeat(2, 1fr))"},
		"grid-column":   {"0", "1 / 2 / 3", "span 0"},
		"box-shadow":    {"2px", "2px 2px #000 #fff", "1px #000 1px"},
		"transform":     {"rotate(10px)", "translate3d(1px, 1px, 1px)", "scale(1 2)"},
		"transition":    {"width 1s", "opacity 1s 2s 3s"},
		"flex":          {"1 2 3", "-1", "1 auto 2"},
		"border-style":  {"groove"},
		"color":         {"reddish", "#12"},
		"font-weight":   {"1001", "heavy"},
		"font-family":   {"12px"},
		"border-radius": {"1px / 2px"},
	}
	for name, vals := range bad {
		for _, v := range vals {
			if err := validate(name, componentsOf(v)); err == nil {
				t.Errorf("%s: %s: accepted", name, v)
			}
		}
	}
}

func TestParseDropsBadValues(t *testing.T) {
	sheet, warns := Parse("t.css", "p { display: table; width: var(--w); color: #fff }")
	if got := warnings(warns); got != `t.css:1:14: display: expected block, inline, inline-block, flex, inline-flex, `+
		`grid, inline-grid, none, contents, list-item, found "table"` {
		t.Errorf("warnings: %s", got)
	}
	if n := len(sheet.Rules[0].Decls); n != 2 {
		t.Errorf("kept %d declarations", n)
	}
}
