package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing/fstest"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/twod/light"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/merlindorin/ember/world"
)

// Tool is the tileset tool: the editor's state and its window, with the
// tiles, the editor, the controls and the preview pane, where the picked
// tile stands lit by a light the mouse drags, and a body walks into it
// with WASD.
type Tool struct {
	ed    *Editor
	tiles *twod.Tileset
	err   error

	win     *platform.Window
	grid    *tileGrid
	editor  *tileEditor
	pane    *previewPane
	status  part
	checks  map[string]part
	sliders map[string]part
	labels  map[string]part
	// items are the menu bar's items by name, file the File menu and recent
	// its recent tilesets' items, all in the dialogs Document.
	items  map[string]*ui.Node
	file   *ui.Node
	recent map[*ui.Node]string
	// recentItems are recent's items in the menu's order.
	recentItems []*ui.Node
	prompt      *prompt
	conflict    *conflict
	dialog      *diskDialog
	saver       *diskDialog
	// dialogs is the Document the file dialogs show in while inside the tool's window.
	dialogs *ui.Document
	// fade is the lamp's shadow fade, fades its items in light.Fade's order.
	fade  *ui.Node
	fades []*ui.Node
	// view is the inspector; inspected is what it last showed, as
	// inspect keys it.
	view      *inspectorDoc
	inspected string
	state     state
	// The layout: the workspace and the inspector's sections; welcome is
	// the start card shown while no tileset is open, in the dialogs
	// Document, welcomeOpen its Open… button, welcomeList its recent rows'
	// column and welcomeRows their tilesets.
	workspace                         part
	welcome, welcomeOpen, welcomeList *ui.Node
	welcomeRows                       map[*ui.Node]string
	previewPanel, tilePanel           part
	// inspectorWin holds the sections, over the footer's fileStatus
	// and editedDot.
	inspectorWin, fileStatus, editedDot *ui.Node
	// paneFull is the preview's whole box, scrolled partly out of view
	// or not.
	paneFull geom.Rect
	// gridSlot, editorSlot and paneSlot are the boxes the widgets draw in.
	gridSlot, editorSlot, paneSlot *ui.Node
	// inspector scrolls the inspector, up and down only.
	inspector part
	// synced is the tiles the panel last showed, as marks says.
	synced string

	// The preview: its camera, its lit layer and the map in it, the light,
	// the walker and its body, and the space they meet in, through shown,
	// which stands for the map of the moment.
	cam    *twod.Camera
	stage  *twod.Layer
	over   *twod.Layer
	tmap   *twod.TileMap
	lamp   *light.Light
	walker *twod.Sprite
	body   *collide.Body
	space  *collide.Space
	shown  *shownMap
	built  int
	// look is the preview's zoom, the wheel's over it, at the game's two
	// points an art pixel when 1; pan where a middle drag moved it.
	look float32
	pan  geom.Vec2
	// room is the .tmj the preview shows the tileset in, "" for the tile
	// alone; rooms is the dialog that picks it.
	room  string
	rooms *diskDialog

	// chosen, folded, toggled and edited are the items and buttons
	// pressed, the sections folded or opened, the switches and fades
	// changed and the sliders moved since the last Update, in order;
	// closed the alerts and the inspector's window closed, and files the
	// files the dialogs picked.
	chosen, folded, toggled, edited []*ui.Node
	closed                          []ui.Closed
	files                           []kit.FileChosen
}

// shownMap is the solid the preview's space holds: whichever maps show
// the picked tile, or the room, now, as each edit makes new ones. Its
// version changes with them, so a static light bakes them again.
type shownMap struct {
	maps    []*twod.TileMap
	version uint64
}

func (s *shownMap) Version() uint64 { return s.version }

func (s *shownMap) SolidIn(area geom.Rect) []geom.Rect {
	var out []geom.Rect
	for _, m := range s.maps {
		out = append(out, m.SolidIn(area)...)
	}
	return out
}

func (s *shownMap) ShadowsIn(area geom.Rect) []twod.ShadowRect {
	var out []twod.ShadowRect
	for _, m := range s.maps {
		out = append(out, m.ShadowsIn(area)...)
	}
	return out
}

