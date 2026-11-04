# Focus and keys

![The settings panel, Save focused by Tab](focus.png "ember · settings")

| Input | Does |
|---|---|
| Tab, Shift-Tab | the next or previous focusable element, in `tabindex` order, once something has the focus |
| the arrows | the focus to the nearest element that way |
| the pad's cross or stick | the same as the arrows |
| Enter, Space, the pad's South | press the focused element |
| Escape, the pad's East | close a menu or a modal dialog, else `BackPressed` |
| a click | focuses quietly, without the outline |

| | |
|---|---|
| `ui.Attr("autofocus", "")` | the first focus, as on a page's load |
| `doc.Focus(n)`, `doc.Focused(n)` | give or read the focus, shown as from Tab; an element shown this frame gets it at its first layout (a later Focus replaces it, leaving the Document drops it), so a screen is shown and focused in one call |
| a modal dialog | keeps the focus inside until it closes, and owns its window's input: other Documents and twod widgets get no hover, clicks or keys, and the game reads `PointerCaptured` and `KeyboardCaptured` true ([decision 0053](../99-decisions/0053-a-modal-owns-its-windows-input.md)) |
| `:focus-visible` | matches only focus from keys or a pad |
| `html[data-input-device="keyboard"]` | follows the last device, keyboard, mouse, gamepad or touch, so CSS shows its glyphs |

A key nothing takes is a `KeyPressed` on the focused element; with nothing focused and no modal dialog, the keys are the game's.

A Document takes no key or pad press in its first frame, nor in the first frame it draws after being hidden: the Escape that opened a pause menu is never also that menu's `BackPressed`.

```
go run ./examples/92-ui-kit -element settings
```
