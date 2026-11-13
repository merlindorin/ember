# Tints, recolours and masks: variants of a picture without a new picture

Status: **approved by the user (2026-11-13), with the answers under "Decided".** Nothing here is built. Decision
number reserved: 0069.

## Why

When a level or a monster needs a variant of pack art, the game writes a one-off program that bakes a new picture:

| Where (Lantervale) | What is baked | What it is |
|---|---|---|
| `levels/act0/derived` (`water-dark.png`) | the pack's water, each colour moved down its ramp | a recolour |
| `crypt/slimes/gen.go` (`crypt/franuka/slimes/{blue,orange,red}`) | every slime sheet in three more colours | the same recolour, three times |

Each costs a generator, its tests, a set of pictures to keep in step with the pack, and for tiles a tileset of its
own that Tiled and the level copy must know. A change of shade means running a program and reviewing a picture diff.
The next variant (a broken edge on a floor tile, a crumbled corner, a puddle's outline) would be baked the same way.

None of these is new art. Each is an existing picture through one of three operations, and the operation is what
should be kept, not its output.

**Out of scope:** the game's UI (`lantervale/gui`: greyed buttons, selected rings) is a different matter, styled by
the ui and its CSS, and stays as it is. New art drawn over tiles, like the crypt's engravings, is also not a
variant: it stays a picture, listed and approved before use.

## The three operations

| Operation | What it does | When it is applied | In the engine today |
|---|---|---|---|
| **Tint** | multiplies the picture's colours and opacity by one colour | when drawn, on the GPU, free | `twod.Tint` on any entity; not on a tile or a Tiled layer |
| **Recolour** | swaps exact colours for others, by a named palette; other colours stay | once, when the picture loads | nothing |
| **Mask** | keeps the picture only where a mask picture is opaque | once, when the picture loads | whole layers only (`AsMask`, `MaskedBy`), as a pass |

They compose in that order: recolour, then mask, then tint.

Recolour and mask run on the CPU at load and give a new `twod.Texture`, cached by its source and its operation, so
two tiles asking for the same variant share one texture and a frame costs nothing more. A recoloured sheet takes
the memory the baked PNG takes today.

## 1. Recolour

```go
// A palette swaps colours: each From becomes its To, exactly; any other colour stays as drawn.
pal := twod.NewPalette(
    twod.Swap(color.Hex("#686f99"), color.Hex("#2c354d")),
    twod.Swap(color.Hex("#404973"), color.Hex("#14182e")),
)
pal, err := twod.LoadPalette(fsys, "palettes/dark-water.png") // two rows of swatches: from above, to below

dark := twod.Recolour(water, pal)                       // a new texture, cached
sheet, err := twod.LoadTexture(fsys, name, twod.Recoloured(pal))
```

- **Exact colours.** Pack art is a fixed palette, so a swap matches a colour exactly, alpha kept. A picture that
  holds none of a palette's colours is an error at load, since the palette then does nothing and was meant for
  another picture.
- **Two ways to write one.** In Go or data as pairs of colours, or as a small picture of two rows of swatches that
  an artist edits in a pixel editor. The engine reads both; a game picks.
- **Animated sheets and atlases** are recoloured whole, every frame and tile where it was.

## 2. Mask

A mask is a picture whose opacity says what to keep: opaque keeps, clear cuts, in between fades.

```go
cut := twod.Masked(tex, region, mask, maskRegion) // a new texture of region's size, cached
```

For tiles, masks are tiles too: a **mask tileset**, cell-sized shapes (a broken edge, a corner bitten off, a
ragged hole), each named by its `name` property. One mask works on any tile of that size, which is the point: ten
floors and five broken edges are fifteen pictures, not fifty.

- `twod.Inverted()` keeps where the mask is clear, so one shape gives the hole and the plug.
- A tile's collision shapes, height and other properties are the uncut tile's. A mask changes what is drawn, never
  what blocks.

## 3. Tint on tiles

`twod.Tint` reaches a `TileMap` as a whole (with step B of [Tiled maps](tiled-maps.md), a Tiled layer's tint and
opacity). A tile gets its own with a `tint` property, drawn through the tint the tile draw already takes.

## 4. In Tiled

The three become properties the engine declares, with the classes of [Tiled maps](tiled-maps.md) §5, on a tile, a
tile object or a tile layer:

| Property | Type | On | Meaning |
|---|---|---|---|
| `tint` | colour | tile, tile object, layer | multiplies its colours; on a layer it adds to Tiled's own layer tint |
| `recolour` | string | tile, tile object, layer | the name of a palette the game gave the loader |
| `mask` | string | tile, tile object | the `name` of a tile of the game's mask tileset |

The game hands the loader its library, and the loader refuses a name it does not hold, with the nearest one:

```go
twod.LoadTiled(fsys, name, twod.Palettes(pals), twod.Masks(maskTileset))
```

**A mask layer**, for cutting cell by cell without touching tiles: a tile layer of class `mask` with the property
`cuts` naming another tile layer. Its tiles come from the mask tileset; each cuts the tile under it in the named
layer. It is painted like any layer, and never drawn itself. The cut is made at load, tile by tile, through the
same cache, so it is not a render pass.

**What Tiled shows.** Tiled previews its own layer tint and nothing else here. A recoloured tile shows in its
original colours, a masked tile whole; a mask layer shows the mask shapes over the tiles, which is a fair sketch of
the cut. The game is where the result is seen. This is the price of not baking: with a baked tileset, Tiled shows
the final picture.

## 5. Sprites and sheets

A monster's or a prop's variant is the same recolour on its sheet: `twod.LoadTexture(…, twod.Recoloured(pal))`. A
game names the palette in its data, beside the sheet.

## What Lantervale does with it

| Today | With this |
|---|---|
| `water-dark.png`, `library/custom/water-dark.tsj`, half of `levels/act0/derived` | the pack's `water.tsj` on the `deep` layer, the layer's `recolour` set to `dark-water`; one palette |
| `crypt/slimes/gen.go`, three folders of baked sheets | the green sheets, and three palettes named in the slimes' data |
| the next broken floor or crumbled edge | a mask from the mask tileset, picked in Tiled |

- **The library** lives once in the game: palettes in `data/palettes.yaml` (pairs of colours, the game designer's,
  readable in a diff), masks in `levels/library/masks/` as one tileset.
- **Masks are pictures nobody has drawn yet.** They are not from a pack, so each is listed and approved by the user
  before use, as the milestone asks for anything not from a pack.
- **The page of classes and properties** (`lantervale/docs/04-world/07-tiled-classes.md`) lists the palettes and the
  masks by name.
- **The rule for the team**, once this is built: no seat bakes a variant of pack art into a new picture when a
  tint, a recolour or a mask gives it. Until then, a seat asks the CTO before generating any picture.

## Alternatives

- **Recolour on the GPU** (a palette lookup in the shader, per draw): no extra texture, a variant per entity for
  free. It adds a lookup to every pixel of every 2D draw or a second shader that breaks batches, for sheets of a
  few kilobytes. Not proposed; it can come later behind the same `Palette` if a game wants hundreds of variants.
- **Masks as layer passes** (`AsMask`, `MaskedBy`, as today): right for a moving mask such as a lantern's circle,
  wrong for static tiles, where it costs a pass per masked layer every frame.
- **Keep baking, with one shared generator**: still pictures to keep in step with the pack, still a tileset per
  variant, still a program to run for a shade.

## Order and owners

| Step | What | Owner | Tests |
|---|---|---|---|
| R1 | `Palette`, `Recolour`, `Recoloured`, `LoadPalette`, the cache | engine-core, in `ember/twod` | exact swap, alpha kept, a palette that matches nothing refused, one texture for two equal asks, a recolour of the water sheet timed |
| R2 | `Masked`, `Inverted`, the mask tileset by name | engine-core, in `ember/twod` | opaque keeps, clear cuts, half fades; a mask of another size refused; collision unchanged |
| R3 | the Tiled properties, `Palettes`, `Masks`, the mask layer; a tile's `tint` | engine-render, after Tiled maps steps G and I | each property on a tile, an object and a layer; unknown names refused with the nearest; a mask layer cutting the named layer |
| R4 | Lantervale: dark water and slimes on palettes; the generators and baked pictures deleted | lantervale-dev, game-designer for `palettes.yaml` | the levels and slimes draw as before, checked against captures; shown to the user before landing |
| R5 | the guide: a page for the three operations, with pictures | doc-writer | the guide test |

R1 and R2 are new files in `ember/twod` and touch nothing of the Tiled work, so they can be built while
engine-render is on Tiled maps.

## R3 as built

- `tint`, `recolour` and `mask` are in `twod.EngineTileProperties()` and read on any layer or object, of a class or
  not; the project file declares `ember.Layer`, `ember.Object` (the three as typed fields) and `mask`. A game's
  layer and object classes take the properties too, added in Tiled by hand: their members stay the game's own.
- Nearer wins: an object's recolour or mask over its layer's (or its nearest group's), a layer's over its tiles'
  own; tints multiply. A layer's recolour leaves a tile whose picture holds none of its colours as it is, and is an
  error when it recolours none of the layer's tiles.
- A mask layer's cells name their mask by the tile's `name`, looked up in `Masks`; a cell turned in Tiled cuts
  turned. `cuts` names a sibling layer, else the one layer of the map so called. Cells keep their tile ids, so
  collision and light read the uncut tile; `Set` and `SetFlip` keep a cell's recolour and cut.
- The tileset tool leaves the three out of its panel: it has no palettes or masks to preview them with, and it
  refuses a tileset whose tiles name them.

## Decided (user, 2026-11-13)

1. Lantervale's palettes are pairs of colours in YAML (`lantervale/data/palettes.yaml`, made in R4). The engine
   reads swatch pictures too.
2. Tiled not showing a recolour or a cut is accepted: no baked tilesets, the game is where the result is seen.
3. engine-core builds R1 and R2 now, in `ember/twod`, with engine-render's agreement on the files.
4. The first masks are a broken edge, a bitten corner and a ragged hole. level-designer proposes them as mockups;
   the user approves each before use.