// build is the tool on the tileset at path, or on none when path is "",
// as main runs it and tests drive it, its World made with opts.
func build(path string, opts ...ember.Option) (*world.World, *Tool, error) {
	ed := blank()
	if path != "" {
		var err error
		if ed, err = open(path); err != nil {
			return nil, nil, err
		}
	}
	t := &Tool{
		ed:      ed,
		checks:  map[string]part{},
		sliders: map[string]part{},
		labels:  map[string]part{},
		items:   map[string]*ui.Node{},
		recent:  map[*ui.Node]string{},
		built:   -1,
		state:   loadState(),
	}
	t.win = platform.NewWindow(
		platform.Size(1280, 800),
		platform.MinSize(1100, 700),
		platform.Background(color.Hex("#101216")),
	)
	style := []ui.DocumentOption{kit.Theme(), ui.Stylesheet(previews{}, previewSheet)}
	sheets := fstest.MapFS{
		"menus.css":     {Data: []byte(menuSheet)},
		"shell.css":     {Data: []byte(shellSheet)},
		"inspector.css": {Data: inspectorCSS},
	}
	t.dialogs = ui.NewDocument(
		slices.Concat(
			style,
			[]ui.DocumentOption{
				ui.Stylesheet(sheets, "menus.css"),
				ui.Stylesheet(sheets, "shell.css"),
				ui.Stylesheet(sheets, "inspector.css"),
			},
		)...)
	t.prompt = newPrompt(t.dialogs)
	t.conflict = newConflict(t.dialogs)
	t.closing(t.prompt.panel.node)
	t.closing(t.conflict.panel.node)
	tilesets := []kit.FileDialogOption{
		kit.FileType("Tiled tilesets", ".tsj"),
		kit.Kind("Tiled tileset", ".tsj"),
		kit.OwnWindow(style...),
		kit.Describe(func(p string) string { return describeTileset(osPath(p)) }),
		kit.Preview(func(p string) string {
			if strings.HasSuffix(p, ".tsj") {
				return previewSrc(p)
			}
			return ""
		}),
	}
	picked := kit.OnChoose(func(e kit.FileChosen) { t.files = append(t.files, e) })
	t.dialog = newDiskDialog(
		t.dialogs,
		append([]kit.FileDialogOption{kit.DialogLabel("open"), picked}, tilesets...)...)
	t.saver = newDiskDialog(
		t.dialogs,
		append([]kit.FileDialogOption{kit.DialogLabel("save"), picked}, tilesets...)...)
	t.rooms = newDiskDialog(
		t.dialogs,
		kit.DialogLabel("room"),
		picked,
		kit.FileType("Tiled maps", ".tmj"),
		kit.Kind("Tiled map", ".tmj"),
		kit.OwnWindow(style...),
	)
	for _, d := range []*diskDialog{t.dialog, t.saver, t.rooms} {
		d.SetOwnWindow(!t.state.Inline)
	}
	t.shell()
	for _, e := range t.widgets() {
		t.win.Add(e)
	}
	t.win.Add(t.dialogs)
	t.preview()
	t.titled()
	if path != "" {
		t.state.opened(path)
		_ = t.state.save()
		t.load()
		t.sync()
	}
	t.fillRecent()
	t.enable()
	w := ember.New(opts...).AddEntities(t.win)
	w.AddSystems(t, collide.NewTracker(t.space))
	w.AddSystems(systems(t.dialog, t.saver, t.rooms)...)
	return w, t, nil
}

// titled names the window after the tileset, a dot when it has changes.
func (t *Tool) titled() {
	name := "no tileset"
	if t.ed.path != "" {
		name = filepath.Base(t.ed.path)
	}
	if t.ed.Changed() {
		name += " •"
	}
	t.win.SetTitle("tileset · " + name)
}

// menuSheet pins the menu bar across the top of the window, and centres
// the ui shell's start card under it.
const menuSheet = `#menus { position: absolute; top: 0; left: 0; right: 0; height: 30px; box-sizing: border-box;
  align-items: center; }
#welcome { position: absolute; top: 30px; left: 0; right: 0; bottom: 0; display: flex; flex-direction: column;
  align-items: center; }
#welcome > .above { flex: 1; }
#welcome > .below { flex: 2; }
#welcome.hidden { display: none; }
#welcome > .kit-card { width: 460px; display: flex; flex-direction: column; gap: 12px; }
#welcome .start-row { display: flex; align-items: baseline; gap: 10px; }
#welcome .recent { display: flex; flex-direction: column; gap: 2px; }
#welcome .recent.hidden { display: none; }
#welcome .list-row { display: flex; align-items: baseline; gap: 8px; margin: 0 -8px; padding: 5px 8px;
  border: 1px solid transparent; border-radius: 4px; background-color: transparent; color: var(--light-text);
  text-align: left; }
#welcome .list-row:hover { background-color: var(--hover-color); border-color: var(--border-color); }
#welcome .list-row:focus-visible { outline: 1px solid var(--accent-color); outline-offset: -1px; }
#welcome .list-row > .path { flex: 1; min-width: 0; }`

// menus are the main window's menu bar, File, Edit and View, at the top of
// the dialogs Document; each item's name is its automation leaf under
// its menu's.
func (t *Tool) menus() *ui.Node {
	item := func(name, text string, args ...ui.Arg) *ui.Node {
		it := t.choosing(kit.MenuItem(text, append([]ui.Arg{ui.Attr("data-name", name)}, args...)...))
		t.items[name] = it
		return it
	}
	check := func(name, text string, on bool) *ui.Node {
		it := t.choosing(kit.CheckMenuItem(text, on, ui.Attr("data-name", name)))
		t.items[name] = it
		return it
	}
	bar := func(name, title string, m *ui.Node) *ui.Node {
		b := kit.MenuBarItem(title, m)
		b.SetAttr("data-name", name)
		return b
	}
	item("open", "Open…", kit.Shortcut("Cmd+O"))
	item("save", "Save", kit.Shortcut("Cmd+S"))
	item("save-as", "Save as…", kit.Shortcut("Cmd+Shift+S"))
	item("quit", "Quit", kit.Shortcut("Cmd+Q"))
	t.file = kit.Menu()
	t.fillFile()
	menus := kit.MenuBar(
		bar("file", "File", t.file),
		bar("edit", "Edit", kit.Menu(
			item("undo", "Undo", kit.Shortcut("Cmd+Z")),
			item("redo", "Redo", kit.Shortcut("Cmd+Shift+Z")),
			kit.Separator(),
			item("copy", "Copy shapes", kit.Shortcut("Cmd+C")),
			item("paste", "Paste shapes", kit.Shortcut("Cmd+V")),
			item("apply", "Apply shapes to selected tiles"),
			kit.Separator(),
			item("delete-shape", "Delete shape", kit.Shortcut("Backspace")),
		)),
		bar("view", "View", kit.Menu(
			check("patch", "5×5 patch", false),
			item("room", "Preview room…"),
			item("tile-alone", "Preview the tile alone"),
			check("shadows", "Shadows", true),
			check("static", "Static light (baked)", false),
			check("effective", "Show effective rects", true),
			check("grid", "Pixel grid", true),
			check("snap", "Snap to art pixels", true),
			kit.Separator(),
			check("debug-picture", "Light picture", false),
			check("debug-occluders", "Occluder picture", false),
			check("debug-shadows", "Shadow maps", false),
			check("debug-outlines", "Occluder outlines", false),
			kit.Separator(),
			check("inspector", "Inspector", true),
			check("own-window", "Dialogs in their own window", !t.state.Inline),
		)),
	)
	ui.ID("menus").Apply(menus)
	return menus
}

