# input

![A settings form: a name, gold, a volume slider, a checkbox, a switch, a search field and a password](input.png "ember · input")

```go
ui.Label(ui.For("name"), ui.Text("Name"))
ui.Input(ui.ID("name"), ui.Value("Ayla"))
ui.Input(ui.Type("range"), ui.Value("70"))
ui.Input(ui.Type("checkbox"), ui.Attr("switch", ""), ui.Checked(true))
ui.Input(ui.Type("radio"), ui.Attr("name", "camera"), ui.Value("follow"), ui.Checked(true))
```

| Type | |
|---|---|
| `text` | one line of text; the default |
| `number` | number characters; Up and Down step it |
| `range` | a slider, in the middle of its range without a value |
| `checkbox` | checked or not; with `switch`, drawn as a switch |
| `radio` | one checked among those of its `name`; arrows move the check, and the group is one Tab stop |
| `search` | text; Escape or the pad's East clears it |
| `password` | text shown as dots |

No spin buttons on `number`, no clear button on `search`: `kit.NumberField` and `kit.SearchField` add them as real elements.

| Attribute | |
|---|---|
| `ui.Type(t)` | one of the types above |
| `ui.Value(v)` | its starting value; `doc.Value(n)` reads it now, `doc.SetValue(n, v)` sets it (a focused text field takes it once changed or submitted, not while typed in) |
| `ui.Placeholder(s)` | a text field's text while empty |
| `ui.Checked(on)` | a checkbox or radio starts checked; `doc.Checked(n)`, `doc.SetChecked(n, on)` (checking a radio unchecks its group) |
| `name` | a radio's group |
| `switch` | a checkbox drawn as a switch |
| `min`, `max`, `step` | a range's (0, 100, 1) and a number's bounds and step; `step="any"` for a free number |
| `readonly`, `maxlength` | a text field kept as is, or held to that many characters |
| `size` | a text field's width in characters (20) |
| `ui.Disabled()` | takes no edit and no focus; `:disabled` |

| Default CSS | |
|---|---|
| `button, input, select, textarea` | `appearance: none; margin: 0; padding: 0; border: 0; background: none`, font, colour, line height and text alignment inherited; `display: inline-block` |
| `input, select, textarea` | `padding: 1px 2px; border: 1px solid #767676; border-radius: 2px; background-color: #fff` |
| `input, select` | `white-space: nowrap; overflow: hidden` |
| `::placeholder` | `color: #757575` |
| `input[type="checkbox"]` | `width: 13px; height: 13px; padding: 0; border-radius: 2px` |
| `input[type="checkbox"]:checked` | `background-color: #0075ff; border-color: #0075ff; color: #fff` |
| `input[type="radio"]` | `width: 13px; height: 13px; padding: 0; border-radius: 50%`; `:checked` a `#0075ff` ring and dot |
| `input[type="checkbox"][switch]` | `width: 32px; height: 18px; border: 1px solid #767676; border-radius: 9px; background-color: #efefef; color: #767676`; `:checked` `#0075ff`, a white thumb |
| `input[type="range"]` | `width: 129px; height: 16px; padding: 0; border: 0; background: none; color: #0075ff` |
| `input:disabled, textarea:disabled` | text `#545454`, background and border at 30%; checkboxes, radios and switches in Chrome's greys |

A range's `background-color` paints its track and its `color` its idle thumb.

| Event | When |
|---|---|
| `ui.Edited{Node}` | every keystroke in a text field, every step of a range, a checkbox toggled, a radio checked |
| `ui.Changed{Node}` | a text field left or submitted after an edit, a range let go after a drag or stepped by a key, a checkbox toggled, a radio checked |
| `ui.Submitted{Node}` | Enter in a text, number, search or password field |

Automation: its `id`, else `data-name`, `aria-label`, its label's text, then its placeholder (`name`); a text field `type`, `set`, `select-all`, `submit`; a checkbox `press`, `set` (true or false); a range `set` (a number).

```
go run ./examples/93-html-elements -element input
```
