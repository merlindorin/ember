# 0054. ember runs in the browser through WebAssembly and WebGPU

- Status: Accepted
- Date: 2026-11-03
- Area: rendering
- Owner: user

## Context

The guide's examples, and later the demo, should run in a browser tab with nothing to install. The renderer is WebGPU already (decision 0041), and every game-side package is pure Go; only the GPU binding, the window runner and the sound device touch the system ([web plan](../design/web.md)).

## Decision

- `GOOS=js GOARCH=wasm`: the same game code, built with `task web`, in one page (`platform/web/index.html`) whose canvas is the primary window.
- `internal/wgpu` has a js twin over `navigator.gpu` through `syscall/js`, with the same API, so `gpu/` builds and draws unchanged. Promises are awaited only between frames, from a goroutine `requestAnimationFrame` wakes; inside a frame, errors come through `TakeError`, and readbacks map a frame later.
- `platform/web` runs the World: canvas, frame loop, captures (`EMBER_*` settings from the page's query). Input, sound and saves follow (W2, W3).
- The browser build is checked against the native goldens: the 2D scenes rendered in headless Chrome match them within the GPU test's tolerance (`go test -tags 'chrome gpu' ./gpu -run GoldenInTheBrowser`).

## Alternatives

- wgpu-native compiled with Emscripten behind cgo: Go can't link cgo for js/wasm.
- TinyGo for smaller binaries: generics, reflection and the parts of the standard library the World relies on are partial.
- WebGL2 for wider support: a second shader language and renderer, the multi-backend cost decision 0041 rejected.

## Consequences

Browsers with WebGPU only (Chrome, Edge, recent Safari; the page says so elsewhere), single-threaded Go, and a 14.8 MB wasm (3.7 MB gzipped) for an example; every GPU call crosses into JS, so batching matters more there.