// fillFile lays the File menu out: Open…, the recent tilesets, then Save,
// Save as… and Quit.
func (t *Tool) fillFile() {
	for _, c := range slices.Clone(t.file.Children()) {
		t.file.Remove(c)
	}
	t.file.Append(t.items["open"])
	t.file.Append(t.recentItems...)
	t.file.Append(kit.Separator(), t.items["save"], t.items["save-as"], kit.Separator(), t.items["quit"])
}

// closing notes n's closes, with their values, for Update.
func (t *Tool) closing(n *ui.Node) *ui.Node {
	ui.On(n, ui.EventClose, func(e ui.Event) { t.closed = append(t.closed, ui.Closed{Node: e.Node, Value: e.Value}) })
	return n
}

// choosing notes n's presses for Update.
func (t *Tool) choosing(n *ui.Node) *ui.Node {
	ui.On(n, ui.EventClick, func(e ui.Event) { t.chosen = append(t.chosen, e.Node) })
	return n
}

// inMenus reports whether n is one of the menu bar's items.
func (t *Tool) inMenus(n *ui.Node) bool {
	if _, ok := t.recent[n]; ok {
		return true
	}
	for _, it := range t.items {
		if it == n {
			return true
		}
	}
	return false
}

// checked reports whether the menu item name is checked, and setChecked
// checks it without choosing it.
func (t *Tool) checked(name string) bool { return t.dialogs.Checked(t.items[name]) }

func (t *Tool) setChecked(name string, on bool) { t.dialogs.SetChecked(t.items[name], on) }

// disable greys the menu items named out, or back in.
func (t *Tool) disable(off bool, names ...string) {
	for _, n := range names {
		if off {
			t.items[n].SetAttr("aria-disabled", "true")
		} else {
			t.items[n].RemoveAttr("aria-disabled")
		}
	}
}

// shortcutText is a shortcut as the menu shows it: Cmd reads Ctrl off
// macOS.
func shortcutText(keys string) string {
	if runtime.GOOS != "darwin" {
		return strings.ReplaceAll(keys, "Cmd", "Ctrl")
	}
	return keys
}

// shell fills the dialogs Document: the menu bar, the start card while no
// tileset is open, else the workspace.
func (t *Tool) shell() {
	t.dialogs.Body().Append(t.menus(), t.welcomeCard(), t.workspaceNode(), t.inspectorWindow())
}

// welcomeCard is the start card: what to do, Open…, and the tilesets
// opened last, their rows named recent-row-<n> and Open… start-open.
func (t *Tool) welcomeCard() *ui.Node {
	t.welcomeRows = map[*ui.Node]string{}
	t.welcomeOpen = t.choosing(kit.Button("Open…", kit.Primary(), ui.Attr("data-name", "start-open")))
	t.welcomeList = ui.Div(ui.Class("recent"), ui.Attr("data-name", "recent"))
	t.welcome = ui.Div(
		ui.ID("welcome"),
		ui.Div(ui.Class("above")),
		kit.Panel(
			ui.Class("kit-card"),
			ui.Attr("data-name", "start"),
			kit.Label("No tileset loaded", kit.AlertTitle()),
			kit.Label(
				"Open a Tiled tileset (.tsj) to draw its tiles' collision and light shapes and set their properties.",
				kit.Wrap(),
				kit.FormLabel(),
			),
			ui.Div(ui.Class("start-row"), t.welcomeOpen, kit.Label(shortcutText("Cmd+O"), kit.FormLabel())),
			kit.Label("RECENT", kit.Caps(), ui.Attr("data-name", "recent-title")),
			t.welcomeList,
		),
		ui.Div(ui.Class("below")),
	)
	return t.welcome
}

// fillWelcome lists the tilesets opened last on the start card.
func (t *Tool) fillWelcome() {
	for _, c := range slices.Clone(t.welcomeList.Children()) {
		t.welcomeList.Remove(c)
	}
	clear(t.welcomeRows)
	others := t.others()
	hide(t.welcomeList, len(others) == 0)
	for i, p := range others {
		row := t.choosing(
			ui.Button(
				ui.Class("list-row"),
				ui.Attr("data-name", fmt.Sprintf("recent-row-%d", i+1)),
				ui.Attr("aria-label", filepath.Base(p)),
				kit.Label(filepath.Base(p)),
				kit.Label(filepath.Dir(p), ui.Class("path", "form-label", "truncate", "truncate-start")),
			),
		)
		t.welcomeRows[row] = p
		t.welcomeList.Append(row)
	}
}

// collides is whether s stops bodies and casts: neither light- nor body-only.
func collides(s Shape) bool { return !s.Shadow && !s.Body }

// autoHeight says the height the engine gives the picked tile when its
// height property is blank: its picture's when tall, its cell's when
// solid, else none.
func (t *Tool) autoHeight() string {
	tile := t.ed.tile
	switch {
	case tile.Tall:
		return fmt.Sprintf("%g (picture)", t.ed.size.Y)
	case tile.Solid || len(tile.Shapes) > 0 && slices.ContainsFunc(tile.Shapes, collides):
		return fmt.Sprintf("%g (cell)", tileSize(t.ed.doc, -1).Y)
	}
	return "0 (flat)"
}

