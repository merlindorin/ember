# Hint

![The same hint read for keyboard, mouse, gamepad and touch, each row set to its device; touch reads the keyboard's text](hint.png "ember · hint")

<!-- chrome -->
Matches Chrome within 0.31% of its pixels (threshold 0.40%).
<!-- /chrome -->

```go
keys := kit.Hint(
	kit.KeyboardHint("Arrows to look · Enter to choose"),
	kit.MouseHint("Click a level to choose"),
	kit.GamepadHint("Stick to look · South to choose"),
)
```

| Option | Default | |
|---|---|---|
| `kit.KeyboardHint(text)` | none | its text before any input, and for every device without one; always give it |
| `kit.MouseHint(text)`, `kit.GamepadHint(text)`, `kit.TouchHint(text)` | the keyboard's | its text when that device was used last |

```go
kit.SetHint(keys, input.GamepadDevice, "Stick to look · Cross to choose")
kit.HintText(keys, input.MouseDevice) // "" when it has no text of its own for it
```

| Device | |
|---|---|
| the one used last | ui keeps it on `html`: `data-input-device="keyboard|mouse|gamepad|touch"` |
| a panel's own | `ui.Attr("data-input-device", "gamepad")` on an ancestor wins over `html`'s: that panel reads for that device whatever was used |

| CSS | |
|---|---|
| `.kit-hint` | a span holding one span per text |
| `.kit-hint > .kit-hint-keyboard`, `.kit-hint-mouse`, `.kit-hint-gamepad`, `.kit-hint-touch` | each device's text, one shown |

Automation: with an `id`, a label reading only the text shown.

```
go run ./examples/92-ui-kit -element hint
```
