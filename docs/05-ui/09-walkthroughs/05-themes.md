# Themes

The same dialog in four looks, switched with T: the kit's own, a warm one written in Go, a cool one read from a CSS file, and a skin of game art. The dialog is built once, in Go; how it looks is CSS, turned on by a class on the dialog and reaching everything in it.

![The dialog in its four looks, one after the other](themes.gif "ember · themes")

## The dialog

`content` is what the dialog says and asks, in a `kit.Panel`. Nothing in it says how it looks.

[main.go](../../../examples/15-themes/main.go)

## A look written in Go

A look is a rule for a class. The kit draws from custom properties, `--panel-color`, `--border-color`, `--button-color`, `--hover-color`, `--accent-color` and the rest of its palette ([Kit](../07-kit/README.md)), so setting them on `.ember` recolours every kit control inside, as CSS variables inherit. The sheet goes in with `ui.CSS(emberLook)`.

[themes.go](../../../examples/15-themes/themes.go)

## A look from a file

`ui.Stylesheet(files, "frost.css")` reads the same from a file, so the look changes without a rebuild. One rule reaches further in: `.frost .kit-button` rounds the kit's buttons ([CSS](../03-css.md) has the selectors).

[frost.css](../../../examples/15-themes/frost.css)

## A skin

`border-image` draws the box as a nine-patch of game art, the wooden frame of [Nine-patch](../../06-2d/04-nine-patch.md), its 8-pixel borders unstretched, and `image-rendering: pixelated` keeps its pixels sharp. `@font-face` names a font file next to the sheet, and `font-family` gives it to everything in the dialog.

[wood.css](../../../examples/15-themes/wood.css)

## Switching looks

`Switch` takes the last look's class off the dialog and puts the next one's on, each time T is pressed. The next frame restyles only what changed.

[switch.go](../../../examples/15-themes/switch.go)

## Try it

[15-themes](../../../examples/15-themes)
