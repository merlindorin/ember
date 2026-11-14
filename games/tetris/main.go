// Tetris: steer the falling pieces, turn them, and fill rows to clear them.
// The pieces fall faster every ten lines.
//
//	go run ./games/tetris
package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ease"
	"github.com/lantervale/ember/games/internal/look"
	"github.com/lantervale/ember/games/internal/save"
	"github.com/lantervale/ember/games/internal/sound"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
)

const cell = 18

// Held arrows move once, then again every repeat after delay.
const delay, repeat = 0.17, 0.05

// The game-over question waits a moment, so the Space that ended the game
// doesn't answer it too.
const askAfter = 0.6

// The board's size: the cells with a 4-point margin.
const boardW, boardH = W*cell + 8, H*cell + 8

// colorOf is a piece's colour, on the board and in the preview: the UI
// pack's palette (blue, gold, red, green, brown, slate, cream).
func colorOf(kind int) color.Color {
	switch kind {
	case I:
		return color.Hex("#5ca2e0")
	case O:
		return color.Hex("#ffb600")
	case T:
		return color.Hex("#cf5e53")
	case S:
		return color.Hex("#72b84e")
	case Z:
		return color.Hex("#c58747")
	case J:
		return color.Hex("#94afc6")
	default:
		return color.Hex("#fff1d2")
	}
}

// A clear flashes its rows and sends sparks for clearTime, the game waiting.
const clearTime = 0.35

// Play reads the keys, lets time pass, sounds what happened and keeps the
// HUD in step.
type Play struct {
	controls            *input.Actions[action]
	game                *Game
	in                  *input.State
	hud                 *ui.Document
	next, over          *ui.Node
	score, lines, level *ui.Node
	seed                func() uint64
	held, ended         float32
	clearing            float32 // since the last clear
	well                *Well
	sparks              *twod.Particles
	data                *save.Data
	sounds              *Sounds
	player              *sound.Player
	menu                *look.Menu
	fit                 *look.Fit
}

// An action is what the player asks with a key or a pad button.
type action int

const (
	moveLeft action = iota
	moveRight
	turn
	faster
	drop
	again
	pause
)

func newControls() *input.Actions[action] {
	return input.NewActions(
		input.Bind(moveLeft, input.KeyLeft, input.GamepadLeft, input.GamepadLeftStickLeft),
		input.Bind(moveRight, input.KeyRight, input.GamepadRight, input.GamepadLeftStickRight),
		input.Bind(turn, input.KeyUp, input.GamepadSouth),
		input.Bind(faster, input.KeyDown, input.GamepadDown, input.GamepadLeftStickDown),
		input.Bind(drop, input.KeySpace, input.GamepadUp),
		input.Bind(again, look.AgainButtons()...),
		input.Bind(pause, look.MenuButtons()...))
}

// Well is the play area, a twod entity drawing the board every frame.
type Well struct {
	twod.Item
	play   *Play
	blocks *twod.Texture
	skin   *look.Skin
}

