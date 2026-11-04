// Flap: tap to flap through the gaps between the pipes; each one passed is
// a point. Space, Up or a click flaps.
//
//	go run ./games/flap
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

// The game-over question waits a moment after the crash.
const askAfter = 0.8

// Play reads the flaps, steps the world on its Tick, sounds what happened and
// keeps the HUD in step.
type Play struct {
	game        *Game
	in          *input.State
	hud         *ui.Document
	over, start *ui.Node
	score, best *ui.Node
	seed        func() uint64
	started     bool
	clock       float32 // time not yet stepped
	ended, time float32
	flapped     float32 // since the last flap, for the wing
	sky         *Sky
	puffs       *twod.Particles
	data        *save.Data
	sounds      *Sounds
	player      *sound.Player
	menu        *look.Menu
	fit         *look.Fit
}

// Sky is the play area, a twod entity drawing the world every frame.
type Sky struct {
	twod.Item
	play *Play
	art  *Art
}

func newPlay(win *platform.Window, seed func() uint64, data *save.Data) *Play {
	pictures, err := loadArt()
	if err != nil {
		panic(err) // embedded files that always decode
	}
	in := win.Input()
	p := &Play{game: New(seed()), in: in, hud: look.HUD(), seed: seed, data: data, flapped: 1,
		sounds: newSounds(), player: sound.NewPlayer(music(), data.Music, data.Effects, !data.MusicOff)}
	scoreCard, score := look.Stat("Score", "score")
	bestCard, best := look.Stat("Best", "best")
	p.score, p.best = score, best
	p.over = look.GameOver(p.restart)
	p.start = ui.P(ui.Class("over-message"), ui.ID("start"), ui.Text("Flap to start"))
	p.hud.Body().Append(look.Side("Flap",
		look.Stats(scoreCard, bestCard),
		p.start,
		look.NewGame(p.hud, p.restart),
		look.Keys("Space", "flap", "Click", "flap too", "N", "new game", "F", "full screen", "Esc", "menu")), p.over)
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
	p.sky = twod.Build(&Sky{Item: twod.Base(), play: p, art: pictures}, "board", look.Place(Width, Height))
	p.puffs = twod.NewParticles(nil, twod.Rate(0), twod.Velocity(30, 120), twod.Direction(geom.Pi, 1.2),
		twod.Lifetime(0.3, 0.7), twod.SizeOverLife(5, 1), twod.Drag(0.08),
		twod.ColorOverLife(color.Hex("#ffffff").Alpha(0.7), color.Hex("#ffffff").Alpha(0)), twod.MaxParticles(400))
	p.show()
	return p
}

func (p *Play) Update(f world.Frame) {
	p.time += f.DT
	p.flapped += f.DT
	if p.menu.Open() {
		return
	}
	if p.in.Pressed(input.KeyN) {
		p.restart()
		return
	}
	if p.game.Over {
		p.ask(f.DT)
		return
	}
	if p.in.Pressed(input.KeySpace) || p.in.Pressed(input.KeyUp) || p.in.Pressed(input.KeyW) ||
		p.in.Pressed(input.MouseLeft) && !p.in.PointerCaptured() {
		p.started = true
		p.game.Flap()
		p.flapped = 0
		p.player.Play(p.sounds.flap)
		p.burst(8)
	}
	if !p.started {
		return
	}
	p.start.SetAttr("hidden", "")
	score := p.game.Score
	for p.clock += f.DT; p.clock >= Tick && !p.game.Over; p.clock -= Tick {
		p.game.Step()
	}
	switch {
	case p.game.Over:
		p.player.Play(p.sounds.hit)
		p.burst(30)
	case p.game.Score != score:
		p.player.Play(p.sounds.point)
	}
	p.show()
}

// burst sends n puffs from the bird.
func (p *Play) burst(n int) {
	p.puffs.SetPosition(p.sky.Position().Add(geom.V2(BirdX-6, p.game.Y)))
	p.puffs.Snap()
	p.puffs.Emit(n)
}

// ask records the score and shows the game-over question once askAfter has
// passed.
func (p *Play) ask(dt float32) {
	if p.ended < askAfter && p.ended+dt >= askAfter {
		message := fmt.Sprintf("Score %d.", p.game.Score)
		if p.data.Add(p.game.Score, time.Now()) == 1 && p.game.Score > 0 {
			message = fmt.Sprintf("New best: %d!", p.game.Score)
		}
		p.store()
		p.show()
		look.SetGameOver(p.over, message)
		p.hud.ShowModal(p.over)
	}
	p.ended += dt
}

func (p *Play) store() {
	if err := save.Store("flap", *p.data); err != nil {
		log.Print(err)
	}
}

func (p *Play) restart() {
	p.game, p.started, p.clock, p.ended = New(p.seed()), false, 0, 0
	p.start.RemoveAttr("hidden")
	p.show()
}

func (p *Play) show() {
	p.score.SetText(strconv.Itoa(p.game.Score))
	best := 0
	if len(p.data.Scores) > 0 {
		best = p.data.Scores[0].Score
	}
	p.best.SetText(strconv.Itoa(max(best, p.game.Score)))
}

