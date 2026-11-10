# 2D drawing: windows, twod entities, the canvas, tile maps, lights

Everything 2D is an entity built with options and shown in the window it is a child of (`win.Add(e)`): `platform`, `twod`, `color`, `geom`, `twod/light`.

## Open a window

```go
win := platform.NewWindow(platform.Title("game"), platform.Size(640, 360), platform.FixedSize())
ember.New().AddEntities(win).AddSystems(play).MustRun()
```

| Option | Use |
|---|---|
| `Title(title string)`, `Size(width, height int)`, `Background(c color.Color)`, `FixedSize()`; else `MinSize(w, h int)`, `MaxSize(w, h int)`, `AspectRatio(w, h int)` | size in points; examples are fixed unless their layout follows the window |
| `Fullscreen()`, `Maximized()`, `Position(x, y int)`, `Undecorated()`, `AlwaysOnTop()`, `Hidden()`, `VSync(on bool)`, `Icon(images ...image.Image)` | |

At run time: `win.SetTitle`, `SetSize(geom.Vec2)`, `SetFullscreen(bool)`, `SetBackground`, `SetCursor(platform.CursorHand)`, `Size()`, `Input()`. `platform.Hidden()` and `twod.Hidden()` differ.
In the browser a resizable window fills the page and follows it, a `FixedSize()` one keeps its size, and full screen (the game's or the page's) fills the screen.

## Draw sprites, shapes and text, and move them

Shared options (`twod.Option`, any entity): `At(x, y float32)`, `Rotation(radians float32)`, `Scale(x, y float32)`, `Tint(c color.Color)`, `Z(z int)`, `SortByY()`, `Blend(m BlendMode)`, `Hidden()`, `Name(id string)`.
Kind options fail to compile on the wrong kind: `Pivot(x, y float32)` (sprite, nine-patch, text), `Size(w, h float32)` (sprite, nine-patch), `Color(c color.Color)` (line, polygon, text).

| Entity | Constructor | Run-time setters |
|---|---|---|
| sprite | `NewSprite(tex *Texture, opts ...SpriteOption) *Sprite` | `SetTexture`, `SetRegion(geom.Rect)`, `SetFlip(x, y bool)`, `SetSize`, `Play(name)`, `Stop()`, `Pause()`, `SetFrame(i)` |
| text | `NewText(text string, opts ...TextOption) *Text` | `SetText`, `SetColor`, `SetFontSize`, `SetTypeface`, `SetWrap`, `Measure() geom.Vec2` |
| line | `NewLine(points []geom.Vec2, opts ...LineOption) *Line` | `SetPoints`, `SetWidth`, `SetColor`, `SetClosed` |
| polygon (filled, concave ok) | `NewPolygon(points []geom.Vec2, opts ...PolygonOption) *Polygon` | `SetPoints`, `SetColor` |
| nine-patch | `NewNinePatch(tex *Texture, opts ...NinePatchOption) *NinePatch` | `SetSize`, `SetMargins(l, t, r, b float32)` |

Every entity also has `SetPosition(geom.Vec2)`, `Translate(delta geom.Vec2)`, `SetRotation`, `SetScale(geom.Vec2)`, `Position()`, `GlobalPosition()`, and from its look `SetTint`, `SetHidden`, `SetZ`, `SetBlend`.
- Sized entities are centred on `At` by default; `Pivot(0, 0)` puts the top-left there (text included).
- Text: `FontSize(size float32)`, `Typeface(twod.Bold())` (`Regular`, `Bold`, `Italic`, `SmallCaps`, `Mono` are functions), `Wrap(width float32)`, `Tracking(points)`, `Baseline()`, `TextShadow(c)`; white 16-point Regular by default.
- Line: `Width(w float32)`, `Joints(twod.Miter|twod.Bevel|twod.RoundJoint)`, `Caps(start, end Cap)` (`Flat`, `Square`, `RoundCap`), `Closed()`.
- Textures: `LoadTexture(fsys fs.FS, name string, opts ...TextureOption)` from an `embed.FS`, `NewTexture(img image.Image, opts...)`; `twod.Pixelated()` for pixel art.
- Moved in `FixedUpdate`, an entity draws interpolated between steps; call `Snap()` after a teleport.

