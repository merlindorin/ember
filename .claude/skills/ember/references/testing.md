# Testing and captures

Tests need no GPU and no real time. A test builds the same window and systems as `main`, sets input, runs frames, and checks the rules and the screen. Captures run the real program hidden and save what it drew.

## Test the rules first

The rules are plain Go, so their tests are plain Go: no window, no World.

## Run frames with simulated input

| Need | Call |
|---|---|
| a World without opening a window | `w := ember.New().AddEntities(win).AddSystems(sys)` (never `MustRun` in a test) |
| advance time | `w.RunFrames(n)`: one frame each, a fixed DT; input set before shows from the next frame |
| hold or release a key | `win.Input().SetKey(input.KeyLeft, true)` / `false` |
| type text | `in.AddText(r)` per rune |
| move and click the mouse | `in.SetMouse(geom.Vec2)` (window points), `in.SetMouseButton(input.MouseLeft, true)` |
| wheel, drop, gamepad, touch | `AddWheel`, `AddDrop`, `SetGamepadButton(id, b, down)`, `SetGamepadStick`, `AddTouch` |

A press is down for one frame, then up: `SetKey(k, true)`, `RunFrames(1)`, `SetKey(k, false)`, `RunFrames(1)`. `Pressed` sees it once.

## Check the screen

| Need | Call |
|---|---|
| find an element | `doc.ByID("board")` (`ui.ID("board")` when built) |
| where it is laid out | `r, ok := doc.BoxOf(node)`: a `geom.Rect` in window points, after a frame |
| click it | `in.SetMouse(r.Center())`, then the press above with `input.MouseLeft` |
| its text | `node.Text()` |
| nothing overflows or misaligns | `align.Check(doc)` (package `ui/align`) returns `[]align.Fault`; empty when the screen is clean |
| the CSS parsed | `doc.Warnings()` returns `[]error`; empty when every rule was understood |

Every screen's test runs `align.Check` and `doc.Warnings`.

## A test of the skeleton

This is the test of SKILL.md's skeleton, run as is: the rules alone, then the arrows through the World and the panel's count.

```go
// main_test.go, beside the skeleton
package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/align"
	"github.com/merlindorin/ember/ui/kit"
)

func TestTheRulesStayOnTheBoard(t *testing.T) {
	g := &Game{}
	g.Move(-1, 0)
	g.Move(1, 0)
	if *g != (Game{X: 1, Moves: 1}) {
		t.Fatalf("a move off the board is no move: %+v", *g)
	}
}

func TestTheArrowsMoveAndThePanelCounts(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 360))
	doc := ui.NewDocument(kit.Theme(), ui.CSS(page))
	play := &Play{game: &Game{}, in: win.Input(), doc: doc, moves: kit.Label("0")}
	play.board = ui.Canvas(play.draw, ui.ID("board"))
	doc.Body().Append(play.board, kit.Panel(kit.Column(play.moves)))
	win.Add(doc)
	w := ember.New().AddEntities(win).AddSystems(play)
	w.RunFrames(1)

	press := func(k input.Key) {
		win.Input().SetKey(k, true)
		w.RunFrames(1)
		win.Input().SetKey(k, false)
		w.RunFrames(1)
	}
	press(input.KeyRight)
	press(input.KeyDown)
	if play.game.X != 1 || play.game.Y != 1 || play.moves.Text() != "2" {
		t.Fatalf("game %+v, panel %q", *play.game, play.moves.Text())
	}
	if r, ok := doc.BoxOf(doc.ByID("board")); !ok || r.W != 256 {
		t.Fatalf("the board is laid out 256 wide: %v", r)
	}
	if faults := align.Check(doc); len(faults) > 0 {
		t.Fatalf("layout faults: %v", faults)
	}
	if warns := doc.Warnings(); len(warns) > 0 {
		t.Fatalf("CSS warnings: %v", warns)
	}
}
```

## Drive it by automation

ui elements record themselves with paths of their names (id, else `data-name`, aria-label, text). Tests act on them without the mouse:

```go
auto := automation.New(automation.Recording())
w := ember.New(ember.Automation(auto)).AddEntities(win)
auto.Queue("settings/name", automation.ActType, "Rowan")   // also ActSet, ActPress
w.RunFrames(3)
_, elements := auto.Snapshot("", "settings")              // []automation.Element: ID, Label, Value, Role
```

`EMBER_AUTOMATION=localhost:7777 go run ./path` serves the same over Connect for scripts (`grpcurl -plaintext localhost:7777 list`). See `examples/18-testing-ui`.

## Capture a screenshot or a GIF

Run the real program hidden; it saves at the given frame and exits.

```sh
EMBER_HIDDEN=1 EMBER_CAPTURE=shot.png EMBER_CAPTURE_FRAME=60 go run ./path
EMBER_HIDDEN=1 EMBER_CAPTURE=walk.gif EMBER_CAPTURE_FRAMES=90 go run ./path   # every 2nd step, 30 fps
```

| Script | Form |
|---|---|
| keys held | `EMBER_CAPTURE_KEYS="enter:5-7;right:10-40"` (key:from-to steps) |
| pointer and buttons | `EMBER_CAPTURE_MOUSE="320,192@0;140,120@40;left:30-40"` (points@step, straight lines between) |
| typed text | `EMBER_CAPTURE_TEXT="Heron@150"` (a character every four steps) |

Captures are at the screen's scale (2× on a Retina Mac): a 640×360 window saves 1280×720. Look at every capture before reporting it; extra windows save as `name-2.png`…

## Before saying it's done

- `go vet ./...` and `go test ./...` from `ember`, clean.
- A capture looked at.
- For an engine change, `task test:all` (vet, tests, GPU goldens, Chrome) when the change touches drawing or ui.
