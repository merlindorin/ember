# ui: screens of HTML elements and CSS

A screen is a `*ui.Document`: HTML elements built from Go, styled by a CSS subset checked against Chrome, added to a window as an entity. `ui/kit` builds ember's components on it. Guide: `docs/05-ui`; examples `examples/11-ui-elements` to `examples/18-testing-ui`, every kit component alone in `examples/92-ui-kit -element <name>`.

## Make a screen

| Need | API |
|---|---|
| a Document | `ui.NewDocument(opts ...ui.DocumentOption) *ui.Document` |
| ember's look (kit components need it) | `kit.Theme() ui.DocumentOption` |
| a sheet written in the code | `ui.CSS(text string) ui.DocumentOption`: an `@font-face` in it is a warning |
| a CSS file, with its fonts | `ui.Stylesheet(fsys fs.FS, name string) ui.DocumentOption`: `@font-face` `src` files read from `fsys` |
| pictures for `url()` and `img` | `ui.Images(fsys fs.FS) ui.DocumentOption` |
| everything bigger, as browser zoom | `ui.Zoom(f float32) ui.DocumentOption`, `doc.SetZoom(f)`, `doc.Zoom()` |
| show it | `doc.Body().Append(nodes...)`, then `win.Add(doc)`: it lays out at the window's size |

- Sheets apply in the order given: `kit.Theme()` first, the game's sheet after it.
- `html, body { height: 100%; }` lets the body centre or stretch what it holds.
- A sheet's mistakes go to `doc.Warnings()`, never a crash.

## Elements and arguments

Every element is `func(args ...ui.Arg) *ui.Node`: `ui.Div`, `Section`, `Header`, `Footer`, `Nav`, `Main`, `Aside`, `Article`, `P`, `H1`–`H4`, `Span`, `Strong`, `Em`, `Code`, `Kbd`, `Samp`, `Pre`, `Label`, `Button`, `Input`, `Select`, `Option`, `Textarea`, `Details`, `Summary`, `Dialog`, `Table`, `Thead`, `Tbody`, `Tfoot`, `Caption`, `Tr`, `Th`, `Td`, `Progress`, `Meter`, `Img`, `Hr`. A `*ui.Node` is itself an Arg: as an argument, it becomes a child.

| Arg | For |
|---|---|
| `ui.ID(id)`, `ui.Class(names...)` | selectors, `doc.ByID`, automation paths |
| `ui.Text(s)` | a text child |
| `ui.Style(css)` | the style attribute; adds to what earlier args (`kit.Padding`) set |
| `ui.Attr(name, value)`, `ui.Type(t)`, `ui.Value(v)`, `ui.Placeholder(s)`, `ui.For(id)` | attributes |
| `ui.Checked(on)`, `ui.Open()`, `ui.Disabled()`, `ui.Selected()`, `ui.Draggable()`, `ui.TabIndex(i)` | flags |
| `ui.Attr("autofocus", "")` | the first focus |

## The kit catalogue

Each returns `*ui.Node` unless said; `args ...ui.Arg` go on the element.

