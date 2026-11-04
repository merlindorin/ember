# Rich text

![Rich texts in dashed boxes: plain, bold, italic, a hex and a named colour, nested and crossed tags, an unknown tag kept as typed, then wrapped, truncated, centred and right-aligned](rich-text.png "ember · rich-text")

<!-- chrome -->
Matches Chrome within 0.43% of its pixels (threshold 0.60%).
<!-- /chrome -->

```go
reward := kit.RichText("Reward: [color=#ffce76]250 gold[/color], [b]Ember Blade[/b]")
```

| Tag | |
|---|---|
| `[b]…[/b]` | bold (a `strong`) |
| `[i]…[/i]` | italic (an `em`) |
| `[color=…]…[/color]` | `#rrggbb`, `#rrggbbaa` or a CSS colour name |
| any other | kept as typed |

| Option | Default | |
|---|---|---|
| `kit.Wrap()`, `kit.Truncate()`, `kit.Center()`, `kit.Right()` | one line, left | Label's layouts |
| `kit.ExpandShortcodes()` | as typed | `:name:` shortcodes shown as their emoji |

Reading it: `reward.Text()` is the text without its tags.

| CSS | |
|---|---|
| `.kit-label.kit-rich-text` | Label's classes; `strong` and `em` keep the browser's look |

Automation: only with an `id`, a label reading its text without tags; no action.

```
go run ./examples/92-ui-kit -element rich-text
```
