# Form

![A form of a name, a volume, a divider and a music check, and below it one lined up by SetLabelWidth with fields one width](form.png "ember · form")

<!-- chrome -->
Matches Chrome within 0.14% of its pixels (threshold 0.20%).
<!-- /chrome -->

```go
settings := kit.Form(
	kit.Field("Name", kit.TextInput("Ayla")),
	kit.Field("Volume", kit.Slider(60, kit.Range(0, 100))),
	kit.Separator(kit.Divider()),
	kit.Field("Music", kit.Check("On", ui.Checked(true))),
)
```

```go
kit.SetLabelWidth(settings, 110) // the labels' column at least 110 points
kit.SetFieldWidth(settings, 120) // every field 120 points; 0 stretches them again
kit.FieldLabel(field)            // the label of field's row
```

| CSS | |
|---|---|
| `.kit-form` | the grid of labels beside fields |
| `.kit-form-row` | a Field: its label and field |
| `.kit-form .kit-form-label` | the label, right-aligned and muted |
| `.kit-form .kit-form-field` | the field, stretched across its column |
| `.kit-form.kit-fixed-fields .kit-form-field` | fields at `SetFieldWidth`'s width |
| `.kit-form > .kit-separator` | a divider across both columns |

Automation: the form adds nothing; its fields are seen on their own.

```
go run ./examples/92-ui-kit -element form
```
