# article

![A quest log of two dashed entries, each with its title](article.png "ember · article")

```go
ui.Article(ui.ID("key"),
	ui.H3(ui.Text("The lost key")),
	ui.P(ui.Text("The smith dropped it in the well.")),
)
```

| Default CSS | |
|---|---|
| `article` | `display: block` |

Laid out as a div: it says the content is whole on its own (a quest, a message, a news item), one of a list.

Automation: not recorded, unless it has an `id` (then a label of its text), as every block; no landmark role yet; its `id` or `data-name` prefixes its descendants' IDs.

```
go run ./examples/93-html-elements -element article
```
