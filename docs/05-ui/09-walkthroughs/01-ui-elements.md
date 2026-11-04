# UI elements

A panel in the middle of a window, with a title, a count and two buttons: Click adds one, Reset goes back to zero.

![A counter: a panel with a title, the count and two buttons](counter.png "ember · ui elements")

## Building the page

The UI is one Document, added to the window. `kit.Panel`, `kit.Column`, `kit.Row`, `kit.Label` and `kit.Button` build its nodes, children as arguments; `kit.Gap` and `kit.Padding` space them, and `ui.Style` sizes the buttons. Placing the panel is CSS: the page's sheet, written in the code with `ui.CSS`, centres the body's content. `ui.ID` names the buttons, which `doc.ByID` finds again.

[main.go](../../../examples/11-ui-elements/main.go)

## Reacting to clicks

Each button takes its callback with `ui.OnClick`. `Counter`'s methods count and write the label with `SetText`; no system reads events.

[counter.go](../../../examples/11-ui-elements/counter.go)

## Try it

[11-ui-elements](../../../examples/11-ui-elements)
