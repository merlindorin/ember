# UI as entities

Status: superseded by [0059](../../99-decisions/0059-ember-ui-is-uiv2.md): ember/ui was removed; the UI is the former uiv2, now `ember/ui` ([ui.md](../ui.md)).

A game builds its UI from ready-made entities, the way it builds the rest of
its world: constructors with options, added to the World, read and changed
by systems, with interaction arriving as World events. The immediate
`world.Resource[ui.UI]` calls stop being what games write.

```go
logs := ui.NewLabel("No logs on the fire", ui.At(16, 16))
add := ui.NewButton("Add a log", ui.Anchor(ui.Center), ui.Size(160, 36))

ember.New(ember.WithTitle("ember · menu"), ember.WithSize(640, 360)).
	AddEntities(logs, add).
	AddSystems(&Fire{label: logs, add: add}).
	MustRun()

type Fire struct {
	label  *ui.Label
	add    *ui.Button
	logs   int
	clicks world.Reader[ui.Clicked]
}

func (s *Fire) Update(f world.Frame) {
	for click := range s.clicks.Read(f.World) {
		if click.Button == s.add {
			s.logs++
			s.label.SetText(fmt.Sprintf("%d logs on the fire", s.logs))
		}
	}
}
```

## Packages

| Package                      | Holds                                                                                                                                               |
|------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------|
| `ui`                         | Components, ready-made entities, layout, themes, events, and the ui system (`ui.Plugin`, installed by `ember.New`).                                 |
| `ui/paint`                   | Today's immediate drawing, moved as is: `Canvas`, text and fonts, shapes, orbs, bars. For the ui system, custom painters, debug overlays and tools. |
| `node/canvas`, `node/node2d` | Unchanged in role; they draw through `ui/paint`.                                                                                                    |

`ui` imports `node` and `node/canvas`. `node/canvas` imports only
`ui/paint`, so there is no cycle. All of it stays pure Go.

## One model with scene nodes

The `control` package goes away and its types become the ui entities.
- **Embedding.** Every ready-made entity embeds `*world.BasicEntity`, `node.Node`, `canvas.CanvasItem` and its ui components.
- **As a scene node.** It saves and loads as `ui.Button`, `ui.Row`… in a scene file, with the same properties as today's control types.
- **As a plain entity.** Added straight to the World, it works without a tree. The ui system only uses the World's parent/child links, which `node` keeps in step inside a tree.
- **What moves.** Control's layout, themes, BBCode text, translation (`tr` and `TranslationServer`) and its tests move into `ui`.
- **Scene files.** Classes are renamed `control.X` → `ui.X`. No scene files are committed, so nothing needs migrating.

## Components

Every element embeds `Element`: the entity, the node, the canvas item, and
two components, `Layout` and `Themed`. Other components hold what kinds of
elements share. A game can build its own elements from them, and the ui
system treats them like the ready-made ones.

| Component | Data | Setters |
|---|---|---|
| `Layout` | anchors and offsets (set by `At`, `Anchor`, `Size`), minimum size, size flags, clipping | `SetAt`, `SetSize`; `Rect()` reads the last layout |
| `Themed` | the element's theme, shared with its descendants, and its own overrides | `SetTheme` |
| `Press` | disabled, toggle mode, pressed | `Enable`, `Disable` |
| `Ranged` | value, minimum, maximum, step | `SetValue` |

Texts, textures and container settings are fields of the elements, with
setters (`SetText`, `SetTexture`); `Show` and `Hide` are on every element.

**Custom drawing.** An element that implements
`Draw(u *paint.UI, name string, r geom.Rect)` draws itself through
`ui/paint` and records itself for automation with `u.Record` or `u.Expose`.

**A game's own elements** embed an `Element` made by `ui.Base()` and are
finished by `ui.Build`, which names them and applies their options, so they
take the same options as the ready-made ones:

