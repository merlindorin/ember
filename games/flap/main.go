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
}

// Sky is the play area, a twod entity drawing the world every frame.
type Sky struct {
	twod.Item
	play *Play
}

func newPlay(in *input.State, seed func() uint64, data *save.Data) *Play {
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
		look.Keys("Space", "flap", "Click", "flap too", "N", "new game", "Esc", "menu")), p.over)
	p.menu = look.NewMenu(p.hud, look.Hooks{NewGame: p.restart, Data: func() *save.Data { return p.data },
		SoundChanged: func() {
			p.player.Set(p.data.Music, p.data.Effects, !p.data.MusicOff)
			p.store()
		}})
	p.sky = twod.Build(&Sky{Item: twod.Base(), play: p}, "board", look.Place(Width, Height))
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

func (s *Sky) Draw(c *twod.Canvas) {
	p := s.play
	g := p.game
	board := geom.Rect{W: Width, H: Height}
	c.PushClipRounded(board, [4]float32{10, 10, 10, 10})
	defer c.PopClip()
	c.Gradient(board, color.Hex("#1f2135"), color.Hex("#3a3557"))

	// The far layers scroll slower than the pipes: clouds, then hills.
	for i := range 5 {
		x := wrap(float32(i)*90-g.Travel*0.15, Width+80) - 40
		y := float32(50 + (i*37)%90)
		for j, r := range []float32{18, 24, 16} {
			c.Circle(geom.V2(x+float32(j)*20, y), r, color.Hex("#ffffff").Alpha(0.05))
		}
	}
	for i := range 6 {
		x := wrap(float32(i)*80-g.Travel*0.4, Width+80) - 40
		c.Circle(geom.V2(x, Height-Ground+30), 70, color.Hex("#2a2c45"))
	}

	for _, pipe := range g.Pipes {
		drawPipe(c, pipe)
	}

	ground := geom.Rect{Y: Height - Ground, W: Width, H: Ground}
	c.Rect(ground, color.Hex("#2f3148"))
	c.Rect(geom.Rect{Y: Height - Ground, W: Width, H: 4}, color.Hex("#80ed99").Alpha(0.6))
	for i := range Width/24 + 2 {
		x := wrap(float32(i)*24-g.Travel, Width+24) - 24
		c.Rect(geom.Rect{X: x, Y: Height - Ground + 10, W: 12, H: 4}, color.Hex("#3a3d58"))
	}

	drawBird(c, geom.V2(BirdX, g.Y), g.VY, p.flapped)
}

func drawPipe(c *twod.Canvas, p Pipe) {
	body, rim := color.Hex("#56cfe1"), color.Hex("#3fa7b8")
	c.Rect(geom.Rect{X: p.X, W: PipeWidth, H: p.Gap}, body)
	c.RoundRect(geom.Rect{X: p.X - 4, Y: p.Gap - 18, W: PipeWidth + 8, H: 18}, 4, rim)
	bottom := p.Gap + GapHeight
	c.Rect(geom.Rect{X: p.X, Y: bottom, W: PipeWidth, H: Height - Ground - bottom}, body)
	c.RoundRect(geom.Rect{X: p.X - 4, Y: bottom, W: PipeWidth + 8, H: 18}, 4, rim)
	c.Rect(geom.Rect{X: p.X + 8, W: 6, H: p.Gap - 18}, color.Hex("#ffffff").Alpha(0.15))
	c.Rect(geom.Rect{X: p.X + 8, Y: bottom + 18, W: 6, H: Height - Ground - bottom - 18}, color.Hex("#ffffff").Alpha(0.15))
}

// drawBird draws the bird at at, tilted up while it rises and down as it
// falls, its wing beating just after a flap.
func drawBird(c *twod.Canvas, at geom.Vec2, vy, flapped float32) {
	tilt := float64(min(max(vy/600, -0.5), 1.1))
	turn := func(v geom.Vec2) geom.Vec2 {
		sin, cos := math.Sincos(tilt)
		return at.Add(geom.V2(v.X*float32(cos)-v.Y*float32(sin), v.X*float32(sin)+v.Y*float32(cos)))
	}
	c.Circle(at, Radius+1, color.Hex("#ff9e64"))
	c.Circle(turn(geom.V2(2, 4)), Radius-4, color.Hex("#ffd6a5"))
	beat := float32(math.Sin(float64(min(flapped*18, math.Pi*2))))
	wing := []geom.Vec2{turn(geom.V2(-8, 0)), turn(geom.V2(4, 0)), turn(geom.V2(-4, 8-12*beat))}
	c.Triangles(wing, []int{0, 1, 2}, color.Hex("#e07a3f"))
	c.Circle(turn(geom.V2(5, -5)), 4, color.Hex("#ffffff"))
	c.Circle(turn(geom.V2(6.5, -5)), 2, color.Hex(look.Background))
	beak := []geom.Vec2{turn(geom.V2(10, -1)), turn(geom.V2(19, 2)), turn(geom.V2(10, 5))}
	c.Triangles(beak, []int{0, 1, 2}, color.Hex("#f2cc8f"))
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
	p := newPlay(win.Input(), seed, &data)
	win.Add(p.sky, p.puffs, p.hud)
	ember.New().AddEntities(win, p.player.Mixer).AddSystems(p).MustRun()
}
