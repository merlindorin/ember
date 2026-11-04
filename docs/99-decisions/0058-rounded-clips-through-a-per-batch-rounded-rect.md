# 0058. Rounded clips through a per-batch rounded rect

- Status: Accepted
- Date: 2026-11-03
- Area: rendering
- Owner: architect

## Context

`overflow: hidden` with `border-radius` clips content to the rounded inner border in Chrome, but a clip in ember was only ever a scissor rectangle, so a scrolled list or an image in a rounded panel showed square corners.

## Decision

- A clip can be rounded: `twod.Canvas.PushClipRounded(r, radii)` (top-left, top-right, bottom-right, bottom-left), popped by `PopClip`. Each draw batch carries the innermost rounded clip over it beside its clip rectangle.
- The GPU keeps the rectangle as the scissor and multiplies each fragment's coverage by an anti-aliased rounded-rect distance, read from the batch's uniforms. Batches without one skip it.
- ui clips an element whose overflow clips on both axes to its padding box rounded by its inner radii (the border radius less the border), as Chrome does; hit tests follow the same curve.
- Limits: only the innermost rounded clip is rounded, outer ones clip as rectangles; under a rotation or a skew the clip falls back to its bounding rectangle, unrounded.

## Alternatives

- A stencil buffer in 2D passes: exact for any nesting, but a stencil attachment and a stencil write and test for every clip push, in every 2D pass.
- The mask-layer passes (`AsMask`/`MaskedBy`): an offscreen pass per clipped element, far too heavy for a list in a panel.

## Consequences

Rounded panels clip as Chrome does at the cost of two more vectors in each batch's uniforms and a distance in the fragments of rounded batches. Content in both an inner and an outer rounded corner at once is cut square by the outer one.
