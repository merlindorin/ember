package kit

import (
	"cmp"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
)

// FileChosen is emitted when the player picks a file in a FileDialog: one to open, or,
// Saving, the path to save under. Path is a path of the dialog's fs.FS.
type FileChosen struct {
	Dialog *FileDialog
	Path   string
	Saving bool
}

// DialogCancelled is emitted when the player closes a FileDialog without picking a file:
// Cancel, Escape or its close light.
type DialogCancelled struct{ Dialog *FileDialog }

// FileDialog is the old ember/ui's FileDialog over an fs.FS: a modal Window whose UnifiedToolbar
// holds back and forward, a PathPopUp of the folder, a list or grid view and a search
// field; places and recent files down a Sidebar; the folder's folders and files in a
// Table (name, date, size, kind; folders first whatever the sort), files of another type
// dimmed, or as Thumbnails; the selected file's picture, name and details under them;
// and Cancel and Open at the bottom. Arrows walk the files, Enter or a double click opens
// a folder or picks a file, the search field filters the folder.
//
// It is kit's stateful kind of component: a value made from the Document, whose node it
// appends to the body, that the game adds to the World as a system. It reacts only to
// events from inside its node, and only while its node is in the Document and the
// Document in the World; otherwise it drops its listing and does nothing until its node
// comes back. Its events carry it, so two dialogs on a page never cross.
type FileDialog struct {
	// doc is the Document holding the dialog's node: home, the game's, or its own window's.
	doc  *ui.Document
	home *ui.Document
	// later is an Open or Save in its own window asked before the dialog was a system,
	// done at its first Update.
	later func()
	fsys  fs.FS
	world *world.World

	types      []fileType
	shown      int
	places     []string
	recent     []string
	describe   func(path string) string
	preview    func(path string) string
	dateFormat func(t time.Time) string
	rootName   string
	kinds      map[string]string
	own        bool
	ownStyle   []ui.DocumentOption
	label      string
	window     *platform.Window

	open          bool
	saving        bool
	folder        string
	back, forward []string
	entries       []fileEntry
	rows          map[*ui.Node]*fileEntry
	sortColumn    int
	sortDesc      bool
	replacing     string

	node, backButton, forwardButton, path, view, search, side, table, grid, note, info, hint, cancel, ok *ui.Node
	typeLabel, typePopUp, nameLabel, name, alert                                                         *ui.Node

	// onChoose and onCancel are the game's OnChoose and OnCancel.
	onChoose func(*world.World, FileChosen)
	onCancel func(*world.World)

	// windows stays a reader: a window closing is the platform's, no
	// element's.
	windows world.Reader[platform.WindowClosed]
}

type fileType struct {
	name string
	exts []string
}

func (t fileType) accepts(name string) bool {
	if len(t.exts) == 0 {
		return true
	}
	ext := strings.ToLower(path.Ext(name))
	return slices.Contains(t.exts, ext)
}

func (t fileType) label() string {
	if t.name == "" {
		return strings.Join(t.exts, ", ")
	}
	return t.name + " (" + strings.Join(t.exts, ", ") + ")"
}

type fileEntry struct {
	path string
	dir  bool
	size int64
	mod  time.Time
}

// FileDialogOption sets up a FileDialog.
type FileDialogOption interface{ applyFileDialog(*FileDialog) }

type fileDialogOption func(*FileDialog)

func (o fileDialogOption) applyFileDialog(d *FileDialog) { o(d) }

// Extensions lets only the files ending in one of exts, such as ".png", be chosen; the
// others show dimmed. Without, any file can be.
func Extensions(exts ...string) FileDialogOption {
	return fileDialogOption(func(d *FileDialog) { d.types = append(d.types, fileType{exts: lower(exts)}) })
}

// FileType adds a type of file named name, ending in one of exts, that the dialog can
// show; the first is shown first.
func FileType(name string, exts ...string) FileDialogOption {
	return fileDialogOption(func(d *FileDialog) { d.types = append(d.types, fileType{name: name, exts: lower(exts)}) })
}

// Places lists folders down the side under Favourites, before the FS's root.
func Places(folders ...string) FileDialogOption {
	return fileDialogOption(func(d *FileDialog) { d.places = slices.Clone(folders) })
}

// Recent lists files down the side, the latest first, for a click to pick.
func Recent(paths ...string) FileDialogOption {
	return fileDialogOption(func(d *FileDialog) { d.recent = slices.Clone(paths) })
}

// Describe gives the line under the selected file's name, "" for the dialog's own.
func Describe(line func(path string) string) FileDialogOption {
	return fileDialogOption(func(d *FileDialog) { d.describe = line })
}

// Preview gives the picture of the file at path, as an image src of the Document's
// stylesheets' files, "" for none.
func Preview(src func(path string) string) FileDialogOption {
	return fileDialogOption(func(d *FileDialog) { d.preview = src })
}

// DateFormat writes a file's modification time, "Today, 15:04", "Yesterday, 15:04" or
// "2 Jan 2006" in the time's own zone by default.
func DateFormat(format func(t time.Time) string) FileDialogOption {
	return fileDialogOption(func(d *FileDialog) { d.dateFormat = format })
}

