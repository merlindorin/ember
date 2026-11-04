# Window

![An Inspector window with a Save and Revert toolbar, two lines and a footer with a Close button; a compact Layers window; a window minimised to its bar; a window whose toolbar holds the lights, back, forward, Documents and Search](window.png "ember · window")

<!-- chrome -->
Matches Chrome within 0.19% of its pixels (threshold 0.25%).
<!-- /chrome -->

```go
inspector := kit.Window("Inspector",
	kit.Toolbar(kit.Button("Save"), kit.Button("Revert")),
	kit.Label("Tile 12 of walls.tsj"),
	kit.Footer(kit.Label("2 changes", kit.Expand()), kit.Button("Close", kit.Primary())))
d.Body().Append(inspector)
d.Show(inspector)
```

| Option | Default | |
|---|---|---|
| `kit.Toolbar(args…)` | none | a row above the body |
| `kit.Footer(args…)` | none | a row under the body |
| `kit.Compact()` | full | the small title bar of a tool |
| `kit.UnifiedToolbar()` | a title bar | the lights at the toolbar's start, which drags it; no title |
| `kit.Resizable()` | fixed | its right and bottom edges resize it |
| `ui.Attr("data-minimised", "")` | unfolded | folded to its bar |
| `ui.Attr("data-zoomed", "")` | its size | filling the game window |

| Reading and changing it | |
|---|---|
| `d.Show(w)`, `d.ShowModal(w)` | open it, over a live page or modal |
| `d.Close(w, value)` | close it, emitting `ui.Closed` |
| `d.Open(w)` | whether it is open |
| `d.PlaceWindow(w, at)` | its top-left at `at`, in the Document's points, without events |
| `d.SizeWindow(w, size)` | its border box, without events; a side of 0 is left as it is (`geom.V2(150, 0)` sets only the width) |
| `d.SetFolded(w, on)` | folded to its bar or not, without events |
| `d.MovedByUser(w)` | whether the player dragged it (`data-moved`); `PlaceWindow` leaves it false, so a game can stack the windows the player hasn't placed |

| Event | When |
|---|---|
| `ui.Closed{Node, Value}` | its close light, `d.Close`, or Escape or the pad's East while modal |
| `ui.WindowMoved{Node, At}` | let go somewhere else after a drag by its bar; `At` its top-left |
| `ui.WindowResized{Node, Size}` | an edge drag let go with a new size: the border box it took within its min and max sizes, not the one the pointer asked for |
| `ui.WindowFolded{Node, Folded}` | its minimise light folded or unfolded it, or the zoom light unfolded it |

Each fires once, when the gesture ends; a press on the bar without moving fires nothing. Callbacks: `ui.OnMove`, `OnResize`, `OnFold`.

| CSS | |
|---|---|
| `.kit-window` | the box, hidden without `[open]`, `[data-placed]` once placed, `[data-moved]` once dragged |
| `.kit-window-bar`, `.kit-window-title` | the title bar and its title |
| `.kit-window-lights`, `.kit-light.kit-close`, `.kit-light.kit-minimise`, `.kit-light.kit-zoom` | the lights |
| `.kit-window.kit-compact`, `.kit-window.kit-unified` | the two bars |
| `.kit-window[data-minimised]`, `.kit-window[data-zoomed]` | folded and filling |
| `.kit-window > .kit-body`, `.kit-toolbar`, `.kit-footer` | its parts |

Automation: the window by the slug of its title (`inspector`), unless an `aria-label` or `data-name` says otherwise; it doesn't scope its children; its lights `close`, `minimise` and `zoom`, `press`.

```
go run ./examples/92-ui-kit -element window
```
