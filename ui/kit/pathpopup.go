package kit

import (
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/merlindorin/ember/ui"
)

// PathPopUp is a button of class "kit-path-pop-up"
// showing a folder glyph, the folder's name in bold and the two folders
// above it, nearest first and dimmer, that opens a menu of the folder and
// every folder above it up to the root, the folder shown checked. A
// chosen folder emits ui.Clicked for its item: FolderOf reads its path,
// and SetFolder shows it. RootName names the root, as a FileDialog's
// file system's. Automation names it "path" (its data-name) and labels it
// with the folders it shows.
func PathPopUp(path string, args ...ui.Arg) *ui.Node {
	p := ui.Button(ui.Class("kit-button", "kit-menu-button", "kit-path-pop-up"), ui.Attr("aria-haspopup", "menu"),
		ui.Attr("data-name", "path"))
	for _, a := range args {
		a.Apply(p)
	}
	SetFolder(p, path)
	return p
}

// SetFolder shows path in the path pop-up p, its folders listed up to the
// root.
func SetFolder(p *ui.Node, path string) {
	for _, c := range slices.Clone(p.Children()) {
		p.Remove(c)
	}
	path = filepath.Clean(path)
	root, named := p.Attr("data-root-name")
	var folders, names []string
	for at := path; ; at = filepath.Dir(at) {
		name := folderName(at)
		if named && filepath.Dir(at) == at {
			name = root
		}
		folders, names = append(folders, at), append(names, name)
		if filepath.Dir(at) == at {
			break
		}
	}
	rest := ""
	if len(names) > 1 {
		rest = " — " + strings.Join(names[1:min(3, len(names))], " › ")
	}
	menu := Menu()
	for i, f := range folders {
		checked, tick := strconv.FormatBool(i == 0), ""
		if i == 0 {
			tick = "✓"
		}
		menu.Append(ui.Div(ui.Class("kit-menu-item"), ui.Attr("role", "menuitemradio"), ui.Attr("aria-checked", checked),
			ui.Attr("aria-label", names[i]), ui.Attr("data-path", f),
			ui.Span(ui.Class("kit-menu-check"), ui.Text(tick)), ui.Span(ui.Class("kit-menu-label"), ui.Text(names[i]))))
	}
	p.SetAttr("data-path", path)
	p.Append(
		ui.Span(ui.Class("kit-folder-glyph"), ui.Span(ui.Class("kit-folder-tab")), ui.Span(ui.Class("kit-folder-body"))),
		ui.Span(ui.Class("kit-path-label"), ui.Strong(ui.Class("kit-path-name"), ui.Text(names[0])),
			ui.Span(ui.Class("kit-path-rest"), ui.Text(rest))),
		ui.Span(ui.Class("kit-pop-up-chevron"), ui.Attr("aria-hidden", "true"), ui.Text("▾")),
		menu,
	)
}

// Folder is the folder the path pop-up p shows.
func Folder(p *ui.Node) string {
	v, _ := p.Attr("data-path")
	return v
}

// FolderOf is the folder a path pop-up's menu item stands for, false for
// any other node.
func FolderOf(item *ui.Node) (string, bool) {
	m := item.Parent()
	if m == nil || m.Parent() == nil || !m.Parent().HasClass("kit-path-pop-up") {
		return "", false
	}
	return item.Attr("data-path")
}

// folderName is how a folder reads in a path pop-up: its last name, or
// the root as it is.
func folderName(path string) string {
	if b := filepath.Base(path); b != string(filepath.Separator) && b != "." && b != "" {
		return b
	}
	return path
}

// RootName names the root of a PathPopUp, or of a FileDialog's FS in its
// path pop-up and down its side; "/" by default.
func RootName(name string) RootNameArg { return RootNameArg(name) }

// RootNameArg is a root's name, both a PathPopUp arg and a
// FileDialogOption.
type RootNameArg string

func (n RootNameArg) Apply(p *ui.Node)              { p.SetAttr("data-root-name", string(n)) }
func (n RootNameArg) applyFileDialog(d *FileDialog) { d.rootName = string(n) }