// DialogLabel names the dialog for automation and assistive tools, "Open" or "Save as" by
// default as it shows; its parts' automation IDs start with label, as label/files/<file>,
// label/file-name, else with file-dialog.
func DialogLabel(label string) FileDialogOption {
	return fileDialogOption(func(d *FileDialog) { d.label = label })
}

// ChooseHandler is what OnChoose takes: a function of the chosen file, and
// of the World too when it needs it.
type ChooseHandler interface {
	func(FileChosen) | func(*world.World, FileChosen)
}

// OnChoose runs fn once a file is chosen, after FileChosen is emitted.
func OnChoose[F ChooseHandler](fn F) FileDialogOption {
	return fileDialogOption(func(d *FileDialog) {
		switch f := any(fn).(type) {
		case func(FileChosen):
			d.onChoose = func(_ *world.World, e FileChosen) { f(e) }
		case func(*world.World, FileChosen):
			d.onChoose = f
		}
	})
}

// CancelHandler is what OnCancel takes: a function, of the World when it
// needs it.
type CancelHandler interface {
	func() | func(*world.World)
}

// OnCancel runs fn once the dialog closes without a file, after
// DialogCancelled is emitted.
func OnCancel[F CancelHandler](fn F) FileDialogOption {
	return fileDialogOption(func(d *FileDialog) {
		switch f := any(fn).(type) {
		case func():
			d.onCancel = func(*world.World) { f() }
		case func(*world.World):
			d.onCancel = f
		}
	})
}

// OwnWindow shows the dialog in a platform window of its own, 900 by 600 points, which
// Open and Save add to the World and a pick or a cancel removes; closing it cancels. Its
// Document is styled with style, the game's own Document options, kit's Theme without.
func OwnWindow(style ...ui.DocumentOption) FileDialogOption {
	return fileDialogOption(func(d *FileDialog) { d.own, d.ownStyle = true, style })
}

// Kind names the files ending in one of exts, such as a game's saves, in the list's Kind
// column and the details; it wins over the dialog's own names of common formats.
func Kind(name string, exts ...string) FileDialogOption {
	return fileDialogOption(func(d *FileDialog) {
		if d.kinds == nil {
			d.kinds = map[string]string{}
		}
		for _, e := range lower(exts) {
			d.kinds[e] = name
		}
	})
}

func lower(exts []string) []string {
	out := make([]string, len(exts))
	for i, e := range exts {
		out[i] = strings.ToLower(e)
	}
	return out
}

// NewFileDialog returns a hidden file dialog over fsys, its node appended to doc's body.
// Add it to the World as a system; Open shows it.
func NewFileDialog(doc *ui.Document, fsys fs.FS, opts ...FileDialogOption) *FileDialog {
	d := &FileDialog{doc: doc, home: doc, fsys: fsys, dateFormat: dateText, rootName: "/"}
	for _, o := range opts {
		o.applyFileDialog(d)
	}
	d.build()
	doc.Body().Append(d.node)
	doc.SortBy(TableGrid(d.table), d.compare)
	return d
}

func (d *FileDialog) build() {
	d.backButton = Button("‹", ui.Class("kit-nav"), ui.Attr("aria-label", "back"), ui.Disabled())
	d.forwardButton = Button("›", ui.Class("kit-nav"), ui.Attr("aria-label", "forward"), ui.Disabled())
	d.path = PathPopUp("/", RootName(d.rootName))
	d.view = Choice(0, []*ui.Node{viewGlyph("list"), viewGlyph("grid")},
		ui.Class("kit-segmented", "kit-view-toggle", "own-height"))
	d.search = SearchField("", ui.Class("kit-file-search"))
	d.side = Sidebar(SidebarSection("Favourites"), SidebarSection("Recent"))
	d.side.AddClass("kit-places")
	d.table = Table([]*ui.Node{
		TableColumn("Name", Sortable()),
		TableColumn("Date modified", ColumnWidth(160), Sortable()),
		TableColumn("Size", ColumnWidth(96), Sortable()),
		TableColumn("Kind", ColumnWidth(100), Sortable()),
	}, nil, ui.Class("kit-file-list"))
	TableGrid(d.table).SetAttr("aria-label", "Files")
	TableGrid(d.table).SetAttr("data-name", "files")
	TableGrid(d.table).SetAttr("autofocus", "")
	TableGrid(d.table).SetAttr("tabindex", "0")
	d.grid = Thumbnails(ui.Class("kit-file-list"), ui.Attr("hidden", ""))
	d.note = Label("", ui.Class("kit-file-note"), ui.Attr("hidden", ""))
	d.info = ui.Div(ui.Class("kit-file-info"), ui.Attr("hidden", ""))
	d.hint = Label("", ui.Class("kit-file-hint"))
	d.cancel = Button("Cancel", ui.Class("kit-file-cancel"), ui.Attr("data-name", "cancel"))
	d.ok = Button(openTitle, ui.Class("kit-primary", "kit-file-ok"), ui.Attr("data-name", "ok"))
	labels := make([]string, len(d.types))
	for i, t := range d.types {
		labels[i] = t.label()
	}
	d.typeLabel = Label("Show", ui.Class("kit-file-label"))
	d.typePopUp = PopUp(0, labels, ui.Class("kit-file-type"))
	d.nameLabel = Label("Save as", ui.Class("kit-file-label"))
	d.name = TextInput("", ui.Class("kit-file-name-field"), ui.Attr("aria-label", "File name"))
	if len(d.types) == 0 {
		setHidden(d.typeLabel, true)
		setHidden(d.typePopUp, true)
	}
	d.node = Window("", UnifiedToolbar(), ui.Class("kit-file-dialog"),
		Toolbar(Row(ui.Class("kit-nav-pair"), d.backButton, d.forwardButton), d.path, d.view,
			ui.Div(ui.Class("kit-expand")), d.search),
		Row(ui.Class("kit-file-main"), d.side, ui.Div(ui.Class("kit-divider")),
			Column(ui.Class("kit-file-pane"), d.table, d.grid, d.note, d.info)),
		Footer(d.typeLabel, d.typePopUp, d.nameLabel, d.name, ui.Div(ui.Class("kit-expand")), d.hint, d.cancel, d.ok))
	d.listen()
}

