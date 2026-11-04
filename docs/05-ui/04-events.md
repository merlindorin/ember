# Events

An element takes a callback for each event it sends:

```go
ui.Button(ui.Text("Save"), ui.OnClick(save))
```

The same, read as World events by a system:

```go
type Settings struct{ clicks world.Reader[ui.Clicked] }

func (s *Settings) Update(f world.Frame) {
	for e := range s.clicks.Read(f.World) {
		if e.Node.ID() == "save" {
			save()
		}
	}
}
```

| Callback | World event | When |
|---|---|---|
| `ui.OnClick` | `ui.Clicked{Node, Clicks}` | a button, a summary or an element with a tabindex pressed: by the pointer, a key, a pad or automation; `Clicks` 2 or 3 for a double or triple click |
| `ui.OnInput` | `ui.Edited{Node}` | a value changing while edited: each keystroke, each step of a range, a checkbox toggled |
| `ui.OnChange` | `ui.Changed{Node}` | a value committed: a checkbox toggled, an option chosen, a text field left or submitted after an edit |
| `ui.OnSubmit` | `ui.Submitted{Node}` | Enter in a one-line text field, or a textarea's submit shortcut |
| `ui.OnToggle` | `ui.Toggled{Node}` | a details element opened or closed |
| `ui.OnClose` | `ui.Closed{Node, Value}` | a dialog closed, with its return value, "" for Escape |
| `ui.OnSort` | `ui.Sorted{Node, Column, Descending}` | a table's column header pressed |
| `ui.OnKey` | `ui.KeyPressed{Node, Key, Text, Command}` | a key nothing took, on the focused element or the open modal dialog |
| `ui.OnBack` | `ui.BackPressed{Node}` | Escape or the pad's East that nothing took: a screen's way back |
| `ui.OnPointerEnter` | `ui.PointerEntered{Node}` | the pointer coming over it, once, not again for its children (mouseenter) |
| `ui.OnPointerLeave` | `ui.PointerLeft{Node}` | the pointer leaving it and all its children (mouseleave) |
| `ui.OnMove` | `ui.WindowMoved{Node, At}` | a window dragged by its bar and let go elsewhere |
| `ui.OnResize` | `ui.WindowResized{Node, Size}` | a window's edge drag let go with a new size |
| `ui.OnFold` | `ui.WindowFolded{Node, Folded}` | a window folded to its bar or unfolded |
| `ui.OnContextMenu` | `ui.ContextClicked{Node}` | the right button pressed and released on it; bubbles like Back |
| `ui.OnDrop` | `ui.Dropped{Target, Dragged}` | a draggable element let go over it, a drop target |
| `ui.OnDragEnd` | `ui.DragEnded{Source, Dropped}` | its drag ended, taken by a target or cancelled |

| A callback is | |
|---|---|
| `func()` | told nothing |
| `func(ui.Event)` | told the event: `Type` (`ui.EventClick`…), `Node`, and what its World event carries (`Clicks`, `Value`, `Column`, `Descending`, `Key`, `Text`, `Command`, `At`, `Size`, `Folded`, `Dragged`, `Dropped`) |
| `func(*world.World, ui.Event)` | told the World too |

```go
remove := ui.On(slot, ui.EventClick, equip) // added to a live element; remove() takes it off
```

Every event is emitted as a World event whether or not it has callbacks. Callbacks run right after their event, in the Document's Update, in the order they were added, and only while their element is in the Document. Most stay on their element; `OnBack`, `OnKey` and `OnContextMenu` bubble: the focused element's handlers run, then each ancestor's up to `html`, innermost first, `e.Node` still the focused element, until one calls `e.StopPropagation()`. So one `OnBack` on a screen's root hears its Escape and pad East; on `body` or `html` it also hears a Back with nothing focused, which emits no World event ([decision 0060](../99-decisions/0060-back-and-key-callbacks-bubble.md)). World events are emitted late in a frame and read the next. A point on a box's right or bottom edge is outside it, as Chrome's `elementFromPoint` has it. `doc.Value(e.Node)` reads what a control now holds.

## Right clicks and drag and drop

```go
slot := kit.Panel(ui.Draggable(), ui.OnDragEnd(func(e ui.Event) { if !e.Dropped { sfx.Play(thud) } }))
bag := kit.Panel(ui.OnDrop(func(e ui.Event) { equip(e.Dragged) }))
card := kit.Panel(ui.OnContextMenu(func(e ui.Event) { doc.ShowModal(menu) }))
```

A `ui.Draggable()` element drags once the left button has moved it 4 points; less is a click. Let go over an element with `OnDrop`, the target's `OnDrop` runs, then the source's `OnDragEnd`; let go over no target, or Escape, cancels it. While it drags, the source carries `data-dragging` and the target under the pointer `data-drag-over`, for CSS, and the pointer is a hand. A right click on the page's `html` or `body` stays the game's. Not yet: a picture following the pointer, drags by keys or pad, and drags between Documents or windows ([decision 0062](../99-decisions/0062-ui-elements-take-a-right-click-and-drag-and-drop-of-their-own.md)).

```
go run ./examples/92-ui-kit -element settings
```
