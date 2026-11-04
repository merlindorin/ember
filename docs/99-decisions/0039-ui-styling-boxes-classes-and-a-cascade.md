# 0039. ui styling: boxes, classes and a cascade

- Status: Accepted
- Date: 2026-10-31
- Area: api
- Owner: user

## Context

The user wants tool-panel UIs built as HTML is: "the coloured rectangle is a div with CSS". Until now every visual idea needed an element of its own, and a theme could style only kinds of elements.

## Decision

- `ui.NewBox` is the div: a flex container with no behaviour, lining its children up (`LineUp(Vertical|Horizontal)`, `Gap`), placing them (`Justify`, `AlignItems`), sizing them (`Width`/`Height` with `Points(n)`, `Full`, `Fit`, and `Grow(n)` as flex-grow), and drawn as its style says.
- `ui.Class(names…)` on any element; `ui.ForClass(name, props…)` in a theme. `ui.Style(props…)` is the inline style. Every property has the same name in all three.
- States (`StateHover`, `StatePressed`, `StateFocus`, `StateSelected`, `StateDisabled`, `StateChecked`) are styled with `ui.On(state, props…)`. Each property goes under its state's key, the names themes already used (`hover_background`, `text_hover_color`, `cell_selected`), so old themes and files keep working.
- Three layers, as in browsers: the engine's theme is the user-agent sheet, below inheritance; the game's themes next; inline styles on top. Text (colour, size, face, tracking) is inherited by children that no game rule styles.
- Specificity counts as CSS does: a class beats a kind across all the game's themes. On a tie a later rule wins, then a nearer theme.
- Each element's resolved style is cached and recomputed only after a change that may alter it (a theme's item, a class, a state, a theme set): one generation counter.
- A plain box is transparent to the pointer. It hovers and presses only when its style has a hover or pressed look.
- `BoxStyle` stays the box model behind `Background`.

## Alternatives

- One element per visual idea (Badge, Divider, Swatch…): many types, each with options for the same few looks, and nothing to share a look between kinds.
- A CSS text parser: a second language to learn and validate, with no type checks; options in Go and keys in JSON say the same.
- Resolving the style every frame without a cache: simpler, but slower with a few hundred elements, the case the user's inspector needs.
- State selectors (`Button:hover`) inside the cascade: closer to CSS, but a game's plain `Button` background would then hide the engine's hover look, changing every themed example.

## Consequences

Existing examples and the demo render byte-identically, and a 300-element frame is faster (about 1.6 ms against 2.0 ms). Badges, dividers, swatches and dashed borders become boxes with classes (phase 2). A state change clears every cached style, which costs one fresh resolution of what is drawn that frame.
