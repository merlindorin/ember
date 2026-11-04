# Document

![The settings panel, one Document in its window](settings.png "ember · settings")

```go
doc := ui.NewDocument(kit.Theme(), ui.Stylesheet(files, "game.css"))
doc.Body().Append(panel)
win.Add(doc)
```

| Option | Default | |
|---|---|---|
| `kit.Theme()` | none | ember's look: the kit's stylesheet and pictures |
| `ui.Stylesheet(fsys, name)` | none | a CSS file, after those before it; its `@font-face` files read from `fsys` |
| `ui.Images(fsys)` | none | where `url()` and an `img`'s `src` find pictures, by name, the last added first; no rules: `ui.Images(crypt.Art)` when a game's art lives apart from its sheets |
| `ui.CSS(text)` | none | a sheet written in the code, as a `<style>` element, after those before it; an `@font-face` in it is a warning, fonts come with `Stylesheet` |
| `ui.Emoji(set)` | none | colour emoji in text, from `twod.LoadEmoji` |

| Reading and changing it | |
|---|---|
| `doc.Body()`, `doc.HTML()` | where the game's elements go; the root |
| `doc.ByID(id)` | the first element with that `id` in tree order, or nil, as `getElementById` |
| `doc.Warnings()` | what the stylesheets and style attributes got wrong |
| `node.Box()` | an element's place on screen, after layout |
| `doc.BoxOf(n)` | its content box in window points, false until laid out or while hidden; where the game draws over an element that only holds room (give it `pointer-events: none` so the game's own widget gets the pointer); a resize reaches it the frame after |
| `doc.Layout(size)` | lays it out now, as each frame does; for tests |

One Document per window: added to it, it lays out at the window's size and draws over the 2D world each frame.

```
go run ./examples/92-ui-kit -element settings
```
