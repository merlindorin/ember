# Status bar

![Three window footers: props.tsj saved; walls.tsj with 100% and the amber dot of unsaved changes; a long name cut with an ellipsis before the dot](status-bar.png "ember · status-bar")

<!-- chrome -->
Matches Chrome within 0.20% of its pixels (threshold 0.26%).
<!-- /chrome -->

```go
bar := kit.StatusBar("walls.tsj · 41 tiles · 3 shapes", kit.Label("100%", kit.Caps()))
```

Nodes after the text sit between it and the dot.

| Reading and changing it | |
|---|---|
| `kit.Status(bar)`, `kit.SetStatus(bar, "props.tsj · 42 tiles")` | its text |
| `kit.Dot(bar)`, `kit.SetDot(bar, true)` | the dot: something is unsaved |

| CSS | |
|---|---|
| `.kit-status-bar` | the line |
| `.kit-status-bar > .kit-status` | its text, muted, cut at its end |
| `.kit-status-dot` | the accent dot, `[hidden]` while saved |

Automation: none.

```
go run ./examples/92-ui-kit -element status-bar
```