// placeholder is what an empty field of d says: its default.
func placeholder(d twod.TileProperty) string {
	switch {
	case d.Name == propHeight:
		return "auto"
	case d.Default == nil || d.Default == "":
		return ""
	}
	return fmt.Sprint(d.Default)
}

// enable turns the editing controls and Save on when a tileset is loaded,
// off when none is.
func (t *Tool) enable() {
	off := t.ed.path == ""
	for _, c := range t.checks {
		c.SetDisabled(off)
	}

	t.disable(off, "save", "save-as", "undo", "redo", "delete-shape", "copy", "paste", "apply")
	hide(t.welcome, !off)
	t.workspace.SetHidden(off)
	t.syncInspector()
}

// preview sets the preview up in the main window, under its UI, clipped
// to the preview pane: a dark floor, the lit layer the picked tile stands
// in, the light and the walker, and outlines over it.
func (t *Tool) preview() {
	t.space = collide.NewSpace()
	t.shown = &shownMap{}
	t.space.AddSolid(t.shown)
	t.cam = twod.NewCamera(light.Ambient(color.RGB(0.05, 0.05, 0.08)), light.OccludersFrom(t.space))
	t.lamp = light.NewPoint(200, color.RGB(1, 0.85, 0.65), light.Falloff(0.3), light.At(-120, -80))
	t.walker = twod.NewSprite(solid(color.White), twod.At(-120, 60), twod.Size(16, 10), light.Unlit())
	t.body = collide.NewBody(collide.Box(16, 10))
	t.walker.Add(t.body)
	floor := twod.NewSprite(solid(color.Hex("#8a8478")), twod.Size(640, 480))
	t.stage = twod.NewLayer(0, light.Lit(), twod.Hidden()).Add(floor)
	t.over = twod.NewLayer(1, twod.Hidden()).
		Add(t.walker, t.lamp, twod.Build(&outlines{Item: twod.Base(), tool: t}, "outlines"))
	t.win.Add(t.cam, t.stage, t.over)
}

// solid is a one-pixel picture of c.
func solid(c color.Color) *twod.Texture {
	s := c.SRGB()
	img := newRGBA(1, 1)
	img.Pix[0], img.Pix[1], img.Pix[2], img.Pix[3] = uint8(s[0]*255), uint8(s[1]*255), uint8(s[2]*255), 255
	return twod.NewTexture(img)
}

// load reads the tileset as edited so far through the engine's own
// reader, and stands the picked tile in the preview with it.
func (t *Tool) load() {
	if t.ed.path == "" {
		return
	}
	fsys, name := t.ed.fs()
	ts, err := twod.LoadTiledTileset(fsys, name, twod.TileProperties(declared[len(twod.EngineTileProperties()):]...))
	t.err = err
	if err != nil {
		return
	}
	t.tiles = ts
	for _, m := range t.shown.maps {
		if w := world.Of(m); w != nil {
			w.RemoveEntity(m)
		} else {
			t.stage.RemoveChild(m.BasicEntity)
		}
	}
	t.shown.maps, t.tmap = nil, nil
	t.shown.version++
	t.built = t.ed.version
	if t.room != "" {
		if roomErr := t.loadRoom(); roomErr != nil {
			t.status.SetText("Can't show the room: " + roomErr.Error())
			t.room = ""
		} else {
			return
		}
	}
	id := t.ed.tile.ID + 1
	cells := [][]int{{id}}
	if t.checked("patch") {
		cells = [][]int{
			{id, id, id, id, id},
			{id, id, id, id, id},
			{id, id, id, id, id},
			{id, id, id, id, id},
			{id, id, id, id, id},
		}
	}
	cell := tileSize(t.ed.doc, -1).Scale(2)
	if g := t.ed.doc.get("grid"); g != nil && t.ed.doc.get("image") == nil {
		cell = geom.V2(float32(g.get("width").num()), float32(g.get(keyHeight).num())).Scale(2)
	}
	n := float32(len(cells))
	t.tmap = twod.NewTileMap(ts, twod.Cells(cells), twod.CellSize(cell.X, cell.Y), twod.At(-cell.X*n/2, -cell.Y*n/2))
	t.shown.maps = []*twod.TileMap{t.tmap}
	t.stage.Add(t.tmap)
}

// loadRoom stands the room t.room in the preview, centred, at two points
// an art pixel, its tilesets read as they are on disk but the one edited,
// read as edited so far.
func (t *Tool) loadRoom() error {
	fsys, _ := t.ed.fs()
	abs, err := filepath.Abs(t.room)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(filepath.VolumeName(abs)+string(filepath.Separator), abs)
	if err != nil {
		return err
	}
	name := filepath.ToSlash(rel)
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return err
	}
	var size struct {
		Width      float32 `json:"width"`
		Height     float32 `json:"height"`
		TileWidth  float32 `json:"tilewidth"`
		TileHeight float32 `json:"tileheight"`
	}
	if err = json.Unmarshal(b, &size); err != nil {
		return err
	}
	cell := geom.V2(size.TileWidth, size.TileHeight).Scale(2)
	room, err := twod.LoadTiled(
		fsys,
		name,
		twod.CellSize(cell.X, cell.Y),
		twod.At(-size.Width*cell.X/2, -size.Height*cell.Y/2),
		twod.TileProperties(declared[len(twod.EngineTileProperties()):]...),
	)
	if err != nil {
		return err
	}
	for _, m := range room.Layers() {
		t.shown.maps = append(t.shown.maps, m)
		t.stage.Add(m)
	}
	return nil
}

// pick makes tile id the one edited, marking no other.
func (t *Tool) pick(id int) {
	t.ed.pick(id)
	t.ed.marked = nil
	t.changed()
}

