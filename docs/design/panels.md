# Collapsible and closable panels

Status: **approved by the user (2026-10-31); being built.** The panels are ui windows since 33eba70b.

The user asked for debug controls in the game, opened with F12, built from the engine's own ui (not ImGui). The engine part is a panel that can fold to its title and be closed. The game part is a debug panel made of those.

## 1. Engine: a titled panel that folds and closes

Today `ui.Panel` is a box with padding around its children. It has no title, and nothing folds or closes. The proposal adds options to the same `*ui.Panel`, not a new type, so a debug panel, a settings section and a game's own window all share one element.

```go
p := ui.NewPanel(
	ui.Title("Debug"),
	ui.Collapsible(),           // a fold arrow in the title bar
	ui.Closable(),              // a × in the title bar
	ui.Anchor(ui.TopRight), ui.Size(260, 0),
).Add(
	ui.NewColumn(ui.Gap(4)).Add(...),
)
```

- **Title bar.** `Title(text)` gives the panel a bar above its children, drawn from its theme (`TitleBackground`, `TitleColor`, `TitleFontSize` in `For[*ui.Panel]`). Without a title, a panel is exactly what it is today.
- **Collapsible.** The bar shows a fold arrow. A click on the bar, or Enter/Space/South while it has focus, folds the panel to its bar or opens it again. Folded, the panel's minimum size is its bar's, so the layout around it closes the gap: a column of folded sections reads as a list of headers.
  - `Collapsed()` starts it folded. `Collapsed()`/`SetCollapsed(on)` read and set it, following the option rules (0018, 0019).
  - Emits `ui.PanelCollapsed{Panel, Collapsed bool}`.
- **Closable.** The bar shows a ×. A click on it, or Escape/East while focus is inside the panel, hides the panel (`SetHidden(true)`) and emits `ui.PanelClosed{Panel}`. The game reopens it with `SetHidden(false)`. Closing never removes the entity, so its state (folded sections, slider values) survives.
- **Focus and pads.** The title bar is focusable when the panel is collapsible or closable, so keyboard and pad navigation reach it like a button. Children of a folded panel can't take focus.
- **Automation.** The bar records as `<path>/title` (role `button`, value `open`/`folded`), the × as `<path>/close`. `Act press` folds or closes, as real input does.
- **Nesting.** A collapsible panel inside another works as a section: same element, no separate "section" type.

### Movable (user, 2026-10-31)

- `ui.Movable()` (a panel option, needs a `Title`): dragging the title bar moves the panel. A click without a drag still folds it, so a press only folds if the pointer moved less than 4 points.
- While moving, the panel leaves its layout parent's flow: it is placed at its own position (top-left, in window points), clamped so its title bar stays inside the window, also when the window shrinks.
- `Position()`/`SetPosition(x, y)` read and set it; `PanelMoved{Panel, Position}` is emitted when a drag ends (not by the setter), so a game can save it.
- Keyboard and pads: none for now (a focused bar doesn't move); automation `<path>/title` takes `move x,y`.
- The demo's debug panel is movable, and its position is kept in debug.json with the folds.

### Resizable (user, 2026-10-31)

- `ui.Resizable()` (a panel option): dragging the panel's right edge, bottom edge or bottom-right corner resizes it. The cursor shows a resize shape over them, and the grab zone is 6 points.
- The size never goes below the panel's minimum size (its content's, folded or not) or past the window. An optional `MaxSize(w, h)` caps it.
- The children lay out into the new size: Fill and Expand take the room, and a Scroll inside shows bars when the content overflows.
- `Size()`/`SetSize(w, h)` read and set it; `PanelResized{Panel, Size}` is emitted when a drag ends, not by the setter. Automation `<path>/resize w,h`.
- The demo's debug panels are resizable, with their sizes kept in debug.json.

### Not in this proposal

- **Docking or tabs:** not needed for the debug panels.

### Alternatives considered

- **A new `ui.Window` type.** It would duplicate Panel's box, padding and theme for a title bar. Options on Panel keep one element and fail to compile on other kinds (options are typed per kind).
- **A separate `ui.Section` for folding.** Two names for one behaviour (0018). A folding panel with a title already is a section.
- **ImGui.** Declined by the user; `gui` was removed in 81d5b62.

### Tests (no GPU)

- Folding: the minimum size becomes the bar's and back; siblings in a column move up and down; `PanelCollapsed` is emitted once per change.
- Closing: × and Escape hide the panel and emit `PanelClosed`; `SetHidden(false)` shows it unchanged.
- Focus: Tab reaches the bar; a folded panel's children are skipped; Enter folds.
- Automation: `title` and `close` are listed and pressable by `Path()`.
- A panel without `Title` lays out and draws exactly as before.

The guide's UI section gets a page step for it (game-doc-writer), and the theme file documents the three new keys.

A game's own debug screen, built from these panels, lives in that game's docs.

## Order

1. The panel options in `ui` (game-engine-developer), with guide docs.
