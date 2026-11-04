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
}

func newPlay(win *platform.Window, seed func() uint64, data *save.Data) *Play {
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

	drawBird(c, geom.V2(BirdX, g.Y), g.VY, p.flapped, p.time)
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
// falls, its wing fluttering and beating hard just after a flap.
func drawBird(c *twod.Canvas, at geom.Vec2, vy, flapped, time float32) {
	tilt := float64(min(max(vy/600, -0.5), 1.1))
	sin, cos := math.Sincos(tilt)
	turn := func(v geom.Vec2) geom.Vec2 {
		return at.Add(geom.V2(v.X*float32(cos)-v.Y*float32(sin), v.X*float32(sin)+v.Y*float32(cos)))
	}
	oval := func(centre geom.Vec2, rx, ry, angle float32, col color.Color) {
		const n = 18
		points := []geom.Vec2{turn(centre)}
		var tris []int
		for i := range n + 1 {
			a := float64(i) * 2 * math.Pi / n
			v := geom.V2(rx*float32(math.Cos(a)), ry*float32(math.Sin(a)))
			s, c := math.Sincos(float64(angle))
			v = geom.V2(v.X*float32(c)-v.Y*float32(s), v.X*float32(s)+v.Y*float32(c))
			points = append(points, turn(centre.Add(v)))
			if i > 0 {
				tris = append(tris, 0, i, i+1)
			}
		}
		c.Triangles(points, tris, col)
	}
	tri := func(a, b, d geom.Vec2, col color.Color) {
		c.Triangles([]geom.Vec2{turn(a), turn(b), turn(d)}, []int{0, 1, 2}, col)
	}
	ink := color.Hex("#3b2a3f")

	tri(geom.V2(-12, -3), geom.V2(-22, -9), geom.V2(-20, 0), ink)
	tri(geom.V2(-12, 1), geom.V2(-21, 4), geom.V2(-14, 6), ink)
	tri(geom.V2(-12, -2), geom.V2(-19, -6), geom.V2(-18, 0), color.Hex("#e07a3f"))
	oval(geom.V2(0, 0), 15, 13, 0, ink)
	oval(geom.V2(0, 0), 13.5, 11.5, 0, color.Hex("#ff9e64"))
	oval(geom.V2(2, 4.5), 9, 5.5, 0, color.Hex("#ffe0b8"))
	c.Circle(turn(geom.V2(4, 3)), 2.2, color.Hex("#ff6b8b").Alpha(0.45))

	beat := -0.4 + 0.25*float32(math.Sin(float64(time*12)))
	if flapped < 0.25 {
		beat = -0.4 + 1.3*float32(math.Sin(float64(flapped/0.25*math.Pi)))
	}
	oval(geom.V2(-5, 2.5), 7.5, 4.2, beat, color.Hex("#b85a2a"))
	oval(geom.V2(-5, 2.5), 6.3, 3.2, beat, color.Hex("#f08a4b"))

	c.Circle(turn(geom.V2(6, -4)), 5.2, ink)
	c.Circle(turn(geom.V2(6, -4)), 4.3, color.Hex("#ffffff"))
	c.Circle(turn(geom.V2(7.4, -4)), 2.2, ink)
	c.Circle(turn(geom.V2(6.6, -5.2)), 0.9, color.Hex("#ffffff"))

	tri(geom.V2(11, -3), geom.V2(21, 0.5), geom.V2(11, 2.5), ink)
	tri(geom.V2(11.5, -2), geom.V2(19.5, 0.5), geom.V2(11.5, 1.5), color.Hex("#ffd166"))
	tri(geom.V2(11, 1.5), geom.V2(18, 2.5), geom.V2(11, 5.5), color.Hex("#f4a259"))
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