// mark adds tile id to the tiles edited with the picked one, or takes it
// away.
func (t *Tool) mark(id int) {
	ed := t.ed
	if id == ed.tile.ID {
		return
	}
	if ed.marked == nil {
		ed.marked = map[int]bool{}
	}
	if ed.marked[id] {
		delete(ed.marked, id)
	} else {
		ed.marked[id] = true
	}
	if n := len(ed.marked); n > 0 {
		tiles, them := "tiles", "them"
		if n == 1 {
			tiles, them = "tile", "it"
		}
		t.status.SetText(fmt.Sprintf("%d more %s selected: properties and Apply shapes reach %s too.", n, tiles, them))
	} else {
		t.status.SetText("")
	}
}

// changed has the panels and the preview follow the editor.
func (t *Tool) changed() { t.built = -1 }

// Priority runs the tool before ui and its dialogs, whose callbacks it
// takes the frame after.
func (t *Tool) Priority() int { return 0 }

func (t *Tool) Update(f world.Frame) {
	w := f.World
	ed := t.ed
	files := t.files
	t.files = nil
	for _, e := range files {
		t.answer(e)
	}
	chosen, folded, toggled, edited, closed := t.chosen, t.folded, t.toggled, t.edited, t.closed
	t.chosen, t.folded, t.toggled, t.edited, t.closed = nil, nil, nil, nil, nil
	for _, n := range chosen {
		switch p, recent := t.welcomeRows[n]; {
		case t.inMenus(n):
			t.menu(f, n)
		case n == t.welcomeOpen:
			t.ask(w, false)
		case recent:
			t.openFile(p)
		}
	}
	for _, e := range closed {
		if e.Node == t.inspectorWin && t.ed.path != "" {
			t.dialogs.SetChecked(t.items["inspector"], false)
			continue
		}
		if !t.prompt.answered(e, t) {
			t.conflict.answered(e, t)
		}
	}
	for _, n := range folded {
		t.fold(n)
	}
	t.lampInput(toggled, edited)
	if t.built != ed.version {
		t.load()
		t.sync()
	} else if t.marks() != t.synced {
		t.sync()
	}
	t.selection()
	t.nudge()
	t.drive(f)
	if (light.DebugOf(t.win) != light.DebugOff || light.OutlinesOf(t.win)) &&
		light.DebugAreaOf(t.win) != t.pane.Rect() {
		light.SetDebugArea(t.win, t.pane.Rect())
	}
	t.showLampValues()
	t.view.Update(w)
	t.inspect()
	t.titled()
	t.showFile()
	t.placeWidgets()
}

// fold keeps whether panel n is folded in the tool's state, and scrolls it
// into view when it opens.
func (t *Tool) fold(n *ui.Node) {
	for _, p := range []part{t.previewPanel, t.tilePanel} {
		if n != p.n {
			continue
		}
		if t.state.Folded == nil {
			t.state.Folded = map[string]bool{}
		}
		name, _ := p.n.Attr("data-name")
		t.state.Folded[name] = !t.dialogs.Open(p.n)
		if t.dialogs.Open(p.n) {
			t.dialogs.ScrollIntoView(p.n)
		}
		if err := t.state.save(); err != nil {
			t.status.SetText(err.Error())
		}
	}
}

// lampInput sets the preview lamp from the checks toggled and the sliders
// edited this frame.
func (t *Tool) lampInput(toggled, edited []*ui.Node) {
	for _, n := range toggled {
		if sh := t.checks["shadows"]; n == sh.n {
			t.lamp.SetShadows(sh.Checked())
			t.setChecked("shadows", sh.Checked())
		}
		if i := slices.Index(t.fades, n); i >= 0 {
			t.lamp.SetShadowFade(light.Fade(i))
		}
	}
	for _, n := range edited {
		switch n {
		case t.sliders["radius"].n:
			t.lamp.SetRadius(t.sliders["radius"].Value())
		case t.sliders["elevation"].n:
			t.lamp.SetElevation(t.sliders["elevation"].Value())
		case t.sliders["size"].n:
			t.lamp.SetSize(t.sliders["size"].Value())
		case t.sliders["body-w"].n, t.sliders["body-h"].n:
			t.resizeBody()
		}
	}
}

// showLampValues writes each lamp slider's value beside it.
func (t *Tool) showLampValues() {
	t.labels["radius"].SetText(fmt.Sprintf("%.0f", t.sliders["radius"].Value()))
	if e := t.sliders["elevation"].Value(); e > 0 {
		t.labels["elevation"].SetText(fmt.Sprintf("%.0f", e))
	} else {
		t.labels["elevation"].SetText("off")
	}
	t.labels["size"].SetText(fmt.Sprintf("%.0f", t.sliders["size"].Value()))
	t.labels["body-w"].SetText(fmt.Sprintf("%.0f", t.sliders["body-w"].Value()))
	t.labels["body-h"].SetText(fmt.Sprintf("%.0f", t.sliders["body-h"].Value()))
}

