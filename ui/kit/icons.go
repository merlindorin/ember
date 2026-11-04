package kit

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"testing/fstest"
	"unicode"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/dom"
)

// Icons is an icon font and the names of its
// glyphs. An icon is a span of class "kit-icon" drawn in the font, at its
// element's font size and colour. The engine ships none; a game loads one
// such as Lucide with LoadIcons, and gives it to its Document with Sheet.
type Icons struct {
	family string
	file   string
	ttf    []byte
	names  map[string]rune
	warns  []error
}

// LoadIcons reads the icon font named font in fsys and its names from the
// JSON file named names: an object of each icon's name and its code point,
// as hex text ("e0a1", "U+E0A1") or a number, as the old ember/ui reads them.
func LoadIcons(fsys fs.FS, font, names string) (*Icons, error) {
	ttf, err := fs.ReadFile(fsys, font)
	if err != nil {
		return nil, fmt.Errorf("kit: %w", err)
	}
	data, err := fs.ReadFile(fsys, names)
	if err != nil {
		return nil, fmt.Errorf("kit: %w", err)
	}
	var raw map[string]any
	if err = json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("kit: %s: %w", names, err)
	}
	out := make(map[string]rune, len(raw))
	for name, v := range raw {
		r, ok := codePoint(v)
		if !ok {
			return nil, fmt.Errorf("kit: %s: %q has no code point: %v", names, name, v)
		}
		out[name] = r
	}
	base := strings.TrimSuffix(path.Base(font), path.Ext(font))
	return &Icons{family: "ember-icons-" + base, file: path.Base(font), ttf: ttf, names: out}, nil
}

func codePoint(v any) (rune, bool) {
	switch v := v.(type) {
	case float64:
		return rune(v), v > 0
	case string:
		s := strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(v), "u+"), "0x")
		n, err := strconv.ParseUint(s, 16, 32)
		if err != nil || n == 0 || n > unicode.MaxRune {
			return 0, false
		}
		return rune(n), true
	}
	return 0, false
}

// Sheet gives a Document the icon font, as an @font-face of its own
// family, and the class of the same name its icons carry.
func (i *Icons) Sheet() ui.DocumentOption {
	css := fmt.Sprintf("@font-face { font-family: %q; src: url(%q); }\n.kit-icon.%s { font-family: %q; }\n", i.family,
		i.file, i.family, i.family)
	return ui.Stylesheet(fstest.MapFS{
		"icons.css": {Data: []byte(css)},
		i.file:      {Data: i.ttf},
	}, "icons.css")
}

// Family is the font family the icons draw in.
func (i *Icons) Family() string { return i.family }

// Warnings are the names Icon was asked for that the set lacks.
func (i *Icons) Warnings() []error { return i.warns }

// Has reports whether name is one of the icons.
func (i *Icons) Has(name string) bool { _, ok := i.names[name]; return ok }

// glyph is the text name draws as, empty for a name the set lacks.
func (i *Icons) glyph(name string) string {
	if r, ok := i.names[name]; ok {
		return string(r)
	}
	return ""
}

// NewIcon is a span showing only the icon named name of set, the old ember/ui's
// NewIcon; empty for a name the set lacks.
func NewIcon(set *Icons, name string, args ...ui.Arg) *ui.Node {
	n := ui.Span(ui.Class("kit-icon", set.family), ui.Attr("data-icon", name), ui.Text(set.glyph(name)))
	for _, a := range args {
		a.Apply(n)
	}
	return n
}

// Icon puts the icon named name of set before an element's text, as
// The old ember/ui's Icon option does for labels, buttons and panels; a name the
// set lacks puts nothing there and is one of the set's Warnings. SetText
// changes the text and keeps the icon.
func Icon(set *Icons, name string) ui.Arg { return iconArg{set, name} }

type iconArg struct {
	set  *Icons
	name string
}

func (a iconArg) Apply(n *ui.Node) {
	if !a.set.Has(a.name) {
		a.set.warns = append(a.set.warns, fmt.Errorf("kit: no icon %q in %s", a.name, a.set.file))
		return
	}
	kids := append([]*dom.Node(nil), n.Children()...)
	for _, k := range kids {
		n.Remove(k)
	}
	n.Append(NewIcon(a.set, a.name, ui.Class("kit-before")))
	n.Append(kids...)
}

// SetText makes n read text after its icons, which it keeps, as
// Node.SetText would without them.
func SetText(n *ui.Node, text string) {
	for _, c := range slices.Clone(n.Children()) {
		if !c.HasClass("kit-icon") {
			n.Remove(c)
		}
	}
	ui.Text(text).Apply(n)
}
