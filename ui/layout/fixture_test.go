package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

//nolint:gochecknoglobals // a read-only test table
var taffySuites = []string{"flex", "grid", "block", "leaf", "blockflex", "blockgrid", "gridflex"}

// taffyAttributes are the style attributes the fixtures use: each one needs a mapping
// onto the computed style once the lead publishes it.
var taffyAttributes = []string{ //nolint:gochecknoglobals // a read-only test table
	"align-content", "align-items", "align-self", "aspect-ratio",
	"border-bottom", "border-left", "border-right", "border-top", "bottom", "box-sizing",
	"column-gap", "display", "flex-basis", "flex-direction", "flex-grow",
	"flex-shrink", "flex-wrap", "grid-auto-columns", "grid-auto-flow",
	"grid-auto-rows", "grid-column-end", "grid-column-start", "grid-row-end",
	"grid-row-start", "grid-template-columns", "grid-template-rows", "height",
	"justify-content", "justify-items", "justify-self", "left", "margin-bottom",
	"margin-left", "margin-right", "margin-top", "max-height", "max-width", "min-height",
	"min-width", "overflow-x", "overflow-y", "padding-bottom", "padding-left",
	"padding-right", "padding-top", "position", "right", "row-gap", "scrollbar-width",
	"top", "width",
}

func TestTaffyFixturesLoad(t *testing.T) {
	known := map[string]bool{}
	for _, a := range taffyAttributes {
		known[a] = true
	}
	used := map[string]bool{}
	for _, suite := range taffySuites {
		fixtures, err := loadTaffySuite(suite)
		if err != nil {
			t.Fatalf("%s: %v", suite, err)
		}
		if len(fixtures) < 20 {
			t.Fatalf("%s: %d fixtures, want the whole suite", suite, len(fixtures))
		}
		names := map[string]bool{}
		for _, f := range fixtures {
			if names[f.Name] {
				t.Errorf("%s: %s twice", suite, f.Name)
			}
			names[f.Name] = true
			if f.Root == nil {
				t.Fatalf("%s/%s: no root", suite, f.Name)
			}
			f.Root.walk(func(n *fixtureNode) {
				for a := range n.Style {
					used[a] = true
					if !known[a] {
						t.Errorf("%s/%s: unknown attribute %q", suite, f.Name, a)
					}
				}
				if n.Want.Width < 0 || n.Want.Height < 0 {
					t.Errorf("%s/%s: negative expected size %+v", suite, f.Name, n.Want)
				}
			})
		}
	}
	for _, a := range taffyAttributes {
		if !used[a] {
			t.Errorf("attribute %q is never used; drop it from the list", a)
		}
	}
}

func TestTaffyFixtureShape(t *testing.T) {
	fixtures, err := loadTaffySuite("flex")
	if err != nil {
		t.Fatal(err)
	}
	f := findFixture(t, fixtures, "absolute_aspect_ratio_fill_height__border_box")
	if !f.Rounding {
		t.Error("rounding should default to on")
	}
	if f.Viewport != [2]string{"max-content", "max-content"} {
		t.Errorf("viewport %v", f.Viewport)
	}
	want := map[string]string{"display": "flex", "position": "absolute", "width": "400px", "height": "300px"}
	if !maps.Equal(f.Root.Style, want) {
		t.Errorf("root style %v, want %v", f.Root.Style, want)
	}
	if f.Root.Want != (fixtureBox{X: 0, Y: 0, Width: 400, Height: 300}) {
		t.Errorf("root box %+v", f.Root.Want)
	}
	if len(f.Root.Children) != 1 {
		t.Fatalf("%d children", len(f.Root.Children))
	}
	child := f.Root.Children[0]
	if child.Want != (fixtureBox{X: 20, Y: 15, Width: 200, Height: 67}) {
		t.Errorf("child box %+v", child.Want)
	}
	if child.Style["aspect-ratio"] != "3" || child.Style["top"] != "5%" {
		t.Errorf("child style %v", child.Style)
	}
}

func TestTaffyFixtureText(t *testing.T) {
	fixtures, err := loadTaffySuite("block")
	if err != nil {
		t.Fatal(err)
	}
	f := findFixture(t, fixtures, "block_absolute_aspect_ratio_fill_max_width__border_box")
	var texts []*fixtureNode
	f.Root.walk(func(n *fixtureNode) {
		if n.Text != "" {
			texts = append(texts, n)
		}
	})
	if len(texts) != 1 {
		t.Fatalf("%d text nodes", len(texts))
	}
	if !texts[0].IsText || len(texts[0].Children) != 0 {
		t.Errorf("text node %+v", texts[0])
	}
	if got := strings.Count(texts[0].Text, ahemBreak); got != 8 {
		t.Errorf("%d break opportunities in %q", got, texts[0].Text)
	}
}

func TestTaffyFixtureGridTracks(t *testing.T) {
	fixtures, err := loadTaffySuite("grid")
	if err != nil {
		t.Fatal(err)
	}
	withTracks := 0
	for _, f := range fixtures {
		f.Root.walk(func(n *fixtureNode) {
			if n.Want.Rows != nil {
				withTracks++
				if n.Want.Columns == nil {
					t.Errorf("%s: rows without columns", f.Name)
				}
			}
		})
	}
	if withTracks == 0 {
		t.Error("no grid container carries its resolved tracks")
	}
}

func findFixture(t *testing.T, fixtures []taffyFixture, name string) taffyFixture {
	t.Helper()
	i := slices.IndexFunc(fixtures, func(f taffyFixture) bool { return f.Name == name })
	if i < 0 {
		t.Fatalf("no fixture %s", name)
	}
	return fixtures[i]
}