// menu does what a menu item asks.
func (t *Tool) menu(f world.Frame, it *ui.Node) {
	ed := t.ed
	switch it {
	case t.items["open"]:
		t.unsaved(func() { t.ask(f.World, false) })
	case t.items["save"]:
		t.save()
	case t.items["save-as"]:
		if ed.path == "" {
			t.status.SetText("Open a tileset first.")
			return
		}
		t.ask(f.World, true)
	case t.items["quit"]:
		t.unsaved(f.Quit)
	case t.items["undo"]:
		ed.Undo()
	case t.items["redo"]:
		ed.Redo()
	case t.items["copy"]:
		t.status.SetText(fmt.Sprintf("Copied %d shapes.", ed.Copy()))
	case t.items["paste"]:
		ed.Paste()
	case t.items["apply"]:
		if n := ed.ApplyShapes(); n == 0 {
			t.status.SetText("Shift- or Cmd-click tiles first to apply these shapes to them.")
		} else {
			t.status.SetText(fmt.Sprintf("Shapes applied to %d tiles.", n))
		}
	case t.items["delete-shape"]:
		t.deleteSelected()
	case t.items["patch"], t.items["tile-alone"]:
		t.room = ""
		t.changed()
	case t.items["room"]:
		t.rooms.Open(t.roomFolder())
	case t.items["shadows"]:
		on := t.dialogs.Checked(it)
		t.lamp.SetShadows(on)
		t.checks["shadows"].SetChecked(on)
	case t.items["static"]:
		t.lamp.SetStatic(t.dialogs.Checked(it))
	case t.items["snap"]:
		ed.free = !t.dialogs.Checked(it)
	case t.items["debug-picture"], t.items["debug-occluders"], t.items["debug-shadows"], t.items["debug-outlines"]:
		t.lightDebug(it)
	case t.items["inspector"]:
		t.syncInspector()
	case t.items["own-window"]:
		t.setOwnWindow(t.dialogs.Checked(it))
	default:
		if p, ok := t.recent[it]; ok {
			t.unsaved(func() { t.openFile(p) })
		}
	}
}

// roomFolder is where the room dialog opens: the shown room's folder, else
// the rooms folder beside the tileset's, else the dialog's start.
func (t *Tool) roomFolder() string {
	folder := t.state.startFolder()
	if t.room != "" {
		folder = filepath.Dir(t.room)
	} else if d := filepath.Join(filepath.Dir(t.ed.path), "..", "..", "dungeon", "rooms"); t.ed.path != "" {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			folder = d
		}
	}
	return folder
}

// setOwnWindow shows the file dialogs in their own windows or in the tool's,
// remembering the choice.
func (t *Tool) setOwnWindow(own bool) {
	t.dialog.SetOwnWindow(own)
	t.saver.SetOwnWindow(own)
	t.rooms.SetOwnWindow(own)
	t.state.Inline = !own
	if err := t.state.save(); err != nil {
		t.status.SetText(err.Error())
	}
}

// debugViews are the View menu's lighting debug views.
//
//nolint:gochecknoglobals // a lookup table
var debugViews = map[string]light.DebugView{
	"debug-picture":   light.DebugLight,
	"debug-occluders": light.DebugOccluders,
	"debug-shadows":   light.DebugShadows,
}

// lightDebug shows the lighting debug view or outlines menu item it asks
// for over the preview: one view at a time.
func (t *Tool) lightDebug(it *ui.Node) {
	if it == t.items["debug-outlines"] {
		light.SetOutlines(t.win, t.dialogs.Checked(it))
	} else {
		light.SetDebug(t.win, t.debugView(it))
	}
	light.SetDebugArea(t.win, t.pane.Rect())
}

// debugView is the debug view item it shows when checked, every other one
// unchecked.
func (t *Tool) debugView(it *ui.Node) light.DebugView {
	view := light.DebugOff
	for name, v := range debugViews {
		if t.items[name] == it {
			if t.dialogs.Checked(it) {
				view = v
			}
		} else {
			t.setChecked(name, false)
		}
	}
	return view
}

// unsaved runs then, first asking whether to save the changes, if any.
func (t *Tool) unsaved(then func()) {
	if !t.ed.Changed() || t.ed.path == "" {
		then()
		return
	}
	t.prompt.ask(filepath.Base(t.ed.path), then)
}

// answered handles the prompt closing in e, reporting whether it was the
// prompt's.
func (p *prompt) answered(e ui.Closed, t *Tool) bool {
	v, ok := p.panel.answer(e)
	if !ok {
		return false
	}
	switch v {
	case p.save.value:
		switch {
		case t.save():
			p.then()
		case !t.conflict.panel.Hidden():
			t.conflict.then = p.then
		default:
			p.panel.show("Save changes to "+filepath.Base(t.ed.path)+"?", "Not saved: "+t.status.Text())
		}
	case p.discard.value:
		p.then()
	}
	return true
}

// ask shows the file dialog, to open a tileset or to save this one as
// another; its answer comes back to Update.
func (t *Tool) ask(_ *world.World, save bool) {
	folder := t.state.startFolder()
	if t.ed.path != "" {
		folder = filepath.Dir(t.ed.path)
	}
	recent := t.others()
	var places []string
	for _, p := range append([]string{t.ed.path}, recent...) {
		if f := filepath.Dir(p); p != "" && !slices.Contains(places, f) && len(places) < 4 {
			places = append(places, f)
		}
	}
	if home, err := os.UserHomeDir(); err == nil && !slices.Contains(places, home) {
		places = append(places, home)
	}
	t.dialog.SetPlaces(places...)
	t.saver.SetPlaces(places...)
	if save {
		t.saver.SetRecent(recent...)
		t.saver.Save(folder, filepath.Base(t.ed.path))
		return
	}
	t.dialog.SetRecent(recent...)
	t.dialog.Open(folder)
}

// previewRoom shows the tileset in the Tiled map at path in the preview,
// as View › Preview room… does.
func (t *Tool) previewRoom(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if _, err = os.Stat(abs); err != nil {
		return fmt.Errorf("room: %w", err)
	}
	t.room = abs
	t.changed()
	return nil
}

