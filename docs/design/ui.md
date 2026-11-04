# uiv2: modern HTML elements and a CSS subset, written from Go

The package this note calls `uiv2` is `ember/ui` since decision [0059](../99-decisions/0059-ember-ui-is-uiv2.md); its `ember/ui` is the old package, removed then.

Status: **design, approved by the user (2026-11-02), decision 0049.** It replaces `ember/ui`'s hand-sized layout once it covers the tileset tool, the dialogs and the demo's screens.

## Why

`ember/ui` sizes layout by hand: a column is "150 points", a field is "Fill+Expand", and nothing stops content from overflowing its parent. Each screen fixes its own widths, and each fix shows up as a new misalignment or a scrollbar where none is meant. CSS already solved this: `grid-template-columns: max-content 1fr`, `min-width: 0` and `text-overflow: ellipsis` rule out the whole class of bug. uiv2 takes that model, and its names, without becoming a browser.

## What it is, and what it is not

- **Written, never read.** The game builds the tree in Go. There is no HTML parser, no DOM query API and no scripting.
- **Modern elements only.** `div section header footer nav main aside article p h1…h4 span strong em code label button input(text number range checkbox search password) select option textarea details summary dialog table thead tbody tr th td progress meter img hr canvas` (a canvas is drawn by the game's `func(*twod.Canvas)`, again when `Invalidate` asks). Nothing presentational (`i b u center font big small`).
- **A modern CSS subset**, written as CSS text: stylesheets (files next to the game, like themes today) plus a `style` attribute.
  - Selectors: type, `.class`, `#id`, descendant, child `>`, attributes `[attr]` and `[attr="value"]` (exact match only), `:hover :focus :focus-visible :active :disabled :checked :open :first-child :last-child :nth-child() :not()`, specificity, inheritance.
  - Values: `px` (= points), `%`, `em`, `rem`, `fr`, `auto`, `min-content`, `max-content`, `fit-content`, `minmax()`, `repeat()`, `calc()`, `var()`, colours (hex, `rgb()`, `oklch()`, the CSS named colours, `transparent`, `currentColor`), `!important`.
  - Layout: `display: block | inline | inline-block | flex | inline-flex | grid | none | contents`, `box-sizing` (border-box by default), `width height min-* max-*`, `margin padding border gap`, `flex` (grow, shrink, basis, wrap, direction, align, justify), `grid-template-columns/rows`, `grid-column/row`, `place-*`, `align-self`, `aspect-ratio`, `overflow` (visible, hidden, clip, auto, scroll) and its axes, `position: static | relative | absolute | fixed | sticky`, `inset`, `z-index`.
  - Text: `font-family` (TrueType faces by file, as themes do), `font-size font-weight line-height letter-spacing text-align text-transform white-space text-overflow overflow-wrap color`. `text-transform` cases what text shows, words split as Unicode splits them for `capitalize`, and keeps the source for automation and the node's text; edit fields (input, textarea) draw their text transformed too, while their value stays as typed (5178fa6e).
  - Look: `background-color` and images (nine-patch as `border-image`), `linear-gradient()` as a `background-image` or in `background` (an angle in deg/rad/grad/turn or `to` a side or corner, two colour stops or more at lengths or percentages, two positions per stop for hard edges; blended in encoded sRGB as Chrome does; no repeating or radial gradients, no transition hints, one background layer), `border` with `border-radius`, `box-shadow`, `outline`, `opacity`, `cursor`, `pointer-events` (`auto` or `none`, inherited: hit testing passes through), `transform` (2D), `transition` on colours, opacity and transforms.
  - Added for M3 (user and architect, 2026-11-02): `text-shadow`, `image-rendering: pixelated`, `accent-color`, `caret-color`, `::selection { background-color }`, `::placeholder { color }`, `object-fit`, `object-position`, `background-repeat`, `background-position`, `background-size` (lengths or auto, a 2x picture drawn at its point size), `border-spacing`, `resize` (Chrome's grip; ember draws none), `scrollbar-width` (`none` scrolls without bars). A range's `background-color` paints its track and its `color` its idle thumb, a progress or meter's `background-color` its track, and an open select's highlighted option takes its `accent-color` (documented conventions, ember only; export has Chrome draw ranges and progress bars the same way).
  - Alignment (kit): controls sharing a flex row (buttons, selects, text fields, switches, search fields) are one height, a control's label sits in the middle of its padding box (across too for a centred button), and a row's single-line text shares a baseline, each within 0.5pt; kit's TestKitPagesLineUp checks every kit page (internal/align). A control, or a row of them inside a row, keeping a height of its own (a taller one on purpose, a window's lights) carries class `own-height`.
  - Tables: the `<table>` elements (`caption`, `thead`, `tbody`, `tfoot`, `tr`, `th`, `td`) with `border-spacing`, `caption-side` (top or bottom) and `table-layout` (auto or fixed); `colspan` and `rowspan`; the caption sits outside the table's border, background and padding, as in Chrome. Collapsed borders are refused.
  - Not supported: floats, `display: table` on other elements (tables as layout: use grid), multi-column, `vertical-align` tricks, old properties.
- **Pure Go, no cgo**, like `ui`, so servers and tests stay GPU-free. It draws through `twod.Canvas`.

## Shape of the API

```go
doc := uiv2.NewDocument(uiv2.Stylesheet(fsys, "tool.css"))
win.Add(doc)

doc.Body().Append(
	uiv2.Section(uiv2.Class("inspector"),
		uiv2.Details(uiv2.Open(),
			uiv2.Summary(uiv2.Text("tile 0")),
			uiv2.Div(uiv2.Class("props"),
				uiv2.Label(uiv2.For("solid"), uiv2.Text("solid")),
				uiv2.Input(uiv2.ID("solid"), uiv2.Type("checkbox"), uiv2.Checked(true)),
			),
		),
	),
)
```

```css
.props { display: grid; grid-template-columns: max-content 1fr; gap: 6px 12px; }
.props > * { min-width: 0; }
```

- Elements are plain values in a tree owned by one `Document` entity per window. That's cheaper than an entity per node, and the tree is the unit that's laid out.
- Constructors take attributes and children as options (decisions 0007, 0018). Setters change a live node (`SetText`, `SetAttr`, `AddClass`, `RemoveClass`, `Append`, `Remove`), and the next frame restyles and relays only what changed.
- Interaction is World events, as in `ui`: `uiv2.Clicked`, `Changed`, `Input`, `Submitted`, `Toggled` (details), `Closed` (dialog), `Sorted` (grid), each carrying the node, and `KeyPressed` for a key no control, menu, grid, listbox or focus move took, on the focused element, else the open modal dialog (none with neither: those keys are the game's). Focus, Tab and spatial navigation, pads, the clipboard, text editing (`internal/textedit`) and automation IDs (the path of names; an element's own part is its `id`, else its `data-name`, `aria-label`, text, label, placeholder, with `aria-hidden` text left out) are reused from `ui`. An element's `data-name` is an automation-only name, like `ui`'s `Name`: its leaf when it is recorded, and the scope of its descendants' IDs, without the uniqueness of an `id`. Automation's press on a grid row or listbox option selects and activates it; select only selects. A Document with an open modal dialog owns its window's input (decision 0053): from `ShowModal` to its close, other Documents, ember/ui and twod widgets in that window get none, and game systems read it as captured. While one of its elements (not its page, nor a `pointer-events: none` box) is under the pointer, a Document covers the layers drawn before it for the pointer, through the next frame (decision 0055): ember/ui and twod widgets under an open menu neither hover nor click; keys are untouched. `ShowModal` runs HTML's dialog focusing steps (its autofocus, else its first control, else the dialog itself, ringed); `Show` doesn't, so a tool window opened with it leaves the focus where it was. Menu items of role menuitemcheckbox turn when chosen and menuitemradio check among their group (Document.Checked reads them, automation records the state as their value); an item's aria-keyshortcuts ("Cmd+S", Cmd being Command on macOS and Control elsewhere; kit.Shortcut sets it) chooses it whenever its Document is drawn and not under another's modal, even with nothing focused, the one exception to unfocused keys being the game's: a focused text field keeps its editing keys, an open menu its moving keys, the keys are captured ahead while their modifiers are held, and of two Documents in a window the last drawn chooses, the frame after.
- A node's box is readable after layout (`n.Box()`), for tests and for games that place things by it.

### Stateful kit components

Most kit components are node builders: their behaviour is uiv2's (a grid, a listbox, a window) and the game reads the events. A component with logic of its own, such as `kit.FileDialog`, is a value made from the Document (`kit.NewFileDialog(doc, fsys, opts…)`, which appends its node to the body) that the game adds to the World as a system, as `collide.NewTracker`. Its `Update` reads uiv2's events and filters them by its subtree, so two such components, or one and the game's own UI, never cross. Two rules:

- **Lifetime follows the node:** while its node is out of the Document, or the Document out of the World, it reacts to nothing, emits nothing and drops what it holds (listings, fs handles); appending the node again brings it back.
- **Its events carry it:** `kit.FileChosen{Dialog}` and `kit.DialogCancelled{Dialog}` tell two dialogs on one page apart. They are emitted in `Update` and read the next frame, like uiv2's.

Its options are typed (`kit.FileDialogOption`), each with a getter of the same name, and setters only where a runtime change is meant (decision 0018). Table sorting stays root behaviour.

## Pipeline per frame

1. **Style:** match selectors and cascade into computed styles, only for nodes marked dirty (an attribute, class, state or sheet change).
2. **Layout:** block, inline, flex and grid formatting contexts with intrinsic sizing, porting Taffy's algorithms (MIT; the reference Bevy and Servo use) into `uiv2/layout`, a pure package with no twod. Text measures through the twod faces.
3. **Paint:** walk the box tree in stacking order into `twod.Canvas` draw lists: backgrounds, borders, shadows, text, clipping for `overflow`, and scroll containers with `ui`'s scrollbars.
4. **Hit testing and events:** from the box tree.

## The reference: Chrome

`uiv2/export` writes any tree as a real HTML page with its stylesheets. Under the `chrome` build tag, a test opens it in headless Chrome, reads every element's `getBoundingClientRect()` and compares it with `n.Box()`, within 0.5 px. Every layout feature lands with Chrome cases, and a screen counts as done only when Chrome and uiv2 agree. That's the end of "laid out the way the author thought". It's a test tool only; the engine never ships a browser.

Known differences, kept on purpose: a missing picture draws nothing (Chrome draws a broken-image icon with the alt text); automation reads alt.

## Packages

| Package | Holds |
|---|---|
| `ember/uiv2` | Document, elements, attributes, events, setters, controls' behaviour |
| `ember/uiv2/css` | tokenizer, parser of the subset, selectors, cascade, computed values |
| `ember/uiv2/layout` | box tree; block, inline, flex, grid; intrinsic sizes (pure) |
| `ember/uiv2/paint` | box tree to `twod.Canvas`, stacking, clipping, scroll |
| `ember/uiv2/export` | tree to HTML for the Chrome reference tests and for docs |

## Team and order

The ui lead owns `ember/uiv2` and this note's open questions, reviews every merge, and keeps the Chrome harness green.

| Milestone | Who | Done when |
|---|---|---|
| M0 skeleton: Document, five elements (div, span, p, button, input text), export, the Chrome harness | lead | a three-element page matches Chrome |
| M1 layout: block, inline text with wrapping and ellipsis, flex, grid, intrinsic sizes, overflow | ui developer 1 | Taffy's flex and grid fixtures ported and matching Chrome |
| M1 css: parser, selectors, cascade, var/calc, states | ui developer 2 | specificity and inheritance cases match Chrome's computed styles |
| M1 paint and controls: paint, scroll, focus, the form controls, details, dialog, table | ui developer 3 | every element's look and keyboard behaviour tested |
| M2 the tileset inspector rebuilt on uiv2 | lead + all | it matches the approved layout at 1100×700 and 1280×800, with Chrome agreeing |
| M3 FileDialog, settings and the demo's screens; `ui` layout retired | all | no screen left on `ember/ui` layout |

## Open questions (the lead's)

- Animation beyond `transition`: likely `@keyframes` later, for game menus.
- Whether `ember/ui`'s element types stay as thin wrappers over uiv2 during M3, or games move to uiv2 directly.
- Right-to-left text: out of scope until a game needs it.
- `placeholder` on `<select>` (lead, 2026-11-02): kept, as a uiv2 extension. A select with a placeholder starts with nothing chosen and shows the placeholder dimmed. Export writes it as the HTML pattern Chrome shows the same way, `<option value="" disabled selected hidden>…</option>`, so pages stay valid HTML and layouts compare.
- The default sheet (lead, 2026-11-03) follows Chrome's user-agent sheet, from Chrome's computed values on a bare page (hr, p, headings, monospace, disabled controls, dialogs and `:modal`, `::backdrop`, cursors, body's 8px margin). One exception: text inputs keep `overflow: hidden` rather than Chrome's `clip`, because uiv2 scrolls an input's own box to keep the caret in view, where Chrome scrolls an inner editor.
- Characters a face lacks (lead, 2026-11-02): they come from Go Mono, which has the arrows, triangles and ticks Go has not. Export gives Chrome the same face for exactly those characters through `unicode-range`. Lines stretch to fit the fallback's metrics, as Chrome's do (inline layout).
