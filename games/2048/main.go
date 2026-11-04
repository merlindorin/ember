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
	"math/bits"
	"strconv"
	"time"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/audio"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/ease"
	"github.com/merlindorin/ember/games/internal/look"
	"github.com/merlindorin/ember/games/internal/save"
	"github.com/merlindorin/ember/games/internal/sound"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/world"
)

const tile, gap = 76, 10

// side is the board's width and height.
const side = 4*tile + 5*gap

// A move draws in two beats: the tiles slide, then merged tiles pop and the
// new one grows in.
const slideTime, popTime = 0.1, 0.15

// Game reads the arrows, plays them on the board and keeps the HUD in step.
type Game struct {
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
}

func newGame(win *platform.Window, seed func() uint64, data *save.Data) (*Game, *Tiles) {
	in := win.Input()
	g := &Game{board: New(seed()), in: in, hud: look.HUD(), seed: seed, since: slideTime + popTime, data: data,
		sounds: newSounds(), player: sound.NewPlayer(music(), data.Music, data.Effects, !data.MusicOff)}
	scoreCard, score := look.Stat("Score", "score")
	bestCard, best := look.Stat("Best tile", "best")
	g.score, g.best = score, best
	g.over = look.GameOver(g.restart)
	g.hud.Body().Append(look.Side("2048",
		look.Stats(scoreCard, bestCard),
		look.NewGame(g.hud, g.restart),
		look.Keys("Arrows", "slide", "N", "new game", "F", "full screen", "Esc", "menu")), g.over)
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
	return g, twod.Build(&Tiles{Item: twod.Base(), game: g}, "board", look.Place(side, side))
}

func (g *Game) Update(f world.Frame) {
	g.since += f.DT
	if g.menu.Open() {
		return
	}
	if g.in.Pressed(input.KeyN) {
		g.restart()
		return
	}
	for _, k := range []struct {
		key input.Key
		dir Dir
	}{{input.KeyLeft, Left}, {input.KeyRight, Right}, {input.KeyUp, Up}, {input.KeyDown, Down}} {
		if !g.in.Pressed(k.key) {
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
	c.RoundRect(geom.Rect{W: side, H: side}, 12, color.Hex(look.Surface))
	for r := range 4 {
		for col := range 4 {
			c.RoundRect(cellRect(geom.V2(float32(col), float32(r)), 1), 8, color.Hex(look.Cell))
		}
	}
	if g.since < slideTime {
		k := ease.OutCubic(g.since / slideTime)
		for _, s := range g.moved.Slides {
			from := geom.V2(float32(s.From[1]), float32(s.From[0]))
			drawTile(c, from.Lerp(geom.V2(float32(s.To[1]), float32(s.To[0])), k), 1, s.Value)
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
			drawTile(c, geom.V2(float32(col), float32(r)), scale, v)
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

func drawTile(c *twod.Canvas, at geom.Vec2, scale float32, v int) {
	r := cellRect(at, scale)
	c.RoundRect(r, 8*scale, tileColor(v))
	label := strconv.Itoa(v)
	opts := []twod.TextOption{twod.Typeface(twod.Bold()), twod.FontSize(fontSize(v) * scale), twod.Color(textColor(v))}
	size := c.Measure(label, opts...)
	c.Text(geom.V2(r.X+(r.W-size.X)/2, r.Y+(r.H-size.Y)/2), label, opts...)
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
	return color.Hex("#f9f6f2")
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
