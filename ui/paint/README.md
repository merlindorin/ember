# ui/paint

Draws a laid-out `box.Tree` onto a `paint.Surface` (`*twod.Canvas` is one), in CSS paint order. `HitTest`, `BarsOf` and `Scrolling` share that order and the old ember/ui's scrollbars.

Controls draw the marks that `appearance: none` leaves to the engine: a checkbox's tick or dash, a switch's thumb (`input[type=checkbox][switch]`), a range's track and thumb, progress and meter fills (a meter's coloured by region), a select's arrow, a summary's triangle, a focused field's caret and selection, a modal dialog's backdrop, which also takes the page's pointer, a canvas element's drawing, which its machine's `Draw` makes in its content box, and an open select's list, drawn over everything and read back by `PickOption`. They read the live control from `Controls(find)`, or else the node's attributes and state. Their boxes, borders and colours come from CSS.

## Known gaps

Each one is either closed or kept on purpose, with its reason, before M3.

| Gap | Today | Closing it needs | Status |
|---|---|---|---|
| Elliptical corners (`border-radius: a / b`) | each corner is round, the smaller of its two radii; corners differ freely (pixel case `internal/pages/radii`) | an elliptical corner shape in the shader | open |
| Group `opacity` | multiplies the alpha of every colour in the subtree, so overlapping children show through each other | drawing the subtree into a pass and compositing it once | open |
| `border-style: dashed` and `dotted` on a rounded box | square boxes are dashed and dotted as Chrome lays them (pixel case `internal/pages/dashes`); a rounded box's dashed or dotted border is drawn solid | dashes along the corner arcs | open |
| Antialiasing of a rounded inset shadow's outer arc | the shadow is clipped to the rounded padding box by polygon, so its arc edge is hard; a border drawn over it usually hides it (pixel case `internal/pages/insets`) | coverage at the arc, or a rounded clip | open |
| Rounded border with sides that differ | square rects, one per side, ignoring the radius | per-side strokes of the rounded outline | open |
| Clips of hoisted positioned descendants | every ancestor's overflow clip applies, including those outside their containing block | clipping only by the ancestors up to the containing block | open |
| Sharp shadow under a translucent rounded box | blurred shadows and sharp ones of square boxes are clipped to outside the border box as Chrome's (pixel case `internal/pages/insets`); a sharp shadow of a rounded box is one fill that shows through it | the sharp shadow as a mesh clipped by the rounded border box | open |
| Caret blink restarts on a move | the caret blinks on the half-second from `Time`, as the old ember/ui's did, even mid-typing | the caret's last move from the control | open |
| Open select's list under a transform | placed and clamped in the root's space, then drawn under the select's transform | placing it in the transformed space, or dropping the transform as browsers do for pickers | open |
| Rounded corners on a split inline | each line's fragment is square | the radius on the first fragment's left corners and the last's right ones, as box-decoration-break: slice does | open |
| A descendant's ink far out of a row out of view | a long list or table walks the rows its scrollport reaches widened by their own ink and 48 points (`inkMargin`); a descendant's shadow, outline or glyphs reaching more than that out of a row out of view are not drawn | a per-subtree ink bound kept by layout, or walking the rows a larger margin reaches | open |
| Open select in a row scrolled out of view | a long list or table walks only the rows its scrollport reaches (`reachable`), so a select open in a skipped row registers no list, though the list hanging below it could reach into view | walking the row holding an open select as `Keep` does the focused box, or closing a picker whose select scrolls out, as Chrome keeps it anchored | open |
| Smooth scaling of pictures | linear sampling, softer than Chrome's upscaling filter; `image-rendering: pixelated` matches it exactly (pixel case `internal/pages/pixelated`) | Chrome's resampling, or mipmaps for downscaling | open |
| `border-image` slices against border widths | each slice drawn at its own size, as when the border width equals the slice | slices scaled to the border widths, as `border-image-width` defaults | open |
| Blurred `text-shadow` | copies of the text over a grid of up to 7×7 taps across ±1.5σ, weighted by the Gaussian; sharp shadows are exact (pixel case `internal/pages/shadows`) | blurring the glyphs’ coverage in a pass | open |
| Opacity over a canvas element | the game's `Draw` paints its own colours; an ancestor's `opacity` below 1 doesn't fade them | a twod.Canvas modulate setter over draw.List's `SetModulate` (twod, outside ui) | open, asked when a screen needs a faded canvas |
