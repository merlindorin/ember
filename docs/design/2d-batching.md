# 2D rendering: batching and what each frame costs

Status: **proposal, for the user's review.** Nothing here is built except item 0.

## How a 2D frame is drawn today

1. **Collect** (`twod/system.go`): every frame, the twod system walks every 2D entity in every window and builds an entry list: transform, tint, layer and z. The children of a `SortByY` layer are sorted by y, and the whole list is sorted by layer and z.
2. **Emit** (`internal/draw/list.go`): each entry draws into a `draw.List`, 4 vertices of 52 bytes and 6 indices per quad. A new batch starts whenever the image changes, along with the font atlas, clip, blend or mask.
3. **Draw** (`gpu/ui.go`): one uber-shader branches per pixel on the shape kind. Each batch binds its image and atlas and draws.

The demo's wave-23 scene, measured by `examples/internal/bench2d` at 1280×720 with about 1,100 entities: **630 batches, 630 texture binds, 18k vertices, 3 passes.** The y-sorted crowd interleaves four enemy sheets with the things map, so nearly every sprite is its own batch.

## 0. Done: one upload per list

Every batch used to rewrite one shared vertex buffer before its draw, and on macOS's GL that waits for the GPU each time. Uploading the whole list once and drawing batches as ranges cut render time from **148 ms to 1.8 ms** for the same 630 draws. This was a plain bug, not a design limit.

## Proposals, most valuable first

### 1. Cull what no camera sees

**Done.** Sprites, nine-patches, lines and polygons outside every camera that draws their layer are skipped while collecting, with a margin of 16 points; a SortByY parent sorts only the children with something in view. bench2d at 1280×720, render before and after:

| scene | render | draws | vertices |
|---|---|---|---|
| wave23 | 1.97 → 1.66 ms | 642 → 575 | 17,394 → 16,434 |
| wide | 2.09 → 0.63 ms | 668 → 195 | 17,394 → 12,942 |
| crowd (5,000) | 8.84 → 7.87 ms | 2,983 → 2,710 | 56,748 → 54,864 |
| 20k | 32.0 → 3.9 ms | 11,404 → 338 | 127,860 → 9,426 |

The twod system itself (collect, sort and emit) fell from 25.7 to 3.9 ms in the 20k scene. In that scene the frame is still the collision step's (38 ms per fixed step): crowd collision, not yet approved.

Nothing culls sprites today; only tile maps draw just the cells in view. In a survivors game, most of the horde is outside the view: enemies spawn in a ring beyond the screen, and flasks are left behind as the hero walks. Every one of them is collected, emitted, uploaded and rasterized off-screen.

- **Change:** while collecting, skip an entity whose drawn bounds (rect under its global transform, plus a margin for glow and rotation) fall outside every camera that draws its layer: the main camera and any `Into` camera such as the minimap. Fixed layers and the UI are always kept.
- **Opt-out:** an entity that draws outside its own bounds (a custom `Draw`) can't be culled by rect, so it is kept unless it reports bounds.
- **Expected:** vertices and batches scale with what's on screen, not with the World. Measure on bench2d with the horde spread 3× wider than the view.

### 2. Several textures per batch

**Done** (decision 0031). bench2d at 1280×720, before → after:

| scene | draws and binds | render | frame |
|---|---|---|---|
| wave23 | 580 → 8 | 1.58 → 0.14 ms | 3.30 → 1.54 ms |
| crowd (5,000) | 2,812 → 8 | 7.38 → 0.23 ms | 11.27 → 3.76 ms |
| 20k | 3,237 → 6 | 7.30 → 0.22 ms | 21.38 → 8.21 ms |

The 20k frame is now within 60 fps; what is left of it is simulation and the twod system's collect (4.5 ms).

A batch holds one image. The proposal is to hold up to 8 images: GL 4.1 guarantees 16 fragment texture units, and the atlas, mask and pass inputs need a few.

- **Vertex:** each one carries a slot index.
- **Shader:** picks the sampler with an if-chain, because GLSL 4.1 can't index a sampler array by a varying.
- **Batch:** a new batch starts only when a run of draws needs a ninth distinct image, or when clip, blend or mask change.
- **Expected:** the crowd's 630 batches fall to a handful (four breed sheets, the things, arrows, flasks and glyphs are about 8 images). No game change, and draw order is untouched, so y-sorting still works.
- **Why before an atlas:** it fixes interleaving without touching how textures are loaded, and it keeps working for images made at runtime: render textures, minimap passes, procedurally drawn cards.

