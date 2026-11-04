package kit

import (
	"slices"

	"github.com/merlindorin/ember/ui"
)

// Window is a dialog of class "kit-window" drawn in the game window, a title
// bar holding the close, minimise and zoom lights and title, then a toolbar, a body and a
// footer. Show it with Document.Show, or ShowModal for the old ember/ui's Modal. Nodes among args
// go in the body, except a Toolbar above it and a Footer under it; the other args apply to
// the dialog, such as Compact or Resizable. ui gives it its behaviour: the lights close
// it (ui.Closed), fold it to its bar and make it fill the window, its bar drags it. Its
// title is its aria-label, so automation names it by its title; an aria-label or data-name
// among args names it otherwise.
func Window(title string, args ...ui.Arg) *ui.Node {
	body := Column(ui.Class("kit-body"))
	var toolbar, footer *ui.Node
	var own []ui.Arg
	unified := false
	for _, a := range args {
		n, ok := a.(*ui.Node)
		switch {
		case a == unifiedToolbar{}:
			unified = true
		case !ok:
			own = append(own, a)
		case n.HasClass("kit-toolbar"):
			toolbar = n
		case n.HasClass("kit-footer"):
			footer = n
		default:
			body.Append(n)
		}
	}
	w := ui.Dialog(ui.Class("kit-window"))
	if title != "" {
		w.SetAttr("aria-label", title)
	}
	switch {
	case unified && toolbar != nil:
		w.AddClass("kit-unified")
		toolbar.SetAttr("data-window-bar", "")
		kids := slices.Clone(toolbar.Children())
		for _, k := range kids {
			toolbar.Remove(k)
		}
		toolbar.Append(windowLights())
		toolbar.Append(kids...)
		w.Append(toolbar)
	default:
		w.Append(windowBar(title))
		if toolbar != nil {
			w.Append(toolbar)
		}
	}
	w.Append(body)
	if footer != nil {
		w.Append(footer)
	}
	for _, a := range own {
		a.Apply(w)
	}
	return w
}

func windowBar(title string) *ui.Node {
	return ui.Header(ui.Class("kit-window-bar"), ui.Attr("data-window-bar", ""), windowLights(),
		ui.Span(ui.Class("kit-window-title"), ui.Text(title)))
}

func windowLights() *ui.Node {
	lights := ui.Span(ui.Class("kit-window-lights", "own-height"))
	for _, action := range []string{"close", "minimise", "zoom"} {
		lights.Append(ui.Button(ui.Class("kit-light", "kit-"+action), ui.Attr("data-window-action", action),
			ui.Attr("aria-label", action)))
	}
	return lights
}

// Toolbar is a Window's toolbar, the old ember/ui's Window.Toolbar: a row of class "kit-toolbar"
// above its body.
func Toolbar(args ...ui.Arg) *ui.Node {
	return Row(append([]ui.Arg{ui.Class("kit-toolbar")}, args...)...)
}

// Footer is a Window's footer, the old ember/ui's Window.Footer: a row of class "kit-footer" under
// its body.
func Footer(args ...ui.Arg) *ui.Node {
	return Row(append([]ui.Arg{ui.Class("kit-footer")}, args...)...)
}

// Compact gives a Window the small title bar of a tool or inspector.
func Compact() ui.Arg { return ui.Class("kit-compact") }

// UnifiedToolbar draws a Window's lights at the start of its Toolbar, which stands for its
// title bar and drags it, without a title: a file dialog's back, path and search on one
// line. Without a Toolbar the Window keeps its title bar.
func UnifiedToolbar() ui.Arg { return unifiedToolbar{} }

type unifiedToolbar struct{}

func (unifiedToolbar) Apply(*ui.Node) {}

// Resizable lets a Window's right and bottom edges resize it.
func Resizable() ui.Arg { return ui.Attr("data-resizable", "") }
