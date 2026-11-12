# CSS

![A card styled by its style attribute: a grid of an orange disc beside a title and a muted line](styled.png "ember · styled")

```go
ui.Div(ui.Style("display: grid; grid-template-columns: 48px 1fr; gap: 4px 12px; border-radius: 8px"),
	ui.Div(ui.Style("grid-row: span 2; width: 48px; height: 48px; border-radius: 24px; background-color: #e0a040")),
	ui.Strong(ui.Text("Ember Quest")),
	ui.Span(ui.Text("Area 2 · 1.5k to home")),
)
```

Stylesheets come from `ui.Stylesheet(fsys, name)`, after the user-agent sheet; `ui.Style(css)` is an element's style attribute, adding to what earlier arguments, such as `kit.Padding`, set. What a sheet gets wrong is in `doc.Warnings()`, never a crash.

| Selectors | |
|---|---|
| `div`, `.class`, `#id`, `[attr]`, `[attr="value"]` | simple |
| `a b`, `a > b` | descendant and child |
| `:hover`, `:focus`, `:focus-visible`, `:active`, `:disabled`, `:checked`, `:open` | states |
| `:first-child`, `:last-child`, `:nth-child()`, `:not()`, `:root` | structure |
| `::placeholder` (color), `::selection` (background-color) | pseudo-elements |

| Values | |
|---|---|
| `px` (a point), `%`, `em`, `rem`, `fr` | units |
| `auto`, `min-content`, `max-content`, `fit-content`, `minmax()`, `repeat()`, `calc()`, `var()` | sizes and functions |
| hex, `rgb()`, `oklch()`, named colours, `transparent`, `currentColor` | colours |
| `linear-gradient(<angle> or to <side or corner>, <colour> [<length or %> [<length or %>]], …)` | in `background-image` or `background`: two stops or more, two positions on one stop a hard edge; `deg`, `rad`, `grad` or `turn`; `to` a corner follows the box's shape; colours blend in encoded sRGB, as Chrome's |
| `radial-gradient([circle or ellipse] [closest-side, closest-corner, farthest-side or farthest-corner] [at <position>], <colour> [<length or %>], …)` | an ellipse to the farthest corner from the middle by default; length stops measure along the horizontal radius |
| `conic-gradient([from <angle>] [at <position>], <colour> [<angle or %> [<angle or %>]], …)` | stops at angles or percentages of a turn, two positions a hard stop: a cooldown sweep is `conic-gradient(transparent 0 70%, rgba(0,0,0,.72) 70% 100%)` |
| | all three blend in encoded sRGB as Chrome's and are cut to the box, rounded or not |

| Properties | |
|---|---|
| layout | `display` (block, inline, inline-block, flex, inline-flex, grid, none, contents), `position` (static, relative, absolute, fixed, sticky), `top` `right` `bottom` `left` `inset`, `z-index`, `box-sizing` (border-box by default), `width` `height` and their `min-` `max-`, `aspect-ratio`, `margin`, `padding`, `overflow` (visible, hidden, clip, auto, scroll) and its axes (clipping on both with a `border-radius` clips to the rounded inner border, the padding box's curve, and a pointer in a cut corner reaches what's under it; nested rounded clips round only the innermost; under rotation or skew the clip is its bounding rectangle, [decision 0058](../99-decisions/0058-rounded-clips-through-a-per-batch-rounded-rect.md)), `visibility`, `order` |
| flex and grid | `flex` and its parts, `gap` `row-gap` `column-gap`, `align-*`, `justify-*`, `place-*`, `grid-template-columns` `-rows`, `grid-column` `grid-row`, `grid-auto-*` |
| text | `font-family` `font-size` `font-weight` `font-style`, `line-height`, `letter-spacing`, `text-align`, `text-transform` (in text fields too: drawn cased, the value kept as typed, the caret and selection following what's drawn), `white-space`, `text-overflow`, `overflow-wrap`, `text-shadow`, `color`, `list-style`, `-webkit-font-smoothing: none` (glyphs unsmoothed on the pixel grid, for pixel fonts) |
| look | `background-color` `-image` `-repeat` `-size` `-position`, `border` and its parts, `border-radius`, `border-image` (nine-patches), `border-image-repeat` and `border-image-width`, `box-shadow`, `outline`, `opacity`, `transform` `transform-origin`, `transition` (colours, opacity, transforms), `cursor`, `pointer-events` (auto, none: the pointer passes through, inherited), `image-rendering` (pixelated), `object-fit` `object-position`, `accent-color`, `caret-color`, `scrollbar-width`, `appearance`, `resize` |
| tables | `border-spacing`, `caption-side` (top, bottom; inherited), `table-layout` (auto, fixed) |

Lay a screen out with flex or grid; tables are for tabular data.

A `border-image` draws its corners and sides `border-image-width` thick and lays its edges and centre along their length, as `border-image-repeat` says, one value for both axes or two (across, then down): `stretch` (the default) stretches the piece, `repeat` lays copies at their size, centred, the ends cut, `round` fits a whole number of copies, scaled, and `space` fits a whole number at their size with the leftover shared out as equal gaps. A patterned frame (ivy, cloth, iron) keeps its pattern with `round`: `border-image: url(frame.png) 8 fill round`.

`border-image-width` takes one to four values, as margins do: a number times the border's width, a length, a percentage of the box's side, or `auto` for the slice's own size. Its initial 1 is the border's widths, so `border: 8px solid` with an 8-texel slice draws the slice as it is, a thinner border draws it thinner, and `border-width: 0 16px` draws no top or bottom row, stretching the slice's middle band over the box. The shorthand takes it after the slice: `border-image: url(frame.png) 8 / 16px`. When opposite sides would overlap, all four shrink by the same factor, and repeated copies scale with their side: an 8-texel slice drawn 16 thick tiles 16-long copies.

| Refused | Use |
|---|---|
| `float`, `clear` | flex or grid |
| `vertical-align` | `align-items` or `align-self` |
| `columns`, `column-count`, `column-width` | grid |
| `border-collapse` | `border-spacing` |
| the `font` shorthand | `font-family`, `font-size`, `font-weight`, `line-height` |
| `text-decoration` | not in the subset |
| `word-wrap`, `grid-gap`, `grid-row-gap`, `grid-column-gap` | `overflow-wrap`, `gap`, `row-gap`, `column-gap` |
| `zoom` | `transform: scale()` |
| `clip` | `overflow: clip` |
| `animation` | `transition` |
| repeating gradients, colour hints, several background layers | one gradient |
| `@media`, `@import` and other at-rules but `@font-face` | not supported |

```
go run ./examples/92-ui-kit -element styled
```