```go
type Hearts struct {
	ui.Element
	Full, Max int
}

func newHearts(opts ...ui.Option) *Hearts {
	return ui.Build(&Hearts{Element: ui.Base()}, "hearts", opts...)
}

func (h *Hearts) MinimumSize() geom.Vec2 { … }
func (h *Hearts) Draw(u *paint.UI, name string, r geom.Rect) { … }
```

## Ready-made entities

| Constructor | Is | Emits |
|---|---|---|
| `NewLabel(text, opts…)` | a text | |
| `NewRichText(text, opts…)` | a text with `[b]`, `[i]`, `[color=…]` | |
| `NewButton(text, opts…)` | a text, Press | `Clicked` |
| `NewCheck(text, on, opts…)` | a text, Press | `Toggled` |
| `NewSlider(value, opts…)` | Ranged | `Changed` |
| `NewProgress(value, opts…)` | Ranged | |
| `NewTextInput(text, opts…)` | an editable text | `Edited`, `Submitted` |
| `NewImage(texture, opts…)` | a texture, stretched | |
| `NewPanel(opts…)` | a themed background; children inside its padding | |
| `NewRow(opts…)`, `NewColumn(opts…)` | children in a row or a column | |
| `NewGrid(columns, opts…)` | children in rows of cells | |
| `NewScroll(opts…)` | its child, clipped and scrolled | |
| `NewElement(opts…)` | a bare box for children placed by anchors | |

Scene files also use `ImageProgress`, `ColorRect` and `NinePatch`, built
the same way. There is no centring container: `Anchor(Center)` centres an
element in its parent or in its container's slot.

**Options** are shared where they make sense:
- placement: `At(x, y)`, `Anchor(a)`, `Size(w, h)`, `MinSize(w, h)`, `Padding(px)`, `Gap(px)`, `Expand()`, `Follow(entity, offset)`;
- naming and state: `Name(id)`, `Hidden()`, `Disabled()`;
- style: `Align(a)`, `WithTheme(t)` (`Theme` is the type), `Face(f)`, `FontSize(px)`, `TextColor(c)`;
- values: `Range(lo, hi)`, `Step(s)`, `Placeholder(text)`, `Margins(l, t, r, b)` for a NinePatch.

There are no exported config structs.

**Containers.** `Add(children…)` appends children and returns the
container, so a menu reads as one expression:

```go
menu := ui.NewPanel(ui.Anchor(ui.Center), ui.Padding(12)).Add(
	ui.NewColumn(ui.Gap(8)).Add(
		ui.NewLabel("Emberfall"),
		play, quit,
	),
)
```

## Events

Each event carries the entity, so a system compares it with the one it holds.

```go
type Clicked struct{ Button *Button }
type Toggled struct{ Check *Check; On bool }
type Changed struct{ Slider *Slider; Value float64 }
type Edited struct{ Input *TextInput; Text string }
type Submitted struct{ Input *TextInput; Text string }
```

Elements detect interaction while the ui system draws them, late in
Update, and emit the events then. Events outlive one frame end, so systems
read them the next frame with a `world.Reader`.

## Layout

- **Which window.** An element added to the World shows in the primary window. An element that is a child of a window entity (`win.AppendChild(label)`) shows in that window, so `For(win)` goes away.
- **Top-level elements** are placed in their window:
  - `Anchor` picks a point: TopLeft (the default), Top, TopRight, Left, Center, Right, BottomLeft, Bottom, BottomRight, or Fill.
  - `At` offsets from that point, in points.
  - `Size` fixes the size; otherwise the content's minimum size is used.
- **Inside Row and Column**, children follow in order, `Gap` apart:
  - `Expand` children share the space that is left;
  - the cross axis follows `Align`.
- **Grid** fills rows of `columns` cells, each row and column as large as its largest child.
- **Panel and Scroll** hold their children inside `Padding`.
- **Pass.** Layout (`ui.Arrange`) runs once per frame in the ui system, from each window's size, before drawing and hit-testing, so `Rect()` always matches what is on screen.

