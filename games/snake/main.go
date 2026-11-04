// Snake: steer with the arrows, eat the apples, don't hit a wall or
// yourself. Every apple makes the snake longer and a little faster.
//
//	go run ./games/snake
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
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

const cell = 20

// The board's size: the cells with a 4-point margin.
const boardW, boardH = W*cell + 8, H*cell + 8

// The game-over question waits a moment after the crash.
const askAfter = 0.6

// Play reads the arrows, steps the snake on time, sounds what happened and
// keeps the HUD in step.
type Play struct {
	controls      *input.Actions[action]
	game          *Game
	in            *input.State
	hud           *ui.Document
	over, start   *ui.Node
	score, length *ui.Node
	seed          func() uint64
	started       bool
	clock         float32 // since the last step
	ended, time   float32
	before        []Cell // the body before the last step, to draw the move
	pit           *Pit
	sparks        *twod.Particles
	data          *save.Data
	sounds        *Sounds
	player        *sound.Player
	menu          *look.Menu
	fit           *look.Fit
}

// An action is what the player asks with a key or a pad button.
type action int

const (
	steerLeft action = iota
	steerRight
	steerUp
	steerDown
	again
	pause
)

func newControls() *input.Actions[action] {
	return input.NewActions(
		input.Bind(steerLeft, input.KeyLeft, input.KeyA, input.GamepadLeft, input.GamepadLeftStickLeft),
		input.Bind(steerRight, input.KeyRight, input.KeyD, input.GamepadRight, input.GamepadLeftStickRight),
		input.Bind(steerUp, input.KeyUp, input.KeyW, input.GamepadUp, input.GamepadLeftStickUp),
		input.Bind(steerDown, input.KeyDown, input.KeyS, input.GamepadDown, input.GamepadLeftStickDown),
		input.Bind(again, look.AgainButtons()...),
		input.Bind(pause, look.MenuButtons()...))
}

// Pit is the play area, a twod entity drawing the board every frame.
type Pit struct {
	twod.Item
	play *Play
	skin *look.Skin
}

func newPlay(win *platform.Window, seed func() uint64, data *save.Data) *Play {
	in := win.Input()
	p := &Play{game: New(seed()), in: in, controls: newControls(), hud: look.HUD(), seed: seed, data: data,
		sounds: newSounds(), player: sound.NewPlayer(music(), data.Music, data.Effects, !data.MusicOff)}
	p.before = p.game.Body
	scoreCard, score := look.Stat("Score", "score")
	lengthCard, length := look.Stat("Length", "length")
	p.score, p.length = score, length
	p.over = look.GameOver(p.restart)
	p.start = ui.P(ui.Class("over-message"), ui.ID("start"), ui.Text("Press an arrow to start"))
	p.hud.Body().Append(look.Side("Snake",
		look.Stats(scoreCard, lengthCard),
		p.start,
		look.NewGame(p.hud, p.restart),
		look.Controls(
			look.Control{Keys: []string{"left", "right", "up", "down"}, Pad: []string{"dpad"}, Does: "steer"},
			look.AgainControl(), look.MenuControl())), p.over)
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
	p.pit = twod.Build(&Pit{Item: twod.Base(), play: p, skin: look.LoadSkin()}, "board", look.Place(boardW, boardH))
	p.sparks = twod.NewParticles(nil, twod.Rate(0), twod.Velocity(40, 160), twod.Direction(0, geom.Pi),
		twod.Lifetime(0.3, 0.7), twod.SizeOverLife(5, 0.5), twod.Drag(0.05),
		twod.ColorOverLife(color.Hex("#fff1d2"), color.Hex("#cf5e53").Alpha(0)), twod.Blend(twod.Add),
		twod.MaxParticles(300))
	p.show()
	return p
}