```go
package main

import (
	"fmt"
	"image"
	stdcolor "image/color"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

type Walk struct {
	hero  *twod.Sprite
	score *twod.Text
	in    *input.State
	steps int
}

func (w *Walk) Update(f world.Frame) {
	dir := geom.V2(w.in.Axis(input.KeyLeft, input.KeyRight), w.in.Axis(input.KeyUp, input.KeyDown))
	if dir != (geom.Vec2{}) {
		w.hero.Translate(dir.Normalize().Scale(160 * f.DT))
		w.hero.SetFlip(dir.X < 0, false)
	}
	if w.in.Pressed(input.KeySpace) {
		w.steps++
		w.score.SetText(fmt.Sprintf("Jumps: %d", w.steps))
	}
}

func square(n int, c stdcolor.RGBA) *twod.Texture {
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for i := range img.Pix {
		img.Pix[i] = []uint8{c.R, c.G, c.B, c.A}[i%4]
	}
	return twod.NewTexture(img, twod.Pixelated())
}

func main() {
	win := platform.NewWindow(platform.Title("2d"), platform.Size(640, 360), platform.FixedSize())
	hero := twod.NewSprite(square(8, stdcolor.RGBA{239, 131, 84, 255}), twod.At(320, 180), twod.Scale(4, 4))
	score := twod.NewText("Jumps: 0", twod.At(16, 12), twod.Pivot(0, 0), twod.FontSize(20))
	ground := twod.NewPolygon(
		[]geom.Vec2{geom.V2(0, 300), geom.V2(640, 300), geom.V2(640, 360), geom.V2(0, 360)},
		twod.Color(color.Hex("#5f8f4e")),
	)
	path := twod.NewLine([]geom.Vec2{geom.V2(40, 280), geom.V2(600, 280)}, twod.Width(3), twod.Color(color.White))
	win.Add(ground, path, hero, score)
	ember.New().AddEntities(win).AddSystems(&Walk{hero: hero, score: score, in: win.Input()}).MustRun()
}
```

Examples: `examples/19-sprites`, `examples/21-lines-and-shapes`, `examples/22-nine-patch`, `examples/43-text`.

## Animate a sprite sheet

```go
frames := twod.Strip(sheet, twod.Cell(16, 16)) // []Frame, row by row
clips := twod.NewAnimations().
	Add("walk", 8, frames[8:12], twod.Loop()). // Add(name string, fps float32, frames []Frame, opts ...ClipOption)
	Add("hit", 12, frames[12:14], twod.Once())
hero := twod.NewSprite(sheet, twod.Animate(clips), twod.Playing("walk"))
hero.Play("hit") // a finished Once clip emits twod.AnimationFinished
```
Still frames: `twod.Still(frames[3])` or `SetStill(f)`; part of a texture: `twod.Region(geom.Rect{...})`. `examples/20-sprite-sheet`.

## Order, layers, a fixed HUD and a camera

