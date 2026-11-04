# 0041. One renderer on WebGPU, through ember's own binding

- Status: Accepted
- Date: 2026-11-01
- Area: rendering
- Owner: user

## Context

The renderer is OpenGL 4.1 core. Apple has deprecated it, and it has no compute shaders on macOS. It fails in VMs and on machines without a proper driver, and it doesn't exist on the web or mobile. The user wants ember for several games, with 3D soon and every platform later ([design note](../design/webgpu.md)).

## Decision

- WebGPU replaces OpenGL as the one GPU API. Natively that is wgpu-native (Metal, D3D12, Vulkan, GLES), and in a browser the browser's own WebGPU.
- `gfx` calls wgpu-native through ember's own small cgo binding to `webgpu.h`, covering only the calls the renderer makes. Nobody else's Go package sits between ember and wgpu-native. The web, when it comes, gets a second binding to the browser's WebGPU behind the same `gfx`.
- Engine shaders are WGSL, and `render.NewEffect` (so also `twod.WithEffect`) takes WGSL.
- The 2D port comes first, beside the OpenGL renderer, until every example's and the demo's captures match. Then OpenGL is removed, and 3D resumes on WebGPU.

## Alternatives

- **Stay on OpenGL:** no compute on macOS, and no web or mobile.
- **go-webgpu (goffi):** no cgo, but its v29 structs miss fields. The spike crashed on bind-group layouts, vertex attributes and depth passes.
- **gogpu/wgpu:** pure Go, with one API for native and browser, but a young reimplementation of wgpu with thin hardware coverage.
- **Native Vulkan, Metal and D3D12:** three renderers to keep alike.
- **ANGLE:** GLES has no compute.
- **Rendering through Ebitengine:** the renderer wouldn't be ember's.

## Consequences

Every build ships the wgpu-native library (6.9 MB on macOS arm64, 9.1 MB on Windows x64) or links it statically. cgo stays, as GLFW needs it anyway. Games with GLSL effects must rewrite them in WGSL; only example 33 has one. The binding moves with wgpu-native's releases, and a changed struct fails to compile rather than crashing.
