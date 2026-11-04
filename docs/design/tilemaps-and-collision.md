# Tilemaps and 2D collision

Status: approved and built (`twod` tile maps, `ember/collide`, `ember/nav`). A design record: the guide documents the API as it is.

Two gaps from the raylib/Ebitengine comparison: tile-based levels, and moving things that walls stop. Both follow the 2D rules:
- ready-made entities, or values the game holds;
- typed options, and no exported config;
- no World resources, and no borrowed names.

The collision side is pure Go with no drawing in it, so servers use it too.

## Tilemaps, in twod

### Tilesets

A tileset is a texture atlas cut into tiles, with tile properties. It is a value shared by maps, like `Animations`.

```go
sheet, _ := twod.LoadTexture(files, "tiles.png", twod.Pixelated())
tiles := twod.NewTileset(sheet, twod.Cell(16, 16),
	twod.Solid(3, 4, 5),
	twod.Tall(7, 8),
	twod.Spacing(1, 1),
)
```

- **Ids:** they count tiles in the atlas left to right, then top to bottom, from 1; 0 is an empty cell.
- **Options:** `Solid(ids…)` marks tiles collision reads. `Tall(ids…)` marks tiles that rise above the ground and sort with characters (see Draw order). `Spacing(x, y)` and `Margin(px)` describe atlases with gutters.
- **Properties:** `tiles.Solid(id)` and `tiles.Tall(id)` read them back. Free-form properties come from Tiled (below) as `tiles.Property(id, name) (string, bool)`.
- **Animated tiles** (water, torches): `twod.TileAnimation(id, fps, frames…)`. The map draws every cell of that id on the clip's current frame, in step.
- **Cells taller than the map cell** (decision 0029): a tile that isn't Tall draws only the bottom of its tileset cell, one map cell's worth at the map cell's shape, never squashed; a Tall one stands its whole picture on the cell. So one wall set of 16×48 pieces serves flat tops and edges and standing walls.

### Tile maps

```go
ground := twod.NewTileMap(tiles, twod.Cells(level), twod.CellSize(32, 32))
win.Add(ground)
```

