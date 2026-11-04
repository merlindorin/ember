---
name: ember
description: Build games, tools and examples with the ember Go engine (github.com/merlindorin/ember) - windows, 2D drawing, ui screens of HTML elements and CSS, input, sound, headless tests and captures. Use whenever code imports github.com/merlindorin/ember or a task asks for a game, example or screen made with ember.
---

# Building with ember

ember is a Go game engine on its own ECS. Read this page first, then the reference for the task; copy the shape of the closest example rather than guessing an API.

| Task | Read |
|---|---|
| entities, systems, phases, events, time, tweens | [references/world.md](references/world.md) |
| windows, sprites, shapes, text, canvas drawing, tile maps, lights | [references/twod.md](references/twod.md) |
| a screen: panels, labels, buttons, forms, CSS, custom drawn elements | [references/ui.md](references/ui.md) |
| keys, mouse, gamepads, action maps, sound | [references/input-audio.md](references/input-audio.md) |
| tests, simulated input, automation, screenshots | [references/testing.md](references/testing.md) |

## Rules

- **The play area is always 2D, never ui.** A board, a level, anything the game draws is twod entities on layers (a custom entity's `Draw`, sprites, tile maps), never ui elements or a `ui.Canvas`, so particles, effects, blend modes and lights can go on it. The ui is for the HUD around it: fine for prototypes, decided case by case for a custom HUD.
- **Rules are plain Go.** Keep a game's rules (a board, a score, a piece) in types that import no ember package; test them as plain Go. Systems only read input, call the rules and show the result.
- **A system** is a struct with `Update(world.Frame)` (or another phase method); add it with `AddSystems`. It reads its window's input through `win.Input()`.
- **Options, not structs.** Constructors take options (`platform.Size(640, 360)`, `twod.At(x, y)`); every option has a getter of the same name, setters exist where change at run time is meant.
- **`Add` returns the parent**, so windows, twod entities and ui containers chain: `win.Add(doc)`, `ember.New().AddEntities(win)`.
- **ui reacts by callbacks** on the element (`ui.OnClick(fn)`), not by reading `ui.Clicked` events in a loop.
- **`kit.Theme()` goes first** in `ui.NewDocument`; the game's `ui.CSS` or `Stylesheet` after it.
- **A plain div takes `OnClick` only with `ui.TabIndex(0)`**; controls, canvases, labels and summaries take it as they are.
- **A `ui.Canvas` (a HUD widget only) draws again only after `doc.Invalidate(node)`**; a twod entity's `Draw` runs every frame.
- **Input is read in `PreUpdate` or `Update`**; `Pressed` is true for one frame, `Down` while held.
- **Example windows are fixed size** (`platform.FixedSize()`) unless their layout follows the window.
- **No raylib, no OS dialogs, no structs of config.** Never import `gpu`, `internal/...` or `platform/desktop` from game code.
- **Comments only where the code can't say it**; the reader is a senior developer.

## Workflow

1. Find the closest example in `examples/` (numbered by guide page; `examples/NN-name/main.go` has a header saying what it shows and how to run it) and its guide page in `docs/`.
2. Write the rules as plain Go with a `_test.go` beside them.
3. Build the screen: a window, twod entities for the play area, a `ui.Document` for the HUD around it.
4. Add one system that reads input and drives the rules.
5. Test headless: input simulated, `RunFrames`, assertions on the rules and on the ui (`doc.ByID`, `BoxOf`, `align.Check`).
6. Capture a screenshot and look at it before saying it works:
   `EMBER_HIDDEN=1 EMBER_CAPTURE=shot.png EMBER_CAPTURE_FRAME=60 EMBER_CAPTURE_KEYS="right:10-11" go run ./path`
7. `go vet ./...` and `go test ./...` from `ember` must stay clean.

## A game skeleton

A board drawn by a twod entity, a HUD panel of kit labels beside it, one system reading the arrows. It builds and runs as is; grow it rather than starting from nothing.

```go
package main

import (
	"fmt"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/merlindorin/ember/world"
)

const cell, cols, rows = 32, 8, 8

// The HUD sits right; the board is drawn left of it, in the 2D world.
const page = `html, body { height: 100%; }
body { display: flex; align-items: center; justify-content: flex-end; padding-right: 48px; }`

// Game is the rules, plain Go: no ember type, so tests need no window.
type Game struct {
	X, Y, Moves int
}

func (g *Game) Move(dx, dy int) {
	x, y := g.X+dx, g.Y+dy
	if x < 0 || y < 0 || x >= cols || y >= rows {
		return
	}
	g.X, g.Y, g.Moves = x, y, g.Moves+1
}

// Board is the play area: a twod entity that draws the game every frame,
// so layers, effects, particles and lights can go on it.
type Board struct {
	twod.Item
	game *Game
}

func (b *Board) Draw(c *twod.Canvas) {
	c.RoundRect(geom.Rect{W: cols * cell, H: rows * cell}, 8, color.Hex("#2b2d42"))
	r := geom.Rect{X: float32(b.game.X*cell) + 3, Y: float32(b.game.Y*cell) + 3, W: cell - 6, H: cell - 6}
	c.RoundRect(r, 6, color.Hex("#ef8354"))
}

// Play reads the keys each frame, changes the game and updates the HUD.
type Play struct {
	game  *Game
	in    *input.State
	moves *ui.Node
}

func (p *Play) Update(world.Frame) {
	switch {
	case p.in.Pressed(input.KeyLeft):
		p.game.Move(-1, 0)
	case p.in.Pressed(input.KeyRight):
		p.game.Move(1, 0)
	case p.in.Pressed(input.KeyUp):
		p.game.Move(0, -1)
	case p.in.Pressed(input.KeyDown):
		p.game.Move(0, 1)
	}
	p.moves.SetText(fmt.Sprint(p.game.Moves))
}

func main() {
	win := platform.NewWindow(platform.Title("skeleton"), platform.Size(640, 360), platform.FixedSize())
	game := &Game{}
	board := twod.Build(&Board{Item: twod.Base(), game: game}, "board", twod.At(64, 52))
	hud := ui.NewDocument(kit.Theme(), ui.CSS(page))
	play := &Play{game: game, in: win.Input(), moves: kit.Label("0", ui.ID("moves"))}
	hud.Body().Append(kit.Panel(kit.Padding(16), kit.Column(kit.Gap(8),
		kit.Label("Moves"), play.moves, kit.Label("Arrows move"))))
	win.Add(board, hud)
	ember.New().AddEntities(win).AddSystems(play).MustRun()
}
```

## Commands

```sh
task wgpu                                  # once: the WebGPU library the renderer loads
go run ./examples/19-sprites               # any example
go test ./...                              # GPU-free tests
task docs                                  # the guide on http://localhost:1313/ember/
task web PKG=./examples/19-sprites         # the same program in the browser (dist/web)
```

The engine's design rules and why they hold are in `AGENT.md` and `docs/99-decisions/`.
