package kit

import (
	"runtime"
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui"
)

// Menu is a div of class "kit-menu" and role menu holding
// MenuItems, SubMenus and Separators. ui gives it WAI-ARIA's menu
// behaviour: it opens from its MenuBarItem, SubMenu or MenuButton (or by
// its open attribute), the arrows, Home, End and typing move in it,
// Escape closes it, and choosing an item emits ui.Clicked for it and
// closes the menus. A menu holding a CheckMenuItem or RadioMenuItem lines
// its other items' text up with theirs, past the check.
func Menu(items ...*ui.Node) *ui.Node {
	m := ui.Div(ui.Class("kit-menu"), ui.Attr("role", "menu"))
	m.Append(items...)
	for _, it := range items {
		if r, _ := it.Attr("role"); r == "menuitemcheckbox" || r == "menuitemradio" {
			m.AddClass("kit-checks")
			break
		}
	}
	return m
}

// MenuItem is a div of class "kit-menu-item" and role
// menuitem showing text, and a Shortcut if given;
// ui.Attr("aria-disabled", "true") greys it out and skips it.
func MenuItem(text string, args ...ui.Arg) *ui.Node {
	return ui.Div(append([]ui.Arg{ui.Class("kit-menu-item"), ui.Attr("role", "menuitem"), ui.Attr("aria-label", text),
		ui.Span(ui.Class("kit-menu-label"), ui.Text(text))}, args...)...)
}

// CheckMenuItem is a MenuItem of role menuitemcheckbox, a check before its text while on:
// choosing it turns it, then emits ui.Clicked; Document.Checked reads it.
func CheckMenuItem(text string, on bool, args ...ui.Arg) *ui.Node {
	return checkable("menuitemcheckbox", text, on, args)
}

// RadioMenuItem is a MenuItem of role menuitemradio: choosing it checks it and unchecks the
// other radio items of its menu, or of its ui.Attr("role", "group") inside the menu.
func RadioMenuItem(text string, on bool, args ...ui.Arg) *ui.Node {
	return checkable("menuitemradio", text, on, args)
}

func checkable(role, text string, on bool, args []ui.Arg) *ui.Node {
	return ui.Div(append([]ui.Arg{ui.Class("kit-menu-item"), ui.Attr("role", role),
		ui.Attr("aria-checked", strconv.FormatBool(on)), ui.Attr("aria-label", text),
		ui.Span(ui.Class("kit-menu-check"), ui.Text("✓")), ui.Span(ui.Class("kit-menu-label"), ui.Text(text))}, args...)...)
}

// Shortcut is a menu item's shortcut, such as "Cmd+S" or "Cmd+Shift+Z" or "Backspace", Cmd
// being Command on macOS and Control elsewhere: shown at its right in the muted colour (Cmd
// reading Ctrl off macOS) and set as its aria-keyshortcuts, which chooses the item when
// pressed while its Document is drawn and not under another's modal (see ui's menus).
func Shortcut(keys string) ui.Arg { return shortcut(keys) }

type shortcut string

func (s shortcut) Apply(n *ui.Node) {
	n.SetAttr("aria-keyshortcuts", string(s))
	label := string(s)
	if runtime.GOOS != "darwin" {
		label = strings.ReplaceAll(label, "Cmd", "Ctrl")
	}
	n.Append(ui.Span(ui.Class("kit-shortcut"), ui.Text(label)))
}

// SubMenu is a menu item opening a menu of items beside it, an arrow at
// its right.
func SubMenu(text string, items ...*ui.Node) *ui.Node {
	return ui.Div(ui.Class("kit-menu-item", "kit-submenu"), ui.Attr("role", "menuitem"), ui.Attr("aria-haspopup", "menu"),
		ui.Attr("aria-label", text), ui.Span(ui.Class("kit-menu-label"), ui.Text(text)),
		ui.Span(ui.Class("kit-submenu-arrow"), ui.Text("▸")), Menu(items...))
}

// MenuBar is a div of class "kit-menu-bar" and role
// menubar holding MenuBarItems, one Tab stop, the arrows moving along it
// and between its open menus, a hovered item switching the open one.
func MenuBar(items ...*ui.Node) *ui.Node {
	b := ui.Div(ui.Class("kit-menu-bar"), ui.Attr("role", "menubar"))
	b.Append(items...)
	return b
}

// MenuBarItem is a menu bar's title, such as File, opening menu under it.
func MenuBarItem(title string, menu *ui.Node) *ui.Node {
	return ui.Span(ui.Class("kit-menu-bar-item"), ui.Attr("role", "menuitem"), ui.Attr("aria-haspopup", "menu"),
		ui.Text(title), menu)
}

// MenuButton is a kit Button opening menu under it, as a pop-up menu.
func MenuButton(text string, menu *ui.Node, args ...ui.Arg) *ui.Node {
	return Button(text, append([]ui.Arg{ui.Class("kit-menu-button"), ui.Attr("aria-haspopup", "menu"), menu}, args...)...)
}