- Draw order: layer number, then `Z`, then `SortByY()` (feet lower draw in front), then the order added.
- `NewLayer(n int, opts ...LayerOption) *Layer`: `Fixed()` ignores the camera (HUD), `Parallax(f)`, `ClipTo(r geom.Rect)`, `Anchor(x, y)` (a fixed layer's origin as a share of the window), `Blend(...)`.
- `NewCamera(opts ...CameraOption) *Camera`, added to the window: `Follow(e)`, `Smoothing(speed)`, `Bounds(r geom.Rect)`, `Zoom(z)`, `ViewSize(w, h float32)` (keep w×h in view, scale to fit), `Resolution(w, h int)` (pixel-perfect, letterboxed), `Offset(x, y)` (shake), `Layers(...)`.
- Setters: `SetFollow`, `SetZoom`, `SetBounds`, `SetOffset`, `SetViewSize`. Mouse to world: `cam.ScreenToWorld(win.Input().Mouse())` (`WorldToScreen` back); without a camera, world points are window points.

```go
package main

import (
	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

type Aim struct {
	cam    *twod.Camera
	marker *twod.Polygon
	in     *input.State
}

func (a *Aim) Update(world.Frame) {
	if a.in.Pressed(input.MouseLeft) {
		a.marker.SetPosition(a.cam.ScreenToWorld(a.in.Mouse()))
	}
}

func main() {
	win := platform.NewWindow(platform.Size(640, 360), platform.FixedSize(), platform.Background(color.Hex("#9cc7e8")))
	diamond := []geom.Vec2{geom.V2(0, -10), geom.V2(10, 0), geom.V2(0, 10), geom.V2(-10, 0)}
	marker := twod.NewPolygon(diamond, twod.Color(color.Hex("#d9534f")))
	cam := twod.NewCamera(twod.Zoom(1.5), twod.Bounds(geom.Rect{W: 1600, H: 360}))
	win.Add(
		twod.NewLayer(0).Add(marker),
		twod.NewLayer(1, twod.Fixed()).Add(twod.NewText("click to mark", twod.At(16, 12), twod.Pivot(0, 0))),
		cam,
	)
	ember.New().AddEntities(win).AddSystems(&Aim{cam: cam, marker: marker, in: win.Input()}).MustRun()
}
```
Examples: `examples/23-2d-camera`, `examples/28-y-sort`, `examples/30-pixel-perfect`.

## Draw a game's board: a custom entity on the canvas

This is how a game draws its board or play area: a twod entity, never ui, so effects, blend modes and lights can go on it.
It embeds `twod.Item` from `twod.Base()`, is finished by `twod.Build(e, name, opts...)`, and draws in `Draw(c *twod.Canvas)`, in its own space, every frame.
A HUD widget drawn by `ui.Canvas(func(c *twod.Canvas))` (see ui.md) uses the same methods, but redraws only after `doc.Invalidate(node)`.

| Canvas method | Draws |
|---|---|
| `Rect(r geom.Rect, c color.Color)`, `RoundRect(r geom.Rect, radius float32, c color.Color)` | filled boxes |
| `Border(r geom.Rect, radius, thickness float32, c color.Color)` | an outline |
| `Circle(center geom.Vec2, radius float32, c color.Color)`, `Ring(center, radius, thickness, c)` | |
| `Line(a, b geom.Vec2, width float32, c color.Color)` | |
| `Gradient(r geom.Rect, top, bottom color.Color)`, `Glow(r, radius, spread, c)` | |
| `Image(r geom.Rect, tex *Texture, uv geom.Rect, tint color.Color)` | uv is 0..1 of tex |
| `NinePatch(r, tex, left, top, right, bottom float32, tint)` | |
| `Text(at geom.Vec2, text string, opts ...TextOption) geom.Vec2` | top-left at `at`, returns its size |
| `Measure(text string, opts ...TextOption) geom.Vec2` | size without drawing (also `twod.Measure`) |
| `PushClip(r)` / `PopClip()`, `Size()`, `Mouse()`, `Input()` | |

```go
package main

import (
	"fmt"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

// Bar is a health bar that draws itself.
type Bar struct {
	twod.Item
	health float32
}

func (b *Bar) Draw(c *twod.Canvas) {
	r := geom.Rect{W: 200, H: 24}
	c.RoundRect(r, 6, color.Hex("#2b2d42"))
	c.RoundRect(geom.Rect{W: 200 * b.health, H: 24}, 6, color.Hex("#c0392b"))
	label := fmt.Sprintf("%.0f%%", b.health*100)
	size := c.Measure(label, twod.FontSize(14))
	c.Text(geom.V2((r.W-size.X)/2, (r.H-size.Y)/2), label, twod.FontSize(14))
}

type Drain struct{ bar *Bar }

func (d *Drain) Update(f world.Frame) { d.bar.health = max(d.bar.health-0.1*f.DT, 0) }

func main() {
	win := platform.NewWindow(platform.Size(640, 360), platform.FixedSize())
	bar := twod.Build(&Bar{Item: twod.Base(), health: 1}, "health", twod.At(220, 160))
	win.Add(bar)
	ember.New().AddEntities(win).AddSystems(&Drain{bar: bar}).MustRun()
}
```
A HUD widget on the same canvas: `examples/17-custom-elements`.

## Colours

`color.Hex("#ef8354")`, `RGB(r, g, b float32)`, `RGBA(r, g, b, a float32)`, `RGB8(r, g, b uint8)`, `Gray(v)`, `HSV(h, s, v)`, `White`, `Black` (0..1); `Alpha(a)`, `Lerp(o, t)`, `Mul(o)`, `Scale(k)`, `Hex()`, `ParseHex(s) (Color, error)`.

## Tile maps and Tiled

- `NewTileset(tex *Texture, cell Grid, opts ...TilesetOption) *Tileset` with `twod.Cell(16, 16)`; tile ids count from 1 along the sheet, 0 is empty.
- Tileset options: `Solid(ids...)`, `Tall(ids...)`, `TileAnimation(id int, fps float32, frames ...int)`, `Collision(id, shapes...)`.
- `NewTileMap(tileset, opts ...TileMapOption)` from `ASCII(text string, legend map[rune]int)`, `CSV(text)` or `Cells(grid [][]int)`, sized by `CellSize(w, h float32)`. Its position is the top-left of cell 0,0 (not centred).
- At run time: `Set(x, y, id int)`, `At(x, y) int`, `Cell(p geom.Vec2) (x, y int, ok bool)`, `Center(x, y) geom.Vec2`, `Solid(x, y) bool`.
- Tiled: `level, err := twod.LoadTiled(files, "level.tmj")`, then `win.Add` each of `level.Layers()`; spawns from `level.Objects("things")` (`Name`, `Type`, `Rect`, `Properties`).
- Objects keep their shape: `o.Shape` is `TiledRectangle`, `TiledEllipse`, `TiledPoint`, `TiledPolygon` or `TiledPolyline`; `o.Points` are a polygon's or polyline's points in map pixels; `Rect` is Tiled's x, y, w, h (a point sized 0). A tile object has `o.Tile` (`*TiledTile{GID, Tileset, ID, Flip}`, nil otherwise; `ID` counts from 1, `Rect.Y` is its picture's bottom); draw it with `o.Tile.Tileset` and `o.Tile.ID`.
- Only used tilesets are loaded (decision 0067): `level.UsedTilesets()` are those a tile layer or tile object takes a tile from. An unused tileset of its own .tsj isn't even read (its file may be absent): in `level.Tilesets()` it's an empty Tileset with its `Source()`; an unused embedded one keeps properties and shapes, no pictures. `tileset.ImageFile(id)` is a tile's picture file in the fsys.
- Collision with tile maps is `collide` (`space.AddSolid(tileMap)`), paths `nav`.

