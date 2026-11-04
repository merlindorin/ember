// 2048: slide the tiles with the arrows; two equal tiles merge into their
// sum. Reach 2048, and keep going until the board is full.
//
//	go run ./games/2048
package main

import (
	"fmt"
	"math/bits"
	"strconv"
	"time"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/games/internal/look"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/merlindorin/ember/world"
)

const tile, gap = 76, 10

// side is the board's width and height.
const side = 4*tile + 5*gap

// Game reads the arrows, plays them on the board and keeps the HUD in step.
type Game struct {
	board       *Board
	in          *input.State
	hud         *ui.Document
	over        *ui.Node
	score, best *ui.Node
	seed        func() uint64
}

// Tiles is the play area, a twod entity drawing the board every frame.
type Tiles struct {
	twod.Item
	game *Game
}

func newGame(in *input.State, seed func() uint64) (*Game, *Tiles) {
	g := &Game{board: New(seed()), in: in, hud: look.HUD(), seed: seed}
	scoreRow, score := look.Stat("Score", "score")
	bestRow, best := look.Stat("Best tile", "best")
	g.score, g.best = score, best
	g.over = look.GameOver(g.restart)
	keys := look.Keys("Arrows: slide", "N: new game")
	g.hud.Body().Append(look.Side("2048", scoreRow, bestRow, kit.Separator(), keys), g.over)
	g.show()
	return g, twod.Build(&Tiles{Item: twod.Base(), game: g}, "board", look.Place(side, side))
}

func (g *Game) Update(world.Frame) {
	if g.in.Pressed(input.KeyN) {
		g.restart()
		return
	}
	for _, k := range []struct {
		key input.Key
		dir Dir
	}{{input.KeyLeft, Left}, {input.KeyRight, Right}, {input.KeyUp, Up}, {input.KeyDown, Down}} {
		if g.in.Pressed(k.key) && g.board.Move(k.dir) {
			g.show()
			if g.board.Over() {
				kit.SetAlert(g.over, "Game over", fmt.Sprintf("Score %d, best tile %d.", g.board.Score, g.board.Best()))
				g.hud.ShowModal(g.over)
			}
			return
		}
	}
}

func (g *Game) restart() {
	g.board = New(g.seed())
	g.show()
}

func (g *Game) show() {
	g.score.SetText(strconv.Itoa(g.board.Score))
	g.best.SetText(strconv.Itoa(g.board.Best()))
}

func (t *Tiles) Draw(c *twod.Canvas) {
	c.RoundRect(geom.Rect{W: side, H: side}, 10, color.Hex(look.Board))
	for r, row := range t.game.board.Cells {
		for col, v := range row {
			cell := geom.Rect{X: float32(gap + col*(tile+gap)), Y: float32(gap + r*(tile+gap)), W: tile, H: tile}
			if v == 0 {
				c.RoundRect(cell, 8, color.Hex(look.Empty))
				continue
			}
			c.RoundRect(cell, 8, tileColor(v))
			label := strconv.Itoa(v)
			opts := []twod.TextOption{twod.Typeface(twod.Bold()), twod.FontSize(fontSize(v)), twod.Color(textColor(v))}
			size := c.Measure(label, opts...)
			c.Text(geom.V2(cell.X+(tile-size.X)/2, cell.Y+(tile-size.Y)/2), label, opts...)
		}
	}
}

func tileColor(v int) color.Color {
	switch v {
	case 2:
		return color.Hex("#eee4da")
	case 4:
		return color.Hex("#ede0c8")
	case 8:
		return color.Hex("#f2b179")
	case 16:
		return color.Hex("#f59563")
	case 32:
		return color.Hex("#f67c5f")
	case 64:
		return color.Hex("#f65e3b")
	case 128, 256, 512, 1024, 2048:
		return color.Hex("#edcf72").Lerp(color.Hex("#edc22e"), float32(bits.Len(uint(v))-8)/4)
	default:
		return color.Hex("#3c3a32")
	}
}

func textColor(v int) color.Color {
	if v <= 4 {
		return color.Hex("#776e65")
	}
	return color.Hex(look.Ink)
}

func fontSize(v int) float32 {
	switch {
	case v < 100:
		return 34
	case v < 1000:
		return 28
	default:
		return 22
	}
}

func main() {
	win := look.Window("2048")
	g, tiles := newGame(win.Input(), func() uint64 { return uint64(time.Now().UnixNano()) })
	win.Add(tiles, g.hud)
	ember.New().AddEntities(win).AddSystems(g).MustRun()
}