| Component | Signature | For |
|---|---|---|
| Panel | `kit.Panel(args...)`, `kit.Title(text) ui.Arg` | a framed box; Title puts a heading above |
| CollapsiblePanel | `kit.CollapsiblePanel(title, open, args...)` | a panel that folds to its title |
| Card | `kit.Card(args...)` | a pressable panel, clicked by pointer or keys |
| Row, Column | `kit.Row(args...)`, `kit.Column(args...)` | flex lines, 4 points apart by default |
| Grid | `kit.Grid(columns int, args...)`, `kit.Columns(tracks string)` | children in rows of columns |
| spacing | `kit.Gap(px)`, `kit.Padding(px)` | the gap and padding of any box |
| placement | `kit.Expand()`, `kit.Fill()`, `kit.AlignStart/Center/End()`, `kit.JustifyStart/Center/End/Between()` | flex children and their line |
| Label | `kit.Label(text, args...)`, `kit.Wrap()`, `kit.Truncate()`, `kit.Center()`, `kit.Right()`, `kit.Caps()` | one line of text unless Wrap |
| RichText | `kit.RichText(text, args...)`, `kit.SetRich(n, text)` | text with inline tags |
| Button | `kit.Button(text, args...)`, `kit.Primary()`, `kit.Destructive()` | a push button |
| Check, Switch | `kit.Check(text, args...)`, `kit.Switch(text, args...)` | on/off; read with `doc.Checked(n)` |
| Slider | `kit.Slider(value, args...)`, `kit.Range(lo, hi)`, `kit.Step(s)` | a value in a range; `doc.Value(n)` |
| NumberField | `kit.NumberField(value, args...)` | a number typed or stepped |
| Progress | `kit.Progress(value, args...)`, `kit.SetProgress(p, value)`, `kit.ShowValue(on)` | a bar, 0–100 unless `kit.Range` |
| TextInput | `kit.TextInput(text, args...)`, `kit.Password()`, `kit.MaxLength(n)`, `kit.ReadOnly()` | one line of text; `doc.Value(n)` |
| SearchField | `kit.SearchField(text, args...)` | a text input with a clear button |
| TextArea | `kit.TextArea(text, args...)`, `kit.Rows(n)`, `kit.NoWrap()` | several lines |
| Choice | `kit.Choice(selected, items, args...)`, `kit.ChoiceItem(content, args...)`, `kit.Chosen(c)`, `kit.SetChosen(c, i)` | a segmented pick of one |
| PopUp | `kit.PopUp(selected int, items []string, args...)` | a select of strings |
| Form | `kit.Form(rows...)`, `kit.Field(label, field)` | labels and fields lined up |
| Tabs | `kit.Tabs(kit.Tab(title, page, args...)...)`, `kit.SelectedTab(doc, tabs)`, `kit.SetSelectedTab(doc, tabs, i)` | one page at a time |
| Table | `kit.Table(columns, rows []*ui.Node, args...)`, `kit.TableColumn(title, args...)`, `kit.TableRow(cells...)` | data in sortable rows |
| Scroll | `kit.Scroll(args...)`, `kit.VerticalBar(m)`, `kit.HorizontalBar(m)` | a scrolled area |
| Separator | `kit.Separator(args...)`, `kit.Vertical()` | a line between groups |
| Image | `kit.Image(src, args...)`, `kit.TileImage(src, args...)`, `kit.NinePatch(src, l, t, r, b, args...)` | pictures from `ui.Images` |
| Tooltip | `kit.Tooltip(tip *ui.Node) ui.Arg` | help beside an element on hover or focus |
| Hint | `kit.Hint(kit.KeyboardHint(s), kit.GamepadHint(s), kit.MouseHint(s), kit.TouchHint(s))`, `kit.SetHint(h, device, s)` | help for the device used last |
| StatusBar | `kit.StatusBar(text, args...)`, `kit.SetStatus(bar, text)` | a line along a window's foot |
| Alert | `kit.Alert(title, message string, buttons ...*ui.Node)`, `kit.SetAlert(a, title, message)` | a question answered by its buttons |
| Window | `kit.Window(title, args...)`, `kit.Toolbar(args...)`, `kit.Footer(args...)`, `kit.Resizable()` | an in-page window, dragged by its bar |
| Menus | `kit.MenuBar(kit.MenuBarItem(title, kit.Menu(kit.MenuItem(text, kit.Shortcut("Cmd+S"))...))...)`, `kit.MenuButton(text, menu)` | app menus |
| Sidebar, Thumbnails | `kit.Sidebar(kit.SidebarSection(title, kit.SidebarItem(text)...)...)`, `kit.Thumbnails(kit.Thumbnail(name, src)...)` | tool screens |
| FileDialog | `kit.NewFileDialog(doc, fsys, opts...) *kit.FileDialog` | open and save, an ember dialog, never the OS's |

## The CSS subset

Full table: `docs/05-ui/03-css.md`. Lay screens out with flex or grid.

- **Works:** selectors `div .c #id [a="v"] a b a>b`, `:hover :focus :focus-visible :active :disabled :checked :open :first-child :last-child :nth-child() :not() :root`; `px % em rem fr`, `calc() var() minmax() repeat()`.
- **Layout:** `display` block, inline, inline-block, flex, inline-flex, grid, none, contents; `position` up to sticky; `overflow`; `z-index`; flex and grid in full.
- **Look:** backgrounds and linear, radial and conic gradients, `border-radius`, `border-image`, `box-shadow`, `opacity`, `transform`, `transition`, `cursor`, `pointer-events`, `image-rendering: pixelated`, `-webkit-font-smoothing: none` for pixel fonts.
- **Refused** (a warning): `float`, `vertical-align`, `columns`, the `font` shorthand, `text-decoration`, `animation`, `@media`, `@import`, several background layers.
- `px` is a point, not a device pixel; `box-sizing` is border-box by default.