- **The entity:** `NewTileMap(tileset, …TileMapOption)` is a 2D entity, with the shared options (At, Z, Tint, Hidden) and its own:
  - `Cells(grid [][]int)`, a grid of ids by row;
  - `CSV(text)`, the same as comma-separated rows;
  - `ASCII(text, legend map[rune]int)`, a picture of the level, `#` for a wall and `.` for floor, as the examples use;
  - `CellSize(w, h)`, its size on screen in points (default the tileset's pixel size);
  - `Isometric()`, diamond tiles.
- **Changing it:**
  - `Set(x, y, id)` and `At(x, y) int` edit and read cells;
  - `Cell(point) (x, y, ok)` and `Center(x, y) geom.Vec2` convert between map and layer space;
  - `Columns()` and `Rows()` give its extent in cells.
- **Culling:**
  - Only the cells that intersect the view are drawn. The view rectangle in the map's space is the inverse of the canvas transform, so a 1000×1000 map draws one screen of cells.
  - Isometric maps draw every cell, back to front, for now.
- **No resource, no tree:** a map is a plain entity, saved as a scene node `twod.TileMap` with its cells and tileset as resources.

### Infinite maps

A TileMap is finite: its cells are a slice the game fills. A world that never ends, such as a Survivors run, needs cells that exist only where someone looks. `InfiniteMap` is that map: it asks the game for chunks of cells on demand, keeps the recent ones and forgets the far ones.

```go
ground := twod.NewInfiniteMap(tiles,
	twod.ChunkSize(16),
	twod.Generator(func(cx, cy int, cells [][]int) {
		r := rand.New(rand.NewPCG(seed, uint64(cx)<<32|uint64(uint32(cy))))
		for y := range cells {
			for x := range cells[y] {
				cells[y][x] = grass[r.IntN(len(grass))]
			}
		}
	}),
	twod.CellSize(32, 32),
	twod.TallIn(world),
)
```

- **Chunks on demand:**
  - The generator fills one chunk, `ChunkSize` cells square (default 16). Its cell `cells[y][x]` is map cell `cx*size+x, cy*size+y`. The generator must be deterministic for a chunk: a chunk that was forgotten and comes back is generated again and must look the same. The usual way is a random source seeded from the game's seed and `cx, cy`.
  - A chunk is generated the first time anything reads it: drawing, `At`, `Solid`, `SolidIn`.
  - `KeepChunks(n)` (default 64) bounds memory. Past n chunks, the least recently used one is forgotten, with its tall-tile sprites, except:
    - chunks the cameras need this frame, so a view never loses its own chunks;
    - chunks changed with `Set`, which are kept for good, since the generator can't rebuild them.
- **Drawing:** only the cells in view, as TileMap. Each frame, before anything is drawn, the map makes the chunks every camera of its window shows (main and `Into` cameras such as a minimap) and the ring just outside, whether or not that camera draws the map's layer, so tall tiles stand before they scroll in. Those chunks are kept until the next frame's, whatever `KeepChunks` says.
- **The same words as TileMap:**
  - the same `Tileset`, its solid, tall and animated tiles;
  - `CellSize` and `TallIn`, shared by both maps;
  - `At`, `Set`, `Solid`, `Center`, `Cell`, `SolidIn`, `TallSprites` and `Version`.
  - `Cell(p)` has no `ok`, since every point is in the map.
  - Getters for every option: `ChunkSize()`, `Generator()`, `KeepChunks()`, `CellSize()`. `Chunks()` is how many chunks are held.
  - Not supported: isometric layout, and scene saving. Its cells are the generator's, so they aren't saved.
- **Collision:** the map is a `collide.Solid`. `SolidIn` generates the chunks it covers, so bodies collide across chunk borders, and off screen too (an enemy spawned beyond the view).
- **Pathfinding:** nav needs a finite grid. `ground.Area(x, y, columns, rows)` is a window of cells that is a `nav.Grid` (and `Version`ed with the map), so `nav.New(ground.Area(…))` finds paths within it. A game re-centres it by making a new area where its chasers are. nav over the whole infinite map is not supported.
- **Precision:**
  - Positions are float32: 24 bits of mantissa, so 1/64 pt of resolution at 131,072 pt from the origin and 1/32 pt at 262,144.
  - A Survivors run of 30 minutes walking straight at 150 pt/s ends 270,000 pt out. There a frame's 2.5 pt step is exact to about 1%, and the camera's translation cancels the entity's to within 1/32 pt on screen: invisible.
  - So there is no floating origin for now. If a game goes much further (hours of travel, fast vehicles), the fix is a floating origin: shift the world back near zero now and then. It would be a twod operation, not a map option.
  - A test draws the map with a camera 270,000 pt out and checks its cells land on whole points.
- **Alternatives:**
  - **An `Infinite` option on TileMap:** a single type, but `Columns`, `Rows`, out-of-map solids and isometric drawing all change meaning, and nav would take an unbounded grid. A type of its own keeps both honest.
  - **A per-cell generator, `func(x, y int) int`:** simpler, but a chunk generator can place things larger than a cell (a pond, a ruin) and seed once per chunk.
  - **Forgetting by distance from the camera:** it needs a camera, and servers have none. A count with LRU works headless too.

### Reading Tiled

`twod.LoadTiled(fsys, "level.tmj") (*twod.Tiled, error)` reads Tiled's JSON format, `.tmj`, with its `.tsj` tilesets. JSON only: `encoding/json`, no XML dependency.
- **Layers:** tile layers become `TileMap`s (CSV or base64 cells), in order. `Tiled.Layers()` returns them, named, and `Tiled.Layer(name)` finds one.
- **Tilesets:** they become `Tileset`s. A tile's boolean `solid` and `tall` properties map onto `Solid` and `Tall`; other properties stay readable through `Property`.
- **Objects:** object layers become `Tiled.Objects(layer)`, each with a name, type, rectangle and properties, for spawn points and triggers. They are data, not entities: the game decides what to spawn.
- **Supported:** orthogonal and isometric maps. Staggered and hexagonal maps, infinite maps and image layers return an error naming the feature.

### Draw order

- **Ground:** a tile map draws with its layer, z and tint like any entity. Floors and walls seen from above need nothing more.
- **Tall tiles and characters:** for tall tiles to sort with characters (trees, pillars, the back walls of a top-down scene), the map does not draw them itself. Instead it adds one child `Sprite` per tall cell, pivoted at its base, `Pivot(0.5, 1)`, at the cell's bottom centre.
  - `TallIn(parent)` stands them in another parent, such as a `SortByY()` layer holding the characters, so y-sort orders them all by their feet.
  - `TileMap.TallSprites()` lists them, and `Set` keeps them in step.
- **Isometric order:** an isometric map draws its cells back to front, by row plus column, and its tall tiles sort by their feet through the same `SortByY`.

## 2D collision

### Recommendation

There are two layers:
1. **Shape tests in `geom`,** pure functions on values, for anyone:
   - `geom.Circle{Center, Radius}`, `Rect.Overlaps(Rect)` and `Circle.Overlaps(Circle)`;
   - `Rect.OverlapsCircle(Circle)`, plus `Contains` for points on both;
   - `geom.Sweep(moving Rect, delta Vec2, still Rect) (t float32, normal Vec2, hit bool)`, the swept AABB, with `SweepCircle` for circles against rects.
2. **Movement with collision in a new pure package, `ember/collide`.** It is a space the game holds, with bodies as entities: no physics, no forces, just "move this much, stop at what is solid, and tell me what you touched".

It isn't a `twod` component because servers move bodies too: Emberfall's shard moves heroes on the ground plane, and must not pull in twod's drawing. It isn't only `geom` either, because moving many bodies against each other needs a broad phase, plus a way to tell which entity was hit.

### API

```go
space := collide.NewSpace(collide.Grid(64))
w.AddSystems(collide.NewTracker(space))

hero := twod.NewSprite(sheet, twod.At(64, 64))
hero.Add(collide.NewBody(collide.Box(12, 8), collide.Offset(0, -4)))
space.AddSolid(walls)

moved, hits := space.Move(hero, geom.V2(dx, dy))
```

- **Bodies:** `collide.NewBody(shape, …BodyOption)` is an entity you add (`Add`) to the entity it belongs to, so it follows it and leaves with it. The tracker files it as soon as it joins the World.
  - Shapes are `Box(w, h)` and `Circle(r)`. Options: `Offset(x, y)`, `Trigger()` (detected but never blocking) and `Group(n)`/`CollidesWith(n…)` (which bodies stop which).
  - Its position is its parent's, read through a small `Positioned2D` interface (`Position`/`SetPosition` in a plane). twod's `Placement` has it, and Emberfall's 3D entities adapt X and Z.
- **The space** is a spatial hash of bodies, kept in step by `NewTracker(space)`, a system late in each fixed step, after the moves. It reports trigger `Entered` and `Left` per step, in a stable order, so a fixed-step simulation stays deterministic; Update still sees them, since events live two frames. Bodies a game moves outside `space.Move` are filed again at the next step.
  - `space.Move(e, delta) (geom.Vec2, []Hit)` sweeps e's body against solid bodies and solid tiles. It slides along what it hits (up to two passes: the blocked axis is cancelled and the rest goes on) and moves the parent.
  - Each `Hit` names the entity or tile, the normal and the time of impact.
  - Triggers in the way are reported, not blocking.
- **Queries:** `space.At(point)`, `space.Overlapping(shape, at)` and `space.Cast(from, to)` (a ray, for line of sight) return what they find.
- **Tile solids:** `space.AddSolid(s collide.Solid)` takes anything with `SolidIn(r geom.Rect) []geom.Rect`. `twod.TileMap` implements it with the rectangles of its solid cells in that area, so the space needs no tiles of its own.
- **Events:** `collide.BodyEntered{Body, Other}` and `collide.BodyLeft{…}` are World events for trigger overlaps, read like `ui.Clicked`.

## Planned examples

Numbered after the doc writer's 19–23.

| # | Example | Shows |
|---|---|---|
| 24 | `26-tilemap` | a tileset from an atlas, a map drawn from an `ASCII` picture, an animated water tile, culling a large map |
| 25 | `27-collision` | the walker of 20-sprite-sheet in a walled room: `collide.NewBody`, `space.Move`, the tile map as a solid, a trigger that opens a door |
| 26 | `28-y-sort` | a top-down scene of trees and pillars pivoted at their feet, from a map's tall tiles, with the hero passing in front of and behind them and blocked by their trunks |

A Tiled example (`29-tiled`) follows once 24–26 settle the API.

## What goes away

Nothing yet. `spatial.Grid` (3D, the ground plane) stays for Emberfall's interest management. `collide`'s hash is its own, in 2D, with shapes.

## Decided

1. **`collide` is its own package,** separate from `spatial`, which it may use inside.
2. **Tiled JSON only:** `.tmj` and `.tsj`, no TMX.
3. **No height offset** for isometric tall tiles for now.
4. **Slopes and one-way platforms come later,** one-way platforms first, when a side-scroller example needs them.
5. **Emberfall keeps its own ground checks;** collide is 2D for games built on the engine.

## Pathfinding

Click-to-move and chasing enemies need paths around walls. `ember/nav` finds them over a grid's solid cells. It is a small package of its own, pure Go, so a server finds paths without twod.

- **Where:** nav, not twod. twod's TileMap is the usual grid: it gains `Solid(x, y)` and a `Version()` that changes with every `Set`. Any type with `Columns`, `Rows`, `Solid`, `Cell` and `Center` is a grid too, so a server's own level works without a TileMap.
- **Finding:** `m := nav.New(grid)` keeps scratch memory sized to the grid. `m.Find(from, to)` returns waypoints in the grid's space, ending at `to`.
  - The search is A* with the octile distance.
  - Diagonal moves are allowed only when both cells beside them are free, so a path never cuts a wall's corner.
  - `nav.NoDiagonals()` turns diagonals off.
  - If `to` is solid or out of reach, the path ends at the reachable cell nearest it, and `Find` reports false. A click on a wall walks to the wall.
- **Smoothing:** the cell path is string-pulled. Each waypoint is kept only when the straight line from the last kept one would cross a solid cell, a corner touched diagonally included. `m.Clear(a, b)` is that line test, for games to use too.
- **Re-planning:** `m.Route(from, to)` is a path to walk.
  - `route.Toward(at, reach)` gives the direction to the next waypoint and moves on to the following one within reach.
  - It plans again when the grid's `Version` changes (a door opened, a wall broke) or the way to the next waypoint is no longer clear.
- **Cost:** a search touches only the cells it opens, with no clearing pass, thanks to generation stamps, and it allocates almost nothing. The worst case, corner to corner across a 128×128 maze of random walls that opens most of the grid, takes about a millisecond on an M1; paths across rooms are much cheaper. So dozens of chasers re-planning a couple of times a second cost about a millisecond a frame. A flow field for hordes can come later if a game needs one.
- **Not yet:** isometric maps, and agent clearance wider than a cell's corner rule. Collision slides a body along walls anyway.
- **Hordes:** `nav.NewFlowField(grid)` serves many walkers at once: one Dijkstra from the target fills every cell's way, and each walker reads `Toward(at)` in constant time. `Around(m.AreaAround, 128, 128)` makes it follow the target over an InfiniteMap. See decision 0024.
- **Alternatives:**
  - **Inside twod:** simpler for the guide, but a server would need twod.
  - **Jump point search:** faster on open maps, but it needs uniform costs and more code. A* with stamps is fast enough at these sizes.
