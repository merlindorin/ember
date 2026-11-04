package main

import (
	"log"
	"testing/fstest"
	"time"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/merlindorin/ember/world"
)

// thumbnails is a level picker: four levels, the second selected, the locked one dimmed.
func thumbnails(g *gallery) *ui.Node {
	d := g.doc
	locked := kit.Thumbnail("Lava keep", "")
	kit.SetDimmed(locked, true)
	levels := kit.Thumbnails(kit.ThumbnailSize(96, 92),
		kit.Thumbnail("Crypt", ""), kit.Thumbnail("Sanctuary", ""), kit.Thumbnail("Ice caves", ""), locked)
	d.SetSelected(levels, 1)
	return levels
}

// statusBar is a tileset editor's footer: what is open, the zoom, and the dot of unsaved changes.
func statusBar(*gallery) *ui.Node {
	bar := kit.StatusBar("walls.tsj · 41 tiles · 3 shapes", kit.Label("100%", kit.Caps()))
	kit.SetDot(bar, true)
	return bar
}

// menu is a tile's actions behind a menu button: shortcuts, a submenu, a separator, a disabled item.
func menu(*gallery) *ui.Node {
	return kit.Row(kit.MenuButton("Tile actions", kit.Menu(
		kit.MenuItem("Duplicate", kit.Shortcut("Cmd+D")),
		kit.MenuItem("Rename…", kit.Shortcut("F2")),
		kit.SubMenu("Move to", kit.MenuItem("Walls"), kit.MenuItem("Floors"), kit.MenuItem("Props")),
		kit.Separator(),
		kit.MenuItem("Paste style", ui.Attr("aria-disabled", "true")),
		kit.MenuItem("Delete", kit.Shortcut("Del")),
	)))
}

// menuBar is a tileset editor's menus.
func menuBar(*gallery) *ui.Node {
	return kit.MenuBar(
		kit.MenuBarItem("File", kit.Menu(
			kit.MenuItem("New tileset", kit.Shortcut("Cmd+N")),
			kit.MenuItem("Open…", kit.Shortcut("Cmd+O")),
			kit.Separator(),
			kit.MenuItem("Save", kit.Shortcut("Cmd+S")),
			kit.MenuItem("Quit", kit.Shortcut("Cmd+Q")),
		)),
		kit.MenuBarItem(
			"Edit",
			kit.Menu(kit.MenuItem("Undo", kit.Shortcut("Cmd+Z")), kit.MenuItem("Redo", kit.Shortcut("Cmd+Shift+Z"))),
		),
		kit.MenuBarItem("View", kit.Menu(kit.MenuItem("Grid"), kit.MenuItem("Collision shapes"))),
	)
}

// window is a tile inspector over the page, resizable; its button shows it again once closed.
func window(g *gallery) *ui.Node {
	d := g.doc
	inspector := kit.Window("Inspector", kit.Resizable(),
		kit.Toolbar(kit.Button("Save"), kit.Button("Revert")),
		kit.Label("Tile 12 of walls.tsj"), kit.Label("solid, occluder group walls"),
		kit.Footer(kit.Label("2 changes", kit.Expand()), kit.Button("Apply", kit.Primary())))
	show := kit.Button("Inspector", ui.OnClick(func() { d.Show(inspector) }))
	page := kit.Column(show, inspector)
	d.Show(inspector)
	return page
}

// alert asks to save before closing a tileset; its button asks again once answered.
func alert(g *gallery) *ui.Node {
	d := g.doc
	ask := kit.Alert("Save the tileset?", "Your changes to walls.tsj will be lost if you close it without saving.",
		kit.Button("Don't save", kit.Destructive()),
		kit.Button("Cancel"),
		kit.Button("Save", kit.Primary()),
	)
	again := kit.Button("Close walls.tsj", ui.OnClick(func() { d.ShowModal(ask) }))
	page := kit.Column(again, ask)
	d.ShowModal(ask)
	return page
}

// fileDialog opens a tileset of a game's project in a window of its own; its button
// opens it again, Save as asks where to save.
func fileDialog(g *gallery) *ui.Node {
	d := g.doc
	at := time.Date(2026, 11, 2, 9, 30, 0, 0, time.Local)
	file := func(size int) *fstest.MapFile { return &fstest.MapFile{Data: make([]byte, size), ModTime: at} }
	project := fstest.MapFS{
		"tilesets/walls.tsj":     file(48200),
		"tilesets/floors.tsj":    file(31000),
		"tilesets/props.tsj":     file(12500),
		"tilesets/notes.txt":     file(420),
		"tilesets/old/crypt.tsj": file(29000),
		"maps/crypt.tmj":         file(210000),
		"art/walls.png":          file(96000),
	}
	dialog := kit.NewFileDialog(d, project,
		kit.FileType("Tilesets", ".tsj"),
		kit.Kind("Tileset", ".tsj"),
		kit.RootName("Project"),
		kit.Places("tilesets", "maps", "art"),
		kit.Recent("tilesets/walls.tsj", "maps/crypt.tmj"),
		kit.OwnWindow(),
		kit.OnChoose(func(e kit.FileChosen) { log.Printf("file chosen %q, saving %v", e.Path, e.Saving) }),
		kit.OnCancel(func() { log.Print("file dialog cancelled") }),
	)
	open := kit.Button("Open tileset…", kit.Primary(), ui.OnClick(func() { dialog.Open("tilesets") }))
	save := kit.Button("Save as…", ui.OnClick(func() { dialog.Save("tilesets", "walls copy") }))
	g.systems = append(g.systems, dialog, &Dialogs{open: func() { dialog.Open("tilesets") }})
	return kit.Row(kit.Gap(12), open, save)
}

// Dialogs opens the file dialog on the first frame, once it is a system of the World.
type Dialogs struct{ open func() }

func (s *Dialogs) Update(world.Frame) {
	if s.open != nil {
		s.open()
		s.open = nil
	}
}