```go
package main

import (
	"image"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

const level = `
..........
..####....
..#..#....
..........`

type Dig struct {
	room *twod.TileMap
	in   *input.State
}

func (d *Dig) Update(world.Frame) {
	if x, y, ok := d.room.Cell(d.in.Mouse()); ok && d.in.Pressed(input.MouseLeft) {
		d.room.Set(x, y, 1)
	}
}

func main() {
	sheet := image.NewRGBA(image.Rect(0, 0, 32, 16)) // two 16×16 tiles: 1 floor, 2 wall
	for i := range sheet.Pix {
		sheet.Pix[i] = []uint8{90, 140, 80, 255, 120, 110, 100, 255}[(i/64%2)*4+i%4]
	}
	tiles := twod.NewTileset(twod.NewTexture(sheet, twod.Pixelated()), twod.Cell(16, 16), twod.Solid(2))
	room := twod.NewTileMap(tiles, twod.ASCII(level, map[rune]int{'.': 1, '#': 2}), twod.CellSize(32, 32))
	win := platform.NewWindow(platform.Size(320, 128), platform.FixedSize())
	win.Add(room)
	ember.New().AddEntities(win).AddSystems(&Dig{room: room, in: win.Input()}).MustRun()
}
```
Examples: `examples/26-tilemap`, `examples/27-collision`, `examples/29-tiled`.

## Particles, blend modes, effects (briefly)

```go
sparks := twod.NewParticles(nil, twod.At(x, y), twod.Burst(80), twod.OneShot(), twod.Lifetime(0.6, 1.4),
	twod.Velocity(60, 220), twod.Direction(-geom.Pi/2, geom.Pi), twod.Gravity(geom.V2(0, 260)),
	twod.ColorOverLife(color.Hex("#fff3b0"), color.RGBA(1, 0.3, 0.05, 0)), twod.SizeOverLife(5, 1), twod.Blend(twod.Add))
night := twod.NewLayer(1, twod.Blend(twod.Multiply))   // Alpha, Add, Multiply, Screen, Lighten
ripples := twod.NewLayer(0, twod.WithEffect(render.NewEffect(wgsl))) // fn effect(vUV: vec2f) -> vec4f
```
A `nil` texture draws round dots; `Emit(n)`/`SetEmitting(bool)` at run time; a finished one-shot emits `twod.ParticlesFinished`. `examples/34-particles`, `examples/31-blend-modes`, `examples/33-2d-shader`.
- Gotcha: an emitter emits 10 a second by default; one meant only for `Emit(n)` needs `twod.Rate(0)`, or it leaves dots where it stands.
- Gotcha: an emitter moved with `SetPosition` just before `Emit` needs `Snap()` too, or it draws from where it was.

## Lights (short)

```go
cam := twod.NewCamera(light.Ambient(color.RGB(0.08, 0.09, 0.16)))      // darkness; ember.New installs lighting
floor := twod.NewLayer(0, light.Lit())                                // layers that receive light
torch := light.NewPoint(140, color.RGB(1, 0.8, 0.5), light.At(320, 180)) // NewPoint(radius, c, opts...)
crate := twod.NewSprite(tex, light.Occluder())                        // casts a shadow
floor.Add(crate, torch)
torch.SetRadius(180) // setters apply the next frame: SetColor, SetIntensity, SetShadows(false)…
```
Also `light.NewCone(radius, c, angle, spread, opts...)`, `NewDirectional(c, angle, opts...)`, `NewShape(shape, c, reach, opts...)`, `light.Unlit()` for a sprite that ignores light; `examples/36-point-lights` to `examples/42-tiled-pictures`.