// listen puts the dialog's reactions on its own elements.
func (d *FileDialog) listen() {
	ui.On(d.backButton, ui.EventClick, d.goBack)
	ui.On(d.forwardButton, ui.EventClick, d.goForward)
	ui.On(d.cancel, ui.EventClick, func(w *world.World, _ ui.Event) { d.cancelled(w) })
	ui.On(d.ok, ui.EventClick, func(w *world.World, _ ui.Event) { d.pick(w) })
	ui.On(d.name, ui.EventSubmit, func(w *world.World, _ ui.Event) { d.pick(w) })
	ui.On(SearchInput(d.search), ui.EventInput, d.refill)
	ui.On(d.side, ui.EventChange, func(w *world.World, _ ui.Event) { d.sideChosen(w) })
	ui.On(d.typePopUp, ui.EventChange, func() {
		d.shown = max(d.doc.Selected(d.typePopUp), 0)
		d.refill()
	})
	for _, item := range d.view.Children() {
		ui.On(item, ui.EventChange, func() { d.showGrid(Chosen(d.view) == 1) })
	}
	table := TableGrid(d.table)
	ui.On(table, ui.EventChange, func() { d.selected(d.doc.Selected(table)) })
	ui.On(table, ui.EventSubmit, func(w *world.World, _ ui.Event) { d.activate(w, d.doc.Selected(table)) })
	ui.On(table, ui.EventSort, func(e ui.Event) {
		d.sortColumn, d.sortDesc = e.Column, e.Descending
		d.fillGrid()
	})
	ui.On(d.grid, ui.EventChange, func() {
		d.doc.SetSelected(table, d.doc.Selected(d.grid))
		d.selected(d.doc.Selected(d.grid))
	})
	ui.On(d.grid, ui.EventSubmit, func(w *world.World, _ ui.Event) { d.activate(w, d.doc.Selected(d.grid)) })
	ui.On(d.node, ui.EventKey, d.key)
	ui.On(d.node, ui.EventClose, func(w *world.World, _ ui.Event) {
		if d.open {
			d.cancelled(w)
		}
	})
}

// listenPath sends a click on one of the path pop-up's folders there, once
// SetFolder has built them.
func (d *FileDialog) listenPath() {
	d.path.Walk(func(n *ui.Node) bool {
		if p, ok := n.Attr("data-path"); ok {
			ui.On(n, ui.EventClick, func() { d.visit(fsPath(p)) })
		}
		return true
	})
}

func viewGlyph(kind string) *ui.Node {
	g := ui.Span(ui.Class("kit-view-glyph", "kit-view-glyph-"+kind), ui.Attr("aria-label", kind))
	n := 3
	if kind == "grid" {
		n = 4
	}
	for range n {
		g.Append(ui.Span())
	}
	return g
}

// Node is the dialog's Window, in the Document's body unless the game moves it.
func (d *FileDialog) Node() *ui.Node { return d.node }

// Table, Thumbnails and SearchField are the dialog's list of files, its grid of them and
// its search field.
func (d *FileDialog) Table() *ui.Node       { return d.table }
func (d *FileDialog) Thumbnails() *ui.Node  { return d.grid }
func (d *FileDialog) SearchField() *ui.Node { return d.search }

// NameField and PathPopUp are the field holding the name to save under and the dialog's
// path pop-up.
func (d *FileDialog) NameField() *ui.Node { return d.name }
func (d *FileDialog) PathPopUp() *ui.Node { return d.path }

// Extensions are the endings of the files the type shown takes, none for any.
func (d *FileDialog) Extensions() []string {
	if d.shown < len(d.types) {
		return slices.Clone(d.types[d.shown].exts)
	}
	return nil
}

