# Tabs

![Tabs of a settings window, Audio chosen and Controls hovered, and a list/grid view toggle](tabs.png "ember · tabs")

<!-- chrome -->
Matches Chrome within 0.23% of its pixels (threshold 0.30%).
<!-- /chrome -->

```go
settings := kit.Tabs(
	kit.Tab("Video", videoPage),
	kit.Tab("Audio", audioPage),
	kit.Tab("Controls", controlsPage),
)
```

| Option | Default | |
|---|---|---|
| `kit.Icon(set, name)` on a `kit.Tab` | no icon | an icon before the tab's title |

```go
kit.SelectedTab(d, settings)       // 0, the first tab, until another is chosen
kit.SetSelectedTab(d, settings, 1) // no event
kit.TabOf(settings, 2)             // the tab itself
kit.TabBar(settings)               // the bar, its events' Node
```

| Event | When |
|---|---|
| `ui.Changed{Node}` | another tab chosen by a click or the arrows, `Node` the `TabBar` |

| CSS | |
|---|---|
| `.kit-tabs` | the bar over the chosen tab's page |
| `.kit-tab-bar-row` | centres the bar |
| `.kit-tab-bar > .kit-choice-item` | a tab, `[aria-selected="true"]` the chosen one |
| `.kit-choice.kit-view-toggle` | a `kit.Choice` as a view toggle, as below the tabs |

Automation: each tab by its `id`, else the slug of its title (`audio`), value whether chosen; `press`.

```
go run ./examples/92-ui-kit -element tabs
```
