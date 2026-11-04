# 0046. Silhouettes are masks subtracted per light

- Status: Accepted
- Date: 2026-11-02
- Area: rendering
- Owner: architect

## Context

`light.Silhouette()` makes a sprite cast its own picture as its shadow, stretched from its foot away from each light, instead of the wedge its box casts. The light picture sums every light after shadowing, and its shadow maps hold one distance per angle, so neither can carry a picture's shape.

## Decision

- On the CPU, the light system picks the lights casting a Silhouette sprite: every directional light and the three strongest reaching its foot, of its groups and casting shadows. For each of them, it draws the sprite's current picture into a list (`draw.Silhouette`), projected: its foot where it stands, its width across, its height along the direction away from the light. The shadow is as long as the sprite is tall, d·h/(e−h) under a light with an Elevation, and ShadowLength for a directional light.
- On the GPU, for each casting light, its silhouettes are drawn into a mask with a max blend, so overlapping shadows never darken twice. That light's own contribution is then drawn again, times the mask, and subtracted from the light picture. Walls' shadows, the light's colour and falloff are all kept.
- Shaded pictures (ShadedBy) take out only the silhouettes of the groups they keep.

## Alternatives

- Drawing the picture into the occluder picture: shadow maps keep one distance per angle from each light, so a figure's shape would come out as a wedge again.
- Darkening the light picture by the mask once, for all lights: a silhouette would darken the light of lamps that don't cast it, and a coloured lamp's shadow would take the other lamps' colours.

## Consequences

Each light casting silhouettes costs a mask pass (cleared only over its reach) and its quad drawn once more, for every light picture of the window: about 0.06 ms of GPU and 0.12 ms of CPU a light. So it suits a hero and a few sprites, not a horde under many lights. The mask atlas (c6f25aa4) packs every casting light's silhouettes into tiles on the CPU, a list a page, and draws them in two passes a page: 1,000 silhouettes under 64 lights went from 9.16 to 2.46 ms of CPU (boxes 1.73) and from 5.99 to 2.21 ms of GPU (boxes 1.83). Then each silhouette became a 64-byte instance that the mask shader projects, its tile the union of its shadows' bounds, and lights with no occluder of their groups within reach skip their shadow rows: 2.05–2.21 ms of CPU (+16–22%) and 1.57–1.72 ms of GPU, under the boxes' 1.82: within the +25% the demo waited for. Static lights cast none. Size and ShadowFade don't soften or fade silhouettes yet.