## React to the player

Callbacks go on the element: `ui.OnClick(fn)`, `OnInput`, `OnChange`, `OnSubmit`, `OnToggle`, `OnClose`, `OnSort`, `OnKey`, `OnBack`, `OnContextMenu`, `OnDrop`, `OnDragEnd`, `OnPointerEnter`, `OnPointerLeave`. `fn` is any of `func()`, `func(ui.Event)` and `func(*world.World, ui.Event)`.

| Need | API |
|---|---|
| what happened | `ui.Event{Type, Node, Clicks, Value, Key, Text, …}`: `Clicks` 2 or 3 for double or triple; `Value` a closed dialog's |
| a callback on an existing node | `remove := ui.On(n, ui.EventClick, fn)` |
| stop a Back or key going up | `e.StopPropagation()`; Back and keys bubble from the focused element to its ancestors, clicks don't |
| from code that didn't build the element | World events `ui.Clicked`, `Changed`, `Edited`, `Submitted`, `Toggled`, `BackPressed`, `Closed` |

- A click reaches the innermost element with behaviour: a control, a canvas, a `label`, a `summary` or an element with a `tabindex`. A plain `div` needs `ui.TabIndex(0)` for its OnClick.
- What a callback changes shows the next frame.
- Values: `doc.Value(n)`, `doc.SetValue(n, v)`, `doc.Checked(n)`, `doc.SetChecked(n, on)`, `doc.Selected(n)`, `doc.SetSelected(n, i)`. The setters emit no event.

## Change a screen at run time

| Need | API |
|---|---|
| text | `n.SetText(s)`, read with `n.Text()` |
| attributes, style | `n.SetAttr(name, v)`, `n.RemoveAttr(name)`, `n.SetStyle(css)` |
| classes | `n.AddClass(c)`, `n.RemoveClass(c)`, `n.HasClass(c)` |
| hide, show | `n.SetAttr("hidden", "")`, `n.RemoveAttr("hidden")` |
| add, remove | `n.Append(children...)`, `n.Remove(child)` |
| dialogs | `doc.ShowModal(d)`, `doc.Show(d)`, `doc.Close(d, value)`, `doc.Open(d)`: a `kit.Alert` or `ui.Dialog` appended to the body first |
| redraw a canvas | `doc.Invalidate(canvas)` |
| a tag that follows a 2D entity | `ui.Follow(target twod.Entity, offset geom.Vec2) ui.Arg`, `ui.SetFollow(n, target, offset)` (`examples/16-hud-over-world`) |

## Draw your own: ui.Canvas

`ui.Canvas(draw func(c *twod.Canvas), args ...ui.Arg) *ui.Node` is an element the game draws (`examples/17-custom-elements`).

- `draw` runs in the content box: the origin is its top-left, and `c.Size()` is the size the layout gave it.
- `draw` runs when the canvas first shows, when its box moves or resizes, and after `doc.Invalidate(n)`. Otherwise the last drawing stays.
- Draw with `c.Rect(r, col)`, `c.RoundRect(r, radius, col)`, `c.Circle(center, radius, col)`, `c.Line(a, b, width, col)`, `c.Border(r, radius, thickness, col)`, `c.Text(at, text, opts...)`, `c.Image(r, tex, uv, tint)`.
- Size it with CSS (`#board { width: 256px; height: 256px; }`); without it, a canvas is 300 by 150.
- `ui.OnClick` on a canvas runs on a click on it; `doc.PointerOn(n) (geom.Vec2, bool)` is then the pointer in the canvas's own points, false when it is off the canvas.
- Tell automation what a canvas shows with `ui.Attr("role", "meter")` and `ui.Attr("aria-valuenow", v)`.

## Focus and keys

