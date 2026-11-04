// Tetris: steer the falling pieces, turn them, and fill rows to clear them.
// The pieces fall faster every ten lines.
//
//	go run ./games/tetris
package main

import (
	"fmt"
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

const cell = 19

// Held arrows move once, then again every repeat after delay.
const delay, repeat = 0.17, 0.05

// The game-over question waits a moment, so the Space that ended the game
// doesn't answer it too.
const askAfter = 0.6

// The board's size: the cells with a 4-point margin.
const boardW, boardH = W*cell + 6, H*cell + 6

// colorOf is a piece's colour, on the board and in the preview.
func colorOf(kind int) color.Color {
	switch kind {
	case I:
		return color.Hex("#56cfe1")
	case O:
		return color.Hex("#f2cc8f")
	case T:
		return color.Hex("#c77dff")
	case S:
		return color.Hex("#80ed99")
	case Z:
		return color.Hex("#ef476f")
	case J:
		return color.Hex("#4895ef")
	default:
		return color.Hex("#ef8354")
	}
}

// Play reads the keys, lets time pass and keeps the HUD in step.
type Play struct {
	game                *Game
	in                  *input.State
	hud                 *ui.Document
	next, over          *ui.Node
	score, lines, level *ui.Node
	seed                func() uint64
	held, ended         float32
}

// Well is the play area, a twod entity drawing the board every frame.
type Well struct {
	twod.Item
	play *Play
}

func newPlay(in *input.State, seed func() uint64) (*Play, *Well) {
	p := &Play{game: New(seed()), in: in, hud: look.HUD(), seed: seed}
	p.next = ui.Canvas(p.drawNext, ui.ID("next"), ui.Style("width: 80px; height: 40px"))
	scoreRow, score := look.Stat("Score", "score")
	linesRow, lines := look.Stat("Lines", "lines")
	levelRow, level := look.Stat("Level", "level")
	p.score, p.lines, p.level = score, lines, level
	p.over = look.GameOver(p.restart)
	side := look.Side("Tetris", scoreRow, linesRow, levelRow, kit.Separator(), kit.Label("Next"), p.next,
		kit.Separator(), look.Keys("← →: move", "↑: turn", "↓: faster", "Space: drop", "N: new game"))
	p.hud.Body().Append(side, p.over)
	p.show()
	return p, twod.Build(&Well{Item: twod.Base(), play: p}, "board", look.Place(boardW, boardH))
}

func (p *Play) Update(f world.Frame) {
	if p.in.Pressed(input.KeyN) {
		p.restart()
		return
	}
	if p.game.Over {
		p.ask(f.DT)
		return
	}
	before := *p.game
	switch {
	case p.in.Pressed(input.KeyUp):
		p.game.Rotate()
	case p.in.Pressed(input.KeySpace):
		p.game.Drop()
	}
	p.steer(f.DT)
	p.game.Step(f.DT, p.in.Down(input.KeyDown))
	if p.game.Next != before.Next || p.game.Score != before.Score || p.game.Lines != before.Lines {
		p.show()
	}
}

// ask shows the game-over question once askAfter has passed.
func (p *Play) ask(dt float32) {
	if p.ended < askAfter && p.ended+dt >= askAfter {
		kit.SetAlert(p.over, "Game over", fmt.Sprintf("Score %d, %d lines.", p.game.Score, p.game.Lines))
		p.hud.ShowModal(p.over)
	}
	p.ended += dt
}

// steer moves the piece on an arrow's press, then repeats while it's held.
func (p *Play) steer(dt float32) {
	var dx int
	switch {
	case p.in.Down(input.KeyLeft):
		dx = -1
	case p.in.Down(input.KeyRight):
		dx = 1
	default:
		p.held = 0
		return
	}
	if p.held == 0 {
		p.game.Move(dx)
	}
	p.held += dt
	if p.held >= delay {
		p.game.Move(dx)
		p.held -= repeat
	}
}

func (p *Play) restart() {
	p.game, p.ended = New(p.seed()), 0
	p.show()
}

func (p *Play) show() {
	p.score.SetText(strconv.Itoa(p.game.Score))
	p.lines.SetText(strconv.Itoa(p.game.Lines))
	p.level.SetText(strconv.Itoa(p.game.Level()))
	p.hud.Invalidate(p.next)
}

func (w *Well) Draw(c *twod.Canvas) {
	p := w.play
	c.RoundRect(geom.Rect{W: boardW, H: boardH}, 8, color.Hex(look.Board))
	for y, row := range p.game.Board {
		for x, kind := range row {
			col := color.Hex(look.Empty)
			if kind != 0 {
				col = colorOf(kind)
			}
			square(c, x, y, col)
		}
	}
	if p.game.Over {
		return
	}
	for _, at := range p.game.Ghost().Cells() {
		square(c, at[0], at[1], colorOf(p.game.Piece.Kind).Alpha(0.25))
	}
	for _, at := range p.game.Piece.Cells() {
		square(c, at[0], at[1], colorOf(p.game.Piece.Kind))
	}
}

func (p *Play) drawNext(c *twod.Canvas) {
	for _, at := range spawn(p.game.Next).cells {
		c.RoundRect(geom.Rect{X: float32(at[0] * 20), Y: float32(at[1] * 20), W: 18, H: 18}, 3, colorOf(p.game.Next))
	}
}

// square draws a board cell; the board has a 4-point margin.
func square(c *twod.Canvas, x, y int, col color.Color) {
	if y < 0 {
		return
	}
	c.RoundRect(geom.Rect{X: float32(4 + x*cell), Y: float32(4 + y*cell), W: cell - 2, H: cell - 2}, 3, col)
}

func main() {
	win := look.Window("tetris")
	p, well := newPlay(win.Input(), func() uint64 { return uint64(time.Now().UnixNano()) })
	win.Add(well, p.hud)
	ember.New().AddEntities(win).AddSystems(p).MustRun()
}
