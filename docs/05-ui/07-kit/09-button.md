# Button

![Buttons default, primary and destructive, in each state: default, hover, focus-visible, active and disabled](button.png "ember · button")

<!-- chrome -->
Matches Chrome within 0.43% of its pixels (threshold 0.66%).
<!-- /chrome -->

```go
save := kit.Button("Save", kit.Primary())
```

| Option | Default | |
|---|---|---|
| `kit.Primary()` | plain | the accent colour: the call to action |
| `kit.Destructive()` | plain | red: an action that loses something |
| `ui.Disabled()` | enabled | greyed out, takes no press |

| Event | When |
|---|---|
| `ui.Clicked{Node}` | a click, Enter or Space while focused, the pad's South, or automation's press |

| CSS | |
|---|---|
| `.kit-button` | its look, with `:hover`, `:active`, `:focus-visible` and `:disabled` |
| `.kit-button.kit-primary`, `.kit-button.kit-destructive` | the two roles |

Automation: its `id`, else the slug of its text (`save`); `press`.

```
go run ./examples/92-ui-kit -element button
```
