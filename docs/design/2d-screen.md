# 2D on screen: resolution, blending, masks, render textures and shaders

Status: approved and built (passes, blending, masks, render textures, effects). A design record: the guide documents the API as it is.

Four gaps from the raylib/Ebitengine comparison, in one mechanism. A **pass** is a set of 2D entities drawn into an offscreen image instead of the window. The image is a value, a `*twod.Texture` with no pixels of its own, and the window shows it like any picture.

The rules are the usual ones:
- ready-made entities and values;
- typed options and no exported config;
- no World resources;
- pure Go in twod; only the GPU layer knows framebuffers.

## Passes

twod's system already sorts each window's entries by layer, z and y. A pass takes the entries of a camera or a layer, draws them into a `DrawList` of their own sized to the pass, and adds one image batch where they stood in the window's order. That batch samples the pass's texture, so the pass shows exactly where its layer would have.

The GPU draws every pass of a window before the window, each into a framebuffer it keeps for that texture, then the window's list. A texture made by a pass is just a texture to everything that draws it.

## 1. Pixel-perfect scaling

```go
eye := twod.NewCamera(twod.Resolution(320, 180), twod.Upscale(twod.Nearest))
```

- **`Resolution(w, h)`** is a CameraOption: the camera's 2D world draws at w×h pixels. Its viewport is w×h units at zoom 1, so a 16-pixel sprite covers 16 of the 320.
- **Upscaling:** the result is scaled by the largest whole factor that fits the window, centred, with the window's background as letterbox bars. `Upscale(Nearest)` keeps pixels sharp (the default) and `Upscale(Linear)` smooths them. `FitWindow()` drops the whole-factor rule to fill the window, keeping the aspect ratio.
- **Fixed layers and the UI** (the HUD and ui elements) still draw at the window's resolution over it, so text stays sharp.
- **Coordinates:** `twod.ToScreen` and `ui.Follow` map through the scaling. `Camera.ScreenToWorld(p)` maps a mouse position to the world.

This matches what `render.Post.Resolution` does for 3D, in 2D terms.

As built:
- **Ordering:** the scaled world shows as one picture, where its first layer stands in the window's order.
- **Drawing inside the pass:** an entity drawn there sees the pass's size as `Canvas.Screen()` and the mouse in its pixels.
- **Presenters:** they read `twod.PassesFor(w, win)`. The GPU stores a pass's picture with premultiplied colours and draws it as `KindPass`.

## 2. Blend modes and masks

```go
glow := twod.NewSprite(light, twod.Blend(twod.Add))
lit := twod.NewLayer(2, twod.MaskedBy(lantern))
lantern := twod.NewLayer(3, twod.AsMask())
```

- **`Blend(mode)`** is a shared option, so it applies to every entity. It is part of the `Look` and applies to the entity itself, not to its children: a layer's `Blend` blends the layer's picture, drawn as a pass. The modes are `Alpha` (the default), `Add`, `Multiply` and `Screen`. Batches carry their mode, and the GPU sets it per batch.
- **Masks:**
  - `AsMask()` makes a layer that is never shown; its entities draw into a pass whose alpha is the mask.
  - `MaskedBy(layer)` draws a layer through it: the layer goes into a pass, and its image batch is multiplied by the mask's alpha at the same screen point. That's a spotlight, a fog of war, or a window into another scene.

As built:
- **Layer pictures:** a layer with a `Blend` other than `Alpha`, `AsMask` or `MaskedBy` draws into a pass the size of the surface it shows on (the window, or a `Resolution` camera's pass), shown as one picture where the layer stands.
- **Empty masks:** a mask with nothing in it hides what it masks.
- **Pass order:** passes are drawn inner first, so a layer's picture is ready before the scaled world that shows it.
- **Blend inside a pass:** the alpha of every mode accumulates as with `Alpha`.
- **Filter:** `twod.Filter` is `render.Filter`, with `twod.Nearest` and `twod.Linear`.

## 3. Render textures

```go
minimap := twod.NewRenderTexture(160, 90)
overview := twod.NewCamera(twod.Into(minimap), twod.Zoom(0.25), twod.Follow(hero))
hud.Add(twod.NewSprite(minimap, twod.At(560, 60)))
```

- **The texture:** `NewRenderTexture(w, h)` returns a `*Texture` of w×h pixels with no image. It is a value games pass around like any texture, never a framebuffer handle.
- **Drawing into it:** `Into(texture)` makes a camera draw its window's 2D world, without fixed layers and the UI, into the texture rather than the window. A sprite showing the texture then shows that view: a minimap or a portrait.
- **Feedback:** a texture is left out of its own pass, rather than showing the previous frame. That needs no second buffer, and no example needs feedback.

As built:
- **Order:** a camera `Into` a texture is not one of its window's cameras, whatever its `Priority`. Its pass is drawn before the window's others.
- **Layers inside a texture:** they draw their entities as they are, not as pictures; mask layers are left out.
- **Drawing passes:** the GPU draws every pass upright, so a render texture samples like any image. It decodes a pass's premultiplied colours for every batch that shows one.
- **Scenes:** a render texture keeps its size in scene files.

## 4. 2D shaders

```go
ripple := render.NewEffect(rippleGLSL, render.Name("ripple"))
water := twod.NewLayer(1, twod.WithEffect(ripple))
ripple.SetFloat("uStrength", 0.5)
```

- **`WithEffect(e)`** is a LayerOption: the layer draws into a pass the size of the window, then shows through e, a `render.Effect` as 3D's post-processing uses.
- **The shader:** the same fragment header as 3D post-processing (`uColor`, `uSize`, `uOutputSize`, `uTime`, its own `Uniforms`), and no depth. It writes the layer's pixels, with their alpha, over what is below.
- **Several effects:** `WithEffect` can be given more than once; they run in order.

As built:
- **Pixels and orientation:** `uColor` holds the layer's pixels with their colours multiplied by their alpha, and the shader writes them back the same way. `vUV.y` grows down the screen, like the 2D world.
- **No depth:** `uDepth` is blank, so `linearDepth` means nothing.
- **Disabled and Screen:** `Disabled` skips an effect; `Screen` means nothing for a layer and is ignored.
- **Where they run:** the effects run on the GPU after the layer's pass, each one taking turns between two targets.
- **Failed shaders:** one that fails to compile is skipped and reports through `render.Effect.Err()`, in 3D post-processing as well.

## Examples

| # | Example | Shows |
|---|---|---|
| 28 | `30-pixel-perfect` | the walker in a 320×180 world scaled sharply to any window size, letterboxed, with a HUD drawn sharp at the window's resolution |
| 29 | `31-blend-modes` | glowing sprites with Add, a Multiply shadow, and a dark scene seen through a lantern mask that follows the mouse |
| 30 | `32-render-to-texture` | a minimap: a second camera into a render texture shown by a HUD sprite |
| 31 | `33-2d-shader` | a water layer drawn through a wave shader |

## Decided

1. **`Resolution` is only on a camera:** no window option. A game that wants it adds a camera.
2. **No `Into` on layers for now:** a camera into a texture covers minimaps and portraits, and the layer pass stays internal. Revisit when an example needs it.
3. **`Blend` applies to the element itself,** not to its children, so a glow's children aren't additive by surprise. A layer's `Blend` blends its composite. Tint keeps its inheritance.
