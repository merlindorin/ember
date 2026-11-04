# Layout

A game's main menu that fills its window at any size: a title at the top, a grid of levels in the middle, and a footer along the bottom. Resize the window and everything finds its place again.

![The menu following its window as it grows wide and shrinks small](layout.gif)

## The rule

Layout is CSS flex and grid. `kit.Column` stacks its children, `kit.Row` lines them up; `kit.Gap` spaces them and `kit.Padding` keeps room inside the edge. A child with `kit.Expand()` takes the room left over, shared alike; `kit.Fill()` stretches one across. `kit.AlignCenter()` centres the children across, `kit.JustifyBetween()` and the rest place them along ([Row and Column](../07-kit/06-row-and-column.md)). A height in percent needs its parents' heights, so the page's sheet, written in the code with `ui.CSS`, gives `html`, `body` and the menu `height: 100%`, and styles the title by its `id`.

## The menu

One padded column holds the title, the grid and the footer. Two empty `ui.Div(kit.Expand())` share the height between them, so the grid stays in the middle; `kit.Grid(4, …)` lays the levels out in rows of four. The footer fills the width, and its label expands, pushing the buttons to the ends.

[main.go](../../../examples/12-layout/main.go)

## Dividers, text, bigger windows

| | |
|---|---|
| a line between parts | `kit.Separator()`, or `kit.Separator(kit.Vertical())` in a row ([Separator](../07-kit/05-separator.md)) |
| a label that wraps | `kit.Label(text, kit.Wrap())`, its height following its lines; `kit.Center()` centres each line ([Label](../07-kit/01-label.md)) |
| bigger windows | `ui.Zoom(f)`, or `doc.SetZoom(win.Size().Y / 720)` each frame to follow a resize: the Document lays out at 1/f of its window and draws f times larger, text sharp, clicks where things show |

## Try it

[12-layout](../../../examples/12-layout)