// FileType is the name of the type of file shown, "" for none.
func (d *FileDialog) FileType() string {
	if d.shown < len(d.types) {
		return d.types[d.shown].name
	}
	return ""
}

// Places is the folders down the dialog's side, as Places set them; SetPlaces sets them,
// shown from the next Open or Save.
func (d *FileDialog) Places() []string            { return slices.Clone(d.places) }
func (d *FileDialog) SetPlaces(folders ...string) { d.places = slices.Clone(folders) }

// Recent is the files under the side's Recent, as Recent set them; SetRecent sets them,
// shown from the next Open or Save.
func (d *FileDialog) Recent() []string          { return slices.Clone(d.recent) }
func (d *FileDialog) SetRecent(paths ...string) { d.recent = slices.Clone(paths) }

// Describe is the function Describe set, nil without one.
func (d *FileDialog) Describe() func(string) string { return d.describe }

// Preview is the function Preview set, nil without one.
func (d *FileDialog) Preview() func(string) string { return d.preview }

// DateFormat is how the dialog writes a file's date: DateFormat's function, else its own.
func (d *FileDialog) DateFormat() func(time.Time) string { return d.dateFormat }

// RootName is the name of the root of the dialog's FS, "/" unless RootName set one.
func (d *FileDialog) RootName() string { return d.rootName }

// DialogLabel is the name DialogLabel gave the dialog, "" without one.
func (d *FileDialog) DialogLabel() string { return d.label }

// OwnWindow reports whether the dialog shows in a window of its own, and Window is that
// window while the dialog shows in it, nil otherwise. SetOwnWindow changes where it shows
// from the next Open or Save: in the game's Document again when off, even after showing in
// a window of its own. An Open or Save in its own window asked before the dialog is added
// as a system opens that window at its first Update.
func (d *FileDialog) OwnWindow() bool          { return d.own }
func (d *FileDialog) SetOwnWindow(on bool)     { d.own = on }
func (d *FileDialog) Window() *platform.Window { return d.window }

// Kind is how the dialog names the files ending in ext, "" when it names them by their
// extension.
func (d *FileDialog) Kind(ext string) string {
	ext = strings.ToLower(ext)
	if k, ok := d.kinds[ext]; ok {
		return k
	}
	return commonKind(ext)
}

// Folder is the folder listed.
func (d *FileDialog) Folder() string { return d.folder }

// Shown reports whether the dialog shows, between Open or Save and a pick or a cancel.
func (d *FileDialog) Shown() bool { return d.open }

// Saving reports whether the dialog asks where to save.
func (d *FileDialog) Saving() bool { return d.saving }

// Open shows the dialog modal in folder, a path of its FS ("." for its root), to pick a
// file to open.
func (d *FileDialog) Open(folder string) { d.show(false, folder, "") }

// Save shows the dialog modal in folder to pick where to save, name filled in; the
// type's first extension is added to a name without one it takes.
func (d *FileDialog) Save(folder, name string) { d.show(true, folder, name) }

func (d *FileDialog) show(saving bool, folder, name string) {
	switch {
	case d.own && d.world == nil:
		d.later = func() { d.show(saving, folder, name) }
		d.saving, d.open = saving, true
		return
	case d.own:
		d.openWindow(saving)
	default:
		d.goHome()
	}
	d.saving, d.open = saving, true
	ok := openTitle
	if saving {
		ok = "Save"
	}
	d.ok.SetText(ok)
	title := openTitle
	if saving {
		title = "Save as"
	}
	scope := "file-dialog"
	if d.label != "" {
		title, scope = d.label, d.label
	}
	d.node.SetAttr("data-name", scope)
	d.node.SetAttr("aria-label", title)
	setHidden(d.nameLabel, !saving)
	setHidden(d.name, !saving)
	d.doc.SetValue(d.name, name)
	setHidden(d.typeLabel, saving || len(d.types) == 0)
	setHidden(d.typePopUp, saving || len(d.types) == 0)
	d.doc.SetSelected(d.typePopUp, d.shown)
	d.hint.SetText("")
	d.doc.SetValue(SearchInput(d.search), "")
	d.back, d.forward, d.folder = nil, nil, ""
	d.fillSide()
	if !d.enter(folder) {
		d.enter(".")
	}
	first, other := TableGrid(d.table), d.name
	if saving {
		first, other = d.name, TableGrid(d.table)
	}
	first.SetAttr("autofocus", "")
	other.RemoveAttr("autofocus")
	d.doc.ShowModal(d.node)
	d.doc.Focus(first)
}

// openWindow moves the dialog into a window of its own, made the first time and again
// after it closed.
func (d *FileDialog) openWindow(saving bool) {
	if d.window != nil {
		return
	}
	title := openTitle
	if saving {
		title = "Save as"
	}
	style := d.ownStyle
	if len(style) == 0 {
		style = []ui.DocumentOption{Theme()}
	}
	own := ui.NewDocument(style...)
	if p := d.node.Parent(); p != nil {
		p.Remove(d.node)
	}
	own.Body().Append(d.node)
	own.SortBy(TableGrid(d.table), d.compare)
	d.node.AddClass("kit-own")
	d.doc = own
	d.window = platform.NewWindow(platform.Title(title), platform.Size(900, 600), platform.MinSize(640, 420))
	d.window.Add(own)
	d.world.AddEntities(d.window)
}

