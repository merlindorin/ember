# UI

`ember/ui` builds screens from Go as HTML elements styled by a CSS subset, checked against Chrome, and drawn through twod. It replaced the first `ember/ui` in [decision 0059](../99-decisions/0059-ember-ui-is-uiv2.md).

![A settings panel: a title, a form of a name, a volume slider and a music switch, then Cancel and Save](settings.png "ember · settings")

```go
doc := ui.NewDocument(kit.Theme())
doc.Body().Append(
	kit.Form(kit.Field("Name", kit.TextInput("Ayla"))),
	kit.Button("Save", kit.Primary()),
)
win.Add(doc)
```

UI as modern HTML elements styled by a CSS subset, written from Go and laid out as Chrome lays out the same page ([decision 0049](../99-decisions/0049-ui-is-modern-html-elements-and-a-css-subset-written-from-go.md)).

| Page | |
|---|---|
| [Document](01-document.md) | the tree a window shows |
| [Elements](02-elements/README.md) | building and changing it |
| [CSS](03-css.md) | what the subset styles, and what it refuses |
| [Events](04-events.md) | what the player did |
| [Focus and keys](05-focus-and-keys.md) | Tab, arrows, pads |
| [Automation](06-automation.md) | IDs and actions |
| [Kit](07-kit/README.md) | ember's components: buttons, forms, tables, windows… |
| [Matches Chrome](08-chrome.md) | how close each component is to Chrome's |
| [Walkthroughs](09-walkthroughs/README.md) | small programs, file by file |

## Try it

[main.go](../../examples/92-ui-kit/main.go)

[elements.go](../../examples/92-ui-kit/elements.go)

[core.go](../../examples/92-ui-kit/core.go)

[controls.go](../../examples/92-ui-kit/controls.go)

[layout.go](../../examples/92-ui-kit/layout.go)

[inputs.go](../../examples/92-ui-kit/inputs.go)

[lists.go](../../examples/92-ui-kit/lists.go)

[windows.go](../../examples/92-ui-kit/windows.go)

[92-ui-kit](../../examples/92-ui-kit)
