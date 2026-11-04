# Separator

![A separator between a menu's groups, a divider in a dialog, and vertical separators in a toolbar](separator.png "ember · separator")

<!-- chrome -->
Matches Chrome within 0.33% of its pixels (threshold 0.50%).
<!-- /chrome -->

```go
menu := kit.Column(open, save, kit.Separator(), quit)
```

| Option | Default | |
|---|---|---|
| `kit.Divider()` | inset 6 points at its ends | the line across, without inset |
| `kit.Vertical()` | across | standing, one line of text tall, between a row's items |

| CSS | |
|---|---|
| `.kit-separator` | a 1-point line in the border colour, 4 points above and below |
| `.kit-separator.kit-divider`, `.kit-separator.kit-vertical` | the two variants |

Automation: not seen.

```
go run ./examples/92-ui-kit -element separator
```