// goHome moves the dialog back into the game's Document after it showed in a window of
// its own.
func (d *FileDialog) goHome() {
	if d.doc == d.home {
		return
	}
	if p := d.node.Parent(); p != nil {
		p.Remove(d.node)
	}
	d.home.Body().Append(d.node)
	d.node.RemoveClass("kit-own")
	d.doc = d.home
}

// closeWindow removes the dialog's own window, if it shows in one.
func (d *FileDialog) closeWindow(w *world.World) {
	if d.window == nil {
		return
	}
	if w != nil && world.Of(d.window) != nil {
		w.RemoveEntity(d.window)
	}
	d.window = nil
}

// Select selects the file or folder at path in the folder listed, "" for none, without an
// event.
func (d *FileDialog) Select(path string) {
	grid := TableGrid(d.table)
	d.doc.SetSelected(grid, -1)
	for i, r := range TableRows(d.table) {
		if e := d.rows[r]; e != nil && e.path == path {
			d.doc.SetSelected(grid, i)
		}
	}
	d.selected(d.doc.Selected(grid))
}

// Pick does what the dialog's Open or Save button does.
func (d *FileDialog) Pick() { d.pick(d.world) }

// New, Phase, Priority and Update make the dialog a system of the World.
func (d *FileDialog) New(w *world.World) { d.world = w }
func (d *FileDialog) Phase() world.Phase { return world.Update }

// Priority runs the dialog after twod's system, where its Documents draw and emit their
// events, so it answers them the same frame.
func (d *FileDialog) Priority() int { return -100 }

// live reports whether the dialog's node is in its Document and the Document in the
// World.
func (d *FileDialog) live() bool {
	if d.world == nil || world.Of(d.doc) != d.world {
		return false
	}
	for p := d.node; p != nil; p = p.Parent() {
		if p == d.doc.HTML() {
			return true
		}
	}
	return false
}

func (d *FileDialog) Update(f world.Frame) {
	w := f.World
	if later := d.later; later != nil {
		d.later = nil
		later()
	}
	live := d.live()
	if live && d.rows == nil && d.open {
		d.enter(d.folder)
	}
	if !live {
		for e := range d.windows.Read(w) {
			if d.window != nil && e.Window == d.window {
				d.cancelled(w)
				d.closeWindow(w)
			}
		}
		d.entries, d.rows = nil, nil
		return
	}
	for e := range d.windows.Read(w) {
		if d.window != nil && e.Window == d.window {
			d.cancelled(w)
			d.closeWindow(w)
		}
	}
}

// key goes up a folder on Command-Up, and sends typing to the search field:
// a key pressed on any of the dialog's elements, bubbled up to its node.
func (d *FileDialog) key(e ui.Event) {
	field := SearchInput(d.search)
	switch {
	case e.Command && e.Key == input.KeyUp:
		d.visit(path.Dir(d.folder))
	case !e.Command && strings.TrimSpace(e.Text) != "":
		d.doc.SetValue(field, d.doc.Value(field)+e.Text)
		d.doc.Focus(field)
		d.refill()
	}
}

// shownPath is how the path pop-up shows the FS path p: from "/", the FS's root.
func shownPath(p string) string {
	if p == "." {
		return "/"
	}
	return "/" + p
}

// fsPath is the FS path the path pop-up's shown path p stands for.
func fsPath(p string) string {
	p = strings.TrimPrefix(path.Clean(p), "/")
	if p == "" {
		return "."
	}
	return p
}

// folderLabel is how the dialog names the folder p: the root by its
// RootName, any other by its own name.
func (d *FileDialog) folderLabel(p string) string {
	if p == "." {
		return d.rootName
	}
	return path.Base(p)
}

// fillSide lists the places, the FS's root and the recent files down the side.
func (d *FileDialog) fillSide() {
	var places []*ui.Node
	for _, f := range append(slices.Clone(d.places), ".") {
		label := d.folderLabel(f)
		places = append(places, glyphed(SidebarItem(label, ui.Attr("data-path", f)), folderGlyph()))
	}
	SetSection(d.side, "Favourites", places...)
	var recent []*ui.Node
	for _, p := range d.recent {
		recent = append(recent,
			glyphed(SidebarItem(path.Base(p), ui.Attr("data-path", p), ui.Attr("data-recent", "")), fileGlyph()))
	}
	SetSection(d.side, "Recent", recent...)
}

// glyphed puts glyph first in n.
func glyphed(n, glyph *ui.Node) *ui.Node {
	kids := slices.Clone(n.Children())
	for _, k := range kids {
		n.Remove(k)
	}
	n.Append(glyph)
	n.Append(kids...)
	return n
}

func folderGlyph() *ui.Node {
	return ui.Span(ui.Class("kit-folder-glyph"), ui.Span(ui.Class("kit-folder-tab")), ui.Span(ui.Class("kit-folder-body")))
}

