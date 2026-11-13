# Sprites

A sprite draws a picture. Here one 16 × 16 pixel-art crate is drawn three times: as it is, turned, and mirrored with a green tint.

![Three crates: plain, turned, and mirrored in green](sprites.png "ember · sprites")

## The program

`twod.LoadTexture` reads the crate from the example's own files, embedded with `go:embed`, and `twod.Pixelated()` keeps its pixels sharp when it is scaled up. Each sprite is made with options: `At` places its centre, `Scale` and `Rotation` size and turn it, `Tint` colours it, and `Flip`, a sprite's own option, mirrors it. At `Scale(1, 1)` one texture pixel is one point, so the 16-pixel crate needs `Scale(6, 6)` to be 96 points wide. They are the window's children, so they show in it. `Hidden()` keeps an entity from drawing, and its children with it; `SetChildrenHidden(true)` draws an entity without its children, which keep their own `Hidden`, as a folded UI panel does with what's inside it. The same picture in other colours, or cut to a shape, is a new texture made at load: see [Recolours and masks](21-recolours-and-masks.md).

[main.go](../../examples/19-sprites/main.go)

## Try it

[19-sprites](../../examples/19-sprites)
