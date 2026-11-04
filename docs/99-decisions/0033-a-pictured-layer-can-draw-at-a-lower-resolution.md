# 0033. A pictured layer can draw at a lower resolution

- Status: Accepted
- Date: 2026-10-30
- Area: rendering
- Owner: architect

## Context

A layer that shows as one picture (a Blend, a mask, an effect) draws into a pass at its window's full resolution. A game's fog and light maps are two such layers covering the whole window, with lights overlapping inside them: at Retina size their fill took about half the frame, though light is soft and needs no such detail.

## Decision

- `twod.PassScale(f)` on a layer draws its pass at f times the window's pixels, from 0 to 1, in the same screen points, and the picture is stretched over the window with linear filtering. `PassScale()` and `SetPassScale(f)` read and change it; the default is 1.
- It applies where the layer already draws as a pass and changes nothing for other layers. The pass follows the window as it resizes.

## Alternatives

- **A second camera with `Resolution`, showing only the light layers:** it draws them small, but its picture composites with Alpha where its first layer stands, so it can't Multiply onto the world as the layers must.
- **A global quality setting for every pass:** one switch, but a blended layer of sharp art, such as fireflies, would blur along with the light.
- **GPU shadows (a 1D shadow map per light):** cuts the triangles, not the fill that costs; kept for later if the CPU polygons ever do.

## Consequences

Light and fog maps cost a quarter of their fill at half scale. Edges inside a scaled layer soften by its scale; a game picks the smallest scale whose capture still looks the same.