func fileGlyph() *ui.Node {
	return ui.Span(ui.Class("kit-file-glyph"), ui.Span(), ui.Span())
}

func (d *FileDialog) sideChosen(w *world.World) {
	items := SidebarItems(d.side)
	i := d.doc.Selected(d.side)
	if i < 0 || i >= len(items) {
		return
	}
	p, _ := items[i].Attr("data-path")
	if _, recent := items[i].Attr("data-recent"); recent {
		d.choose(w, p)
		return
	}
	d.visit(p)
}

// visit lists folder, the one listed now going into the back history.
func (d *FileDialog) visit(folder string) {
	from := d.folder
	if d.enter(folder) && from != "" && from != d.folder {
		d.back, d.forward = append(d.back, from), nil
		d.navButtons()
	}
}

func (d *FileDialog) goBack() {
	if len(d.back) == 0 {
		return
	}
	to, from := d.back[len(d.back)-1], d.folder
	if d.enter(to) {
		d.back, d.forward = d.back[:len(d.back)-1], append(d.forward, from)
		d.navButtons()
	}
}

func (d *FileDialog) goForward() {
	if len(d.forward) == 0 {
		return
	}
	to, from := d.forward[len(d.forward)-1], d.folder
	if d.enter(to) {
		d.forward, d.back = d.forward[:len(d.forward)-1], append(d.back, from)
		d.navButtons()
	}
}

func (d *FileDialog) navButtons() {
	setDisabled(d.backButton, len(d.back) == 0)
	setDisabled(d.forwardButton, len(d.forward) == 0)
}

func setDisabled(n *ui.Node, on bool) {
	if on {
		n.SetAttr("disabled", "")
	} else {
		n.RemoveAttr("disabled")
	}
}

// enter lists folder, reporting whether it could.
func (d *FileDialog) enter(folder string) bool {
	folder = path.Clean(folder)
	files, err := fs.ReadDir(d.fsys, folder)
	if err != nil {
		d.hint.SetText(err.Error())
		return false
	}
	var entries []fileEntry
	for _, f := range files {
		if strings.HasPrefix(f.Name(), ".") {
			continue
		}
		e := fileEntry{path: path.Join(folder, f.Name()), dir: f.IsDir()}
		if info, infoErr := f.Info(); infoErr == nil {
			e.size, e.mod = info.Size(), info.ModTime()
		}
		entries = append(entries, e)
	}
	if folder != d.folder {
		d.doc.SetValue(SearchInput(d.search), "")
	}
	d.folder, d.entries = folder, entries
	SetFolder(d.path, shownPath(folder))
	d.listenPath()
	d.hint.SetText("")
	d.doc.SetSelected(d.side, -1)
	for i, it := range SidebarItems(d.side) {
		if p, _ := it.Attr("data-path"); p == folder {
			if _, recent := it.Attr("data-recent"); !recent {
				d.doc.SetSelected(d.side, i)
			}
		}
	}
	d.refill()
	d.navButtons()
	return true
}

// refill lists the folder's entries matching the search.
func (d *FileDialog) refill() {
	grid := TableGrid(d.table)
	was := ""
	if e := d.entryAt(d.doc.Selected(grid)); e != nil {
		was = e.path
	}
	body := tableBody(grid)
	for _, r := range slices.Clone(body.Children()) {
		body.Remove(r)
	}
	term := strings.ToLower(strings.TrimSpace(d.doc.Value(SearchInput(d.search))))
	d.rows = map[*ui.Node]*fileEntry{}
	dimmed, shown := false, 0
	for i := range d.entries {
		e := &d.entries[i]
		if term != "" && !strings.Contains(strings.ToLower(path.Base(e.path)), term) {
			continue
		}
		r := d.row(e)
		if _, ok := r.Attr("aria-disabled"); ok {
			dimmed = true
		}
		body.Append(r)
		d.rows[r] = e
		shown++
	}
	d.doc.Sort(grid, d.sortColumn, d.sortDesc)
	d.doc.SetSelected(grid, -1)
	for i, r := range TableRows(d.table) {
		_, off := r.Attr("aria-disabled")
		if d.rows[r].path == was || was == "" && term != "" && !off {
			d.doc.SetSelected(grid, i)
			break
		}
	}
	switch {
	case term != "" && shown == 0:
		d.note.SetText(fmt.Sprintf("Nothing in %s matches “%s”.", d.folderLabel(d.folder), strings.TrimSpace(term)))
	case dimmed && d.shown < len(d.types):
		d.note.SetText("Files that aren't " + d.typeName() + " are dimmed and can't be chosen.")
	default:
		d.note.SetText("")
	}
	setHidden(d.note, d.note.Text() == "")
	d.fillGrid()
	d.selected(d.doc.Selected(grid))
}

func setHidden(n *ui.Node, on bool) {
	if on {
		n.SetAttr("hidden", "")
	} else {
		n.RemoveAttr("hidden")
	}
}

func (d *FileDialog) typeName() string {
	if t := d.types[d.shown]; t.name != "" {
		return t.name
	}
	return strings.Join(d.types[d.shown].exts, " or ") + " files"
}

