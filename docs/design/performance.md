# Performance: 600 fps at the start, 200,000 entities above 60 fps

Status: **done (2026-10-31), apart from clean re-measurements at low load.** Both targets are met: about 340 fps at the start of a run (hidden window) and 200,000 entities at 118 fps. D4's packed colour and atlas pages (2d-batching items 3 and 4) were measured as not worth doing at today's counts.

## Targets

| Case | Target | Frame budget |
|---|---|---|
| The demo at the start (sanctuary, about 1,000 entities), 2000×1125 points on a Retina screen | about 600 fps | 1.7 ms |
| 200,000 entities (mostly still, a few thousand moving), same window | above 60 fps | 16.7 ms |

The demo window runs without vsync, so fps is the true cost.

## Where a frame goes today

The user's F3 at the start of a run (Area 4, 989 entities, 34k vertices):

| Item | ms |
|---|---|
| frame | 9.84 (105 fps) |
| CPU tick (all systems) | 2.53 |
| CPU render | 1.62 |
| ui system | 1.39, with almost nothing on screen |
| twod system (collect, sort, emit) | 0.72 |
| GPU, drawing the 2D window | 4.69 |
| GPU, light and minimap passes | 0.26 |

bench2d at 2000×1125 (hidden window): wave23 (1,146 entities) 8.2–10 ms; 20k scene (60,146 entities) 16.8 ms, of which twod Update 5.2 ms, FixedUpdate step 2.3 ms, collide Tracker 1.3 ms.

## What blocks the targets

1. **The CPU and the GPU take turns.** Frame ≈ CPU (4.2) + GPU (4.7). On macOS GL, the swap waits for the GPU to finish the frame, so the next frame's CPU work only starts after it. Nothing overlaps.
2. **The GPU spends 4.7 ms on a mostly dark screen.** At Retina scale the window is 4000×2250 device pixels, about 9 M pixels. The floor, the light layer's composite, the ambient mesh and the HUD each cover most of the screen, and the uber-shader does two `pow` calls per textured pixel (2d-batching item 5).
3. **The ui costs 1.4 ms per frame while idle.** Theme lookups (`ui.lookup`, `faceOf`, `themeType`), text measuring and layout run for every element every frame, hidden screens included.
4. **Work grows with every entity, not with what is visible.**
   - twod collects every 2D entity each frame (2d-batching "O(visible)" (a)).
   - The World snapshots every entity each fixed step for interpolation ((b)).
   - The collide Tracker and the ECS groups walk everything.
5. **Smaller costs.**
   - GPU timer queries run every frame even with F3 closed.
   - About 1.3 MB of vertices are uploaded each frame, at 52 bytes a vertex and 6 vertices per quad (2d-batching item 4).
   - Collect allocates every frame (item 6).

## Changes that don't change the game (being done now)

| # | Change | Owner |
|---|---|---|
| A | ui: skip hidden subtrees; cache resolved theme values per element until its kind, theme or options change; cache text measures; lay out only when something changed | game-engine-developer |
| B | twod collect O(visible): a per-layer grid of cells, so collect visits only the cells under each camera; static entities never re-sorted | game-engine-developer-2 (done, see "B and D6" below) |
| C | World snapshots only of entities moved in that step | game-engine-developer-2 (paths shared with game-engine-developer) |
| D | 2d-batching items 4 (4 vertices per quad, an index buffer, packed colour) and 6 (no allocation in collect, done by game-engine-developer-2 with B). Item 5 (no `pow`) is dropped: with both pows removed, GPU time didn't move (3.6 ms either way), because the GPU is bound by fill, not arithmetic | game-engine-developer |
| E | GPU timers only while F3 shows them | game-engine-developer |
| F | collide: the Tracker and space only step bodies that moved or are awake; static bodies sleep | game-engine-developer-2 |
| G | bench2d gains a 200k scene (190k still sprites, 10k moving), a 600-fps start scene that matches the demo's sanctuary, and prints GPU ms; every change reports before and after | game-engine-developer-2 |
| K | **Lighting is slow by its algorithms** (top priority). Walking: open crypt 1.9–3.9 ms, the Drowned Warrens 3.6 ms of a 5.2 ms tick, spikes 8–10 ms. Per cell in view, every frame: rooms found by scanning dungeon plans, areas by `AreaOf` searches, thin bars by re-reading tile collision shapes; the ambient is a vertex per cell; each light walks cells. Plan (user, 2026-10-31): lighting is part of the map, baked at generation in layers. Rooms and areas are indexed cell grids at load, and tile shapes precomputed. Per chunk: a light-map texture (ambient per area or room, plus every placed light clipped to its area) and a room-id texture. At runtime the shader multiplies by a per-room and per-door level table (fades and spill), each visible torch adds a small flicker disc, and the hero's torch is one disc masked by room ids. Re-bake only edited chunks. Target under 0.2 ms | game-engine-developer |
| J | The big map and the minimap paint the explored map once into cached image tiles, updated only where new squares are explored, instead of a rect per square and per wall cell every frame | game-engine-developer |
| I | Sprites without an animation never step it (22% of the 200k tick after C) | game-engine-developer-2 (paths shared with game-engine-developer) |
| H | Overlap CPU and GPU: measure where the swap waits; try drawing frame N while simulating N+1, and swap before the next frame's simulation, without adding input lag beyond one frame | game-engine-developer, with the architect |