// The sheets' regions, in pixels: a pipe's cap and body, the ground strip,
// a bird's frame. Everything draws at px points a pixel.
const (
	px                   = 2
	capH, pipeW          = 16, 32
	bodyTop, bodyEnd     = 16, 64
	groundY, groundW     = 80, 64
	birdSize, frames     = 16, 4
	sheetW, sheetH       = 128, 112
	birdsW, birdsH       = 64, 112
	skyW, skyH, skyShown = 256, 256, 165 // the sky and clouds, above the picture's city
)

// uv is a region of a sheet w by h pixels, as the 0 to 1 a canvas takes.
func uv(x, y, w, h, sheetW, sheetH float32) geom.Rect {
	return geom.Rect{X: x / sheetW, Y: y / sheetH, W: w / sheetW, H: h / sheetH}
}

func (s *Sky) Draw(c *twod.Canvas) {
	p := s.play
	g := p.game
	board := geom.Rect{W: Width, H: Height}
	c.PushClipRounded(board, [4]float32{10, 10, 10, 10})
	defer c.PopClip()
	white := color.Hex("#ffffff")

	// The sky and its clouds scroll slower than the pipes, standing on the
	// ground, the sky's top colour above them.
	sky := geom.Rect{Y: Height - Ground - skyShown*px, W: skyW * px, H: skyShown * px}
	c.Rect(geom.Rect{W: Width, H: sky.Y + 1}, color.Hex("#0cf1ff"))
	for x := -wrap(g.Travel*0.25, sky.W); x < Width; x += sky.W {
		c.Image(geom.Rect{X: x, Y: sky.Y, W: sky.W, H: sky.H}, s.art.sky, uv(0, 0, skyW, skyShown, skyW, skyH), white)
	}

	for _, pipe := range g.Pipes {
		s.drawPipe(c, pipe)
	}

	strip := uv(0, groundY, groundW, sheetH-groundY, sheetW, sheetH)
	for x := -wrap(g.Travel, groundW*px); x < Width; x += groundW * px {
		c.Image(geom.Rect{X: x, Y: Height - Ground, W: groundW * px, H: Ground}, s.art.pipes, strip, white)
	}

	s.drawBird(c, geom.V2(BirdX, g.Y), g.VY, p.flapped, p.time)
}

// drawPipe draws a pair, each cap at the gap and the body stretched to the
// sky and to the ground.
func (s *Sky) drawPipe(c *twod.Canvas, p Pipe) {
	white := color.Hex("#ffffff")
	body := uv(0, bodyTop, pipeW, bodyEnd-bodyTop, sheetW, sheetH)
	rim := float32(capH * px)
	c.Image(geom.Rect{X: p.X, W: PipeWidth, H: p.Gap - rim}, s.art.pipes, body, white)
	bottomCap := uv(0, bodyEnd, pipeW, capH, sheetW, sheetH)
	c.Image(geom.Rect{X: p.X, Y: p.Gap - rim, W: PipeWidth, H: rim}, s.art.pipes, bottomCap, white)
	bottom := p.Gap + GapHeight
	c.Image(geom.Rect{X: p.X, Y: bottom, W: PipeWidth, H: rim}, s.art.pipes, uv(0, 0, pipeW, capH, sheetW, sheetH), white)
	c.Image(geom.Rect{X: p.X, Y: bottom + rim, W: PipeWidth, H: Height - Ground - bottom - rim}, s.art.pipes, body, white)
}

// drawBird draws the bird at at, tilted up while it rises and down as it
// falls, its wings beating fast just after a flap and slowly in between.
func (s *Sky) drawBird(c *twod.Canvas, at geom.Vec2, vy, flapped, time float32) {
	tilt := min(max(vy/600, -0.5), 1.1)
	frame := int(time/0.12) % frames
	if flapped < 0.24 {
		frame = int(flapped/0.06) % frames
	}
	saved := c.Transform()
	c.SetTransform(saved.Mul(geom.Transform2DFrom(at, tilt, 0, geom.V2(1, 1))))
	size := float32(birdSize * px)
	c.Image(geom.Rect{X: -size / 2, Y: -size / 2, W: size, H: size}, s.art.birds,
		uv(float32(frame*birdSize), 0, birdSize, birdSize, birdsW, birdsH), color.Hex("#ffffff"))
	c.SetTransform(saved)
}

func wrap(x, n float32) float32 { return x - n*float32(math.Floor(float64(x/n))) }

func main() {
	fixed := flag.Uint64("seed", 0, "replay the game this seed deals (0: a new game each time)")
	flag.Parse()
	seed := func() uint64 {
		if *fixed != 0 {
			return *fixed
		}
		return uint64(time.Now().UnixNano())
	}
	win := look.Window("flap")
	data, err := save.Load("flap")
	if err != nil {
		log.Print(err)
	}
	p := newPlay(win, seed, &data)
	win.Add(look.Camera(), p.sky, p.puffs, p.hud)
	ember.New().AddEntities(win, p.player.Mixer).AddSystems(p, p.fit).MustRun()
}
