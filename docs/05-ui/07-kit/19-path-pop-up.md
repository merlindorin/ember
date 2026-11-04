# Path pop-up

![Path pop-ups closed, one cut at its width, paths cut at their start, and one open on its folders](path-pop-up.png "ember · path-pop-up")

<!-- chrome -->
Matches Chrome within 0.24% of its pixels (threshold 0.32%).
<!-- /chrome -->

```go
folder := kit.PathPopUp("/Users/ada/assets/survivors/crypt")
```

| Option | Default | |
|---|---|---|
| `kit.RootName("assets")` | the root as it is (`/`) | the root's name in the button and the menu |

```go
kit.Folder(folder)             // "/Users/ada/assets/survivors/crypt"
kit.SetFolder(folder, "/Users/ada/assets")
path, ok := kit.FolderOf(item) // the folder of a chosen item
```

| Event | When |
|---|---|
| `ui.Clicked{Node}` | a folder chosen from its menu, `Node` the item: `kit.FolderOf` reads its path |

| CSS | |
|---|---|
| `.kit-path-pop-up` | the button, at most 340 points wide |
| `.kit-folder-glyph`, `.kit-folder-tab`, `.kit-folder-body` | the folder drawn before the name |
| `.kit-path-label`, `.kit-path-name`, `.kit-path-rest` | the bold name, then the two folders above it, dimmer, cut with an ellipsis |
| `.kit-pop-up-chevron` | the ▾ |
| `.kit-path-pop-up .kit-menu-item`, `.kit-menu-check` | the menu's folders and the ✓ of the one shown |

Automation: its `id`, else the slug of its text (`crypt-survivors-assets`); `press`; each folder is a menu item named by the folder's name, `press`.

```
go run ./examples/92-ui-kit -element path-pop-up
```
