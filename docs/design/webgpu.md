# One renderer on WebGPU

Status: **approved by the user (2026-11-01), [decision 0041](../99-decisions/0041-one-renderer-on-webgpu.md): ember's own cgo binding to wgpu-native, WGSL effects.** Phase 1 (the 2D port) is done: all 43 2D scenes match OpenGL within one level (65430070). The user then decided to remove OpenGL entirely ("I do not want to maintain multiple gfx"): phase 2 ports 3D, keeps GL goldens, and deletes OpenGL.

The user wants ember to be their own engine, for several games, with 3D soon, and on every platform: Windows, macOS, Linux, then the web, Android and iOS. Today the renderer is OpenGL 4.1 core. Apple has deprecated OpenGL, which has no compute shaders on macOS. Windows VMs and machines without a proper driver get it only through emulation layers, and the web and mobile have no OpenGL 4.1 at all. This note proposes moving the renderer onto WebGPU before 3D resumes, so 3D is written once, on the renderer it will ship on.

## What exists

The renderer is small and already behind a seam:

- **Games never touch OpenGL.** 2D and UI draw through `twod.Canvas` into `internal/draw` lists: vertices of 13 floats, batches keyed by font atlas, up to 8 images, a clip, a blend mode and a mask. Offscreen `draw.Pass`es carry their own list, target texture and effects. `draw.Source` is the only thing `gpu` reads for 2D.
- **`gfx` is about 800 lines:** contexts, programs, meshes, dynamic buffers, targets and GPU timers. **`gpu` is about 2,000 lines:**
  - `UIRenderer`, which draws the 2D lists, passes, masks, blends and render textures;
  - `Renderer`, the 3D forward HDR renderer with MSAA, a shadow map, bloom, tonemapping and the post chain;
  - 17 GLSL shaders.
- **The window code holds little OpenGL.** `platform/desktop` uses it only to create the context, read the version strings for the crash log and read pixels back for captures.
- **One game-facing GLSL surface.** `render.NewEffect(glsl)`, used by `twod.WithEffect` and by camera post chains, takes GLSL 4.1 fragment source. Only example 33 uses it today, and the demo doesn't.

## Proposal

### 1. WebGPU is the one GPU API

`gfx` becomes a thin layer over WebGPU, which provides:

- devices, a surface per window, buffers, textures and samplers;
- render pipelines cached by their state (blend, mask, depth, target format);
- bind groups, and command encoders recording one frame.

Every backend comes from the WebGPU implementation rather than from ember:

