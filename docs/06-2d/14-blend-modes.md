# Blend modes and masks

How a picture mixes with what is under it. Here night falls on the meadow, a lantern lights a soft circle of day around it that follows the mouse, and fireflies glow.

![The lantern's soft circle of daylight sweeping across the meadow at night, among glowing fireflies](blend-modes.gif "ember · blend modes")

## The layers

From the bottom up:
- the meadow;
- the night, a dark blue polygon on a layer that `Blend(Multiply)`, so everything under it is darkened as one picture;
- the day, a second meadow on a layer `MaskedBy(lantern)`, which shows only where the lantern's picture is opaque;
- the lantern, a layer `AsMask()` holding one soft disc, which is never shown itself;
- the fireflies, small discs that each `Blend(Add)` and light up what is under them.

The blend modes are `Alpha`, the default, `Add` for glows, `Multiply` for shadows and night, `Screen` for soft light, and `Lighten`, which keeps the brighter of the entity and what is under it, channel by channel; its transparent parts leave what is under unchanged. In a `Multiply` light layer, an ambient shape drawn with `Lighten` after the `Add` glows of the lights gives the brighter of the ambient level and the lights' sum, so the dark never falls below the ambient. A scene names it `lighten`. For lights that cast shadows, see [Light](19-lights/README.md). On an entity, `Blend` applies to it alone, not its children; on a layer, it blends the whole layer as one picture. A mask shows through its alpha, so the disc's soft edge makes a soft edge of light. `MaskedBy` isn't saved when the layer is saved as a scene.

A layer shown as one picture, by a `Blend` other than `Alpha`, `AsMask`, `MaskedBy` or `WithEffect`, can draw that picture at fewer pixels. `twod.PassScale(0.5)` draws it at half the window's pixels each way, a quarter of them, in the same screen points, and stretches it over the window smoothly; it follows the window as it resizes. Light and fog maps are soft anyway, so they lose nothing and cost less: `twod.NewLayer(4, twod.Blend(twod.Multiply), twod.PassScale(0.5))`. Edges soften by the scale, so pick the smallest one whose capture still looks the same. It is 1 by default, `SetPassScale` changes it, other layers ignore it, and a scene saves it as `pass_scale` ([decision 0033](../99-decisions/0033-a-pictured-layer-can-draw-at-a-lower-resolution.md)).

[main.go](../../examples/31-blend-modes/main.go)

## The glow

`glow` draws the soft disc in code, white and opaque in the middle, fading to nothing at its edge, and makes a texture of it. The lantern and the fireflies share it. `Lantern` moves the lantern's disc to the mouse whenever the mouse moves.

[glow.go](../../examples/31-blend-modes/glow.go)

## As before

The meadow of [Tile maps](08-tile-maps.md).

[level.go](../../examples/31-blend-modes/level.go)

## Try it

[31-blend-modes](../../examples/31-blend-modes)
