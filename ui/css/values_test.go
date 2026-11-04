package css //nolint:testpackage // tests the unexported value parsers (parseColor, parseDim, angle, duration)

import (
	"math"
	"testing"
)

func valueOf(t *testing.T, src string) Component {
	t.Helper()
	decls, warns := ParseInline("--v: " + src)
	if len(warns) > 0 || len(decls) != 1 || len(decls[0].Value) != 1 {
		t.Fatalf("%q: %v %v", src, decls, warns)
	}
	return decls[0].Value[0]
}

func TestParseDim(t *testing.T) {
	cases := []struct {
		src  string
		want Dim
	}{
		{"12px", Dim{Px: 12, kind: dimLength}},
		{"50%", Dim{Pct: 50, kind: dimPercent}},
		{"1.5em", Dim{Em: 1.5, kind: dimLength}},
		{"2rem", Dim{Rem: 2, kind: dimLength}},
		{"1fr", Dim{Fr: 1, kind: dimFlex}},
		{"3", Dim{Num: 3, kind: dimNumber}},
		{"calc(100% - 2rem)", Dim{Pct: 100, Rem: -2, kind: dimPercent | dimLength}},
		{"calc(4px * 3 + 1em / 2)", Dim{Px: 12, Em: 0.5, kind: dimLength}},
		{"calc(2 * (10px + 5%))", Dim{Px: 20, Pct: 10, kind: dimLength | dimPercent}},
		{"calc(calc(1px + 1px) * 2)", Dim{Px: 4, kind: dimLength}},
		{"calc(1 + 2 * 3)", Dim{Num: 7, kind: dimNumber}},
	}
	for _, c := range cases {
		got, err := parseDim(valueOf(t, c.src))
		if err != nil || got != c.want {
			t.Errorf("%q: got %+v, %v; want %+v", c.src, got, err, c.want)
		}
	}
}

func TestParseDimErrors(t *testing.T) {
	for _, src := range []string{"12pt", "calc(1px+2px)", "calc(1px + 2)", "calc(2px * 3px)", "calc(1px / 0)",
		"calc(1px / 2px)", "calc(1fr + 1px)", "calc()", "calc(1px 2px)", "red", "min(1px, 2px)"} {
		if _, err := parseDim(valueOf(t, src)); err == nil {
			t.Errorf("%q: no error", src)
		}
	}
}

func TestDimResolve(t *testing.T) {
	d, _ := parseDim(valueOf(t, "calc(100% - 2rem + 1em)"))
	if got := d.Absolute(10, 16).Resolve(200); got != 200-32+10 {
		t.Errorf("got %v", got)
	}
}

func near(a, b float64) bool { return math.Abs(a-b) < 0.002 }

func TestParseColor(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{"#f00", "rgb(255, 0, 0)"},
		{"#ff000080", "rgba(255, 0, 0, 0.502)"},
		{"#0f08", "rgba(0, 255, 0, 0.533)"},
		{"#f7efdd", "rgb(247, 239, 221)"},
		{"transparent", "rgba(0, 0, 0, 0)"},
		{"red", "rgb(255, 0, 0)"},
		{"RebeccaPurple", "rgb(102, 51, 153)"},
		{"lightgoldenrodyellow", "rgb(250, 250, 210)"},
		{"rgb(255, 128, 0)", "rgb(255, 128, 0)"},
		{"rgba(255, 128, 0, 0.5)", "rgba(255, 128, 0, 0.5)"},
		{"rgb(100%, 50%, 0%)", "rgb(255, 128, 0)"},
		{"rgb(255 128 0 / 25%)", "rgba(255, 128, 0, 0.25)"},
		{"rgb(300 -5 none)", "rgb(255, 0, 0)"},
		{"oklch(62.8% 0.2577 29.23)", "rgb(255, 0, 0)"},
		{"oklch(1 0 0)", "rgb(255, 255, 255)"},
		{"oklch(0 0 0 / 0.5)", "rgba(0, 0, 0, 0.5)"},
		{"oklch(0.5198 0.1769 142.5deg)", "rgb(0, 128, 0)"},
	}
	for _, c := range cases {
		got, err := parseColor(valueOf(t, c.src))
		if err != nil || got.String() != c.want {
			t.Errorf("%q: got %v, %v; want %s", c.src, got, err, c.want)
		}
	}
}

func TestParseColorErrors(t *testing.T) {
	for _, src := range []string{"reddish", "#12", "#ggg", "rgb(1, 2)", "rgb(1 2 3 4)", "rgb(1, 2%, 3)", "rgb(1 2, 3)",
		"oklch(1, 0, 0)", "hsl(0 0% 0%)", "12px"} {
		if _, err := parseColor(valueOf(t, src)); err == nil {
			t.Errorf("%q: no error", src)
		}
	}
}

func TestAngleAndDuration(t *testing.T) {
	if a, _ := angle(valueOf(t, "0.5turn")); a != 180 {
		t.Errorf("turn: %v", a)
	}
	if a, _ := angle(valueOf(t, "3.14159265rad")); !near(a, 180) {
		t.Errorf("rad: %v", a)
	}
	if d, _ := duration(valueOf(t, "150ms")); d != 0.15 {
		t.Errorf("ms: %v", d)
	}
	if _, err := duration(valueOf(t, "3")); err == nil {
		t.Error("unitless duration accepted")
	}
}
