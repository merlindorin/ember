# UI components for tool panels, and an inspector example

Status: **built; the inspector example was removed at the user's request (2026-10-31). ember/ui itself was removed by [0059](../../99-decisions/0059-ember-ui-is-uiv2.md); kit's components took its place.**

The user wants dark tool-panel UIs (an inspector: sections, sliders, cards, swatches) built from ember's own ui, with a sample as example 90-inspector on a new guide page, "An inspector". The code stays as simple as an example should be; the theme drives colours, spacing, radii, fonts and the accent. The user's reference images are private and never enter the repo or the docs.

## What exists

Rows, columns, grids and scrolls; titled panels that fold, close, move and resize; pressable cards with hover and press styles; buttons, checks, disabled state, sliders, text inputs, images and colour rects; wrapped labels and rich text; any TrueType face; text tooltips; file drops.

## Direction: boxes and classes, like HTML and CSS (user, 2026-10-31)

"Be inspired more by HTML… the coloured rectangle is a div with CSS." Instead of one element per visual idea, the ui gets one generic box and CSS-like styling:

- **`ui.NewBox(opts…)`, the div.** A box holds children, has no behaviour of its own, and draws what its style says. Panels, cards, swatches, badges and dividers are boxes with different styles.
- **Style properties, inline or in the theme**, with the same names in both: `Background(colour or gradient)`, `Border(width, colour, dashed)`, `Radius`, `Padding`, `Gap`, `Width`/`Height` (fixed, fill, or fit), `Opacity`, `Shadow`. `ui.Style(props…)` on any element is the inline `style=""`.
- **Classes.** `ui.Class("swatch", "selected")` on any element; the theme styles classes as well as kinds, `ui.ForClass("swatch", props…)`, and more specific rules win (kind < class < inline), as in CSS.
- **States as pseudo-classes:** hover, pressed, focus, selected, disabled, checked. `ui.ForClass("card", ui.On(ui.Hover, props…))` is `.card:hover`.
- **Text styles cascade:** colour, size, face, tracking and caps are inherited by children unless a child sets its own, so a section's labels follow it.

Settled points (with game-engine-developer-2):

1. **Three layers, as in browsers:** the engine theme is the user-agent sheet, below inheritance; the game's theme comes next; then inline styles. Text set on a container reaches its labels unless the game's theme or the label itself sets it.
2. **Specificity counts as CSS does:** inline beats classes plus states, which beat a kind. On a tie the later rule wins, and a nested theme beats an outer one.
3. **Resolved once:** each element's style is resolved into a cached value, recomputed only when its classes, state, own style or a theme's generation change, never per frame.
4. **Pointer events:** a plain box is transparent to the pointer. It takes hover and press, and hit-testing, only when a rule uses a state or it has a handler.
5. **A box is a flex container:** `Direction(Row|Column)` (Column by default), `Gap`, `Align`, `Justify`. `Width` and `Height` take a size, `Fill` or `Fit`, and `Grow(n)` is flex-grow, mapped onto today's Size, Fill and Expand so a Grid's rules stay explicit.
6. **One box model:** `BoxStyle` stays the value behind Background and Border, so Panel, Button and theme.json keep working. Choice's cell keys become shorthands for its cell's `:hover`/`:selected`.

What this changes below: items 7 (badge), 9 (divider), 12 (swatch), 14 (dashed border) are no longer elements. They become a box with a class, and their looks come from the theme. `ColorRect` stays as a shorthand for a box with a background. The choice group (item 2) marks its selected item with the `selected` state, so any styled box can be an item. An existing element keeps its options, which become shorthands for style properties, so nothing breaks.

## New components

| # | Component | Shape of the API | Status |
|---|---|---|---|
| 1 | Icons | `ui.LoadIcons(fsys, font, names)` reads an icon font and its name table into an icon set; `ui.NewIcon(set, name)`; an `Icon(set, name)` option on buttons, labels, cards and panel titles. The engine ships no icons; the example loads Lucide (ISC) next to its theme | building |
| 2 | One-of-several choice | `ui.NewChoice(items…)` holds pressable items, keeps one selected, and emits `ChoiceChanged{Choice, Index}`. Its theme gives a selected style (border, fill, text colour), so cards, tabs and swatches only differ by their items | building |
| 3 | Segmented control | the same choice with `ui.Segmented()`, drawn as joined tabs | building |
| 4 | Slider look from the theme | theme keys for the track height and colour, the fill colour, the thumb shape and size, plus `ui.Accent(c)` per slider | building |
| 5 | Letter spacing | `twod.Tracking(points)` text option, also in themes | building |
| 6 | Drag to change a value | `ui.Scrub()` on a label or value: dragging sideways steps the slider or number it is bound to | approved |
| 7 | Badge / tag | a box with class `tag`: border, radius and small text, from the theme | box + class |
| 8 | Reset to default | a slider's `Default(v)` and a reset icon button that shows when the value differs | approved |
| 9 | Divider | a box with class `divider`, Height(1) and the line colour | box + class |
| 10 | Flat section header | a panel title style with no box: caps, tracking, chevron | approved |
| 11 | Toggle card | a panel whose title holds an icon and toggles it on and off, highlighted while on | approved |
| 12 | Colour swatch | a box with class `swatch` and an inline Background (a colour, a gradient, or the checkerboard pattern); its selected outline is `.swatch:selected` | box + class (gradient and pattern backgrounds are new style values) |
| 13 | XY pad | `ui.NewPad2D(opts…)`, a 2D value in a box with grid lines and a dot; `Pad2DChanged` | approved |
| 14 | Dashed border | `Border(width, colour, ui.Dashed(dash, gap))`, a style value | style value |
| 15 | Rich tooltip | `ui.TooltipOf(element)`, any element as a tooltip's content | approved |

Kept in the example rather than the engine: value text formatting ("0.0°", "1.3×"), the hex colour field, undo and redo, the label–badge–value–slider row, and the drop zone's thumbnail and file name.

Each component lands with tests that need no GPU and with its step in the guide (game-doc-writer is briefed at spec time).