### 3. Pack small images into atlas pages

The deeper fix: twod packs images loaded through it (`LoadTexture`, sheets, separate animation frames like 0x72's) into shared 2048² pages, with 1–2 px edge extrusion so filtering never bleeds. UVs are remapped inside `twod.Texture`, invisible to games.

- **Benefit:** combined with item 2, a whole scene often fits in one or two pages, and binds go to near zero.
- **Cost:** a packer, extrusion, per-texture filter rules (Pixelated vs smooth pages), and textures that change at runtime staying unpacked.
- **Recommendation:** do it only if bench2d at 5,000 entities still shows binds as the cost after items 1 and 2.

### 4. A lighter vertex, and quads by index

A quad is 6 vertices × 52 bytes = 312 bytes. With 4 vertices and a shared static index buffer, plus colour packed as 4 normalized bytes and the shape kind and slot packed together, it's about 4 × 28 = 112 bytes. That's under half the bandwidth and the CPU writes, which matters at 10k sprites, particles and glyphs.

- **Done (indices):** a quad is 4 vertices and 6 indices, 232 bytes, and a coloured mesh (the fog's fans and ambient) makes one vertex per point it shares. The GPU draws each batch as a range of an index buffer uploaded once per list. Captures are byte-identical. Bench2d's render time moves within noise (20k: 0.34–0.37 → 0.30 ms), so vertex bandwidth isn't what limits a frame at these counts.
- **Left (packed vertex):** colour as 4 bytes must be stored in gamma, not linear, or the dark ambient and the fog's fades will band. Colours above 1 (HDR) can't be packed. Worth doing only if a heavier scene shows upload cost.

### 5. Fix alpha once, at upload

Images are already uploaded as `SRGB8_ALPHA8` (`gfx.NewTexture`), so the texture unit linearizes them. Yet for every image pixel, the 2D fragment shader goes back to gamma, divides by alpha and comes back to linear (two `pow` calls), because Go's `image.RGBA` is premultiplied in gamma space. Converting once on the CPU at upload, to straight alpha or to alpha premultiplied in linear space, removes both `pow` calls from every pixel. Full-screen floors pay them on every pixel, at every resolution.

The 2D blend itself happens in gamma space: the shader writes `pow(c, 1/2.2)`, and Add, Screen and Multiply combine those values, not linear ones. Anything that computes colours meant to combine through a blend must work in that space. The demo's sight disc does so with `blendGamma` in `demo/fog`. Changing the blend space here means changing those too.

### 6. No per-frame allocation in collect

The twod system rebuilds its camera and feed maps every frame, and the y-sorted children slice every frame. Reusing them keeps the garbage collector out of busy frames.

## Not proposed

- **Sorting draws by texture:** this would batch better, but it breaks the painter's order that y-sorting and z rely on. Item 2 gets the batching without it.
- **GPU instancing per sprite:** this would need a second shader path beside the uber-shader for shapes and text. Items 2 and 4 get most of the win in one path.
- **Moving to Metal or Vulkan:** out of scope. The bug in item 0 was ours, not GL's.

## Baselines after item 0 (259ba5f, 1280×720)

| Scene | Entities | Render | Frame | Draws and binds | Vertices |
|---|---|---|---|---|---|
| wave23 | 1,092 | 1.97 ms | 4.29 ms | 642 | 17,394 |
| wide (horde 3× the view) | 1,092 | 2.09 ms | 4.34 ms | 668 | 17,394 |
| crowd | 5,092 | 8.84 ms | 16.25 ms | 2,983 | 56,748 |

- **wide:** it draws as many vertices as wave23 although most of the horde is off screen. Nothing is culled, which is item 1's case.
- **crowd:** about 3,000 draws, each with its own bind, put the frame at the edge of 60 fps. That's the case for item 2, then item 3 if needed.

## How each item is judged

`examples/internal/bench2d` gains two more scenes: the horde spread 3× wider than the view (for culling), and 5,000 entities (for headroom). Each item lands with before-and-after numbers for render ms, batches, binds and vertices, and with tests on the draw list (batch counts, culled counts) that need no GPU.
