// 2048: slide the tiles with the arrows; two equal tiles merge into their
// sum. Reach 2048, and keep going until the board is full.
//
//	go run ./games/2048
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/audio"
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

const tile, gap = 76, 10

// side is the board's width and height.
const side = 4*tile + 5*gap

// A move draws in two beats: the tiles slide, then merged tiles pop and the
// new one grows in.
const slideTime, popTime = 0.1, 0.15

// An action is what the player asks with a key or a pad button.
type action int

const (
	slideLeft action = iota
	slideRight
	slideUp
	slideDown
	again
	pause
)

func newControls() *input.Actions[action] {
	return input.NewActions(
		input.Bind(slideLeft, input.KeyLeft, input.GamepadLeft, input.GamepadLeftStickLeft),
		input.Bind(slideRight, input.KeyRight, input.GamepadRight, input.GamepadLeftStickRight),
		input.Bind(slideUp, input.KeyUp, input.GamepadUp, input.GamepadLeftStickUp),
		input.Bind(slideDown, input.KeyDown, input.GamepadDown, input.GamepadLeftStickDown),
		input.Bind(again, look.AgainButtons()...),
		input.Bind(pause, look.MenuButtons()...))
}

// Game reads the arrows, plays them on the board and keeps the HUD in step.
type Game struct {
	controls    *input.Actions[action]
	board       *Board
	in          *input.State
	hud         *ui.Document
	over        *ui.Node
	score, best *ui.Node
	seed        func() uint64
	moved       Moved
	since       float32 // since the last move, in seconds
	data        *save.Data
	sounds      *Sounds
	player      *sound.Player
	menu        *look.Menu
	fit         *look.Fit
}

// Tiles is the play area, a twod entity drawing the board every frame.
type Tiles struct {
	twod.Item
	game *Game
	skin *look.Skin
}

func newGame(win *platform.Window, seed func() uint64, data *save.Data) (*Game, *Tiles) {
	in := win.Input()
	g := &Game{board: New(seed()), in: in, controls: newControls(), hud: look.HUD(), seed: seed,
		since: slideTime + popTime, data: data,
		sounds: newSounds(), player: sound.NewPlayer(music(), data.Music, data.Effects, !data.MusicOff)}
	scoreCard, score := look.Stat("Score", "score")
	bestCard, best := look.Stat("Best tile", "best")
	g.score, g.best = score, best
	g.over = look.GameOver(g.restart)
	g.hud.Body().Append(look.Side("2048",
		look.Stats(scoreCard, bestCard),
		look.NewGame(g.hud, g.restart),
		look.Controls(
			look.Control{Keys: []string{"left", "right", "up", "down"}, Pad: []string{"dpad"}, Does: "slide"},
			look.AgainControl(), look.MenuControl())), g.over)
	g.menu = look.NewMenu(g.hud, look.Hooks{NewGame: g.restart, Data: func() *save.Data { return g.data },
		SoundChanged: func() {
			g.player.Set(g.data.Music, g.data.Effects, !g.data.MusicOff)
			g.store()
		},
		Fullscreen: func() { g.fit.Toggle() }})
	g.fit = look.NewFit(win, g.hud, func(on bool) {
		g.data.Fullscreen = on
		g.store()
	})
	if data.Fullscreen {
		win.SetFullscreen(true)
	}
	g.show()
	return g, twod.Build(&Tiles{Item: twod.Base(), game: g, skin: look.LoadSkin()}, "board", look.Place(side, side))
}

func (g *Game) Update(f world.Frame) {
	g.since += f.DT
	if g.menu.Open() {
		return
	}
	if g.controls.Pressed(g.in, pause) {
		g.menu.Show()
		return
	}
	if g.controls.Pressed(g.in, again) {
		g.restart()
		return
	}
	for _, k := range []struct {
		slide action
		dir   Dir
	}{{slideLeft, Left}, {slideRight, Right}, {slideUp, Up}, {slideDown, Down}} {
		if !g.controls.Pressed(g.in, k.slide) {
			continue
		}
		moved, ok := g.board.Move(k.dir)
		if !ok {
			return
		}
		g.moved, g.since = moved, 0
		g.show()
		g.play(moved)
		if g.board.Over() {
			g.end()
		}
		return
	}
}

// play sounds a move: a slide, and the biggest of its merges.
func (g *Game) play(m Moved) {
	g.player.Play(g.sounds.slide, audio.Volume(0.6))
	biggest := 0
	for _, at := range m.Merged {
		biggest = max(biggest, g.board.Cells[at[0]][at[1]])
	}
	if biggest > 0 {
		g.player.Play(g.sounds.mergeOf(biggest))
	}
}

