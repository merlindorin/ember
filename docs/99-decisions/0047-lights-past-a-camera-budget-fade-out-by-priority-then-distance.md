# 0047. Lights past a camera's budget fade out by priority, then distance

- Status: Accepted
- Date: 2026-11-02
- Area: rendering
- Owner: architect

## Context

Every light reaching a view costs its quad, its shadow rows and, with silhouettes, its tile, so a crowded view needs a cap; but a light simply cut when the cap is reached pops off, and two lights at the edge of it would swap each frame.

## Decision

- `light.Budget(n)` on a camera caps the point, cone and shape lights its view draws; 0, the default, draws all. Directional and `Static` lights count against none.
- Past it, lights are ranked by `light.Priority(n)`, then by how far the middle of the view is from their disc (0 inside it). A light kept last frame ranks as if 10% nearer, so lights about as near never swap back and forth.
- A light falling past the budget fades out over `light.BudgetFade(seconds)` (0.3 s by default, smoothstep on its colour) and fades back in when it makes it again; one first showing comes in at once. Fades run on the World's scaled time, so a paused game holds them.

## Alternatives

- Ranking by each light's brightness at the middle of the view: costs a falloff a light, and flickering lights would swap places.
- Cutting lights without a fade: they pop, which a budget is meant to hide.
- One cap for the whole window or engine: a minimap or split screen needs its own.

## Consequences

A fading light still draws, so a view briefly holds up to its budget plus those fading out. The F3 overlay shows the budget and how many are fading.
