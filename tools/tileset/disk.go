package main

import (
	"bytes"
	"image/png"
	"io/fs"
	"os"
	"path"
	"runtime"
	"slices"
	"strings"
	"testing/fstest"
	"time"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/merlindorin/ember/world"
)

// diskDialog is a kit.FileDialog over the whole disk: kit's paths are slash
// paths from its root, the tool's the OS's (fsPath, osPath).
type diskDialog struct {
	*kit.FileDialog
}

const windows = runtime.GOOS == "windows"

// fsPath is the dialog's slash path of the OS path p: from "/" ("." for
// it), or on Windows from the drives ("C:" for C:\).
func fsPath(p string) string { return fsPathFor(windows, p) }

// osPath is the OS path of the dialog's slash path p.
func osPath(p string) string { return osPathFor(windows, p) }

// fsPathFor is fsPath on Windows or not, so both test anywhere.
func fsPathFor(win bool, p string) string {
	if !win {
		rel := strings.Trim(p, "/")
		if rel == "" {
			return "."
		}
		return path.Clean(rel)
	}
	rel := strings.Trim(strings.ReplaceAll(p, `\`, "/"), "/")
	if rel == "" {
		return "."
	}
	return path.Clean(rel)
}

// osPathFor is osPath on Windows or not; the root of drives has none.
func osPathFor(win bool, p string) string {
	switch {
	case !win && (p == "." || p == ""):
		return "/"
	case !win:
		return "/" + path.Clean(p)
	case p == "." || p == "":
		return ""
	}
	p = path.Clean(p)
	if !strings.Contains(p, "/") {
		return p + `\`
	}
	return strings.ReplaceAll(p, "/", `\`)
}

// diskFS is the disk as the dialog lists it: "/" on Unix, the drives on
// Windows.
func diskFS() fs.FS {
	if !windows {
		return os.DirFS("/")
	}
	return volumes{list: drives, open: func(name string) fs.FS { return os.DirFS(name + `\`) }}
}

// drives are Windows's drives, by name: "C:", "D:" and on.
func drives() []string {
	var out []string
	for l := 'A'; l <= 'Z'; l++ {
		if info, err := os.Stat(string(l) + `:\`); err == nil && info.IsDir() {
			out = append(out, string(l)+":")
		}
	}
	return out
}

// opOpen is an open's name in its errors.
const opOpen = "open"

// volumes is a file system of several, named: its root lists them, and
// "C:/Users/heron" is Users/heron of the one named "C:".
type volumes struct {
	list func() []string
	open func(name string) fs.FS
}

// on is the volume name p is on and p's path on it.
func (v volumes) on(op, p string) (fs.FS, string, error) {
	if !fs.ValidPath(p) {
		return nil, "", &fs.PathError{Op: op, Path: p, Err: fs.ErrInvalid}
	}
	name, rest, _ := strings.Cut(p, "/")
	if !slices.Contains(v.list(), name) {
		return nil, "", &fs.PathError{Op: op, Path: p, Err: fs.ErrNotExist}
	}
	if rest == "" {
		rest = "."
	}
	return v.open(name), rest, nil
}

func (v volumes) Open(p string) (fs.File, error) {
	if p == "." {
		return nil, &fs.PathError{Op: opOpen, Path: p, Err: fs.ErrInvalid}
	}
	f, rest, err := v.on(opOpen, p)
	if err != nil {
		return nil, err
	}
	return f.Open(rest)
}

func (v volumes) ReadDir(p string) ([]fs.DirEntry, error) {
	if p != "." {
		f, rest, err := v.on("readdir", p)
		if err != nil {
			return nil, err
		}
		return fs.ReadDir(f, rest)
	}
	var out []fs.DirEntry
	for _, name := range v.list() {
		if info, err := fs.Stat(v.open(name), "."); err == nil {
			out = append(out, fs.FileInfoToDirEntry(volumeInfo{info, name}))
		}
	}
	return out, nil
}

func (v volumes) Stat(p string) (fs.FileInfo, error) {
	if p == "." {
		return volumeInfo{name: "."}, nil
	}
	f, rest, err := v.on("stat", p)
	if err != nil {
		return nil, err
	}
	info, err := fs.Stat(f, rest)
	if err != nil || rest != "." {
		return info, err
	}
	return volumeInfo{info, path.Base(p)}, nil
}

// volumeInfo is a volume's root, or the root of them all, as a folder
// named name.
type volumeInfo struct {
	fs.FileInfo
	name string
}

func (i volumeInfo) Name() string { return i.name }
func (i volumeInfo) IsDir() bool  { return true }
func (i volumeInfo) Mode() fs.FileMode {
	return fs.ModeDir | 0o555
}
func (i volumeInfo) Size() int64 { return 0 }
func (i volumeInfo) ModTime() time.Time {
	if i.FileInfo == nil {
		return time.Time{}
	}
	return i.FileInfo.ModTime()
}
func (i volumeInfo) Sys() any { return nil }

func newDiskDialog(doc *ui.Document, opts ...kit.FileDialogOption) *diskDialog {
	name := "/"
	if windows {
		name = "This PC"
	}
	return &diskDialog{FileDialog: kit.NewFileDialog(doc, diskFS(), append(opts, kit.RootName(name))...)}
}

func (d *diskDialog) Open(folder string)       { d.FileDialog.Open(fsPath(folder)) }
func (d *diskDialog) Save(folder, name string) { d.FileDialog.Save(fsPath(folder), name) }
func (d *diskDialog) Folder() string           { return osPath(d.FileDialog.Folder()) }
func (d *diskDialog) Hidden() bool             { return !d.Shown() }

func (d *diskDialog) SetPlaces(folders ...string) { d.FileDialog.SetPlaces(d.fsPaths(folders)...) }
func (d *diskDialog) SetRecent(paths ...string)   { d.FileDialog.SetRecent(d.fsPaths(paths)...) }

func (d *diskDialog) fsPaths(ps []string) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = fsPath(p)
	}
	return out
}

// chosen is e's path on the disk, when e is this dialog's.
func (d *diskDialog) chosen(e kit.FileChosen) (string, bool) {
	if e.Dialog != d.FileDialog {
		return "", false
	}
	return osPath(e.Path), true
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
// "preview/<the dialog's slash path>.png", and an empty stylesheet that brings it in.
type previews struct{}

const previewSheet = "previews.css"

func (p previews) Open(name string) (fs.File, error) {
	if name == previewSheet {
		return fstest.MapFS{name: {Data: nil}}.Open(name)
	}
	rel, ok := strings.CutPrefix(name, "preview/")
	if !ok || !strings.HasSuffix(rel, ".png") {
		return nil, &fs.PathError{Op: opOpen, Path: name, Err: fs.ErrNotExist}
	}
	img := tilesetImage(osPath(strings.TrimSuffix(rel, ".png")))
	if img == nil {
		return nil, &fs.PathError{Op: opOpen, Path: name, Err: fs.ErrNotExist}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return nil, err
	}
	return fstest.MapFS{name: {Data: b.Bytes(), ModTime: time.Now()}}.Open(name)
}

// previewSrc is the image src of the tileset at the slash path p's preview.
func previewSrc(p string) string { return "preview/" + path.Clean(p) + ".png" }
