# Sidebar

![A sidebar of Places and Recent, Tilesets chosen and Desktop hovered, beside the same in the file dialog's places look](sidebar.png "ember · sidebar")

<!-- chrome -->
Matches Chrome within 0.22% of its pixels (threshold 0.29%).
<!-- /chrome -->

```go
side := kit.Sidebar(
	kit.SidebarSection("Places", kit.SidebarItem("Home"), kit.SidebarItem("Desktop"), kit.SidebarItem("Tilesets")),
	kit.SidebarSection("Recent", kit.SidebarItem("walls.tsj"), kit.SidebarItem("props.tsj")),
)
```

| Option | Default | |
|---|---|---|
| `ui.Class("kit-places")` on the sidebar | plain | the file dialog's darker look |
| `""` as a section's title | — | no heading |
| `kit.Icon(set, name)` on an item | no icon | an icon before its text |

```go
d.Selected(side)                         // the item across every section, -1 for none
d.SetSelected(side, 2)                   // no event
kit.SetSection(side, "Recent", items...) // replaces that section, or adds it at the end
kit.SectionItems(side, "Recent")
kit.SidebarItems(side)
```

| Event | When |
|---|---|
| `ui.Changed{Node}` | an item chosen by a click or the arrows, `Node` the sidebar |
| `ui.Submitted{Node}` | Enter, a double click or the pad's South on the chosen item |

| CSS | |
|---|---|
| `.kit-sidebar` | the list, `:focus-visible`; `.kit-sidebar.kit-places` the file dialog's |
| `.kit-sidebar-section`, `.kit-sidebar-title` | a section and its heading, in capitals |
| `.kit-sidebar-item` | an item, with `:hover`, `[aria-selected="true"]` and `[aria-disabled="true"]` |

Automation: each item by its `id`, else the slug of its text (`walls-tsj`), value whether chosen; `press`.

```
go run ./examples/92-ui-kit -element sidebar
```
