# 0045. ShadedBy draws a light picture per set of groups

- Status: Accepted
- Date: 2026-11-02
- Area: rendering
- Owner: architect

## Context

A monster casting from a box at its feet shaded its own lower body: lit layers read one light picture per window, already summed over every light's shadows, so they can't tell which occluder group darkened a pixel.

## Decision

- `light.ShadedBy(groups…)` on a lit layer or entity picks the occluder groups whose shadows fall on it (all by default), inherited like the light mode: the mirror of `BlockedBy` on a light.
- Each lit vertex carries the groups it leaves out, packed into its light spread, and each list marks which sets it drew (`draw.List.Shades`).
- The GPU draws one more light picture for each set used in a window, up to three. The shadow maps and the directional lights' shadows are redrawn with `groups & mask`, and ambient, emissive and static tiles are drawn again as they are. The 2D shader then picks a picture per vertex through a small shade map.

## Alternatives

- Keeping each group's shadow apart in the light picture's channels: there are three groups but only one colour picture, so lights could no longer be coloured.
- Testing shadows per pixel in the 2D shader against every light's shadow map: the cost grows with the number of lights on every lit pixel, for every drawing, not only those ShadedBy.
- Leaving the monster's own box out with a per-entity exception: other monsters' boxes would still shade it, and every caster would need its owner tracked through the GPU.

## Consequences

A window pays nothing until something uses ShadedBy. After that, each set of groups used costs a copy of the light picture, plus a redraw of only the lights an excluded occluder reaches (043b842f): +10% GPU on the demo's crypt walk. Directional lights still redraw the whole picture. Static lights are baked once with every group, so their shadows fall on everything. The next step, if needed, is to copy only the reached lights' rectangles rather than the whole picture.
