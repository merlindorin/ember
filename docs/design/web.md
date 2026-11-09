# Web: ember in the browser through WebAssembly and WebGPU

Status: **accepted, [decision 0054](../99-decisions/0054-ember-runs-in-the-browser-through-webassembly-and-webgpu.md); W0–W3 done (2026-11-03); W4 waits until after the Prologue, and publishing the guide's live frames waits on the user.**

## Building

`GOOS=js GOARCH=wasm go build ./examples/19-sprites` builds a game for the browser, and `task web PKG=…` puts it beside `wasm_exec.js` and the page. Go turns cgo off by itself for js; a shell that exports `CGO_ENABLED=1` overrides that and fails with "cgo: unknown ptrSize for wasm", so unset it there or prefix `CGO_ENABLED=0` (`task web` does).

To check by hand: a real pad (PS4, Xbox, Switch Pro) in Chrome with example 49. DevTools can't fake the Gamepad API, so the standard mapping is checked by reading only.

## Size

Every example built for the browser (`-trimpath -ldflags "-s -w"`), 56 of them (2026-11-03):

| | Raw total | Gzipped total | Per example |
|---|---|---|---|
| automation's Connect server linked | 1.6 GB | 340 MB | 2.6–38.9 MB raw; ui ones about 37 MB |
| server left out of js builds (`automation/server_js.go`) | 515 MB | 145 MB | 2.6–16.1 MB raw, mean 9.2; 16 over 10 MB, all ui ones (11–16 MB raw, 3.5–4.5 MB gzipped) |

What's left in a ui example (13-controls, 13.8 MB raw, 3.7 MB gzipped): 7.7 MB of code, mostly the engine's own (ui/css, layout, paint, gpu, twod; the Go runtime 1.2 MB), and 6.1 MB of data, mostly the runtime's tables; embedded fonts are about 0.76 MB, uniseg's tables 0.27 MB, emoji names 0.08 MB. The guide builds every example, gzipped, and loads one on click (architect, 2026-11-03). Later, if size matters more: binaryen's `wasm-opt -Oz` usually saves 10–20% of a Go wasm, at the cost of one more build tool; not used for now.

## Goal

The same game code runs in a browser tab: `GOOS=js GOARCH=wasm go build`, one HTML page, nothing else to install. First the guide's examples, run live on their pages, then the demo.

## What already works there

Every game-side package is pure Go by rule (CLAUDE.md, "keep the pure packages free of cgo"): `world`, `twod`, `twod/light`, `ui`, `ui`, `render`, `collide`, `nav`, `ease`, `audio`, `input`, `anim`, `gltf`. Shaders are WGSL, the browser's WebGPU language. Examples load their assets from `embed`.

Only three places touch the operating system, and each gets a web twin chosen by build tag:

| Today (desktop) | Web twin | Note |
|---|---|---|
| `internal/wgpu`: cgo over wgpu-native's `webgpu.h` (≈1,700 lines, 70 methods) | `internal/wgpu` with `//go:build js`: the same Go API over `navigator.gpu` through `syscall/js` | `webgpu.h` mirrors the JS API one to one, so `gpu/` doesn't change |
| `platform/desktop`: GLFW windows, input, pacing, crash log | `platform/web`: a `<canvas>` as the window | `ember.go` picks one by tag |
| `internal/audiodevice`: oto | oto's own js backend (Web Audio) | sound starts after the first click |

## Pieces

