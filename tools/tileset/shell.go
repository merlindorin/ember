package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/twod/light"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

// shellSheet lays the workspace out under the menu bar: the tiles, the
// editor over the status line, and the inspector's sections. The boxes
// the tile grid, the editor and the preview pane draw in only reserve
// room: transparent, and the pointer passes through them to the widget.
const shellSheet = `
#workspace { position: absolute; top: 30px; left: 0; right: 0; bottom: 0; display: flex; gap: 12px; padding: 12px; }
#workspace.hidden { display: none; }
.slot { pointer-events: none; }
.tiles { width: 250px; flex: none; }
.centre { flex: 1 1 0; min-width: 260px; display: flex; flex-direction: column; gap: 8px; }
.editor { flex: 1 1 0; min-height: 300px; }
.status { color: var(--muted-color); }
.side { width: 440px; flex: none; }
#inspector-window { inset: auto; top: 42px; right: 12px; bottom: 12px; margin: 0; width: 440px; height: auto;
  background-color: transparent; }
#inspector-window > .kit-body { flex: 1 1 0; min-height: 0; padding: 0; background-color: transparent; }
.sections { flex: 1 1 0; min-height: 0; display: flex; flex-direction: column; overflow-x: hidden; overflow-y: auto; }
.sections > .kit-panel { flex: none; background-color: var(--window-color); border: 0;
  border-bottom: 1px solid var(--window-edge); border-radius: 0; }
.sections > .filler { flex: 1 0 auto; background-color: var(--window-color); }
.sections > .kit-panel.see-through { background-color: transparent; }
.sections > .kit-panel.see-through > .kit-panel-title,
.see-through > .section-body > :not(.slot) { background-color: var(--window-color); }
.see-through > .section-body { gap: 0; padding: 0; }
.see-through > .section-body > :not(.slot) { padding: 8px 12px; }
.sections > .kit-panel > .kit-panel-title { padding: 6px 12px; background-color: transparent;
  color: var(--light-text); font-size: 13px; font-weight: bold; }
.section-body { display: flex; flex-direction: column; gap: 10px; padding: 4px 12px 12px; }
.preview { height: 230px; }
.value { width: 48px; text-align: right; }
.slider-row { display: flex; gap: 8px; align-items: center; }
.slider-row > .kit-slider { flex: 1 1 0; min-width: 0; }
#inspector-window > .kit-footer { justify-content: space-between; align-items: center; padding: 6px 12px;
  border-radius: 0 0 9px 9px; background-color: var(--window-color); font-size: 11px; color: var(--muted-color); }
.edited-dot { width: 8px; height: 8px; border-radius: 4px; background-color: var(--accent-color); }
.edited-dot.clean { visibility: hidden; }
`

// inspectorLabels is the width of every inspector form's label column.
const inspectorLabels = 120

// part is an element of the shell, read and set the way the tool and its
// tests do: its automation ID, its box, its text, its state.
type part struct {
	doc *ui.Document
	n   *ui.Node
}

// Path is the element's automation ID: the ids and data-names from the
// page down to it.
func (p part) Path() string {
	var parts []string
	for n := p.n; n != nil; n = n.Parent() {
		if v := n.ID(); v != "" {
			parts = append(parts, v)
		} else if name, _ := n.Attr("data-name"); name != "" {
			parts = append(parts, name)
		}
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, "/")
}

// Rect is the element's box in window points.
func (p part) Rect() geom.Rect { r, _ := p.doc.BoxOf(p.n); return r }

func (p part) Hidden() bool          { return p.n.HasClass("hidden") }
func (p part) SetHidden(hidden bool) { hide(p.n, hidden) }

func (p part) Text() string { return p.n.Text() }

func (p part) SetText(s string) {
	if p.n.Text() != s {
		p.n.SetText(s)
	}
}

func (p part) Checked() bool      { return p.doc.Checked(p.n) }
func (p part) SetChecked(on bool) { p.doc.SetChecked(p.n, on) }
func (p part) SetDisabled(off bool) {
	switch _, on := p.n.Attr("disabled"); {
	case off && !on:
		p.n.SetAttr("disabled", "")
	case !off && on:
		p.n.RemoveAttr("disabled")
	}
}

// Value is a slider's value.
func (p part) Value() float32 {
	v, _ := strconv.ParseFloat(p.doc.Value(p.n), 32)
	return float32(v)
}

