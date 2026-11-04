# 0031. A batch draws several textures

- Status: Accepted
- Date: 2026-10-30
- Area: rendering
- Owner: architect

## Context

A 2D batch held one image, so a y-sorted crowd of several sprite sheets broke into a batch, a draw call and a texture bind per sprite: 3,237 draws for the 20k bench scene, and a 21 ms frame.

## Decision

- A batch draws up to 8 images, each in its own texture unit (GL 4.1 guarantees 16; the font atlas and the mask take two). Each image vertex carries its image's slot, packed with its shape kind in `Shape.X` as kind + 16 × slot, so a vertex stays the same size.
- The shader picks the image by slot with an if-chain, since GLSL 4.1 can't index samplers by a varying.
- A new batch starts only for a ninth image, a font atlas change, or a change of clip, blend or mask. Pass pictures and render textures, which the shader reads as premultiplied, keep a batch to themselves. Draw order never changes.
- Bindings are skipped when a batch uses the same textures as the one before.

## Alternatives

- **Atlas pages first,** packing loaded images into shared textures: fewer textures for everything, but it rewrites loading and UVs and still breaks on a second page; it can come on top.
- **Sorting by texture:** fewer breaks without shader work, but it changes draw order, which y-sorting and layers depend on.

## Consequences

The 20k bench scene draws in 6 draw calls (was 3,237), renders in 0.22 ms (was 7.3) and runs at 8.2 ms a frame. A ninth distinct texture in a run still breaks the batch; atlas pages would lift that.