func (p *Play) Update(f world.Frame) {
	p.time += f.DT
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
	for _, k := range []struct {
		steer action
		dir   Dir
	}{{steerRight, Right}, {steerDown, Down}, {steerLeft, Left}, {steerUp, Up}} {
		if p.controls.Pressed(p.in, k.steer) {
			p.started = true
			p.game.Turn(k.dir)
			p.player.Play(p.sounds.turn)
		}
	}
	if !p.started {
		return
	}
	p.start.SetAttr("hidden", "")
	p.clock += f.DT
	if p.clock < p.game.Interval() {
		return
	}
	p.clock -= p.game.Interval()
	p.before = p.game.Body
	eaten, apple := p.game.Eaten, p.game.Apple
	p.game.Step()
	switch {
	case p.game.Over:
		p.player.Play(p.sounds.over)
	case p.game.Eaten != eaten:
		p.player.Play(p.sounds.eatOf(eaten))
		p.sparks.SetPosition(p.pit.Position().Add(centre(apple)))
		p.sparks.Snap()
		p.sparks.Emit(24)
	}
	p.show()
}

// ask records the score and shows the game-over question once askAfter has
// passed.
func (p *Play) ask(dt float32) {
	if p.ended < askAfter && p.ended+dt >= askAfter {
		message := fmt.Sprintf("Score %d, %d long.", p.game.Score, len(p.game.Body))
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
	if err := save.Store("snake", *p.data); err != nil {
		log.Print(err)
	}
}

func (p *Play) restart() {
	p.game, p.started, p.clock, p.ended = New(p.seed()), false, 0, 0
	p.before = p.game.Body
	p.start.RemoveAttr("hidden")
	p.show()
}

func (p *Play) show() {
	p.score.SetText(strconv.Itoa(p.game.Score))
	p.length.SetText(strconv.Itoa(len(p.game.Body)))
}

func (pit *Pit) Draw(c *twod.Canvas) {
	p := pit.play
	pit.skin.Board(c, geom.Rect{W: boardW, H: boardH})
	for y := range H {
		for x := range W {
			if (x+y)%2 == 0 {
				square := geom.Rect{X: float32(4 + x*cell), Y: float32(4 + y*cell), W: cell, H: cell}
				c.Rect(square, color.Hex(look.Surface).Alpha(0.3))
			}
		}
	}
	pulse := 1 + 0.08*float32(math.Sin(float64(p.time)*6))
	c.Circle(centre(p.game.Apple), cell*0.38*pulse, color.Hex("#cf5e53"))
	c.Circle(centre(p.game.Apple).Add(geom.V2(-2, -2)), cell*0.1, color.Hex("#ffffff").Alpha(0.5))

	k := min(p.clock/p.game.Interval(), 1)
	if p.game.Over || !p.started {
		k = 1
	}
	n := len(p.game.Body)
	at := make([]geom.Vec2, n)
	for i, b := range p.game.Body {
		at[i] = centre(b)
		if i < len(p.before) && !p.game.Over {
			at[i] = centre(p.before[i]).Lerp(at[i], k)
		}
	}
	for i := n - 1; i >= 0; i-- {
		t := float32(i) / float32(max(n-1, 1))
		col := color.Hex("#72b84e").Lerp(color.Hex("#3f6e2b"), t)
		width := cell - 4 - 4*t
		if i > 0 {
			c.Line(at[i-1], at[i], width, col)
		}
		c.Circle(at[i], width/2, col)
	}
	drawEyes(c, at[0], p.game.Heading())
}

func drawEyes(c *twod.Canvas, head geom.Vec2, d Dir) {
	ahead := d.step()
	forward := geom.V2(float32(ahead.X), float32(ahead.Y))
	side := geom.V2(-forward.Y, forward.X)
	for _, s := range []float32{-1, 1} {
		eye := head.Add(forward.Scale(3)).Add(side.Scale(4 * s))
		c.Circle(eye, 2.6, color.Hex("#ffffff"))
		c.Circle(eye.Add(forward.Scale(1)), 1.3, color.Hex(look.Background))
	}
}

// centre is the middle of a cell, in the board's space.
func centre(at Cell) geom.Vec2 {
	return geom.V2(float32(4+at.X*cell)+cell/2, float32(4+at.Y*cell)+cell/2)
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
	win := look.Window("snake")
	data, err := save.Load("snake")
	if err != nil {
		log.Print(err)
	}
	p := newPlay(win, seed, &data)
	win.Add(look.Camera(), p.pit, p.sparks, p.hud)
	ember.New().AddEntities(win, p.player.Mixer).AddSystems(p, p.fit).MustRun()
}
