# 0044. Names made consistent

- Status: Accepted
- Date: 2026-11-02
- Area: api
- Owner: user

## Context

[0019](0019-options-read-back-and-pair-with-setters.md) set the naming rules, but packages written before or beside it drifted: `With` prefixes in render, net and automation, `XRay()` against `Shadows()`, three words for an occluder's group, two penumbra options, events without a subject. The user flagged it.

## Decision

- No `With` prefix. Kept only where the bare name is a type of the same package: `ember.WithTheme`, `ui.WithTheme`, `twod.WithEffect`, `audio.WithBus`, `net.WithCodec`, `net.WithListener`, `net.WithEndpoint`.
- One positive word per idea, the same for every kind: `light.Shadows(on)` (no `XRay`), `twod.CastShadow(on, ids…)` for tiles, `render.Shadows(distance)` with 0 for off, one group word (`OccluderGroup`, tile property `occluder_group`, which updates 0043), one penumbra (`light.Size`), one sampling word (`twod.Pixelated()` for textures and tiled pictures, both smooth by default).
- Getters of a window's or camera's state read `XOf(win)`: `light.DebugOf`, `OutlinesOf`, `DebugAreaOf`, beside `AmbientOf` and `AmbientMapOf`.
- Events name their subject: `collide.BodyEntered`/`BodyLeft`, `net.ServerDisconnected`, `ui.MenuItemChosen`, `ui.Pad2DChanged`. ui's `Clicked`, `Pressed`, `Toggled` and `Submitted` stay as they are: the element is in the event, and the words are what every UI kit says.
- The full table of old and new names is in the commits of 2026-11-02 (`refactor(…): … (decision 0044)`).

## Alternatives

- Keep the old names beside the new ones for a while: no game outside the repo uses ember yet, so deprecations would only add noise.
- `XRay()` beside `Shadows()`: two words for one flag, the inconsistency the user flagged.

## Consequences

Every name follows 0019 again, and a tiled picture now samples smoothly unless `Pixelated()`, like textures. Callers in the examples, tools and demo were fixed in the same commits; guide pages follow from the doc writer.