## Themes and style

- **Theme items** are kept by element type (`Button`, `Label`, `Row`) and item name (`font_color`, `separation`, `normal`), as control's themes are today.
- **Lookup order:** the element's `Theme`, then its ancestors', then the World's `ui.Theme` resource, then `ui.Default()`.
- **Per-element overrides:** a few options (`FontSize`, `TextColor`) cover the common cases. Anything more is a theme.
- **Default look:** the default theme matches today's immediate UI.

## Automation IDs

- **IDs are paths of names** from the window down, such as `menu/add-a-log`.
- **An element's name:**
  - is `Name(id)` when given;
  - otherwise it is the slug of the text it was built with, so `SetText` never changes it;
  - otherwise it is the type (`row`, `progress`).
- **Duplicate names among siblings** are numbered: `add-a-log2`.
- **Roles** come from the type (`button`, `label`, `meter`, `checkbox`, `slider`, `textbox`). Values come from the components, or from `Expose` for custom elements.
- **Actions:** `press` on a button reaches the same code as a click and emits `Clicked`.
- **Recording.** The ui system records every visible element each frame through the existing automation recorder, so `GetTree`, `Press` and `SetValue` work unchanged.

## Emberfall's HUD (outline)

The HUD is about 350 lines of immediate calls. It moves panel by panel:
1. **Static panels become entities, built once in `client.New`.** These are the character panel, the skill bar (a Row of slots with a custom `Paint` for cooldowns), the needs and load bars (Progress), and the craft panel (a Panel and Column with Buttons). Its `ButtonAt` checks become a system reading `ui.Clicked`.
2. **Painted parts become one custom element each:** the orbs, minimap, dial, nameplates and floating numbers. Each `Paint` reads game state.
3. **Transient texts** (notices, killer warnings, the death screen) become Labels and Panels that the HUD system shows, hides or adds.
4. **`Expose` calls** become element names and roles. The test IDs (`health`, `mana`, `craft`…) stay the same, because the names are chosen to match.

The game uses no immediate calls once that's done.

## The window examples in the new API

`09-window`:

```go
func main() {
	size := ui.NewLabel("", ui.At(16, 16))

	ember.
		New(
			ember.WithTitle("ember · window"),
			ember.WithSize(640, 360),
			ember.WithBackground(color.Hex("#2b2f3a")),
		).
		AddEntities(size).
		AddSystems(&ShowSize{label: size}).
		MustRun()
}

type ShowSize struct{ label *ui.Label }

func (s *ShowSize) Update(f world.Frame) {
	size := world.Resource[platform.Window](f.World).Size()
	s.label.SetText(fmt.Sprintf("%.0f × %.0f", size.X, size.Y))
}
```

`10-multi-window`: `labels.go` goes away, and each label belongs to its window.

```go
func main() {
	ember.
		New(
			ember.WithTitle("ember · scene"),
			ember.WithSize(900, 600),
			ember.WithPosition(60, 80),
		).
		AddEntities(
			newGround(),
			newCrate(-2, 0),
			newCrate(2, 1),
			&Eye{world.NewBasic(), scene.At(6, 5, 8).LookingAt(geom.Zero3), render.Camera{}},
			ui.NewLabel("scene", ui.At(16, 16)),
			newMapWindow(),
		).
		MustRun()
}

// window.go
func newMapWindow() *Window {
	w := &Window{world.NewBasic(), platform.NewWindow(
		platform.Title("ember · map"),
		platform.Size(400, 400),
		platform.Position(980, 80),
	)}
	w.AppendChild(&Eye{world.NewBasic(), scene.At(0, 20, 0.01).LookingAt(geom.Zero3), render.Camera{OrthoHeight: 12, Window: &w.Window}})
	w.AppendChild(ui.NewLabel("map", ui.At(16, 16)))
	return w
}
```