func (d *FileDialog) accepts(name string) bool {
	return d.shown >= len(d.types) || d.types[d.shown].accepts(name)
}

func (d *FileDialog) row(e *fileEntry) *ui.Node {
	size, glyph := sizeText(e.size), fileGlyph()
	if e.dir {
		size, glyph = "—", folderGlyph()
	}
	r := ui.Tr(
		ui.Td(ui.Class("kit-file-name"), glyph, ui.Span(ui.Text(path.Base(e.path)))),
		ui.Td(ui.Text(d.date(e.mod))),
		ui.Td(ui.Text(size)),
		ui.Td(ui.Text(d.fileKind(*e))),
	)
	if !e.dir && !d.accepts(e.path) {
		SetDimmed(r, true)
	}
	return r
}

func (d *FileDialog) date(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return d.dateFormat(t)
}

// compare sorts the rows: folders first, then by column, reversed when descending.
func (d *FileDialog) compare(a, b *ui.Node, column int, descending bool) int {
	x, y := d.rows[a], d.rows[b]
	if x == nil || y == nil {
		return 0
	}
	if x.dir != y.dir {
		if x.dir {
			return -1
		}
		return 1
	}
	var c int
	switch column {
	case 1:
		c = x.mod.Compare(y.mod)
	case 2:
		c = cmp.Compare(x.size, y.size)
	case 3:
		c = cmp.Compare(strings.ToLower(d.fileKind(*x)), strings.ToLower(d.fileKind(*y)))
	}
	if c == 0 {
		c = cmp.Compare(strings.ToLower(path.Base(x.path)), strings.ToLower(path.Base(y.path)))
	}
	if descending {
		return -c
	}
	return c
}

func (d *FileDialog) entryAt(i int) *fileEntry {
	rows := TableRows(d.table)
	if i < 0 || i >= len(rows) {
		return nil
	}
	return d.rows[rows[i]]
}

// fillGrid shows the table's rows, in its order, as thumbnails.
func (d *FileDialog) fillGrid() {
	for _, c := range slices.Clone(d.grid.Children()) {
		d.grid.Remove(c)
	}
	for _, r := range TableRows(d.table) {
		e := d.rows[r]
		if e == nil {
			continue
		}
		src := ""
		if !e.dir && d.preview != nil {
			src = d.preview(e.path)
		}
		cell := Thumbnail(path.Base(e.path), src)
		if src == "" {
			pic := cell.Children()[0]
			pic.AddClass("kit-glyph-picture")
			if e.dir {
				pic.Append(folderGlyph())
			} else {
				pic.Append(fileGlyph())
			}
		}
		if _, off := r.Attr("aria-disabled"); off {
			SetDimmed(cell, true)
		}
		d.grid.Append(cell)
	}
	d.doc.SetSelected(d.grid, d.doc.Selected(TableGrid(d.table)))
}

func (d *FileDialog) showGrid(on bool) {
	setHidden(d.grid, !on)
	setHidden(d.table, on)
	if on {
		d.doc.Focus(d.grid)
	} else {
		d.doc.Focus(TableGrid(d.table))
	}
}

// selected shows the file at i in the info pane under the list.
func (d *FileDialog) selected(i int) {
	if d.doc.Selected(d.grid) != i {
		d.doc.SetSelected(d.grid, i)
	}
	for _, c := range slices.Clone(d.info.Children()) {
		d.info.Remove(c)
	}
	e := d.entryAt(i)
	setHidden(d.info, e == nil)
	if e == nil {
		return
	}
	if d.saving && !e.dir {
		d.doc.SetValue(d.name, path.Base(e.path))
	}
	thumb := ui.Div(ui.Class("kit-file-thumb"))
	switch src := ""; {
	case e.dir:
		thumb.Append(folderGlyph())
	case d.preview != nil && func() bool { src = d.preview(e.path); return src != "" }():
		thumb.Append(Image(src, ui.Class("kit-keep-aspect-centered")))
	default:
		thumb.Append(fileGlyph())
	}
	d.info.Append(thumb, ui.Div(ui.Class("kit-file-texts"),
		ui.Div(ui.Class("kit-file-title"), ui.Text(path.Base(e.path))),
		ui.Div(ui.Class("kit-file-details"), ui.Text(d.details(e))),
		ui.Div(ui.Class("kit-file-path"), ui.Text(e.path))))
}

func (d *FileDialog) details(e *fileEntry) string {
	if d.describe != nil {
		if s := d.describe(e.path); s != "" {
			return s
		}
	}
	if e.dir {
		return "Folder · modified " + d.date(e.mod)
	}
	return d.fileKind(*e) + " · " + sizeText(e.size) + " · modified " + d.date(e.mod)
}

// activate goes into a folder or picks a file.
func (d *FileDialog) activate(w *world.World, i int) {
	e := d.entryAt(i)
	switch {
	case e == nil:
	case e.dir:
		d.visit(e.path)
	case d.saving:
		d.doc.SetValue(d.name, path.Base(e.path))
	case d.accepts(e.path):
		d.choose(w, e.path)
	}
}