| Platform | API underneath |
|---|---|
| macOS, iOS | Metal |
| Windows | Direct3D 12 or Vulkan |
| Linux, Android | Vulkan |
| Old drivers, VMs | OpenGL ES (the implementation's fallback) |
| Browser | the browser's own WebGPU |

Natively that implementation is wgpu-native (Rust, the one behind Firefox). In the browser it is the browser's.

### 2. Shaders are WGSL, the engine's and the games'

- The 17 engine shaders are rewritten in WGSL.
- `render.NewEffect` takes WGSL: a fragment function the engine wraps with the same inputs as today (`uv`, the previous picture, depth, sizes, time, the effect's own uniforms).
- Browsers accept only WGSL, so taking GLSL would mean translating it on every platform, or the web never running a game's effects.
- This is the one breaking change, and it is cheapest now: one example uses it, and nothing is published.

### 3. The 2D passes map one for one

| Today (OpenGL) | On WebGPU |
|---|---|
| a batch: atlas, 8 image units, clip, blend | a bind group for its textures (atlas, images, mask), a scissor rectangle, a pipeline per blend mode |
| `draw.Pass` into a texture | a render pass into a texture, then read as a texture by the next pass |
| masks (`AsMask`, `MaskedBy`) | the mask texture in the batch's bind group, as today |
| blend modes (Multiply, Add…) | blend state in the pipeline |
| render textures (`twod.Into`) | the same textures used as render attachments |
| per-frame vertex upload | one mapped vertex buffer per frame, written once |
| GPU timers | timestamp queries, where the adapter has them |

The draw lists don't change, so `twod` and `ui` don't change.

### 4. 3D is built on it, not ported to it

When 3D resumes (decision pending, [The 3D API](3d.md)), it starts on WebGPU:

- depth and MSAA targets;
- a shadow pass;
- HDR, bloom and tonemapping as render passes;
- and, which OpenGL 4.1 on macOS never had, compute shaders. Skinning (on the CPU today in `anim`), particles and culling can move to the GPU.

Pipelines come from material and mesh layout, and are cached.

### 5. The binding: cgo first

Two Go bindings to wgpu-native were looked at:

- **cgo**, through the C header, the way `go-gl` works today. It needs a C compiler per target, which CI's native runners have. It works next to GLFW and oto on every desktop.
- **go-webgpu with goffi**, which needs no C compiler. It is in beta. It conflicts with purego, which oto uses ("duplicated definition of `_cgo_init`"; workarounds are a build tag or `pureffi`). On Linux it doesn't link next to other cgo code such as GLFW (goffi issue #72). Its callbacks don't work on Android yet.

The binding sits behind `gfx`, so it can change without touching anything else. The spike (section 8) found go-webgpu usable but broken in three places on wgpu-native v29. The proposal is therefore **ember's own thin binding** to `webgpu.h`: internal to `gfx`, through cgo (GLFW needs cgo anyway), covering only the calls the renderer makes. The spike's own render-pass call shows that works. goffi stays an option for later, once its Linux link issue is fixed.

Either way, every build ships the wgpu-native library next to the game: a `.dll`, `.dylib` or `.so` per OS and architecture, the way the VirtualBox build ships Mesa today. It is 6.9 MB on macOS arm64 and 9.1 MB on Windows x64.

### 6. Platforms after the renderer

Each new platform is a backend behind the window entities and `internal/inputfeed`, the seam the desktop runner uses today:

| Platform | How |
|---|---|
| Windows, macOS, Linux | GLFW as today, a WebGPU surface from its native window |
| Web | Go WASM: a canvas, browser input and the browser's WebGPU through `syscall/js`, sound through WebAudio (oto) |
| Android | the engine as a library (`gomobile bind`) in a small host app that hands it a surface, touch and the app lifecycle |
| iOS, iPadOS | the same in an Xcode host app (Metal surface); needs an Apple developer account |

### 7. The order

1. **Spike** (done, section 8).
2. **Decision 0041** (recorded): WebGPU replaces OpenGL, through ember's own binding, and effects take WGSL.
3. **The 2D port**, in a private worktree, with the OpenGL renderer kept beside it. `EMBER_CAPTURE` shots of every example and the demo are compared between the two until they match. Then OpenGL, `gfx`'s GL code and the GLSL shaders are deleted.
4. **CI on native runners**, for macOS, Windows and Linux on amd64 and arm64. Linux renders through the implementation's software or GL fallback, so every shader compiles on every push. goreleaser packages each build with its wgpu-native library.
5. **3D resumes** on WebGPU.
6. **Web**, then **Android**, then **iOS**.

### 8. What the spike found

The spike was about 620 lines outside the repo, on an M1 Max. Through go-webgpu v0.5.5 and wgpu-native v29, with GLFW and oto, it drew 10,000 alpha-blended sprites (one instanced draw) over a depth-tested cube, into a render texture, then a WGSL tint-and-ripple pass onto the window.

- **Builds:** with cgo, GLFW, go-webgpu and oto link and run on macOS. A Windows amd64 build (mingw) ran under Wine Staging at 60 fps and captured the same picture. Without cgo (no GLFW), goffi and purego clash on `_cgo_init` unless built with `-tags nofakecgo`. Linux wasn't testable here.
- **Speed** (Retina 1600×1200, three passes):

  | Sprites | Vsync off | GPU-bound, no present | Go work per frame |
  |---|---|---|---|
  | 10,000 | 4.9 ms | 2.6 ms (387 fps) | 0.43 ms |
  | 100,000 | 6.2 ms | 4.4 ms (226 fps) | 1.25 ms |

  Render pipelines take 0.4–3 ms each to create. The cost of the calls through goffi is negligible. Vsync off on macOS sometimes waits 30–50 ms for a drawable from the window compositor.
- **Bindings broken on v29:** three descriptor structs miss v29 fields, each crashing every time.
  - Explicit bind-group layouts abort in Rust. Open issue #34, five weeks old.
  - Vertex attributes abort.
  - Depth attachments segfault.

  The spike worked around them: automatic layouts, vertices read from storage buffers, and its own `BeginRenderPass` call.
- **Errors:** WGSL compile errors surface only through error scopes, and their message text came back garbled. A bad descriptor aborts the process rather than returning an error; the crash-log watch (427ac3c8) still records it.
- **Surfaces:** no GLFW helper. macOS needs a `CAMetalLayer` on the Cocoa window, about 10 lines of cgo Objective-C. Windows needs the HWND and the module handle.
- **Captures:** copying a texture to a buffer and mapping it works. Rows are padded to 256 bytes, and pixels are BGRA.

## Estimates

For one engine developer, from the size of today's code (not prototyped yet):

| Step | Weeks |
|---|---|
| `gfx` on WebGPU, the 2D renderer, WGSL shaders, desktop surfaces | 4–6 |
| Captures, render textures, effects, crash info, CI | 1–2 |
| Web platform | 3–5 |
| Android | 3–4 |
| iOS | 3–4 |

## What changes for games

- `render.NewEffect` takes WGSL instead of GLSL. `Effect.Err()` reports WGSL compile errors with their line, as today.
- Every build ships one more file, the wgpu-native library.
- Nothing else. Entities, `twod`, `ui`, input and audio stay as they are.

## Alternatives

- **Stay on OpenGL 4.1.** No work now, but no compute shaders on macOS, a deprecated API on Apple, and nothing for the web or mobile. 3D would be written on it and ported later anyway.
- **Native Vulkan, Metal and Direct3D backends.** The most control, at three times the work and three renderers to keep alike.
- **ANGLE** (OpenGL ES over Metal, Vulkan and D3D). It keeps a GL-style API, but GLES 3 has no compute, and the web would still be WebGL.
- **Render through another engine** (Ebitengine). Every platform at once, but the renderer and its shader language (Kage) wouldn't be ember's. Ruled out: the user wants ember as their own engine.
