# 0042. 2D lights and shadows on the GPU

- Status: Accepted
- Date: 2026-11-01
- Area: rendering
- Owner: user

## Context

The demo lit its world on the CPU, cell by cell, through areas, doors and baked shapes, and the user found it too complex for an engine. They want lighting any game can use, computed on the GPU as far as possible, with real shadows (`docs/design/2d-lighting.md`).

## Decision

- Lighting lives in `twod/light`, pure Go, read by `gpu` through `internal/draw`:
  - lights are 2D entities;
  - layers are `Lit`, sprites `Unlit` or `LitAt` a point;
  - the camera has an `Ambient` and the collide spaces whose solids block light;
  - sprites can be an `Occluder`.
- Each frame the GPU draws, per window:
  - an occluder picture;
  - one polar shadow-map row per shadow-casting light, marching that picture;
  - a half-resolution float light picture: the ambient plus each light's disc, dark past its shadow.
- Lit sprites multiply their colour by the light picture in the 2D shader itself ("forward"), at each pixel or at their anchor. Sorting and batching are unchanged.
- No CPU ray casts. The CPU lists the lights and occluders in view.

## Alternatives

- **The demo's CPU light maps, moved into the engine:** the complexity the user objected to, made into API.
- **Multiplying a finished layer by a light picture:** it can't light a sprite at one point without breaking the y-sort, and it costs passes per layer.
- **CPU ray casts per light (box2dlights' own way):** CPU time grows with lights times rays. The GPU marches a picture instead.

## Consequences

Supersedes 0037. The 2D vertex grows from 13 to 16 floats. At most 256 lights a window cast shadows. Shadows are as sharp as the occluder picture's texel (one a point) and 512 angles a light.