| Need | API |
|---|---|
| give or read the focus | `doc.Focus(n)`, `doc.Focused(n) bool`: an element shown this frame takes it at its first layout |
| one Back for a whole screen | `ui.OnBack(fn)` on its root; with nothing focused, Back reaches the body's: `ui.On(doc.Body(), ui.EventBack, fn)` |
| a key nothing took | `ui.OnKey(func(e ui.Event) { … e.Key … })` |
| keyboard order | Tab and Shift-Tab once something has the focus; the arrows move it spatially |
| keys left to the game | with nothing focused and no modal open; read `win.Input()` as usual |

Escape and the pad's East are Back. Enter, Space and the pad's South press the focused element. A modal dialog keeps the focus and its window's input until it closes.

## Test a screen

See [testing.md](testing.md): `doc.ByID`, `doc.BoxOf`, simulated clicks, `align.Check(doc)` and `doc.Warnings()`, which every screen's test runs. Automation names an element by the path of its ancestors' IDs, `settings/save` (`examples/18-testing-ui`).

## A game screen: a board and a score

A board on a canvas, toggled by clicks; a side panel with the score, a bar and a restart button that asks first. Escape asks too.

```go
package main

import (
	"fmt"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

const cell, size = 48, 5

const page = `html, body { height: 100%; }
body { display: flex; align-items: center; justify-content: center; gap: 24px; }
#board { width: 240px; height: 240px; }
#side { min-width: 200px; }`

// Board is the rules: plain Go, tested without a window.
type Board struct {
	Lit   [size][size]bool
	Score int
}

func (b *Board) Toggle(x, y int) {
	if x < 0 || y < 0 || x >= size || y >= size {
		return
	}
	b.Lit[x][y] = !b.Lit[x][y]
	if b.Lit[x][y] {
		b.Score++
	}
}

type Screen struct {
	board                   Board
	doc                     *ui.Document
	canvas, score, bar, ask *ui.Node
}

func (s *Screen) draw(c *twod.Canvas) {
	c.RoundRect(geom.Rect{W: c.Size().X, H: c.Size().Y}, 8, color.Hex("#2b2d42"))
	for x := range size {
		for y := range size {
			col := color.Hex("#3d405b")
			if s.board.Lit[x][y] {
				col = color.Hex("#f2cc8f")
			}
			c.RoundRect(geom.Rect{X: float32(x*cell) + 4, Y: float32(y*cell) + 4, W: cell - 8, H: cell - 8}, 6, col)
		}
	}
}

func (s *Screen) click() {
	at, ok := s.doc.PointerOn(s.canvas)
	if !ok {
		return
	}
	s.board.Toggle(int(at.X)/cell, int(at.Y)/cell)
	s.show()
}

func (s *Screen) show() {
	s.score.SetText(fmt.Sprint(s.board.Score))
	kit.SetProgress(s.bar, float64(min(s.board.Score, 25)))
	s.doc.Invalidate(s.canvas)
}

func (s *Screen) answer(e ui.Event) {
	if e.Value == "restart" {
		s.board = Board{}
		s.show()
	}
}

func newScreen() *Screen {
	s := &Screen{doc: ui.NewDocument(kit.Theme(), ui.CSS(page))}
	s.canvas = ui.Canvas(s.draw, ui.ID("board"), ui.OnClick(s.click))
	s.score = kit.Label("0", ui.ID("score"))
	s.bar = kit.Progress(0, kit.Range(0, 25))
	s.ask = kit.Alert("Restart?", "The score goes back to zero.",
		kit.Button("Cancel"), kit.Button("Restart", kit.Primary(), ui.Value("restart")))
	ui.On(s.ask, ui.EventClose, s.answer)
	restart := kit.Button("Restart", ui.ID("restart"), ui.OnClick(func() { s.doc.ShowModal(s.ask) }))
	side := kit.Panel(kit.Title("Score"), ui.ID("side"), kit.Padding(16), kit.Column(kit.Gap(8),
		s.score, s.bar, restart, kit.Hint(kit.KeyboardHint("Click a cell to light it"))))
	s.doc.Body().Append(s.canvas, side, s.ask)
	ui.On(s.doc.Body(), ui.EventBack, func() { s.doc.ShowModal(s.ask) })
	return s
}

func main() {
	win := platform.NewWindow(platform.Title("lights"), platform.Size(640, 360), platform.FixedSize())
	win.Add(newScreen().doc)
	ember.New().AddEntities(win).MustRun()
}
```
