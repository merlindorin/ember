# Text

Words on screen: a title, a score, a chat line, a button's label. This section shows how ember draws them, and with which fonts.

## Faces and sizes

A face is a font: its letters, ready to draw. The engine ships five, `twod.Regular()`, `Bold()`, `Italic()`, `SmallCaps()` and `Mono()`. Text picks one with `Typeface`, and its size in points with `FontSize`; with neither, it is 16-point Regular.

## Your own fonts

`twod.LoadFace(files, "fonts/X.ttf")` reads a TrueType font, a `.ttf`, from a file system such as the game's embedded files; `twod.NewFace` takes the same bytes from anywhere. Only TrueType is read: OpenType with CFF outlines, font collections and WOFF are refused with an error that says so. Keep the font's licence next to it.

## Fallbacks and emoji

A face draws only the characters it has. `With(twod.Fallback(other))` gives it faces to fall back on, character by character, which is how text holds emoji, other alphabets or symbols. Colour emoji are pictures, added with `twod.Emoji(set)`.

## Bitmap fonts

A bitmap font is a picture of its characters, drawn pixel for pixel, as old arcade games did. It suits pixel art, and needs no font file.

## Where text appears

- In the 2D world: `twod.NewText`, placed and drawn like any 2D entity.
- In the UI: labels, buttons, checks and text inputs take the same `Typeface` and `FontSize`.
- For a whole game: a Document's `@font-face` and `font-family` give every element a font at once, as on the [Themes](../05-ui/09-walkthroughs/05-themes.md) page.
