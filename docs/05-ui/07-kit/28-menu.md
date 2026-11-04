# Menu

![The Tile actions menu open: Duplicate Cmd+D, Rename… F2 hovered, Move to with its submenu open on Walls, Floors and Props, a separator, Paste style greyed out, Delete Del](menu.png "ember · menu")

<!-- chrome -->
Matches Chrome within 0.24% of its pixels (threshold 0.31%).
<!-- /chrome -->

```go
actions := kit.MenuButton("Tile actions", kit.Menu(
	kit.MenuItem("Duplicate", kit.Shortcut("Cmd+D")),
	kit.SubMenu("Move to", kit.MenuItem("Walls"), kit.MenuItem("Floors")),
	kit.Separator(),
	kit.MenuItem("Delete", kit.Shortcut("Del")),
))
```

| Option | Default | |
|---|---|---|
| `kit.Shortcut("Cmd+D")` | none | the key at the item's right, muted; the game reads it |
| `ui.Attr("aria-disabled", "true")` | enabled | greyed out, skipped |
| `kit.SubMenu(text, items…)` | | an item opening a menu beside it, `▸` at its right |
| `kit.Separator()` | | a line between items |
| `ui.Attr("open", "")` on the Menu | closed | open without its button |

| Event | When |
|---|---|
| `ui.Clicked{Node}` | an item chosen by a click, Enter, Space, the pad's South or automation's press; Node is the item, and the menus close |

| CSS | |
|---|---|
| `.kit-menu` | the box, hidden without `[open]` |
| `.kit-menu-item` | an item, with `:hover`, `:focus-visible`, `[aria-expanded="true"]` and `[aria-disabled="true"]` |
| `.kit-menu-label`, `.kit-shortcut`, `.kit-submenu-arrow` | its text, its key, its arrow |
| `.kit-menu-item.kit-submenu`, `.kit-menu .kit-menu` | a submenu and the menu beside it |
| `.kit-menu-button` | the button opening it |

Automation: each item by the slug of its text (`rename`); `press`.

```
go run ./examples/92-ui-kit -element menu
```