// define gives automation the tool's own actions: preview-room, with a
// "path" to a .tmj, shows the tileset in that room.
func (t *Tool) define(a *automation.Automation) {
	a.Define(
		automation.Action{ID: "preview-room", Label: "Preview a room", Run: func(args map[string]any) (any, error) {
			path, _ := args["path"].(string)
			if path == "" {
				return nil, errors.New("preview-room: a path to a .tmj")
			}
			return nil, t.previewRoom(path)
		}},
	)
}

// answer opens the tileset the file dialog picked, or saves under it.
func (t *Tool) answer(e kit.FileChosen) {
	if p, ok := t.rooms.chosen(e); ok {
		t.room = p
		t.changed()
		return
	}
	p, ok := t.dialog.chosen(e)
	if !ok {
		if p, ok = t.saver.chosen(e); !ok {
			return
		}
	}
	if !e.Saving {
		t.openFile(p)
		return
	}
	if err := t.ed.SaveAs(p); err != nil {
		t.status.SetText(err.Error())
		return
	}
	t.status.SetText("Saved " + filepath.Base(p))
	t.remember(p)
}

// openFile makes the tileset at path the one edited.
func (t *Tool) openFile(path string) {
	ed, err := open(path)
	if err != nil {
		t.status.SetText("Can't open it: " + err.Error())
		return
	}
	t.ed = ed
	t.status.SetText("")
	t.changed()
	t.remember(path)
	t.enable()
}

// remember puts path first among the recent tilesets, in the File menu
// and on disk.
func (t *Tool) remember(path string) {
	t.state.opened(path)
	if err := t.state.save(); err != nil {
		t.status.SetText(err.Error())
	}
	t.fillRecent()
	t.titled()
}

// others are the recent tilesets but the one open.
func (t *Tool) others() []string {
	var out []string
	for _, p := range t.state.Recent {
		if p != t.ed.path {
			if abs, err := filepath.Abs(t.ed.path); t.ed.path == "" || err != nil || abs != p {
				out = append(out, p)
			}
		}
	}
	return out
}

// fillRecent lists the recent tilesets in the File menu, under Open….
func (t *Tool) fillRecent() {
	clear(t.recent)
	t.recentItems = t.recentItems[:0]
	for i, p := range t.others() {
		it := t.choosing(
			kit.MenuItem(
				filepath.Base(p)+"  ·  "+filepath.Base(filepath.Dir(p)),
				ui.Attr("data-name", fmt.Sprintf("recent-%d", i+1)),
			),
		)
		t.recent[it] = p
		t.recentItems = append(t.recentItems, it)
	}
	t.fillFile()
	t.fillWelcome()
}

// sync notes the tiles the inspector shows, and says when the tileset
// can't be read.
func (t *Tool) sync() {
	if t.err != nil {
		t.status.SetText("Can't read it: " + t.err.Error())
	}
	t.synced = t.marks()
}

// marks names the tiles the panel shows: the picked one and those marked.
func (t *Tool) marks() string {
	var ids []string
	for _, tile := range t.ed.targets() {
		ids = append(ids, strconv.Itoa(tile.ID))
	}
	return strings.Join(ids, ",")
}