1. **GPU binding (W1).** Adapter and device requests are promises: await them once at start-up. Buffers, textures, bind groups, pipelines and passes become JS objects held in a handle table. Writes go through `queue.writeBuffer` with `js.CopyBytesToJS` from one reusable staging `Uint8Array`, so no per-call allocation. The surface is the canvas's `webgpu` context; the preferred format comes from `navigator.gpu.getPreferredCanvasFormat()`. Readbacks (probes, captures) map buffers asynchronously and arrive a frame late, as probes already do.
2. **Platform (W2).**
   - **Frame loop:** `requestAnimationFrame` drives `World.Step`; the blocking `Run` becomes a callback chain on js.
   - **Windows:** the primary window is a canvas whose backing is its CSS size times `devicePixelRatio` (or `EMBER_SCALE`). The page is its frame: a resizable window fills the page and follows it (no smaller than its `MinSize`), a `FixedSize()` one keeps its size, and while the page is full screen, by `SetFullscreen` or by the page itself, it fills the screen. The runner checks each frame, resizes the canvas and the surface, and syncs the window's size and full screen state. `SetFullscreen` asks for the whole page full screen, never the canvas, which the browser would stretch past the size the game reads. A second `platform.Window` is an error on web.
   - **The end:** when the World quits, the runner removes and releases every page handler it added (keys, pointer, wheel, paste, sound), cancels its animation frame and closes the sound, and `main` returns, so `go.run`'s promise resolves: ember's page then says the game has ended with a Play again link, and a page that reloads or runs the game again finds no handler left behind.
   - **Input:** keyboard by `KeyboardEvent.code` (physical positions, as today's Keys), pointer events with capture, wheel, touch (`Touchscreen`), the Gamepad API (button and axis order "standard" maps to our positions), clipboard through `navigator.clipboard`, and cursor shapes through CSS `cursor`.
   - **Text input:** `beforeinput`/`input` events from a hidden textarea, so the browser's IME and the emoji picker work.
3. **Audio (W2).** oto's js backend. The AudioContext stays suspended, playing silence, until the player's first click, key or touch on the canvas resumes it. The canvas keeps mousedown and mouseup from the page, so oto's own unlock on mouseup never sees a click; checked under Chrome's autoplay rule by `TestAClickOnTheCanvasStartsTheSound` (52–54).
4. **Files and saves (W3).** Assets from `embed`. Saves and tool state in `localStorage`, behind the same small store the desktop writes to files. The crash log goes to the console and a visible overlay.
5. **Build and host (W3).** `task web PKG=./examples/NN-name`: the wasm, `wasm_exec.js`, and a page that shows a "WebGPU unavailable" message where needed. The guide embeds one live frame per page, built at site build time; the GIFs stay for browsers without WebGPU.

## Milestones

| Step | Done when | Owner |
|---|---|---|
| W0 | a spike: one WGSL triangle drawn by `gpu/` through the js binding in Chrome | game-engine-developer-2 |
| W1 | the binding is complete: the 2D golden scenes render in headless Chrome, and their pictures match the native goldens within the GPU test's tolerance | game-engine-developer-2 |
| W2 | `platform/web`: input, frame loop, audio; examples 19, 20, 26, 36 and 46 run in Chrome with keyboard, mouse and gamepad | game-engine-developer-2 with a ui developer |
| W3 | `task web`, the guide's live frames, saves; every 2D example and ui's 92-ui-kit run in the browser | game-engine-developer-2, game-doc-writer |
| W4 | the demo runs in the browser at 60 fps on an M-series Mac in Chrome | game-engine-developer |

CI: a `web` job builds every example for js/wasm and runs W1's golden scenes in the pinned headless Chrome that `task test:chrome` already uses.

## Limits accepted

- **Browsers:** WebGPU ships in Chrome and Edge, and in recent Safari and Firefox (Firefox 159 on macOS runs the guide's examples, checked by `TestExamplesRunInFirefox`, which needs a window: headless Firefox has no adapter). Without it, the page says so instead of failing silently.
- **Secure pages only:** browsers offer WebGPU only over https or from localhost, so a page served over http from a network address (a phone on the LAN, `hugo server --baseURL http://192.168.…`) has none; the page names the origin as the reason.
- **Size:** a Go wasm build is about 10–20 MB before compression, 3–6 MB gzipped. That's fine for examples; the demo can stream assets later.
- **Speed:** every GPU call crosses into JS. Batching (draw lists, replayed ui frames) keeps the calls per frame low; W1 measures calls per frame on the busiest golden scene.
- **One thread:** Go's wasm runs single-threaded. The flow-field rebuild that runs in the background today runs inline on web.

## Alternatives

- **Emscripten wgpu-native:** compile the C library to wasm and keep cgo. Go can't link cgo for js/wasm, so it isn't possible.
- **TinyGo:** smaller binaries, but generics, reflection and the parts of the standard library that ember uses are partial, and the World relies on reflection.
- **WebGL2 instead of WebGPU:** wider browser support, but a second shader language and renderer: the multi-backend cost the user rejected with decision 0041.
