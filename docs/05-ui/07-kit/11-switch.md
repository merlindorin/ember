# Switch

![Switches off and on, in each state: default, hover, focus-visible and disabled](switch.png "ember · switch")

<!-- chrome -->
Matches Chrome within 0.29% of its pixels (threshold 0.40%).
<!-- /chrome -->

```go
music := kit.Switch("Music", ui.Checked(true))
```

| Option | Default | |
|---|---|---|
| `ui.Checked(true)` | off | the knob right, the track in the accent colour |
| `ui.Disabled()` | enabled | text greyed out, takes no press |

```go
track := kit.SwitchTrack(music)
on := d.Checked(track)
d.SetChecked(track, false)
```

| Event | When |
|---|---|
| `ui.Edited{Node}`, `ui.Changed{Node}` | flipped: a click on the track or its text, Enter or Space while focused, the pad's South, or automation; `Node` is `kit.SwitchTrack(music)` |

| CSS | |
|---|---|
| `.kit-switch` | the label round track and text, with `:hover` and `.kit-disabled` |
| `.kit-switch-track` | the track and knob, with `:checked` and `:focus-visible` |
| `.kit-switch-text` | the text |

Automation: a toggle named by its text (`music`); `press`, `set`.

```
go run ./examples/92-ui-kit -element switch
```