## What goes away

- **For games:** `world.Resource[ui.UI]` and its widget calls (`Label`, `Button`, `ButtonIf`, `ButtonAt`, `Check`, `Slider`, `TextField`, `Panel`, `Title`, `Bar`, `Orb`, `Tooltip`), together with `For(win)`, `Push`/`Pop` and `Expose`.
- **Moved to `ui/paint`:** the drawing those calls did, for the ui system and custom painters. `ui.CanvasFor` becomes `paint.CanvasFor`.
- **Merged into `ui`:** the `control` package, and the `control.*` scene classes (now `ui.*`).
- **Rewritten:** the immediate calls in 08, 09, 90-pixel-isle, 92-sword and Emberfall. The editor's canvas F3 overlay and debug drawing use `ui/paint`.

## Decided

1. `Image` and `Panel` keep their names.
2. `ui.Follow(entity, offset)` is an option: the element sits at the 3D entity's screen position in its window, plus the offset, for nameplates and floating numbers.
3. Keyboard focus and Tab order came with gamepad navigation; see Focus and navigation.
4. UI scale is per window, from its pixel density: layout is in points.
5. `ui/paint` stays open to games for debug overlays.
6. The orb is game code, drawn through `Paint`; there is no `NewOrb`.
7. The remaining borrowed names change in this work:
   - `canvas.CanvasItem` becomes `canvas.Item` (`GetItem`), and the interface that was `canvas.Item` becomes `canvas.Drawable`.
   - Theme types: `StyleBoxFlat` becomes `BoxStyle` (`Fill`, borders, corner radii, `Padding*`), and the theme's maps are `Colors`, `Constants`, `TextSizes` and `Boxes`.
   - Theme items:

     | Was | Is |
     |---|---|
     | `font_color`, `default_color` | `text_color` |
     | `font_hover_color`, `font_pressed_color`, `font_disabled_color` | `text_hover_color`, `text_pressed_color`, `text_disabled_color` |
     | `font_size` | `text_size` |
     | `separation` | `gap` |
     | `h_separation`, `v_separation` | `column_gap`, `row_gap` |
     | `margin_left`… | `padding_left`… |
     | `panel` | `background` |
     | `normal`, `hover`, `pressed`, `disabled` | `background`, `hover_background`, `pressed_background`, `disabled_background` |

## Focus and navigation

A menu works by keyboard and pad as well as by mouse, so the tutorial's level-up menu and a pause menu need no extra code.

- **Focus:** each window has at most one focused element.
  - Buttons, checks, sliders and text inputs can take focus. A game's own element opts in with `ui.Focusable()`, and `ui.Unfocusable()` skips a built-in one.
  - Every element has `Focus()` and `Focused()`. The World emits `ui.FocusGained` and `ui.FocusLost`.
- **Moving:**
  - Tab and Shift+Tab follow tree order.
  - The arrows, the pad's cross and its left stick move spatially, to the nearest focusable element that way by rectangles.
  - Disabled, hidden and unfocusable elements are skipped.
  - In a focused text input, left and right move the caret instead.
- **Acting:**
  - Enter, Space and `GamepadSouth` press the focused button or toggle the focused check.
  - Left and right step a slider by its `Step`, else a twentieth of its range.
  - Escape and `GamepadEast` emit `ui.BackPressed{Window}`, which a game reads to close a menu.
- **Mouse players first:**
  - A click focuses quietly: the outline doesn't show, and the arrows still belong to the game.
  - Focus becomes active, outlined and capturing, when it's reached by Tab, by navigation, by `Focus()` or by `AutoFocus()`. A mouse press makes it quiet again.
  - Tab and Shift+Tab move the focus only while an element in the window has it. With nothing focused, Tab is the game's, through `win.Input()`; a menu starts its focus with `AutoFocus()`, `Focus()` or a click.
  - `ui.AutoFocus()` on a menu's first button focuses it whenever it appears, so a pad player never needs the mouse.