func newPlay(win *platform.Window, seed func() uint64, data *save.Data) *Play {
	in := win.Input()
	p := &Play{game: New(seed()), in: in, controls: newControls(), hud: look.HUD(), seed: seed,
		clearing: clearTime, data: data,
		sounds: newSounds(), player: sound.NewPlayer(music(), data.Music, data.Effects, !data.MusicOff)}
	p.next = ui.Canvas(p.drawNext, ui.ID("next"), ui.Style("width: 64px; height: 32px"))
	scoreCard, score := look.Stat("Score", "score")
	linesCard, lines := look.Stat("Lines", "lines")
	levelCard, level := look.Stat("Level", "level")
	p.score, p.lines, p.level = score, lines, level
	p.over = look.GameOver(p.restart)
	side := look.Side("Tetris",
		look.Stats(scoreCard, look.Card("Next", p.next)),
		look.Stats(linesCard, levelCard),
		look.NewGame(p.hud, p.restart),
		look.Controls(
			look.Control{Keys: []string{"left", "right"}, Pad: []string{"left-right"}, Does: "move"},
			look.Control{Keys: []string{"up"}, Pad: []string{"a"}, Does: "turn"},
			look.Control{Keys: []string{"down"}, Pad: []string{"down"}, Does: "faster"},
			look.Control{Keys: []string{"space"}, Pad: []string{"up"}, Does: "drop"},
			look.AgainControl(), look.MenuControl()))
	p.hud.Body().Append(side, p.over)
	p.menu = look.NewMenu(p.hud, look.Hooks{NewGame: p.restart, Data: func() *save.Data { return p.data },
		SoundChanged: func() {
			p.player.Set(p.data.Music, p.data.Effects, !p.data.MusicOff)
			p.store()
		},
		Fullscreen: func() { p.fit.Toggle() }})
	p.fit = look.NewFit(win, p.hud, func(on bool) {
		p.data.Fullscreen = on
		p.store()
	})
	if data.Fullscreen {
		win.SetFullscreen(true)
	}
	well := &Well{Item: twod.Base(), play: p, blocks: loadBlocks(), skin: look.LoadSkin()}
	p.well = twod.Build(well, "board", look.Place(boardW, boardH))
	p.sparks = twod.NewParticles(nil, twod.Rate(0), twod.Velocity(40, 180), twod.Direction(-geom.Pi/2, geom.Pi),
		twod.Gravity(geom.V2(0, 320)), twod.Lifetime(0.4, 0.9), twod.SizeOverLife(5, 0.5),
		twod.ColorOverLife(color.Hex("#fff3d6"), color.Hex(look.Accent).Alpha(0)), twod.Blend(twod.Add),
		twod.MaxParticles(800))
	p.show()
	return p
}

func (p *Play) Update(f world.Frame) {
	p.clearing += f.DT
	if p.menu.Open() {
		return
	}
	if p.controls.Pressed(p.in, pause) {
		p.menu.Show()
		return
	}
	if p.controls.Pressed(p.in, again) {
		p.restart()
		return
	}
	if p.game.Over {
		p.ask(f.DT)
		return
	}
	if p.clearing < clearTime {
		return
	}
	before := *p.game
	switch {
	case p.controls.Pressed(p.in, turn):
		if p.game.Rotate() {
			p.player.Play(p.sounds.turn)
		}
	case p.controls.Pressed(p.in, drop):
		p.game.Drop()
		p.player.Play(p.sounds.drop)
	}
	p.steer(f.DT)
	if p.game.Piece.X != before.Piece.X && p.game.Next == before.Next {
		p.player.Play(p.sounds.move)
	}
	p.game.Step(f.DT, p.controls.Down(p.in, faster))
	p.heard(before)
	if p.game.Next != before.Next || p.game.Score != before.Score || p.game.Lines != before.Lines {
		p.show()
	}
}

// heard sounds a lock and starts a clear's flash and sparks.
func (p *Play) heard(before Game) {
	switch {
	case p.game.Clears != before.Clears:
		rows := p.game.Cleared.Rows
		p.player.Play(p.sounds.clear[min(len(rows), 4)])
		p.clearing = 0
		origin := p.well.Position()
		for _, y := range rows {
			for x := range W {
				p.sparks.SetPosition(origin.Add(geom.V2(float32(4+x*cell+cell/2), float32(4+y*cell+cell/2))))
				p.sparks.Snap()
				p.sparks.Emit(6)
			}
		}
	case p.game.Over:
		p.player.Play(p.sounds.over)
	case p.game.Next != before.Next && p.game.Score-before.Score < 2:
		p.player.Play(p.sounds.lock)
	}
}

// ask records the score and shows the game-over question once askAfter has
// passed.
func (p *Play) ask(dt float32) {
	if p.ended < askAfter && p.ended+dt >= askAfter {
		message := fmt.Sprintf("Score %d, %d lines.", p.game.Score, p.game.Lines)
		if p.data.Add(p.game.Score, time.Now()) == 1 {
			message = fmt.Sprintf("New best: %d!", p.game.Score)
		}
		p.store()
		look.SetGameOver(p.over, message)
		p.hud.ShowModal(p.over)
	}
	p.ended += dt
}

