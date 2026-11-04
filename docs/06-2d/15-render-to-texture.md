# Render to texture

A camera can draw into a texture instead of the window, and the texture then shows like any other. Here a second camera draws the whole meadow, small, into a minimap in the corner; the hero walking shows up in it by itself.

![The meadow up close, and the whole meadow small in a framed minimap](render-to-texture.png "ember · render to texture")

## The program

The window's camera follows the hero at three times the size; `SnapToPixels` keeps the pixel art on whole pixels as it moves. `twod.NewRenderTexture(240, 128, twod.Pixelated())` is a texture with no picture of its own. A second camera with `Into(minimap)` draws into it at half size, centred on the meadow; a camera into a texture is never the window's camera, and leaves out fixed layers and the UI. On a fixed layer, a dark frame and a sprite of the minimap show it, like any texture. Inside a render texture, blended, masked and effect layers draw plainly, and masks are skipped.

[main.go](../../examples/32-render-to-texture/main.go)

A texture shown much smaller than its own size, such as a minimap drawn at a third of the world's scale, looks better with the default smooth filter than with `Pixelated()`, which shimmers as things move.

## Drawing only some layers

A camera draws every layer by default. `twod.Layers(…)` makes it draw only the layers listed, and what's in them; a window's fixed layers and its UI always show. Give a minimap's camera a layer of markers, and the main camera the world's layers, and each view shows only its own. `Layers()` reads them back and `SetLayers(…)` changes them.

## As before

The meadow and the walking hero of [Pixel-perfect](13-pixel-perfect.md).

[level.go](../../examples/32-render-to-texture/level.go)

[walk.go](../../examples/32-render-to-texture/walk.go)

## Try it

[32-render-to-texture](../../examples/32-render-to-texture)
