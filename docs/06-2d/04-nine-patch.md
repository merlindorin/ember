# Nine-patch

A nine-patch stretches a small framed picture to any size without stretching its frame. Here one 24 × 24 panel becomes a dialog, a button, a pillar and a long bar.

The panel, with its 8-pixel frame:

![The 24 × 24 panel](../../examples/22-nine-patch/panel.png)

![Four panels of different sizes from the same picture](nine-patch.png "ember · nine-patch")

## The program

`twod.NewNinePatch` takes the texture, and `Margins` the width of its left, top, right and bottom borders, in texture pixels. The four corners are drawn as they are, the edges stretch along their length, and the middle fills the rest; `Size` sets the size it fills. Here `Pivot(0, 0)` places each panel by its top-left corner. As with sprites, `Scale` enlarges the whole of it, here three times, so the pixel art stays crisp: with pixel-art frames, keep the scale a whole number, so every corner pixel stays whole.

[main.go](../../examples/22-nine-patch/main.go)

## Try it

[22-nine-patch](../../examples/22-nine-patch)
