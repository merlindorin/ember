# File dialog

![A file dialog in Pictures: places and recent files on the left, the folder's files in a table with list.txt dimmed, beach.png selected with its picture and details under the list, Show Pictures (.png, .jpg), Cancel and Open](file-dialog.png "ember · file-dialog")

<!-- chrome -->
Matches Chrome within 0.25% of its pixels (threshold 0.29%).
<!-- /chrome -->

```go
pictures := kit.NewFileDialog(doc, os.DirFS(dir),
	kit.FileType("Pictures", ".png", ".jpg"),
	kit.Places("Pictures", "Music", "Notes"),
	kit.OnChoose(func(e kit.FileChosen) { load(e.Path) }),
	kit.OnCancel(func() { status.SetText("cancelled") }))
w.AddSystems(pictures)
pictures.Open("Pictures")
```

| Option | Default | |
|---|---|---|
| `kit.FileType("Pictures", ".png", ".jpg")` | any file | a type in the Show pop-up; other files dimmed |
| `kit.Extensions(".png")` | any file | a type without a name |
| `kit.Kind("Tileset", ".tsj")` | its own names, else "XYZ file" | the Kind column and the details |
| `kit.RootName("Documents")` | `/` | the root in the path pop-up and the side |
| `kit.Places("Pictures", "Music")` | none | folders under Favourites |
| `kit.Recent("Pictures/beach.png")` | none | files under Recent, a click picks one |
| `kit.Describe(func(path string) string)` | kind · size · modified date | the line under the selected file's name |
| `kit.Preview(func(path string) string)` | a glyph | the selected file's picture, an image of the Document's files |
| `kit.DateFormat(func(time.Time) string)` | "Today, 15:04", "Yesterday, 15:04", "2 Jan 2006" | the dates |
| `kit.OwnWindow(style…)` | in the game's window | in a 900 × 600 window of its own |
| `kit.OnChoose(fn)` | none | runs with the `kit.FileChosen` when a file is picked; `func(kit.FileChosen)` or `func(*world.World, kit.FileChosen)` |
| `kit.OnCancel(fn)` | none | runs when it's cancelled; `func()` or `func(*world.World)` |

| Reading and changing it | |
|---|---|
| `fd.Open(folder)`, `fd.Save(folder, name)` | show it modal, to open or to save |
| `fd.Select(path)`, `fd.Pick()` | select a file, press Open or Save |
| `fd.Folder()`, `fd.Saving()`, `fd.FileType()`, `fd.Extensions()` | where it is and what it takes |
| `fd.SetPlaces(…)`, `fd.SetRecent(…)`, `fd.SetOwnWindow(on)` | change the side and the window; seen at the next Open |
| `fd.Node()`, `fd.Window()` | its Window node, its own platform window |
| `fd.Table()`, `fd.Thumbnails()`, `fd.SearchField()`, `fd.NameField()`, `fd.PathPopUp()` | its parts |

| Event | When |
|---|---|
| `kit.FileChosen{Dialog, Path, Saving}` | a file picked, or where to save |
| `kit.DialogCancelled{Dialog}` | Cancel, Escape or its close light |

Both are emitted whether or not the dialog has callbacks, for systems elsewhere; `OnChoose` and `OnCancel` run right after.

| CSS | |
|---|---|
| `.kit-window.kit-file-dialog` | the window, `.kit-own` in a window of its own |
| `.kit-file-main`, `.kit-file-pane`, `.kit-file-list` | the side and the list |
| `.kit-file-info`, `.kit-file-thumb`, `.kit-file-title`, `.kit-file-details`, `.kit-file-path` | the selected file |
| `.kit-file-note`, `.kit-file-hint` | the dimmed files' note and the footer's hint |
| `.kit-file-type`, `.kit-file-name-field`, `.kit-file-cancel`, `.kit-file-ok` | the footer |
| `.kit-nav-pair`, `.kit-view-toggle`, `.kit-file-search` | the toolbar |

Automation: its parts under `file-dialog`, or the name `kit.DialogLabel(name)` gives: `file-dialog/files/beach-png`, `file-dialog/ok`, `file-dialog/cancel`, `back`, `forward`: `press`; `file-dialog/file-name` and its search field: `type`, `set`.

```
go run ./examples/92-ui-kit -element file-dialog
```

## Saving

![The dialog saving into Pictures: Save as holiday.png in the footer, Cancel and Save](file-dialog-save.png "ember · file-dialog")

<!-- chrome 32b-file-dialog-save -->
Matches Chrome within 0.25% of its pixels (threshold 0.28%).
<!-- /chrome -->

```go
pictures.Save("Pictures", "holiday")
```

## Replacing

![The Replace “sunset.png”? alert over the dialog, Replace in red and Keep](file-dialog-replace.png "ember · file-dialog")

<!-- chrome 32c-file-dialog-replace -->
Matches Chrome within 0.24% of its pixels (threshold 0.28%).
<!-- /chrome -->

```go
pictures.Save("Pictures", "sunset.png")
pictures.Pick()
```

## In its own window

![The dialog filling a 900 × 600 window, without lights of its own](file-dialog-own-window.png "ember · file-dialog")

<!-- chrome 32d-file-dialog-own-window -->
Matches Chrome within 0.27% of its pixels (threshold 0.31%).
<!-- /chrome -->

```go
pictures := kit.NewFileDialog(doc, os.DirFS(dir), kit.OwnWindow())
```