Each lands with numbers from bench2d and F3, and with tests that need no GPU (batch counts, culled counts, allocation counts).

## Proposals that could change the look (for review)

1. **Draw the world at point resolution.** Pixel art is drawn at 2 points per art pixel, so at Retina each art pixel is 4×4 device pixels. Drawing the world layers at 2000×1125 and scaling up by 2 with nearest filtering shows the same pixels with 4× fewer fragments. The HUD and text stay at full resolution. What could change: sub-point motion steps in whole points instead of half points, and the soft light edges become a little coarser. `PassScale` (decision 0033) already does this per pass.
   - Measured: GPU about 3.8 → 2.75 ms (−28%) at the start; the frame is within noise today, since it sits close to both the CPU and the GPU. To land it, the camera's resolution must follow the window size in points.
2. **Draw the light layer at half resolution:** already done (`PassScale(0.5)` on the Light layer, 0.2 ms).
3. **Pack the crypt art into atlas pages** (2d-batching item 3), if binds still show after D.

## How progress is judged

The two target scenes in bench2d plus the user's F3 at the start of a run, before and after each change, in a table appended here.

## Measurements

A game's own measurements (its start, gating its changes, its maps and its lighting) live in that game's docs.

### bench2d

bench2d, `-w 2000 -h 1125`, hidden window, 300 frames after a 60-frame warm-up (200k: 120 frames), in ms.