// pick is the Open or Save button: the folder or file selected, or the name typed to save
// under, the type's first extension added, asking first before replacing a file.
func (d *FileDialog) pick(w *world.World) {
	if d.saving {
		n := strings.TrimSpace(d.doc.Value(d.name))
		if n == "" {
			d.hint.SetText("Type a name to save under.")
			return
		}
		if exts := d.Extensions(); len(exts) > 0 && !d.accepts(n) {
			n += exts[0]
		}
		p := path.Join(d.folder, n)
		if info, err := fs.Stat(d.fsys, p); err == nil && !info.IsDir() {
			d.confirm(p)
			return
		}
		d.choose(w, p)
		return
	}
	e := d.entryAt(d.doc.Selected(TableGrid(d.table)))
	switch {
	case e == nil || !e.dir && !d.accepts(e.path):
		d.hint.SetText("Choose a file in the list.")
	case e.dir:
		d.visit(e.path)
	default:
		d.choose(w, e.path)
	}
}

func (d *FileDialog) choose(w *world.World, p string) {
	if !d.open {
		return
	}
	d.open = false
	if w != nil {
		world.Emit(w, FileChosen{Dialog: d, Path: p, Saving: d.saving})
	}
	if d.onChoose != nil {
		d.onChoose(w, FileChosen{Dialog: d, Path: p, Saving: d.saving})
	}
	d.doc.Close(d.node, p)
	d.closeWindow(w)
}

// confirm asks before saving over the file at p.
func (d *FileDialog) confirm(p string) {
	d.replacing = p
	d.alert = Alert("Replace “"+path.Base(p)+"”?",
		"A file with that name already exists in "+d.folderLabel(path.Dir(p))+". Replacing it overwrites what it holds.",
		Button("Keep"), Button("Replace", ui.Class("kit-destructive")))
	ui.On(d.alert, ui.EventClose, func(w *world.World, e ui.Event) {
		d.node.Remove(d.alert)
		d.alert = nil
		if e.Value == "Replace" {
			d.choose(w, d.replacing)
		}
	})
	d.node.Append(d.alert)
	d.doc.ShowModal(d.alert)
}

// cancelled closes the dialog without a file: Cancel, Escape or its close light.
func (d *FileDialog) cancelled(w *world.World) {
	if !d.open {
		return
	}
	d.open, d.later = false, nil
	if w != nil {
		world.Emit(w, DialogCancelled{Dialog: d})
	}
	if d.onCancel != nil {
		d.onCancel(w)
	}
	d.doc.Close(d.node, "")
	d.closeWindow(w)
}

// openTitle is the dialog's title and its button's text when it opens.
const openTitle = "Open"

// commonKind names the common formats by extension; a game names its own
// with Kind.
func commonKind(ext string) string {
	switch ext {
	case ".png":
		return "PNG image"
	case ".jpg", ".jpeg":
		return "JPEG image"
	case ".gif":
		return "GIF image"
	case ".webp":
		return "WebP image"
	case ".wav":
		return "WAV sound"
	case ".ogg":
		return "Ogg sound"
	case ".mp3":
		return "MP3 sound"
	case ".flac":
		return "FLAC sound"
	case ".txt":
		return "Text"
	case ".md":
		return "Markdown"
	case ".json":
		return "JSON"
	case ".yaml", ".yml":
		return "YAML"
	case ".csv":
		return "CSV"
	case ".ttf", ".otf":
		return "Font"
	case ".pdf":
		return "PDF document"
	case ".zip":
		return "ZIP archive"
	}
	return ""
}

// fileKind is how the list names e's kind: Folder, the game's Kind, a common format, or
// its extension's "XYZ file".
func (d *FileDialog) fileKind(e fileEntry) string {
	if e.dir {
		return "Folder"
	}
	ext := strings.ToLower(path.Ext(e.path))
	if k, ok := d.kinds[ext]; ok {
		return k
	}
	if k := commonKind(ext); k != "" {
		return k
	}
	if ext == "" {
		return "File"
	}
	return strings.ToUpper(ext[1:]) + " file"
}

func sizeText(n int64) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d bytes", n)
	case n < 1000*1000:
		return fmt.Sprintf("%d KB", (n+500)/1000)
	case n < 1000*1000*1000:
		return fmt.Sprintf("%.1f MB", float64(n)/1e6)
	}
	return fmt.Sprintf("%.1f GB", float64(n)/1e9)
}

// dateText is the old ember/ui's date: today's and yesterday's by their time, older ones by day,
// in t's zone.
func dateText(t time.Time) string {
	now := time.Now().In(t.Location())
	y, m, day := now.Date()
	today := time.Date(y, m, day, 0, 0, 0, 0, now.Location())
	switch {
	case !t.Before(today):
		return "Today, " + t.Format("15:04")
	case !t.Before(today.AddDate(0, 0, -1)):
		return "Yesterday, " + t.Format("15:04")
	}
	return t.Format("2 Jan 2006")
}
