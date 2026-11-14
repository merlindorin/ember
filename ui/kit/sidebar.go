package kit

import (
	"slices"

	"github.com/lantervale/ember/ui"
)

// Sidebar is a div of class "kit-sidebar" listing places
// in titled SidebarSections, one chosen at a time. ui gives it
// WAI-ARIA's listbox behaviour: a click or Up and Down choose an item,
// emitting ui.Changed for the sidebar (Document.Selected reads it);
// Enter, a double click or South emit ui.Submitted. Class "kit-places" is
// The old ember/ui's file-dialog look.
func Sidebar(sections ...*ui.Node) *ui.Node {
	s := ui.Div(ui.Class("kit-sidebar"), ui.Attr("role", "listbox"))
	s.Append(sections...)
	return s
}

// SidebarSection is a sidebar's section: title, shown in capitals, then
// items; no title, no heading.
func SidebarSection(title string, items ...*ui.Node) *ui.Node {
	sec := ui.Div(ui.Class("kit-sidebar-section"), ui.Attr("role", "group"), ui.Attr("aria-label", title))
	if title != "" {
		sec.Append(ui.Div(ui.Class("kit-sidebar-title"), ui.Text(title)))
	}
	sec.Append(items...)
	return sec
}

// SidebarItem is a sidebar's place reading text; args go on it, such as
// an Icon before the text.
func SidebarItem(text string, args ...ui.Arg) *ui.Node {
	return ui.Div(append([]ui.Arg{ui.Class("kit-sidebar-item"), ui.Attr("role", "option"),
		ui.Span(ui.Class("kit-sidebar-label"), ui.Text(text))}, args...)...)
}

// SetSection replaces sidebar's section titled title with one of items,
// as a Recent list changes, or adds it at the end.
func SetSection(sidebar *ui.Node, title string, items ...*ui.Node) {
	fresh := SidebarSection(title, items...)
	secs := sidebar.Children()
	i := slices.IndexFunc(secs, func(s *ui.Node) bool { v, _ := s.Attr("aria-label"); return v == title })
	if i < 0 {
		sidebar.Append(fresh)
		return
	}
	after := slices.Clone(secs[i+1:])
	sidebar.Remove(secs[i])
	sidebar.Append(fresh)
	sidebar.Append(after...)
}

// SectionItems are the items of sidebar's section titled title.
func SectionItems(sidebar *ui.Node, title string) []*ui.Node {
	for _, s := range sidebar.Children() {
		if v, _ := s.Attr("aria-label"); v == title {
			return sectionItems(s)
		}
	}
	return nil
}

// SidebarItems are sidebar's items, in order.
func SidebarItems(sidebar *ui.Node) []*ui.Node {
	var out []*ui.Node
	for _, s := range sidebar.Children() {
		out = append(out, sectionItems(s)...)
	}
	return out
}

func sectionItems(sec *ui.Node) []*ui.Node {
	var out []*ui.Node
	for _, c := range sec.Children() {
		if c.HasClass("kit-sidebar-item") {
			out = append(out, c)
		}
	}
	return out
}