// SetCollapsed folds a section to its title bar, or opens it and
// scrolls it into view.
func (p part) SetCollapsed(folded bool) {
	p.doc.SetOpen(p.n, !folded)
	if !folded {
		p.doc.ScrollIntoView(p.n)
	}
}

// named is a data-name argument, the automation name of an element.
func named(name string) ui.Arg { return ui.Attr("data-name", name) }

// workspaceNode is the shell's workspace: the slots the widgets draw in
// and the status line, with room on the right for the inspector's
// window, in the dialogs Document.
func (t *Tool) workspaceNode() *ui.Node {
	d := t.dialogs
	t.gridSlot = ui.Div(ui.Class("slot", "tiles"), named("tiles"))
	t.editorSlot = ui.Div(ui.Class("slot", "editor"), named("editor"))
	t.paneSlot = ui.Div(ui.Class("slot", "preview"), named("preview"))
	status := kit.Label("", ui.Class("status"), kit.Truncate(), named("status"))
	t.status = part{d, status}
	ws := ui.Div(ui.ID("workspace"), named("panel"), t.gridSlot,
		ui.Div(ui.Class("centre"), t.editorSlot, status), ui.Div(ui.Class("side")))
	t.workspace = part{d, ws}
	return ws
}

// inspectorWindow is the inspector's compact window: the tile's section
// and the preview's, folding, an opened one scrolled into view, over a footer
// naming the file, its tiles and whether it has changes.
func (t *Tool) inspectorWindow() *ui.Node {
	d := t.dialogs
	tile := t.section("tile", "Tile")
	t.tilePanel = part{d, t.folding(tile)}
	t.view = newInspectorDoc(t, d, tile)
	preview := t.section("preview-section", "Preview", t.paneSlot,
		kit.Label("Click it, WASD walks; drag to move the lamp", kit.FormLabel()), t.lampForm())
	preview.AddClass("see-through")
	t.previewPanel = part{d, t.folding(preview)}
	sections := kit.Scroll(ui.Class("sections"), named("sections"), tile, preview, ui.Div(ui.Class("filler")))
	t.inspector = part{d, sections}
	t.fileStatus = kit.Label("", kit.Truncate(), named("file"))
	t.editedDot = ui.Span(ui.Class("edited-dot", "clean"))
	t.inspectorWin = t.closing(kit.Window("Inspector", kit.Compact(), ui.ID("inspector-window"), sections,
		kit.Footer(t.fileStatus, t.editedDot)))
	return t.inspectorWin
}

// syncInspector shows the inspector's window while a tileset is open and
// the View menu's Inspector is checked.
func (t *Tool) syncInspector() {
	want := t.ed.path != "" && t.dialogs.Checked(t.items["inspector"])
	switch open := t.dialogs.Open(t.inspectorWin); {
	case want && !open:
		t.dialogs.Show(t.inspectorWin)
	case !want && open:
		t.dialogs.Close(t.inspectorWin, "")
	}
}

// tileCount is the open tileset's tile count.
func (t *Tool) tileCount() int { return int(t.ed.doc.get("tilecount").num()) }

// showFile is the footer: the file's name, its tiles, and "edited" with
// the dot while it has changes.
func (t *Tool) showFile() {
	text := ""
	if t.ed.path != "" {
		text = fmt.Sprintf("%s · %d tiles", filepath.Base(t.ed.path), t.tileCount())
		if t.ed.Changed() {
			text += " · edited"
		}
	}
	t.fileStatus.SetText(text)
	if t.ed.Changed() {
		t.editedDot.RemoveClass("clean")
	} else {
		t.editedDot.AddClass("clean")
	}
}

// section is a foldable part of the inspector, folded as it was when the
// tool last closed.
func (t *Tool) section(name, title string, children ...*ui.Node) *ui.Node {
	folded, known := t.state.Folded[name]
	open := !folded && (known || name == "tile")
	p := kit.CollapsiblePanel(
		title,
		open,
		named(name),
		ui.Div(append([]ui.Arg{ui.Class("section-body")}, args(children)...)...),
	)
	p.Children()[0].SetAttr("data-name", "title")
	return p
}

func args(nodes []*ui.Node) []ui.Arg {
	out := make([]ui.Arg, len(nodes))
	for i, n := range nodes {
		out[i] = n
	}
	return out
}

// folding notes the section n's folds for Update.
func (t *Tool) folding(n *ui.Node) *ui.Node {
	ui.On(n, ui.EventToggle, func(e ui.Event) { t.folded = append(t.folded, e.Node) })
	return n
}

