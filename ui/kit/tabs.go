package kit

import (
	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/ui"
)

// TabPage is one tab of Tabs: its title and the page it shows.
type TabPage struct {
	tab, page *ui.Node
	title     string
}

// Tab is a page shown when the tab reading title is chosen; args go on
// the tab, such as an Icon before the title.
func Tab(title string, page *ui.Node, args ...ui.Arg) TabPage {
	tab := ui.Button(append([]ui.Arg{ui.Class("kit-choice-item"), ui.Attr("role", "tab"), ui.Text(title)}, args...)...)
	return TabPage{tab: tab, page: page, title: title}
}

// Tabs is a div of class "kit-tabs" holding a centred
// segmented bar of tabs, role tablist, and each tab's page in a div of
// role tabpanel, the first tab chosen. ui gives it WAI-ARIA's tabs
// behaviour: only the chosen tab's page shows, a click or the arrows
// choose another, and a change emits ui.Changed for the bar. A page's
// id is its tab's id, else its title's slug, followed by "-panel".
func Tabs(tabs ...TabPage) *ui.Node {
	bar := ui.Div(ui.Class("kit-choice", "kit-segmented", "kit-tab-bar"), ui.Attr("role", "tablist"))
	t := ui.Div(ui.Class("kit-tabs"), ui.Div(ui.Class("kit-tab-bar-row"), bar))
	for i, tp := range tabs {
		id := tp.tab.ID()
		if id == "" {
			id = automation.Slug(tp.title)
		}
		id += "-panel"
		panel := ui.Div(ui.ID(id), ui.Class("kit-tab-panel"), ui.Attr("role", "tabpanel"), tp.page)
		tp.tab.SetAttr("aria-controls", id)
		if i == 0 {
			tp.tab.SetAttr("aria-selected", "true")
		} else {
			tp.tab.SetAttr("aria-selected", "false")
			panel.SetAttr("hidden", "")
		}
		bar.Append(tp.tab)
		t.Append(panel)
	}
	return t
}

// TabBar is the bar of tabs, role tablist, whose ui.Changed reports a
// change of tab.
func TabBar(tabs *ui.Node) *ui.Node { return tabs.Children()[0].Children()[0] }

// TabOf is the tab i of tabs.
func TabOf(tabs *ui.Node, i int) *ui.Node { return TabBar(tabs).Children()[i] }

// SelectedTab is the place of the tab tabs shows.
func SelectedTab(d *ui.Document, tabs *ui.Node) int { return d.Selected(TabBar(tabs)) }

// SetSelectedTab shows tab i of tabs, without an event.
func SetSelectedTab(d *ui.Document, tabs *ui.Node, i int) { d.SetSelected(TabBar(tabs), i) }
