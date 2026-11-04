# 2D shaders

A layer can run a shader, a small WGSL function of the game's own, on what it draws. Here clicking the meadow sends out ripples: rings of light that bend the meadow as they spread and fade. The game tells the shader where and when each ripple started, from Go, every frame.

![Clicks on the meadow send out rings that bend the pond, the path and the yard](2d-shader.gif "ember · 2D shader")

## The scene

The meadow of [Tile maps](08-tile-maps.md) and the hero standing on the path, a sprite showing one `Still` frame of the sheet, are on one layer, and that layer runs the shader: `render.NewEffect` makes it from the WGSL source, and `twod.WithEffect` puts it on the layer. The help text is on a fixed layer of its own, so the ripples leave it alone: an effect belongs to its layer. On a layer, an effect's `WithScreen` is ignored: it only means something on a camera.

[main.go](../../examples/33-2d-shader/main.go)

## The shader

The layer is first drawn into a picture, and the shader turns it into the layer's pixels. Its source defines one function, `fn effect(vUV: vec2f) -> vec4f`, the colour of the pixel at `vUV`, which runs 0 to 1 from the top-left, on a layer as on a camera. Beside it, the engine declares `color(uv)`, the layer's picture, and `uSize`, `uOutputSize` and `uTime`, as in 3D post-processing, which also has `linearDepth(uv)` and `uDepthRange`; each uniform the game sets appears under its name, as an `f32`, a `vec2f` to `vec4f`, or an `array<…, N>` for the slices. For each live ripple, the shader pushes the pixels in a ring outwards and lights the ring's crest; the ring grows with the ripple's age. Colours are premultiplied by their alpha, so the light is scaled by alpha too. A source that doesn't compile is reported by `Err()`, and the layer isn't drawn.

WGSL is the shading language of WebGPU, which ember draws with ([decision 0041](../99-decisions/0041-one-renderer-on-webgpu.md)).

[shader.go](../../examples/33-2d-shader/shader.go)

## Driving it from Go

`Ripples` starts a ripple where the window is clicked, ages them, and hands them to the shader every frame with `effect.SetVec4s("uRipples", ripples)`: four `geom.Vec4`s of the centre, the age, and whether each lives, for the shader's `uRipples[4]`. The effect has a setter for each kind of uniform, `SetFloat`, `SetVec2` to `SetVec4`, `SetColor` and their slices; they can be called at any time, and the shader reads them on the next frame. With four ripples alive, a click replaces the oldest.

[ripples.go](../../examples/33-2d-shader/ripples.go)

## As before

The meadow's level, as in [Tile maps](08-tile-maps.md).

[level.go](../../examples/33-2d-shader/level.go)

## Try it

[33-2d-shader](../../examples/33-2d-shader)