func (p *Play) store() {
	if err := save.Store("tetris", *p.data); err != nil {
		log.Print(err)
	}
}

// steer moves the piece on an arrow's press, then repeats while it's held.
func (p *Play) steer(dt float32) {
	var dx int
	switch {
	case p.controls.Down(p.in, moveLeft):
		dx = -1
	case p.controls.Down(p.in, moveRight):
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
	p.game, p.ended, p.clearing = New(p.seed()), 0, clearTime
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
	w.skin.Board(c, geom.Rect{W: boardW, H: boardH})
	if p.clearing < clearTime {
		w.drawClear(c, p.clearing/clearTime)
		return
	}
	drawBoard(c, w.blocks, &p.game.Board)
	if p.game.Over {
		return
	}
	for _, at := range p.game.Ghost().Cells() {
		square(c, w.blocks, at[0], at[1], 1, colorOf(p.game.Piece.Kind).Alpha(0.25))
	}
	for _, at := range p.game.Piece.Cells() {
		square(c, w.blocks, at[0], at[1], 1, colorOf(p.game.Piece.Kind))
	}
}

// drawClear draws the board as it was, its full rows flashing white and
// shrinking away, k from 0 to 1.
func (w *Well) drawClear(c *twod.Canvas, k float32) {
	cleared := &w.play.game.Cleared
	drawBoard(c, w.blocks, &cleared.Board)
	flash := color.Hex("#ffffff").Alpha(1 - ease.OutQuad(k))
	for _, y := range cleared.Rows {
		for x := range W {
			empty(c, x, y)
			square(c, w.blocks, x, y, 1-ease.InQuad(k), colorOf(cleared.Board[y][x]).Lerp(color.Hex("#ffffff"), 1-k))
			square(c, w.blocks, x, y, 1, flash.Alpha(0.5*(1-k)))
		}
	}
}

func drawBoard(c *twod.Canvas, blocks *twod.Texture, board *[H][W]int) {
	for y, row := range board {
		for x, kind := range row {
			if kind == 0 {
				empty(c, x, y)
				continue
			}
			square(c, blocks, x, y, 1, colorOf(kind))
		}
	}
}

func (p *Play) drawNext(c *twod.Canvas) {
	for _, at := range spawn(p.game.Next).cells {
		r := geom.Rect{X: float32(at[0] * 16), Y: float32(at[1] * 16), W: 16, H: 16}
		c.Image(r, p.well.blocks, blockUV, colorOf(p.game.Next))
	}
}

// square draws a board cell, Kenney's block tinted col, at a scale about its
// centre; the board has a 4-point margin.
func square(c *twod.Canvas, blocks *twod.Texture, x, y int, scale float32, col color.Color) {
	if y < 0 || scale <= 0 {
		return
	}
	size := cell * scale
	at := geom.V2(float32(4+x*cell)+(cell-size)/2, float32(4+y*cell)+(cell-size)/2)
	c.Image(geom.Rect{X: at.X, Y: at.Y, W: size, H: size}, blocks, blockUV, col)
}

// empty draws an empty cell, a faint square sunk into the board.
func empty(c *twod.Canvas, x, y int) {
	if y < 0 {
		return
	}
	r := geom.Rect{X: float32(4+x*cell) + 1, Y: float32(4+y*cell) + 1, W: cell - 2, H: cell - 2}
	c.Rect(r, color.Hex(look.Surface).Alpha(0.45))
}

func main() {
	fixed := flag.Uint64("seed", 0, "replay the game this seed deals (0: a new game each time)")
	flag.Parse()
	seed := func() uint64 {
		if *fixed != 0 {
			return *fixed
		}
		return uint64(time.Now().UnixNano())
	}
	win := look.Window("tetris")
	data, err := save.Load("tetris")
	if err != nil {
		log.Print(err)
	}
	p := newPlay(win, seed, &data)
	win.Add(look.Camera(), p.well, p.sparks, p.hud)
	ember.New().AddEntities(win, p.player.Mixer).AddSystems(p, p.fit).MustRun()
}