// end records the score and asks for a new game.
func (g *Game) end() {
	g.player.Play(g.sounds.over)
	message := fmt.Sprintf("Score %d, best tile %d.", g.board.Score, g.board.Best())
	if g.data.Add(g.board.Score, time.Now()) == 1 {
		message = fmt.Sprintf("New best: %d!", g.board.Score)
	}
	g.store()
	look.SetGameOver(g.over, message)
	g.hud.ShowModal(g.over)
}

func (g *Game) store() {
	if err := save.Store("2048", *g.data); err != nil {
		log.Print(err)
	}
}

func (g *Game) restart() {
	g.board, g.moved, g.since = New(g.seed()), Moved{}, slideTime+popTime
	g.show()
}

func (g *Game) show() {
	g.score.SetText(strconv.Itoa(g.board.Score))
	g.best.SetText(strconv.Itoa(g.board.Best()))
}

func (t *Tiles) Draw(c *twod.Canvas) {
	g := t.game
	t.skin.Tray(c, geom.Rect{W: side, H: side})
	for r := range 4 {
		for col := range 4 {
			t.skin.Slot(c, cellRect(geom.V2(float32(col), float32(r)), 1))
		}
	}
	if g.since < slideTime {
		k := ease.OutCubic(g.since / slideTime)
		for _, s := range g.moved.Slides {
			from := geom.V2(float32(s.From[1]), float32(s.From[0]))
			t.drawTile(c, from.Lerp(geom.V2(float32(s.To[1]), float32(s.To[0])), k), 1, s.Value)
		}
		return
	}
	pop := min((g.since-slideTime)/popTime, 1)
	for r, row := range g.board.Cells {
		for col, v := range row {
			if v == 0 {
				continue
			}
			scale := float32(1)
			switch at := [2]int{r, col}; {
			case at == g.moved.Added:
				scale = ease.OutBack(pop)
			case merged(g.moved, at):
				scale = 1 + 0.12*float32(math.Sin(math.Pi*float64(pop)))
			}
			t.drawTile(c, geom.V2(float32(col), float32(r)), scale, v)
		}
	}
}

func merged(m Moved, at [2]int) bool {
	for _, c := range m.Merged {
		if c == at {
			return true
		}
	}
	return false
}

// cellRect is the square of the cell at {column, row}, scaled about its centre.
func cellRect(at geom.Vec2, scale float32) geom.Rect {
	size := tile * scale
	x := gap + at.X*(tile+gap) + (tile-size)/2
	y := gap + at.Y*(tile+gap) + (tile-size)/2
	return geom.Rect{X: x, Y: y, W: size, H: size}
}

// drawTile draws a tile at {column, row}: the UI pack's parchment, its
// number inked in its value's colour in the HUD's pixel font; 2048 and up
// on the parchment with metal corners.
func (t *Tiles) drawTile(c *twod.Canvas, at geom.Vec2, scale float32, v int) {
	r := cellRect(at, scale)
	if v >= 2048 {
		t.skin.Plate(c, r)
	} else {
		t.skin.Piece(c, r)
	}
	label := strconv.Itoa(v)
	opts := []twod.TextOption{twod.Typeface(t.skin.Face), twod.FontSize(fontSize(v) * scale), twod.Color(ink(v)),
		twod.Pixelated()}
	size := c.Measure(label, opts...)
	c.Text(geom.V2(r.X+(r.W-size.X)/2, r.Y+(r.H-size.Y)/2), label, opts...)
}

// ink is a value's colour, warm to cold as it grows.
func ink(v int) color.Color {
	switch v {
	case 2:
		return color.Hex("#7d5a36")
	case 4:
		return color.Hex("#6d4b27")
	case 8:
		return color.Hex("#c86a1e")
	case 16:
		return color.Hex("#c8562a")
	case 32:
		return color.Hex("#c8393f")
	case 64:
		return color.Hex("#9c1f33")
	case 128:
		return color.Hex("#4f9a2f")
	case 256:
		return color.Hex("#2f7a26")
	case 512:
		return color.Hex("#3a7fd0")
	case 1024:
		return color.Hex("#2a4fa8")
	default:
		return color.Hex("#7b3cc0")
	}
}

// fontSize is a value's text size, a multiple of 8 so Silkscreen's pixels
// stay whole.
func fontSize(v int) float32 {
	switch {
	case v < 100:
		return 40
	case v < 1000:
		return 24
	default:
		return 16
	}
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
	win := look.Window("2048")
	data, err := save.Load("2048")
	if err != nil {
		log.Print(err)
	}
	g, tiles := newGame(win, seed, &data)
	win.Add(look.Camera(), tiles, g.hud)
	ember.New().AddEntities(win, g.player.Mixer).AddSystems(g, g.fit).MustRun()
}
