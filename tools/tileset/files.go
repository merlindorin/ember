package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
)

// state is what the tool remembers between runs, in its folder: the
// tilesets opened last, the latest first, the folder the dialog was last
// in, and whether dialogs show inside the main window rather than in
// their own.
type state struct {
	Recent []string `json:"recent"`
	Folder string   `json:"folder"`
	Inline bool     `json:"dialogs_inline,omitempty"`
	// Folded are the inspector's sections folded, by name.
	Folded map[string]bool `json:"folded,omitempty"`
}

// maxRecent is how many tilesets the dialog lists as recent.
const maxRecent = 8

// home is the tool's folder: EMBER_TILESET_DIR, else ~/.ember-tileset;
// none under go test without EMBER_TILESET_DIR, so tests never write the
// player's own.
func home() string {
	if d := os.Getenv("EMBER_TILESET_DIR"); d != "" {
		return d
	}
	if testing.Testing() {
		return ""
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(h, ".ember-tileset")
}

func loadState() state {
	var s state
	if home() == "" {
		return s
	}
	if b, err := os.ReadFile(filepath.Join(home(), "state.json")); err == nil {
		_ = json.Unmarshal(b, &s)
	}
	return s
}

func (s state) save() error {
	dir := home()
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	//nolint:gosec // recent files only, no secret
	return os.WriteFile(filepath.Join(dir, "state.json"), append(b, '\n'), 0o644)
}

// opened puts path first among the recent tilesets and remembers its
// folder.
func (s *state) opened(path string) {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	s.Recent = slices.DeleteFunc(s.Recent, func(p string) bool { return p == path })
	s.Recent = append([]string{path}, s.Recent...)
	if len(s.Recent) > maxRecent {
		s.Recent = s.Recent[:maxRecent]
	}
	s.Folder = filepath.Dir(path)
}

// startDir is -dir: the folder the open dialog starts in before the tool
// has opened a tileset.
//
//nolint:gochecknoglobals // set once from -dir, as main parses its flags
var startDir string

// startFolder is where the dialog opens: the folder it was last in, else
// -dir, else here.
func (s state) startFolder() string {
	for _, d := range []string{s.Folder, startDir, "."} {
		if info, err := os.Stat(d); d != "" && err == nil && info.IsDir() {
			if abs, absErr := filepath.Abs(d); absErr == nil {
				return abs
			}
			return d
		}
	}
	return "."
}

// alert is a kit.Alert in the tool's dialogs Document, named name for
// automation: its buttons are name/<value>.
type alert struct {
	doc  *ui.Document
	node *ui.Node
}

func newAlert(doc *ui.Document, name string, buttons ...*ui.Node) *alert {
	a := &alert{doc: doc, node: kit.Alert("", "", buttons...)}
	a.node.SetAttr("data-name", name)
	doc.Body().Append(a.node)
	return a
}

// Hidden reports whether the alert is closed.
func (a *alert) Hidden() bool { return !a.doc.Open(a.node) }

func (a *alert) show(title, message string) {
	kit.SetAlert(a.node, title, message)
	a.doc.ShowModal(a.node)
}

// answer is the button that closed the alert in e, "" for Escape or
// another alert.
func (a *alert) answer(e ui.Closed) (string, bool) {
	return e.Value, e.Node == a.node
}

// alertButton is an alert's button: the value it answers and its
// automation path.
type alertButton struct{ value, path string }

func (b alertButton) Path() string { return b.path }

func (b alertButton) node(text string, args ...ui.Arg) *ui.Node {
	return kit.Button(text, append([]ui.Arg{ui.Value(b.value), ui.Attr("data-name", b.value)}, args...)...)
}

func button(alert, value string) alertButton { return alertButton{value, alert + "/" + value} }

// prompt asks what to do with unsaved changes before going on: save them,
// throw them away, or stay.
type prompt struct {
	panel                 *alert
	save, discard, cancel alertButton
	then                  func()
}

func newPrompt(doc *ui.Document) *prompt {
	p := &prompt{
		save:    button("unsaved", "save"),
		discard: button("unsaved", "discard"),
		cancel:  button("unsaved", "cancel"),
	}
	p.panel = newAlert(doc, "unsaved",
		p.discard.node("Don't save", kit.Destructive()),
		p.cancel.node("Cancel"),
		p.save.node("Save", kit.Primary()))
	return p
}

// ask shows the prompt about name, then runs then unless cancelled.
func (p *prompt) ask(name string, then func()) {
	p.then = then
	p.panel.show("Save changes to "+name+"?", "Your changes are lost if you don't save them.")
}

// conflict asks what to do when the file changed on disk since it was
// read: load it again, losing the changes here, write over it, or stay.
type conflict struct {
	panel                     *alert
	reload, overwrite, cancel alertButton
	// then is what was waiting on the save, run once the file is settled.
	then func()
}

func newConflict(doc *ui.Document) *conflict {
	c := &conflict{
		reload:    button("changed-on-disk", "reload"),
		overwrite: button("changed-on-disk", "overwrite"),
		cancel:    button("changed-on-disk", "cancel"),
	}
	c.panel = newAlert(doc, "changed-on-disk",
		c.overwrite.node("Overwrite", kit.Destructive()),
		c.reload.node("Reload"),
		c.cancel.node("Cancel", kit.Primary()))
	return c
}

func (c *conflict) ask(name string) {
	c.panel.show(
		name+" changed on disk",
		"It changed since it was opened: reload it and lose the changes here, or overwrite it.",
	)
}

// answered handles the conflict closing in e, reporting whether it was
// the conflict's.
func (c *conflict) answered(e ui.Closed, t *Tool) bool {
	v, ok := c.panel.answer(e)
	if !ok {
		return false
	}
	then := c.then
	c.then = nil
	switch v {
	case c.reload.value:
		t.openFile(t.ed.path)
	case c.overwrite.value:
		if err := t.ed.Overwrite(); err != nil {
			t.status.SetText(err.Error())
			return true
		}
		t.status.SetText("Saved " + filepath.Base(t.ed.path))
	default:
		return true
	}
	if then != nil {
		then()
	}
	return true
}

// readProperties reads a game's tile properties from a JSON list:
// [{"name": "loot", "kind": "enum", "values": ["none", "gold"],
// "default": "none", "doc": "…"}], kind one of bool, int, float, string,
// enum, color, with "min" and "max" for a number's range.
func readProperties(path string) ([]twod.Property, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var list []struct {
		Name    string   `json:"name"`
		Kind    string   `json:"kind"`
		Default any      `json:"default"`
		Values  []string `json:"values"`
		Min     float64  `json:"min"`
		Max     float64  `json:"max"`
		Doc     string   `json:"doc"`
	}
	if err = json.Unmarshal(b, &list); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	kinds := map[string]twod.PropertyKind{
		"bool":   twod.KindBool,
		"int":    twod.KindInt,
		"float":  twod.KindFloat,
		"string": twod.KindString,
		"enum":   twod.KindEnum,
		"color":  twod.KindColor,
	}
	var out []twod.Property
	for _, p := range list {
		k, ok := kinds[p.Kind]
		if !ok || p.Name == "" {
			return nil, fmt.Errorf(
				"%s: property %q: kind %q is not one of bool, int, float, string, enum, color",
				path,
				p.Name,
				p.Kind,
			)
		}
		d := twod.Property{
			Name:    p.Name,
			Kind:    k,
			Default: p.Default,
			Values:  p.Values,
			Min:     p.Min,
			Max:     p.Max,
			Doc:     p.Doc,
		}
		switch k {
		case twod.KindInt:
			f, _ := p.Default.(float64)
			d.Default = int(f)
		case twod.KindFloat:
			f, _ := p.Default.(float64)
			d.Default = f
		case twod.KindBool:
			b, _ := p.Default.(bool)
			d.Default = b
		case twod.KindString, twod.KindEnum, twod.KindColor, twod.KindFile:
			s, _ := p.Default.(string)
			if k == twod.KindEnum && s == "" && len(p.Values) > 0 {
				s = p.Values[0]
			}
			d.Default = s
		case twod.KindClass, twod.KindObject:
		}
		out = append(out, d)
	}
	return out, nil
}
