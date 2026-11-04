# Elements

![The settings panel: a section, a heading, a form and a row of buttons](../settings.png "ember · settings")

```go
ui.Section(ui.Class("kit-panel"),
	ui.H2(ui.Text("Settings")),
	ui.Label(ui.For("name"), ui.Text("Name")),
	ui.Input(ui.ID("name"), ui.Type("text"), ui.Value("Ayla")),
)
```

One constructor per element, taking `Arg`s: attributes, and children, since a `*Node` is one.

| Elements | |
|---|---|
| [`div`](01-div.md), [`section`](02-section.md), [`header`](03-header.md), [`footer`](04-footer.md), [`nav`](05-nav.md), [`main`](06-main.md), [`aside`](07-aside.md), [`article`](08-article.md) | blocks |
| [`p`](09-p.md), [`h1`](10-h1.md), [`h2`](11-h2.md), [`h3`](12-h3.md), [`h4`](13-h4.md), [`span`](14-span.md), [`strong`](15-strong.md), [`em`](16-em.md), [`code`](17-code.md), [`label`](18-label.md), [`kbd`](38-kbd.md), [`samp`](39-samp.md), [`pre`](40-pre.md) | text |
| [`button`](19-button.md), [`input`](20-input.md), [`select`](21-select.md), [`option`](22-option.md), [`textarea`](23-textarea.md), [`progress`](24-progress.md), [`meter`](25-meter.md) | controls |
| [`details`](26-details.md), [`summary`](27-summary.md), [`dialog`](28-dialog.md), [`table`](29-table.md), [`thead`](30-thead.md), [`tbody`](31-tbody.md), [`tr`](32-tr.md), [`th`](33-th.md), [`td`](34-td.md), [`caption`](41-caption.md), [`tfoot`](42-tfoot.md) | folding, dialogs and tables |
| [`img`](35-img.md), [`hr`](36-hr.md), [`canvas`](37-canvas.md) | pictures |

Each constructor is the tag's name: `ui.Div`, `ui.H1`, `ui.Textarea`, `ui.Canvas`.

| Arg | |
|---|---|
| `ID(id)`, `Class(names…)`, `Style(css)`, `Attr(name, value)` | identity and look |
| `Text(s)` | its text |
| `Type(t)`, `Value(v)`, `Placeholder(s)`, `Checked(on)`, `Selected()`, `Disabled()`, `Open()`, `For(id)`, `TabIndex(i)` | controls |

| Changing it | |
|---|---|
| `n.Append(children…)`, `n.Remove(child)` | the tree |
| `n.SetText(s)`, `n.SetAttr(name, v)`, `n.RemoveAttr(name)`, `n.AddClass(c)`, `n.RemoveClass(c)`, `n.SetStyle(css)` | a node; the next frame restyles only what changed |
| `doc.SetValue(n, v)`, `doc.Value(n)`, `doc.SetChecked(n, on)`, `doc.Checked(n)`, `doc.SetSelected(n, i)`, `doc.Selected(n)` | a control's value, without events |
| `doc.Show(n)`, `doc.ShowModal(n)`, `doc.Close(n, value)` | a dialog |
| `doc.SetOpen(n, on)` | a details, without events |
| `doc.Open(n)` | a dialog or details open |
| `doc.Focus(n)`, `doc.Focused(n)` | the focus |
| `ui.Follow(entity, offset)`, `ui.SetFollow(n, entity, offset)` | keep an element over a 2D entity: its box's bottom centre on the entity's screen point plus `offset` (CSS px), hidden while the point is off screen, in place from its second frame; `SetFollow` on an element already shown follows from the next frame; a nil entity stops it where it is |

```
go run ./examples/92-ui-kit -element settings
```

## Try it

Each element alone, in its default look: `-element` takes its tag.

[main.go](../../../examples/93-html-elements/main.go)

[elements.go](../../../examples/93-html-elements/elements.go)

[blocks.go](../../../examples/93-html-elements/blocks.go)

[text.go](../../../examples/93-html-elements/text.go)

[controls.go](../../../examples/93-html-elements/controls.go)

[structure.go](../../../examples/93-html-elements/structure.go)

[93-html-elements](../../../examples/93-html-elements)
