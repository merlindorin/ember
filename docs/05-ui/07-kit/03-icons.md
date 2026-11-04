# Icons

![Icons from a two-glyph test font at the default size, 13, 20 and 30 points, in accent and muted colours, then before a label, a caps label and a rich text, and a missing name drawing nothing](icons.png "ember · icons")

<!-- chrome -->
Matches Chrome within 0.39% of its pixels (threshold 0.50%).
<!-- /chrome -->

```go
set, err := kit.LoadIcons(files, "lucide.ttf", "codepoints.json")
doc := ui.NewDocument(kit.Theme(), set.Sheet())

heart := kit.NewIcon(set, "heart", ui.Style("color: tomato; font-size: 20px"))
save := kit.Button("Save", kit.Icon(set, "save"))
```

| Call | |
|---|---|
| `kit.LoadIcons(fsys, font, names)` | a TrueType icon font and a JSON of names to code points (`"e0a1"`, `"U+E0A1"` or a number) |
| `set.Sheet()` | the Document option that gives it the font |
| `kit.NewIcon(set, name, args…)` | an icon alone, at its font size and colour |
| `kit.Icon(set, name)` | an icon before a label's, button's or panel's text |
| `set.Has(name)`, `set.Family()` | whether the set has an icon, its font family |

| CSS | |
|---|---|
| `.kit-icon` | the glyph, upright and regular |
| `.kit-icon.kit-before` | an icon before text, 0.3 em from it |

Automation: the icon adds nothing to its element's ID: `kit.Button("Save", kit.Icon(set, "save"))` is `save`.

```
go run ./examples/92-ui-kit -element icons
```