**Baseline** (3876346's scenes, load 6, before 53b6a1b, so the GPU was timed every frame and frames include that cost):

| Scene | Entities | Frame | CPU tick | Render | GPU 2d window | Top systems |
|---|---|---|---|---|---|---|
| start (the demo's sanctuary) | 709 | 9.98 | 2.61 | 1.66 | 3.34 | twod 1.20, ui 0.93, lighting 0.10 |
| wave23 | 1,146 | 8.25 | 1.05 | 1.82 | 2.31 | twod 1.25 |
| 20k | 60,146 | 16.44 | 10.18 | 2.29 | 3.22 | twod 5.69 (collect 4.02), step 2.43, Tracker 1.35, stepEnd 0.45, stepStart 0.43 |
| 200k | 220,161 | 67.97 | 60.03 | 2.80 | 3.64 | twod 27.62 (collect 25.08), stepEnd 3.29, stepStart 2.67, step 1.53, Tracker 0.61 |

**The 200k tick** (CPU profile, 8 s): twod collect and cull walk every 2D entity, about 40% of samples in the Group iteration alone. stepStart and stepEnd snapshot every entity each fixed step, about 25%. Sprites step their animations, 7%. Allocations reach `madvise`, 5%. Collide's Move is 6% and its Tracker under 1%. At 68 ms a frame the World runs about four fixed steps a frame, so every FixedUpdate cost counts four times. Once the frame is under 16.7 ms, it falls to one.

**After F** (d55b578), the Tracker per fixed step: 20k 1.35 → 0.80 (load 18–24), 200k 0.62 → 0.36 (load 14).

**After E, A, F, C and H** (93ee797, `EMBER_STATS=cpu`, load 8):

| Scene | Entities | Frame (fps) | CPU tick | Render | Swap | Top systems |
|---|---|---|---|---|---|---|
| start | 709 | 3.02 (331) | 1.42 | 0.32 | 1.13 | twod 1.15 (draw 1.12), lighting 0.09, ui 0.06 |
| wave23 | 1,146 | 2.14 (467) | 0.75 | 0.18 | 1.06 | twod 0.76 |
| 20k | 60,146 | 9.53 (105) | 7.80 | 0.21 | 1.36 | twod 5.47 (collect 4.23), step 2.70, Tracker 0.89, look 0.37, stepEnd 0.14 |
| 200k | 220,146 | 27.76 (36) | 26.16 | 0.20 | 1.26 | twod 23.61 (collect 22.88), step 1.02, Tracker 0.33, stepEnd 0.03 |

**The 200k tick now** (CPU profile): twod is 80% of samples.
1. collect walks every child of every layer: 55%, of which culling (`cull.visible`, `inView`, `Camera.shows`) is 28%. B's grid takes this.
2. gather steps every sprite's animation: 22%, with `Sprite.step` alone at 13%, the 190,000 still sprites included.
3. Allocation and clearing (`memclrNoHeapPointers`, `madvise`): 9%.

Collide's crowd rebuild is 2.4%.

**The window's GPU time, layer by layer** (93ee797, the demo's start, `EMBER_STATS=1`, so timed and inflated about 1.5×; each row hides some layers):

| Hidden | Draws | Vertices | GPU 2d window ms |
|---|---|---|---|
| nothing | 20 | 30,648 | 3.45 |
| the floor | 20 | 14,928 | 2.43 |
| the light layer's composite | 12 | 18,096 | 2.74 |
| the shards and shots | 20 | 30,648 | 2.63 |
| the crowd, the sparks, or the HUD overlay | 16–20 | 30,312–30,648 | 3.5–3.6 (no change) |
| every world layer | 8 | 2,040 | 1.47 |
| every world layer and the HUD overlay | 8 | 2,040 | 1.40 |
| the minimap (separately) | 15 | 29,466 | 2.42, from 3.05 |

An almost empty window (example 11) costs 0.20 ms, so the 1.4 ms left with every layer hidden was the minimap's two window-sized passes and their full-screen composite, now cropped. A full-screen solid quad of alpha 0 is also drawn each frame, but skipping it gained nothing. What remains is the floor (about 1 ms) and the light's full-screen multiply (about 0.7 ms): proposal 1 halves both.

**B and D6: collect visits what the views show, and allocates nothing** (`EMBER_STATS=cpu`, `-w 2000 -h 1125`, 300 frames). The load was 90–127 throughout, so base (5775e34f) and new ran back to back per scene; read the ratios more than the absolute ms.

| Scene | Base frame (fps) | New frame (fps) | Base collect | New collect |
|---|---|---|---|---|
| start | 2.17 (461) | 2.11 (474) | 0.05 | 0.04 |
| wave23 | 1.82–1.97 | 1.83–2.06 | 0.21–0.23 | 0.21–0.24 |
| 20k | 9.90 (101) | 9.73 (103) | 4.37 | 4.44 |
| 200k | 25.44 (39) | 8.47 (118) | 20.84 | 5.45 |

How it works: a non-fixed layer with at least 128 children files its still sprites (those without 2D children, at most two cells wide) in 256-point cells by the middle of their bounds. A sprite's placement setters, its size-changing setters and its animation steps put it on its grid's dirty list once; before collecting, the grid files the dirty again, and a move within its cell costs one bounds check. The layer's children list carries a change count (`node.ChildrenChanged`), so a child added, removed or reparented brings the grid in step with one walk. Collect visits, in the children's order, the cells round what each view shows (grown by the largest sprite filed and 64 points for drawing between fixed steps), plus every other child, so the emitted lists are byte-identical: `TestEveryBenchSceneDrawsTheSameWithGridsAndWithout` compares the window's list, its batches and every pass, frame by frame, for every bench scene, and twod's `TestTheGridDrawsExactlyWhatTheWholeWalkDraws` does the same for a scene that moves, resizes, adds, removes and reparents sprites, with a parallax layer, a zoomed follow camera and a minimap camera. A grid whose views take in more than half its children for 30 frames is dropped for 120, so wave23's crowd, all in view and all moving, costs what it did. What the 200k collect has left is its 10,000 enemies, entities with a sprite and a body under them, visited every frame.

D6: gather and collect reuse the frame's camera and feed maps, the per-window culls, the y-sort's stretches and the maps' views; `TestGatherAndCollectAllocateNothingOnceWarm` counts 0 allocations a frame (17 before).

Building the 200k scene took a minute: adding a child appends it twice (the node, then twod's own append for the whole entity), and `BasicEntity.RemoveChild` scanned from the first child. It looks from the last now, so the 190,000 still sprites build in about a second.

**Sharp pixels** (a4d314e9, the pause menu's setting: the world camera's resolution is the window's points). The demo at 2000×1125 points on a 2× framebuffer, `EMBER_STATS=1`, two rounds each at load 15:

| | Frame (fps) | GPU 2d window | World pass | Light pass | GPU total |
|---|---|---|---|---|---|
| off | 8.60–8.79 (114–116) | 2.41–2.43 | none | 0.34–0.35 | about 2.85 |
| on | 7.58–7.98 (125–132) | 1.47 | 0.46–0.48 | 0.17 | about 2.15 |

The world and its light draw a quarter of the pixels; the new pass that shows the world scaled up costs about 0.5 ms, so the GPU saves about 0.7 ms a frame. On a 1× screen the setting changes nothing but snapping the world to whole pixels.

**OpenGL and WebGPU** (decision 0041). OpenGL was measured at 0ab11cb1, its last commit, and WebGPU at 2427fe40. Both are bench2d at `-w 2000 -h 1125` with `EMBER_STATS=1`, so the GPU is timed every frame. The two renderers ran in turn per scene, two rounds each (ranges below). The scenes ran in a visible window, except start, which plays the demo's sound and so ran hidden. The load was 10 at the start and rose to 11–18 during the runs, so read the ratios more than the absolute ms. An M1 Max on a 2× screen, wgpu-native 29.0.0.0 on Metal.

| Scene | Renderer | Frame (fps) | CPU tick | Render (CPU) | GPU 2d window | GPU passes |
|---|---|---|---|---|---|---|
| wave23 | OpenGL | 5.68–6.02 (166–177) | 0.87 | 2.13–2.19 | 1.44 | camera 0.11–0.12, layer 0.09–0.10 |
| wave23 | WebGPU | 8.76–9.17 (109–114) | 0.85–0.86 | 0.69–0.71 | 1.17–1.45 | camera 0.14–0.31, layer 0.19–0.37 |
| 20k | OpenGL | 17.65–18.39 (54–57) | 11.95–12.25 | 1.96–1.98 | 2.41–2.48 | layer 0.07 |
| 20k | WebGPU | 12.85–13.08 (76–78) | 9.65–10.04 | 0.77–0.83 | 2.28–2.35 | layer 0.43–0.50 |
| 200k | OpenGL | 16.56–17.09 (59–60) | 9.57–9.99 | 1.91–2.13 | 2.22–2.45 | layer 0.03–0.04 |
| 200k | WebGPU | 11.36–12.06 (83–88) | 8.38–8.63 | 0.74–0.77 | 2.02–2.11 | layer 0.57 |
| start, hidden | OpenGL | 6.23–6.78 (147–161) | 1.12 | 2.43 | 1.49–1.53 | light 0.24–0.27, minimap 0.24–0.35 in three passes |
| start, hidden | WebGPU | 1.94–1.95 (513–517) | 0.95–0.97 | 0.93–0.95 | 1.98–2.13 | light 0.83, minimap 1.60–1.66 in three passes |

- **The CPU's render cost falls by 2.5–3×**, from 2.0–2.4 ms to 0.7–0.95 ms: one recorded command buffer a frame instead of GL calls through the driver. In the busy scenes, the frame falls by a quarter to a third (20k: 57 → 77 fps, 200k: 60 → 85).
- **The window's own GPU time is about the same** (within 15% either way).
- **Pass spans read higher on WebGPU.** Each pass is timed by timestamps at its begin and end, so on Apple's tile-based GPU a span also covers loading and storing its attachments. GL's timer queries didn't count those. The window's span is comparable; the small passes' spans are not.
- **wave23 in a visible window is slower on WebGPU**, 114 fps against 177. Its CPU work and GPU time are both smaller. The frame waits about 7 ms for a drawable from the window compositor, though the surface presents Immediate (vsync off), as the spike saw. That's open: a drawable count or present mode the compositor serves sooner. The demo doesn't ask for vsync either, so a visible demo meets the same wait.
- **The drawable wait, fixed** (after 8007c266 made vsync the default). The 7 ms was all in acquiring the surface's next texture, spread unevenly: one frame in 4–8 waited 15–60 ms. The macOS compositor lets about two drawables through per refresh of a 60 Hz screen at Immediate (114–122 fps), and wgpu already sets the layer's displaySyncEnabled off, three drawables and no timeout. desiredMaximumFrameLatency 1, 2 or 3 changed nothing, and Mailbox isn't offered on Metal. Two changes fix it for a window with `VSync(false)`. Visible wave23 at load 9–20, two rounds each:

  | | Frame (fps) | Worst | Acquire |
  |---|---|---|---|
  | before | 8.18–8.42 (119–122) | 48–66 ms | 6.6–6.8 ms |
  | surface only drawn into and shown (framebufferOnly) | 8.54–8.68 (115–117) | 56–63 ms | 6.9–7.1 ms |
  | at most one frame shown a refresh, the others drawn offscreen | 1.50–1.51 (664–666) | 10–17 ms | 1.7–2.2 ms |
  | both (landed) | 1.37–1.62 (615–728) | 4–58 ms | 0.9–2.7 ms |

  framebufferOnly alone does nothing, but with pacing it halves what is left of the wait. A frame read back (a capture or a GIF) draws offscreen, since the surface's textures can no longer be copied. With vsync on, the default, two windows run at a steady 60 fps (10-multi-window, not halved), F3 and EMBER_STATS show the work apart, and EMBER_STATS leaves the display's wait out of its spike reports.
- **Hidden start isn't comparable as a frame.** Hidden GL still swaps and waits, while hidden WebGPU draws offscreen without presenting. Only its GPU columns compare.

### 2D lights and shadows

The user's target is 10,000 shadow-casting solid objects at 60 fps or more. It's met: the worst case, all 10,000 in view with 256 shadowed lights, runs at about 125 fps.

Measured by the architect's tester on milestone 1 (b0f277b3), with bench2d's `lights` scene: `-scene lights -lights N [-noshadows] [-radius R] [-occluders 10000 -occkind cells|sprites]`. Runs were hidden, at 1280×720 points on a 2× screen, an M1 Max, load 6.4–10, with `EMBER_STATS=1`. Each row gives the two rounds' frame times.

| Scene | Frame (fps) |
|---|---|
| no lights | 0.61–0.65 ms (1,542–1,632) |
| 16 lights, shadows / no shadows | 0.69–0.71 (1,412–1,452) / 0.70 (1,431–1,433) |
| 64 lights, shadows / no shadows | 0.83–0.89 (1,129–1,207) / 0.72 (1,385–1,394) |
| 256 lights, shadows / no shadows | 1.41–1.51 (664–710) / 0.96–0.97 (1,034–1,040) |
| 64 lights of radius 600, shadows / no shadows | 1.14–1.22 (823–879) / 0.90–0.91 (1,096–1,115) |
| 10,000 solid cells, those in view, 16 / 64 / 256 lights | 1.18–1.21 / 1.16–1.29 / 1.70–1.73 (577–860) |
| 10,000 solid cells all in view, 0 / 16 / 64 / 256 lights | 7.03 / 7.84–7.99 / 7.89–8.00 / 7.97 (125–142) |
| 10,000 Occluder sprites, half moving, 0 / 16 / 64 / 256 lights | 6.57 / 7.71–8.02 / 7.72–7.79 / 8.22 (122–152) |

- **What the lights cost:** lighting itself is about 1–1.2 ms at its worst. The rest of the 10,000-object frames is twod collecting and drawing 10,000 sprites or tiles, which costs the same with no light at all (the 0-light rows).
- **The light picture grows with lights × area:** 64 lights of radius 600 cost about as much as 256 of radius 250.
- **Occluders when nothing is lit:** 10,000 Occluder sprites cost the light system 0.83 ms a frame even with nothing lit. Since milestone 2, a frame with no light and no ambient in any window skips them.
- **Reading the GPU rows:** "gpu 2d window" grew from 0.32 to 2.6 ms with 256 lights, even with lights casting no shadows, close to "gpu light picture" (2.48 ms), while that pass only reads one more texture a pixel. The likely reason is that on Apple's tiled GPUs a pass begins before the one before it ends, so the per-pass spans overlap and shouldn't be added up. Since milestone 2, `gpu frame` is the frame's whole GPU time, from the first stamp to the last. The probe runs below confirm the overlap: a 1,000-texel pass reads 1.37 ms on its own while `gpu frame` doesn't grow.

**Light probes, against a CPU mirror (milestone 3).** Rules such as "an enemy lit under 25% shows only its eyes" read the light through `light.NewProbe()`: the GPU samples the light picture at each probe in view, and a readback brings the values back 1–3 frames later. The probes answer exactly what was drawn, groups, softness, shapes, the sun and glows included. The alternative was mirroring the shaders on the CPU, with a collide.Space.Cast per point and light for shadows. Both were measured at load 27–39, so read the ratios:

| | Cost |
|---|---|
| CPU mirror, 100 points × 64 point lights, 10,000 solid cells (`BenchmarkCPUMirror`, point lights and solid cells only) | 1.9 ms a frame |
| GPU probes, 1,000 probes, 64 lights (bench2d `-scene lights -lights 64 -probes 1000`) | CPU 0.08 ms a frame (the light system lists them); GPU none measurable: `gpu frame` 1.73–1.86 ms with them, 2.03–2.08 without |

The probes are the engine's way, and the mirror stays a benchmark. The same runs back the note on reading the GPU rows above: "gpu light probes", a 1,000-texel pass, reads 1.37 ms on its own while the frame's whole GPU time doesn't grow. The per-pass spans overlap, and `gpu frame` is the number to read.

**Height-based shadows, before and after (2026-11-01).** Steps 1–2 of the lighting Tier 1 (`2d-lighting.md` §8): heights in the occluder picture, two shadowed stretches per angle, Size and fade curves, wall faces lit around their box, the 72-byte vertex. A first run at load ~10 read the light picture at 4.5× the milestone-1 number; most of that was load. Alternating runs of 2305472e (before) and the round committed with this table (after), bench2d `-scene lights -w 2000 -h 1125`, hidden, 300 frames, load 5–8, in ms:

| Scene | light picture | shadows | 2d window | CPU work |
|---|---|---|---|---|
| 64 lights | 1.31–1.43 → 1.35–1.41 | 0.34–0.36 → 0.34–0.35 | 1.76–1.89 → 1.86–1.93 | ≈ |
| 256 lights | 5.88–6.24 → 6.05–6.49 | 1.09–1.14 → 1.10–1.16 | ≈ | ≈ |
| 64 without shadows | 0.39 → 0.40 | — | 0.93 → 0.98 | 0.94 → 1.03 |
| 64, 10,000 solid cells | 1.17–1.19 → 1.18–1.22 | 0.20 → 0.20 | 2.23–2.26 → 2.44–2.49 | 1.82–1.94 → 1.96–2.01 |
| 64, 10,000 Occluder sprites | 2.20 → 2.23 | 0.45 → 0.45 | 3.71 → 3.80 | 7.88–7.99 → 8.45–8.63 |

- **What kept the plain path where it was:** the shadow and light-picture shaders come in two variants made from one source. The plain one (an R32F map, one distance a texel, stopping at the first occluder, one tap) runs unless a light in the window has an Elevation, a Size or a fade, which switches it to RGBA32F and two stretches. Before that split, every light read 16-byte texels and the light picture cost 10–25% more.
- **What stays:** the 2d window pass is up to 10% slower with many solid cells. Their faces take the brightest of three light samples (`draw.LitFacing`, 15eca6cb), and the vertex grew by 8 bytes for it. The CPU work is up 4–8% with 10,000 cells: their face boxes come from a per-tile cache, and the light system writes each occluder's height. A light with the new options pays for the stretches only in the windows it lights.

**After packing the vertex (68 bytes) and reading round-a-box light once a vertex.** The light mode now rides in LightSpread's signs, and a tile lit round its box (a wall's face, a tall tile) reads the light picture in the vertex shader, four times a quad instead of three or four times a pixel. Same scenes, `gpu frame` (the whole frame's GPU time, the number to read: the per-pass spans overlap), load 10–17, so read the ratios:

| Scene | before (2305472e) | after |
|---|---|---|
| 64 lights | 2.21–2.25 | 2.34–2.47 (+4–11%, at load 16) |
| 64, 10,000 solid cells | 2.38–2.46 | 2.62–2.70 (+8–10%) |

The "gpu 2d window" span of the cells scene fell from 2.23 to 1.30 ms, but as above it overlaps the light picture's.

**Clean rerun, bisected, and the light round a box back in the fragment shader (2026-11-01).** The rows above were taken at load 10–17; rerun at load 6–8 (bench2d `-scene lights -w 2000 -h 1125`, hidden, 300 frames, medians of three alternating runs), they were over 10%: 64 lights 2.32 → 2.56 ms, 10,000 cells 2.36 → 2.66. A build at each commit since 2305472e split it:

| Commit | 64 lights, gpu frame | 10,000 cells, gpu frame | 10,000 cells, CPU work |
|---|---|---|---|
| 91f7a021 ambient map | 2.31 | 2.36 | 1.86 |
| 2d915301 tile shadow shapes | 2.27 | 2.44 | 1.86 |
| 15eca6cb wall faces lit round their box | 2.28 | 2.55 | 2.03 |
| 7966810d plain and full variants | 2.21 | 2.61 | 1.98 |
| 74f5e2ff 68-byte vertex, light read once a vertex | 2.48 | 2.67 | 2.02 |

- **Reading the light round a box once a vertex cost more than it saved:** sampling the light picture in the vertex stage made the 64-lights frame 12% slower, though few of its vertices are lit round a box. The fragment shader reads it again (three or four samples a pixel of a wall's face or tall tile); the vertex stays 68 bytes.
- **The CPU half is not CPU work:** a profile of the cells scene puts the extra in wgpu-native (`wgpuCommandEncoderFinish` 0.19 → 0.37 s over 1,200 frames) while the 2D pass's commands are unchanged: back-pressure from the busier GPU, not the engine's Go code, whose tile loop is flat. Caching face boxes per chunk would save nothing.

Against 2d915301, just before wall faces were lit round their box, the same way:

| Scene | 2d915301 | now | |
|---|---|---|---|
| 64 lights, gpu frame | 2.20 | 2.34 | +6% |
| 64, 10,000 solid cells, gpu frame | 2.40 | 2.57 | +7%, most of it the wall faces' light samples |
| 64, 10,000 solid cells, CPU work | 1.86 | 1.97 | +6% |

**ShadedBy on the demo's crowd** (785882fd, decision 0045). The demo's crypt walk, 2000×1125 points, `EMBER_STATS=1`, 600 frames, god mode, three rounds interleaved at load 10–11:

| | GPU frame | CPU frame | Light shade pass |
|---|---|---|---|
| no ShadedBy (785882fd reverted) | 2.82 | 1.84 | — |
| ShadedBy, the shaded picture drawn whole (a7758fad) | 4.66 (+65%) | 3.32 | 2.97, plus 1.74 shadow rows |
| ShadedBy, only the lights reached redrawn (043b842f) | 3.11 (+10%) | 1.96 | 1.72 |

bench2d's lights scene (64 lights, 10,000 cells, `-shaded`) goes from 4.76 to 3.38 ms of GPU frame: its occluders are all group 0, so nothing is reached and the shaded picture is the copy alone.

**Silhouettes** (41c4c928, decision 0046). bench2d's lights scene, 2000×1125 points, `EMBER_STATS=1`, 300 frames, three rounds at load 10–12:

| Scene | GPU frame | CPU frame |
|---|---|---|
| 64 lights, 10,000 cells, before | 2.67 | 2.06 |
| 64 lights, 10,000 cells, with Silhouette (none used) | 2.59 | 2.02 |
| 64 lights, 1,000 Occluder sprites, boxes | 1.85 | 1.68 |
| 64 lights, 1,000 Occluder sprites, `-silhouette` | 5.99 | 9.16 |

With 1,000 silhouettes almost every light casts some: two render passes a light, so 128 passes a frame. The CPU time is wgpu's command encoding (`CommandEncoderFinish`, buffer writes), not the engine's Go code, and the GPU time was mostly each mask pass clearing a whole texture (each now clears only the light's reach). The mask atlas planned in decision 0046 brings it to two passes a picture.

**The silhouette mask atlas** (c6f25aa4). The same scene, load 7–9, two runs each:

| | Boxes | Silhouettes, a mask a light (d831d050) | Silhouettes, the atlas (c6f25aa4) |
|---|---|---|---|
| CPU frame | 1.73 | 9.16 | 2.46 (+42%) |
| GPU frame | 1.83 | 5.99 | 2.21 (+20%) |

The light system packs each casting light's silhouettes into a tile, on the light picture's pixel grid, one list a page; the GPU draws a page's lists in one pass and subtracts all its lights in another. What the CPU still spends is the shadow quads themselves: 3,000 a frame, projected and written as four 68-byte vertices each, about 0.8 MB uploaded. Two lights a sprite instead of three: 2.34 ms (+35%).

**Instanced silhouettes and unblocked lights.** The same scene, load 7–11, two runs each:

| | Boxes | The atlas (c6f25aa4) | Instanced | Instanced, unblocked lights |
|---|---|---|---|---|
| CPU frame | 1.77–1.81 | 2.46 | 2.22–2.26 | 2.05–2.21 (+16–22%) |
| GPU frame | 1.82–1.85 | 2.21 | 4.02–4.04 | 1.57–1.72 |

Instanced: the CPU sends each silhouette as one 64-byte instance (foot, size, rectangle, uv, light), bucketed by page, groups and texture with a counting sort, and the mask shader projects it; a tile is the union of its shadows' bounds, from the foot to the shadow's end, half the sprite's width more each side. Its GPU frame doubled, but not from the silhouettes (1.26 ms): with no box occluder left, every shadow ray marched its light's whole radius, and the shadow rows went from 0.31 to 1.12 ms. So a light none of whose occluders lies within its reach is now `draw.Light.Unblocked`, from a 64-point grid of the window's occluders by group, and gets no shadow rows, as if `Shadows(false)`. Here that is every light: silhouettes 0.52–0.57 ms, the light picture 0.51 ms instead of 1.24. The 10,000-occluder rows stay as they were (cells 1.49–1.53 ms of CPU, sprites 8.2–8.5, GPU 1.67–1.82 and 3.48–3.52).

**Affects** (decision 0048). bench2d's lights scene, 64 lights of radius 250, load 6–10, two runs each:

| | Before | Affects, none used | `-affects`: two sets of 16 lights left out |
|---|---|---|---|
| 10,000 cells, CPU frame | 1.52–1.53 | 1.49–1.52 | 1.82–1.92 |
| 10,000 cells, GPU frame | 1.78–1.88 | 1.82–1.84 | 2.60–2.78 (shaded pictures 1.47–1.50) |
| 1,000 silhouettes, CPU frame | 2.07–2.11 | 2.04–2.08 | — |
| 1,000 silhouettes, GPU frame | 1.53–1.55 | 1.54–1.55 | — |

Each set is a copy of the light picture and its 16 left-out lights drawn again, shadows and all. The CPU difference is the GPU's back-pressure: at radius 80, where the GPU keeps up, the two sets cost 0.09 ms of render.

**uiv2 layout** (`go test -run xxx -bench InspectorLayout ./uiv2/layout`): an inspector-sized tree (a title, a max-content 1fr grid of 24 names and cut values, four rows of three buttons) laid out at 360 points, measured at load 37–57, so the times are noisy and the allocations are the steady figure:

| | Time per layout | Bytes | Allocations |
|---|---|---|---|
| Before | 0.47–0.56 ms | 486 KB | 4,074 |
| One anonymous-block style per parent style; no anonymous block for collapsible whitespace | 0.43–0.53 ms | 326 KB | 3,714 |
| A shared whitespace replacer in cut, preallocated slices in ellipsize (2b48cdfa, ui-developer-2) | — | 279 KB | 3,363 |

The real inspector's `Document.Layout` measured 0.46–0.47 ms at load 57, about 97% of it layout and css near nothing when no style is dirty. About half of layout is building its node tree anew each frame. A frame in which nothing is dirty and the viewport is the same now skips layout: the Document keeps its tree and calls layout.Refresh (236ec6c6, 8c1eb5bc).

**Quick check at load 93–118, noisy (2026-11-02).** One round, at the user's request, of 65606d63 (Tier 1 lighting closed) against 7f5f4e60 (wip, after the text batch, decision 0050's sRGB curve, per-corner radii and the Windows start checks), base and now run in turn per row in clean worktrees. bench2d at `-w 2000 -h 1125`, `EMBER_STATS=1`, 300 frames (200k: 120); the crypt walk is the demo at 600 frames, god mode, scratch saves. At this load the absolute times are about 1.5× those above; read the deltas, and only those past 25% count as flagged.

| Row | Load | Frame, base → now | GPU frame, base → now | Other |
|---|---|---|---|---|
| 64 lights, 10,000 cells | 115–118 | 2.58 → 2.54 (−2%) | 2.89 → 3.06 (+6%) | tick 1.67 → 1.63 |
| 64 lights, 1,000 silhouette sprites | 118 | 2.50 → 2.58 (+3%) | 1.92 → 1.93 | tick 1.59 → 1.63 |
| 64 lights, 10,000 cells, `-shaded` | 116 | 3.10 → 2.79 (−10%) | 3.84 → 3.87 (+1%) | render 1.34 → 1.10 |
| 64 lights, 10,000 cells, `-affects` | 116–118 | 2.95 → 2.93 (−1%) | 5.03 → 4.53 (−10%) | |
| 200k | 116–118 | 11.52 → 11.17 (−3%) | 2.50 → 2.65 (+6%) | tick 10.70 → 10.34 |
| start | 93–116 | 1.01–1.16 → 1.08–1.25 | 1.00–1.02 → 1.10–1.20 (+10–18%), four runs each | 2d window 0.90–0.97 → 1.05–1.18 |
| the demo's crypt walk | 115 | 2.83 → 2.90 (+2%) | 5.18 → 4.21 (−19%) | tick 0.92 → 1.05 (+14%), but 922 → 1,188 entities: the demo grew, so not like for like |
| BulletHell (CPU) | 109 | 1.36 → 1.57 ms (+15%) | — | 1,282 → 1,336 allocations |
| ui's 300 elements, the text-heavy frame (CPU) | 109 | 1.90 → 1.96 ms (+3%) | — | 6,221 allocations both |

Nothing passes 25%. Two to watch at a quiet load: the start scene's GPU frame, +10–18% (its first run read +46%, then three more runs each settled it), most likely the text's quarter-pixel phases and stem darkening or the sRGB curve's encode per fragment; and BulletHell's CPU frame, +15% in one run with 54 more allocations a frame.

**BulletHell's bars change kind: a new baseline (2026-11-03).** The scene's 300 health bars were ember/ui Progress elements placed by `ui.Follow`; they are now drawn as the demo draws its marks over the horde (`hud/pips.go`): one twod item, a back and a fill rectangle over each enemy, read from its position as it draws. This is a new baseline for BulletHell, not a speed-up of the same scene. Before and after were run in turn at `-benchtime 500x` (allocations drift with b.N, so they compare only at one benchtime), three rounds at load 26:

| | Time per frame | Bytes | Allocations |
|---|---|---|---|
| ember/ui Progress with Follow | 1.20–1.33 ms | 56.5 KB | 1,086 |
| one twod item drawing the bars | 0.89–0.95 ms | 33.7 KB | 484 |

Its bar colours are package values: `color.Hex` parses and allocates, and called in Draw for two colours per bar it cost 600 allocations a frame.

**The start scene's GPU time, bisected at load 10.7–11.6 (2026-11-03).** bench2d `-scene start`, `EMBER_STATS=1`, three rounds of ten builds in turn; the GPU's 2d window, ms:

| Build | Round 1 | Round 2 | Round 3 |
|---|---|---|---|
| 65606d63 (Tier 1 closed) | 1.01 | 0.91 | 0.84 |
| 28bd0812 (before the sRGB curve) | 1.01 | 0.94 | 0.81 |
| d0e20733 (decision 0050's exact sRGB curve) | 1.20 | 0.98 | 0.92 |
| 7f5f4e60 (wip after the text batch) | 1.02 | 0.98 | 0.94 |

The quick check's +10–18% was the load. Base to now is +1–12% (about +6%); the text batch's commits are flat, and the step is the exact sRGB encode per fragment, about 0.04–0.1 ms, the cost decision 0050 accepted. Not flagged.

**Rounded clips** (6af3c598, decision 0058: the fragment shader cuts overflow to a rounded padding box). It landed at load 54 before its bench; measured on 2026-11-09 against 6af3c598~1, bench2d at `-w 2000 -h 1125`, hidden, `EMBER_STATS=1`, 300 frames, before and after in turn, two rounds at load 10, in ms:

| Scene | Frame before | Frame after | GPU frame before | GPU frame after | 2d window before | 2d window after |
|---|---|---|---|---|---|---|
| lights, 10,000 occluders | 2.08, 2.01 | 2.01, 1.89 | 1.83, 1.81 | 1.83, 1.80 | 1.67, 1.66 | 1.71, 1.72 |
| 200k | 9.84, 9.84 | 9.87, 9.53 | 2.02, 2.01 | 2.04, 2.07 | | |

Frames are unchanged; the 2d window's pass is about 3% slower, inside the noise of two rounds. Not flagged.