// toggling notes n's changes for Update.
func (t *Tool) toggling(n *ui.Node) *ui.Node {
	ui.On(n, ui.EventChange, func(e ui.Event) { t.toggled = append(t.toggled, e.Node) })
	return n
}

// lampForm is the preview lamp's and walker's settings.
func (t *Tool) lampForm() *ui.Node {
	d := t.dialogs
	slider := func(name string, v, lo, hi, step float64) *ui.Node {
		s := kit.Slider(
			v,
			kit.Range(lo, hi),
			kit.Step(step),
			ui.ID(name),
			ui.OnInput(func(e ui.Event) { t.edited = append(t.edited, e.Node) }),
		)
		l := kit.Label("", ui.Class("value"), named(name+"-label"))
		t.sliders[name], t.labels[name] = part{d, s}, part{d, l}
		return ui.Div(ui.Class("slider-row"), s, l)
	}
	shadows := kit.Switch("", ui.Checked(true), ui.ID("shadows"))
	t.checks["shadows"] = part{d, t.toggling(kit.SwitchTrack(shadows))}
	var fades []*ui.Node
	for _, f := range []light.Fade{light.FadeNone, light.FadeLinear, light.FadeSmooth, light.FadeSoft} {
		fades = append(fades, t.toggling(kit.ChoiceItem(kit.Label(f.String()), named(f.String()))))
	}
	t.fade = kit.Choice(0, fades, kit.Segmented(), ui.ID("fade"))
	t.fades = fades
	form := kit.Form(
		kit.Field("light radius", slider("radius", 200, 40, 500, 10)),
		kit.Field("elevation", slider("elevation", 0, 0, 256, 4)),
		kit.Field("lamp size", slider("size", 0, 0, 32, 1)),
		kit.Field("shadows", shadows),
		kit.Field("shadows fade", t.fade),
		kit.Separator(),
		kit.Field("walker width", slider("body-w", 16, 2, 64, 1)),
		kit.Field("walker height", slider("body-h", 10, 2, 64, 1)),
	)
	form.SetAttr("data-name", "lamp")
	kit.SetLabelWidth(form, inspectorLabels)
	return form
}

// placeWidgets puts the tile grid, the editor and the preview pane in the
// boxes the shell laid out for them, hidden where their box is. The
// shell lays out first when it hasn't yet, so the widgets draw from the
// first frame. The preview scrolls with the inspector: its pane is the
// part of its box the sections show, and paneFull the whole box, which
// the preview's camera centres on.
func (t *Tool) placeWidgets() {
	if _, ok := t.dialogs.BoxOf(t.dialogs.Body()); !ok {
		t.dialogs.Layout(t.win.Size())
	}
	t.paneFull = geom.Rect{}
	for _, w := range []struct {
		s    *slot
		node *ui.Node
	}{{&t.grid.slot, t.gridSlot}, {&t.editor.slot, t.editorSlot}, {&t.pane.slot, t.paneSlot}} {
		r, ok := t.dialogs.BoxOf(w.node)
		if ok && w.s == &t.pane.slot {
			t.paneFull = r
			view, shown := t.dialogs.BoxOf(t.inspector.n)
			r, ok = clipRect(r, view), ok && shown
		}
		w.s.SetHidden(!ok || r.W <= 0 || r.H <= 0)
		if ok {
			w.s.place(r)
		}
	}
}

// clipRect is the part of r inside within, empty when they don't meet.
func clipRect(r, within geom.Rect) geom.Rect {
	x0, y0 := max(r.X, within.X), max(r.Y, within.Y)
	x1, y1 := min(r.X+r.W, within.X+within.W), min(r.Y+r.H, within.Y+within.H)
	if x1 <= x0 || y1 <= y0 {
		return geom.Rect{}
	}
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// widgets are the tile grid, the editor and the preview pane, drawn under
// the shell's Document.
func (t *Tool) widgets() []twod.Entity {
	t.grid = twod.Build(&tileGrid{slot: slot{Item: twod.Base()}, tool: t}, "tiles", twod.Overlay())
	t.editor = twod.Build(&tileEditor{slot: slot{Item: twod.Base()}, tool: t}, "editor", twod.Overlay())
	t.pane = twod.Build(&previewPane{slot: slot{Item: twod.Base()}, tool: t}, "preview", twod.Overlay())
	return []twod.Entity{t.grid, t.editor, t.pane}
}