- **Capture** (decision 0027): while focus is active, ui captures only what it navigates by: the arrows, Tab, Enter, Space and Escape, and the pad's cross, left stick, South and East. Every other key and pad button reaches the game. A focused text field captures the whole keyboard. `Captured(button)` reports it, and `KeyboardCaptured()` means the whole keyboard.
  - Capture set during a frame now applies from the next frame. It used to be reset as each frame began, after the UI had set it late in the previous one, so systems in Update never saw it.
- **Look:** `For[*ui.Button](ui.FocusOutline(box))` is the box drawn around a focused element. The default theme outlines in its accent colour.
- **Automation:** presses work as before. A snapshot marks the focused element with `focused`.
- **Alternatives:**
  - A `ui.Cancel` option on a panel instead of the BackPressed event. The event lets a game decide what back means, closing or asking.
  - Always-visible focus. It would put an outline on every clicked button for mouse players.

## Text editing

A one-line `TextInput` and a many-line `TextArea` edit text the way a desktop does, with one editing core behind both (`internal/textedit`, pure Go), by grapheme cluster as before.

- **Keys follow the platform:** macOS's set on darwin, the Windows and Linux set elsewhere, chosen by `runtime.GOOS`.

  | | macOS | Windows, Linux |
  |---|---|---|
  | character, word | ←/→, Alt+←/→ | ←/→, Ctrl+←/→ |
  | line start, end | Cmd+←/→, Home/End | Home/End |
  | text start, end | Cmd+↑/↓ | Ctrl+Home/End |
  | line up, down (area) | ↑/↓, PageUp/PageDown | ↑/↓, PageUp/PageDown |
  | select | Shift with any move, Cmd+A | Shift with any move, Ctrl+A |
  | delete | Backspace/Delete, Alt+Backspace a word, Cmd+Backspace to line start | Backspace/Delete, Ctrl+Backspace a word |
  | clipboard | Cmd+C/X/V | Ctrl+C/X/V |
  | undo, redo | Cmd+Z, Shift+Cmd+Z | Ctrl+Z, Ctrl+Y or Shift+Ctrl+Z |
  | submit (area, with `SubmitShortcut()`) | Cmd+Enter | Ctrl+Enter |
- **Moving and selecting:**
  - Up and down move by visual line, keeping the column.
  - The mouse places the caret with a click, selects by dragging, takes a word on a double click and a line on a triple click, and extends with Shift+click.
- **Editing:**
  - Typing replaces the selection.
  - Undo groups a typed run, and each other edit is a step of its own.
  - Paste keeps emoji, and keeps line breaks only in the area.
  - Tab inserts a tab in the area, or moves focus with `TabFocus()`.
  - Shortcodes work in both.
- **Options shared by both:** `Placeholder`, `MaxLength(n)` (in characters), `ReadOnly()` (select and copy only), `Password()` (masked, copy and cut blocked), `Shortcodes()`.
- **Area only:**
  - It soft-wraps at words, or scrolls sideways with `NoWrap()`.
  - It scrolls vertically by wheel, trackpad or a draggable scrollbar, and keeps the caret in view.
  - `LineNumbers()` adds a gutter, and `Rows(n)` is its natural height.
  - `TextChanged` and `Submitted` carry the area in `Area`, as they carry an input in `Input`.
- **Looks:** `SelectionColor` and `CaretColor` through `For[...]`. The caret blinks, and the focus outline is as before.
- **Automation:** `type` and `submit` as before, plus `set` (replace the text) and `select-all`. The snapshot's value is the text, or bullets for a password.
- **Not yet:** IME composition, marked text for Japanese, Chinese or Korean input, is left for later. Characters typed through the system still arrive, but not the pre-edit underline.
