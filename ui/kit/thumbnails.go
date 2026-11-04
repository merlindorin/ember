package kit

import "github.com/merlindorin/ember/ui"

// Thumbnails is a div of class "kit-thumbnails", a listbox of Thumbnail
// cells laid out in lines, 118 by 112 points each, that scrolls when they run past its
// height. ui gives it WAI-ARIA's listbox behaviour: a click or the arrows select a cell,
// emitting Changed for the Thumbnails (Document.Selected reads it); a double click,
// Enter or South activates it, emitting Submitted. SetDimmed shows a cell that can't be
// selected.
func Thumbnails(args ...ui.Arg) *ui.Node {
	return ui.Div(append([]ui.Arg{ui.Class("kit-thumbnails"), ui.Attr("role", "listbox")}, args...)...)
}

// Thumbnail is a cell of Thumbnails: the picture src, an image of the Document's
// stylesheets' files ("" for none), fitted above name.
func Thumbnail(name, src string, args ...ui.Arg) *ui.Node {
	head := []ui.Arg{ui.Class("kit-thumbnail"), ui.Attr("role", "option")}
	if src != "" {
		head = append(head, Image(src, ui.Class("kit-keep-aspect-centered", "kit-thumbnail-picture")))
	} else {
		head = append(head, ui.Div(ui.Class("kit-thumbnail-picture")))
	}
	head = append(head, ui.Span(ui.Class("kit-thumbnail-name"), ui.Text(name)))
	return ui.Div(append(head, args...)...)
}

// ThumbnailSize sizes Thumbnails' cells, the old ember/ui's ThumbnailSize: w by h points, the
// picture above, the name under it.
func ThumbnailSize(w, h float32) ui.Arg {
	return declarations{
		{gridTemplateColumns, "repeat(auto-fill, " + pxs(w) + ")"},
		{"grid-auto-rows", pxs(h)},
	}
}