// propertyOf is property name of tile as the panel shows it.
func propertyOf(tile Tile, name string) string {
	switch name {
	case propSolid:
		return fmt.Sprint(tile.Solid)
	case propTall:
		return fmt.Sprint(tile.Tall)
	case propShadow:
		return fmt.Sprint(tile.NoShadow)
	case propOccluderGroup:
		return fmt.Sprint(tile.Group)
	}
	if v, ok := tile.Values[name]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

// save saves the tileset, reporting whether it did.
func (t *Tool) save() bool {
	if t.ed.path == "" {
		t.status.SetText("Open a tileset first.")
		return false
	}
	if err := t.ed.Save(); err != nil {
		if errors.Is(err, errChangedOnDisk) {
			t.conflict.ask(filepath.Base(t.ed.path))
		}
		t.status.SetText(err.Error())
		return false
	}
	t.status.SetText("Saved " + filepath.Base(t.ed.path))
	t.titled()
	return true
}

// property sets declared property name from text typed in its field.
func (t *Tool) property(name, text string) {
	d, ok := definition(name)
	if !ok {
		return
	}
	text = strings.TrimSpace(text)
	var v any = text
	switch d.Kind {
	case twod.KindInt:
		n, err := strconv.Atoi(text)
		if text == "" {
			n, err = 0, nil
		}
		if err != nil {
			t.status.SetText(name + ": not a whole number")
			return
		}
		v = n
	case twod.KindFloat:
		f, err := strconv.ParseFloat(text, 64)
		if text == "" {
			f, err = 0, nil
		}
		if err != nil {
			t.status.SetText(name + ": not a number")
			return
		}
		v = f
	case twod.KindBool, twod.KindString, twod.KindEnum, twod.KindColor:
	}
	if f, isFloat := v.(float64); isFloat && d.Min < d.Max && (f < d.Min || f > d.Max) {
		t.status.SetText(fmt.Sprintf("%s: between %v and %v", name, d.Min, d.Max))
		return
	}
	if n, isInt := v.(int); isInt && d.Min < d.Max && (float64(n) < d.Min || float64(n) > d.Max) {
		t.status.SetText(fmt.Sprintf("%s: between %v and %v", name, d.Min, d.Max))
		return
	}
	t.ed.SetValue(name, v)
}

// deleteSelected deletes the selected point of a polygon, saying why when
// it can't, else the selected shape.
func (t *Tool) deleteSelected() {
	ed := t.ed
	if _, ok := ed.polygon(); ok && ed.point >= 0 {
		if err := ed.DeletePoint(); err != nil {
			t.status.SetText(err.Error())
		}
		return
	}
	ed.Delete()
}

// nudge moves the selected shape by an art pixel per arrow, four with
// Shift, while no element has the keys.
func (t *Tool) nudge() {
	in := t.win.Input()
	if t.ed.selected < 0 || in.KeyboardCaptured() || t.pane.Focused() {
		return
	}
	step := float32(1)
	if in.Down(input.KeyLeftShift) || in.Down(input.KeyRightShift) {
		step = 4
	}
	var d geom.Vec2
	switch {
	case in.Pressed(input.KeyLeft):
		d.X = -step
	case in.Pressed(input.KeyRight):
		d.X = step
	case in.Pressed(input.KeyUp):
		d.Y = -step
	case in.Pressed(input.KeyDown):
		d.Y = step
	}
	t.ed.Move(d)
}

// selection turns Delete on only while a shape is selected.
func (t *Tool) selection() {
	none := t.ed.path == "" || t.ed.selected < 0
	t.disable(none, "delete-shape")
}

// resizeBody gives the walker a body of the sliders' size.
func (t *Tool) resizeBody() {
	t.walker.RemoveChild(t.body.BasicEntity)
	size := geom.V2(t.sliders["body-w"].Value(), t.sliders["body-h"].Value())
	t.body = collide.NewBody(collide.Box(size.X, size.Y))
	t.walker.Add(t.body)
	t.walker.SetSize(size)
}

// drive moves the light where the mouse drags it in the preview, and the
// walker with WASD, through the space as a game moves it.
func (t *Tool) drive(f world.Frame) {
	r := t.pane.Rect()
	in := t.win.Input()
	if t.look == 0 {
		t.look = 1
	}
	if r.Contains(in.Mouse()) {
		if w := in.Wheel().Y; w != 0 {
			t.look = min(max(t.look*float32(math.Pow(1.25, float64(w))), 0.25), 8)
		}
		if in.Down(input.MouseMiddle) {
			t.pan = t.pan.Add(in.MouseDelta().Scale(1 / t.look))
		}
	}
	t.cam.SetZoom(t.look)
	full := t.paneFull
	if full.W <= 0 {
		full = r
	}
	mid := full.Center()
	half := t.win.Size().Scale(0.5)
	t.cam.SetOffset((half.X-mid.X)/t.look-t.pan.X, (half.Y-mid.Y)/t.look-t.pan.Y)
	clip := geom.Rect{
		X: (r.X-mid.X)/t.look - t.pan.X,
		Y: (r.Y-mid.Y)/t.look - t.pan.Y,
		W: r.W / t.look,
		H: r.H / t.look,
	}
	t.stage.SetClipTo(clip)
	t.over.SetClipTo(clip)
	hidden := t.ed.path == "" || r.W <= 0 || r.H <= 0
	t.stage.SetHidden(hidden)
	t.over.SetHidden(hidden)
	if in.Down(input.MouseLeft) && r.Contains(in.Mouse()) {
		t.lamp.SetPosition(t.cam.ScreenToWorld(in.Mouse()))
	}
	if !t.pane.Focused() {
		return
	}
	var d geom.Vec2
	if in.Down(input.KeyA) {
		d.X--
	}
	if in.Down(input.KeyD) {
		d.X++
	}
	if in.Down(input.KeyW) {
		d.Y--
	}
	if in.Down(input.KeyS) {
		d.Y++
	}
	if d != (geom.Vec2{}) {
		t.space.Move(t.walker, d.Scale(120*f.DT/d.Len()))
	}
}

// outlines draws, over the preview, the shapes bodies meet (orange) and
// those that only block light (blue), as the engine places them, and the
// walker's body.
type outlines struct {
	twod.Item
	tool *Tool
}

func (o *outlines) Draw(c *twod.Canvas) {
	m := o.tool.shown
	everything := geom.Rect{X: -2000, Y: -2000, W: 4000, H: 4000}
	solids := map[geom.Rect]bool{}
	for _, r := range m.SolidIn(everything) {
		solids[r] = true
		c.Border(r, 0, 1, collisionColor)
	}
	for _, s := range m.ShadowsIn(everything) {
		if !solids[s.Rect] {
			c.Border(s.Rect, 0, 1, shadowColor)
		}
	}
	b := o.tool.body.Bounds()
	c.Border(b, 0, 1, color.White)
	c.Circle(o.tool.lamp.Position(), 4, color.Hex("#ffd34a"))
}

// recorder is the automation the tool serves: on EMBER_AUTOMATION, a
// client drives it by its panels' IDs.
func recorder() *automation.Automation { return automation.New() }

// shownValue is engine property d of tile as its pop-up shows it, and
// whether the tile sets it; its default when not.
func shownValue(tile Tile, d twod.TileProperty) (string, bool) {
	switch d.Name {
	case propShadow:
		if tile.NoShadow {
			return shadowNone, true
		}
		return "cast", false
	case propSolid:
		return fmt.Sprint(tile.Solid), tile.Solid || tile.SolidSet
	case propTall:
		return fmt.Sprint(tile.Tall), tile.Tall || tile.TallSet
	case propOccluderGroup:
		return fmt.Sprint(max(tile.Group, 0)), tile.Group >= 0
	}
	if v, ok := tile.Values[d.Name]; ok {
		return fmt.Sprint(v), true
	}
	return fmt.Sprint(d.Default), false
}

// chose sets engine property name, an enum or the occluder group, to the
// value picked at index of its pop-up.
func (t *Tool) chose(name string, index int) {
	d, ok := definition(name)
	values := valuesOf(d)
	if !ok || index < 0 || index >= len(values) {
		return
	}
	ed := t.ed
	switch name {
	case propOccluderGroup:
		if index != ed.tile.Group || len(ed.marked) > 0 {
			ed.SetGroup(index)
		}
	default:
		ed.SetValue(name, values[index])
	}
}
