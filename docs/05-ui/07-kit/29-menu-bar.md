# Menu bar

![A menu bar with File open on New tileset, Open…, Save, Save as… and Quit with their shortcuts; under it the same bar with Edit hovered](menu-bar.png "ember · menu-bar")

<!-- chrome -->
Matches Chrome within 0.27% of its pixels (threshold 0.35%).
<!-- /chrome -->

```go
bar := kit.MenuBar(
	kit.MenuBarItem("File", kit.Menu(kit.MenuItem("Save", kit.Shortcut("Cmd+S")), kit.MenuItem("Quit"))),
	kit.MenuBarItem("Edit", kit.Menu(kit.MenuItem("Undo"), kit.MenuItem("Redo"))),
)
```

Its menus are [Menus](28-menu.md): one Tab stop, the arrows move along the bar and between open menus, a hovered title switches the open one. Pressing on its title opens it, and releasing over an item chooses it, as native menus do; a plain click opens it, then a click chooses; releasing over nothing closes it. A title's own `OnClick` doesn't fire for a press and release on it.

| Event | When |
|---|---|
| `ui.Clicked{Node}` | an item of its menus chosen; Node is the item |

| CSS | |
|---|---|
| `.kit-menu-bar` | the bar |
| `.kit-menu-bar-item` | a title, with `:hover`, `:focus-visible` and `[aria-expanded="true"]` |
| `.kit-menu-bar-item > .kit-menu` | its menu |

Automation: each title by the slug of its text (`file`), kept while its menu is open; each item as in a Menu; `press`.

```
go run ./examples/92-ui-kit -element menu-bar
```
