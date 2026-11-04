# Text

Text is a 2D entity: words drawn with a typeface, at a size and a colour. Here a title, every face the engine ships, a pixel font read from its file, a paragraph wrapped to a width, and a clock that changes every frame.

![A title, six faces, a wrapped paragraph and a clock](text.png "ember · text")

## The program

`twod.NewText` is white 16-point `twod.Regular` text. Here `Pivot(0, 0)` places each text by its top-left corner. `twod.Baseline()` makes its position the start of its first baseline instead, the pivot then only moving it across, so texts of different sizes on one line share it. `Measure` and `Lines` measure it. `Typeface` picks a face: the engine ships `twod.Regular()`, `Bold()`, `Italic()`, `SmallCaps()` and `Mono()`, and a game's own font is a face too, here the pixel font of the last label and the clock. `FontSize` sets the size in points and `Color` the colour. `Wrap` breaks the text between words so no line is wider than it. `Tracking(points)` adds that much room after each character, a whole emoji counting as one, as CSS letter-spacing does; a negative value tightens it, and `SetTracking` changes it. Measuring, wrapping and drawing all count it, so `Measure` agrees with what's drawn, and `twod.Measure(text, opts…)` measures a text without a canvas.

[main.go](../../examples/43-text/main.go)

## A font of your own

`twod.LoadFace` reads a TrueType font, a `.ttf`, from a file system, here the example's embedded `fonts` folder; `twod.NewFace` takes the same bytes from anywhere else. Other formats, such as OpenType with CFF outlines or WOFF, are refused with an error saying so. `twod.Pixelated()` on a text draws its glyphs unsmoothed on the pixel grid, for a pixel font such as Silkscreen at its own sizes; `Pixelated()` reads it back. Silkscreen comes under the SIL Open Font Licence, which travels with the font, so its licence file sits next to it.

[font.go](../../examples/43-text/font.go)

## Changing it

`SetText` replaces the words while the game runs, here with the time since the start.

[clock.go](../../examples/43-text/clock.go)

## Try it

[43-text](../../examples/43-text)
