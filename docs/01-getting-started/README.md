# Getting started

ember is a Go game engine for online, open-world games. A game is plain Go: entities are structs, systems are small objects, and the same World runs in a window or headless on a server.

## Requirements

- Go 1.27.0 or later.
- A C compiler for the windowed build, which uses GLFW and WebGPU through cgo. On macOS, install the Xcode command line tools with `xcode-select --install`.
- A graphics card that WebGPU draws with: Metal on macOS; Direct3D 12, Vulkan or else OpenGL on Windows; Vulkan on Linux, or OpenGL ES where there's no Vulkan driver, with the Vulkan loader (`libvulkan1`) installed either way.
- The WebGPU library, wgpu-native, which ember opens when it starts: `task wgpu` fetches it for `go run`, and a built game ships it beside its executable.
- Servers need neither: the packages they use are pure Go and build with `CGO_ENABLED=0`.

## Running the examples

Every page of this guide is a complete example from the repository's `examples`. Clone it and run them from its folder:

```sh
git clone https://github.com/lantervale/ember
cd ember
task wgpu
go run ./examples/01-hello-cube
```

`task wgpu` fetches the WebGPU library once, into the source tree.

The code on each page is the example's source, included when the site is built, so it always compiles and always matches what you run.

Start with [Hello cube](01-hello-cube.md), then [The World](../02-world/README.md).

## Shipping a game

`task package` builds a game into `dist/` with its WebGPU library: `task package PKG=./examples/19-sprites NAME=sprites`, and `GOOS` and `GOARCH` build for another system.

- **macOS:** `sprites-darwin-<arch>/sprites.app`, so builds for arm64 and amd64 keep apart, the game in `Contents/MacOS` and `libwgpu_native.dylib` in `Contents/Frameworks`, with its `Info.plist`. It is signed ad hoc, so it runs on the machine that built it; a copy downloaded elsewhere shows Gatekeeper's warning until it is signed with a Developer ID and notarized.
- **Windows:** `sprites-windows-<arch>/`, holding `sprites.exe`, a windowed build with no console, beside `wgpu_native.dll`. It builds from macOS too, with the Windows C compiler named in `CC`, or else mingw-w64 for amd64 (`brew install mingw-w64`) and zig for arm64 (`brew install zig`), since mingw-w64 has no arm64 compiler.
- **Linux:** `sprites-linux-<arch>/`, the game beside `libwgpu_native.so`. It builds on Linux, with GLFW's X11 and Wayland headers and `libvulkan-dev`: the game links the Vulkan loader rather than OpenGL, so players need `libvulkan1` for the game to start, even on a card that draws through OpenGL ES; nearly every desktop distribution has it.

A game looks for the library in `EMBER_WGPU_LIB` first, then beside its executable, then in a macOS app's `Contents/Frameworks`, then in the source tree, where `task wgpu` put it. `EMBER_STATS=1` names what it drew with when it quits: "graphics: WebGPU (wgpu-native 29.0.0.0, Metal)" and the graphics card.

## In the browser

The guide's windowed examples also run in the page, built for WebAssembly and drawn with WebGPU. When a page can't run them, it shows the screenshot and says why.

### What it needs

- **A browser with WebGPU.** Recent desktop Chrome, Edge, Safari and Firefox have it.
- **A secure page.** Browsers give WebGPU only to pages over https or from localhost. A guide served over http from a network address, such as a phone reaching `http://192.168.…`, has none.

### Turning WebGPU on

| Browser | WebGPU |
|---|---|
| Chrome, Edge | On by default on Windows, macOS, ChromeOS and Android. On Linux, turn on `chrome://flags/#enable-unsafe-webgpu` and restart. |
| Safari | On by default from Safari 26. Before that: Settings › Advanced › "Show features for web developers", then Develop › Feature Flags › WebGPU. |
| Firefox | On by default on Windows and macOS in recent versions. Elsewhere, set `dom.webgpu.enabled` to true in `about:config` and restart. |

A graphics driver the browser blocks also turns WebGPU off, and so does hardware acceleration turned off (Chrome and Edge: Settings › System). Updating the driver or turning acceleration back on brings it back. [webgpureport.org](https://webgpureport.org) shows what your browser offers.

### Building an example for the browser

`task web PKG=./examples/19-sprites` builds one into `dist/web`, with its page. Serve that folder over localhost (`python3 -m http.server -d dist/web`) and open it.
