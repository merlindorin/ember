package main

import (
	"bytes"
	"image/png"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing/fstest"
	"time"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/merlindorin/ember/world"
)

// diskDialog is a kit.FileDialog over the disk: kit's paths are slash paths under the
// volume root, the tool's are the OS's.
type diskDialog struct {
	*kit.FileDialog
	root string
}

// volumeRoot is the root of the volume holding dir: "/", or "C:\" on Windows.
func volumeRoot(dir string) string {
	return filepath.VolumeName(dir) + string(filepath.Separator)
}

// fsPath is the slash path under root of the OS path p, "." for root itself.
func fsPath(root, p string) string { return fsPathIn(root, filepath.Separator, p) }

// osPath is the OS path of the slash path p under root.
func osPath(root, p string) string { return osPathIn(root, filepath.Separator, p) }

// fsPathIn is fsPath for an OS whose separator is sep, so both kinds of disk test anywhere.
func fsPathIn(root string, sep byte, p string) string {
	rel, ok := strings.CutPrefix(p, root)
	if !ok {
		return "."
	}
	rel = strings.Trim(strings.ReplaceAll(rel, string(sep), "/"), "/")
	if rel == "" {
		return "."
	}
	return path.Clean(rel)
}

// osPathIn is osPath for an OS whose separator is sep.
func osPathIn(root string, sep byte, p string) string {
	if p == "." || p == "" {
		return root
	}
	return root + strings.ReplaceAll(path.Clean(p), "/", string(sep))
}

func newDiskDialog(doc *ui.Document, root string, opts ...kit.FileDialogOption) *diskDialog {
	name := strings.TrimSuffix(root, string(filepath.Separator))
	if name == "" {
		name = string(filepath.Separator)
	}
	return &diskDialog{
		FileDialog: kit.NewFileDialog(doc, os.DirFS(root), append(opts, kit.RootName(name))...),
		root:       root,
	}
}

func (d *diskDialog) Open(folder string)       { d.FileDialog.Open(fsPath(d.root, folder)) }
func (d *diskDialog) Save(folder, name string) { d.FileDialog.Save(fsPath(d.root, folder), name) }
func (d *diskDialog) Folder() string           { return osPath(d.root, d.FileDialog.Folder()) }
func (d *diskDialog) Hidden() bool             { return !d.Shown() }

func (d *diskDialog) SetPlaces(folders ...string) { d.FileDialog.SetPlaces(d.fsPaths(folders)...) }
func (d *diskDialog) SetRecent(paths ...string)   { d.FileDialog.SetRecent(d.fsPaths(paths)...) }

func (d *diskDialog) fsPaths(ps []string) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = fsPath(d.root, p)
	}
	return out
}

// chosen is e's path on the disk, when e is this dialog's.
func (d *diskDialog) chosen(e kit.FileChosen) (string, bool) {
	if e.Dialog != d.FileDialog {
		return "", false
	}
	return osPath(d.root, e.Path), true
}

// systems are the dialogs as the World's systems.
func systems(ds ...*diskDialog) []world.System {
	out := make([]world.System, len(ds))
	for i, d := range ds {
		out[i] = d.FileDialog
	}
	return out
}

// previews serves a dialog's Document the tilesets' pictures as PNGs, at
// "preview/<slash path under root>.png", and an empty stylesheet that brings it in.
type previews struct{ root string }

const previewSheet = "previews.css"

func (p previews) Open(name string) (fs.File, error) {
	if name == previewSheet {
		return fstest.MapFS{name: {Data: nil}}.Open(name)
	}
	rel, ok := strings.CutPrefix(name, "preview/")
	if !ok || !strings.HasSuffix(rel, ".png") {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	img := tilesetImage(osPath(p.root, strings.TrimSuffix(rel, ".png")))
	if img == nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return nil, err
	}
	return fstest.MapFS{name: {Data: b.Bytes(), ModTime: time.Now()}}.Open(name)
}

// previewSrc is the image src of the tileset at the slash path p's preview.
func previewSrc(p string) string { return "preview/" + path.Clean(p) + ".png" }
