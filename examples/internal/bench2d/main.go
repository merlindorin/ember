// Bench2d is the demo's late-run scene, rebuilt from the crypt's art and the
// same engine pieces, for measuring the 2D renderer: 600 animated enemies of
// four breeds y-sorted with the hero, 200 arrows, 150 flasks, 50 damage
// numbers, the endless floor and its things, and a round minimap drawn by a
// camera into a texture. It runs a few hundred frames on the GPU and prints
// the average render time and draw counts. -scene wide spreads the horde to
// three times the view's width, for culling; -scene crowd has about 5,000
// entities, for headroom; -scene 20k is 20,000 enemies built and moved as
// the demo does (chasing the hero as a soft crowd through a collide space,
// y-sorted), about 2,000 of them in view; -scene 200k is 190,000 still
// sprites over a wide world and 10,000 enemies moving and colliding round
// the hero. A game measures its own scenes with a bench of its own. Every
// scene prints its CPU tick, its render, the costs the World records
// (twod's collect and draw, and the GPU's passes when timed) and its
// slowest systems. GPU timing costs frame time, so take frames from
// a plain run and GPU ms from one with EMBER_STATS=1.
//
//	EMBER_HIDDEN=1 [EMBER_STATS=1] go run ./examples/internal/bench2d \
//		[-scene wave23|wide|crowd|20k|200k] [-w 2000 -h 1125 -frames 300] [-cpuprofile f]
package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"runtime/pprof"
	"time"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/examples/internal/crypt"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

type walker struct {
	sprite *twod.Sprite
	speed  float32
}

type flier struct {
	sprite *twod.Sprite
	vel    geom.Vec2
}

// dots draws the minimap's marks, one square each.
type dots struct {
	twod.Item
	at []geom.Vec2
}

func (d *dots) Draw(c *twod.Canvas) {
	for _, p := range d.at {
		c.Rect(geom.Rect{X: p.X - 10, Y: p.Y - 10, W: 20, H: 20}, color.Hex("#d04040"))
	}
}

func main() {
	width := flag.Int("w", 1280, "window width in points")
	height := flag.Int("h", 720, "window height in points")
	frames := flag.Int("frames", 300, "frames measured after a 60-frame warm-up")
	scene := flag.String(
		"scene",
		"wave23",
		"wave23, wide (the horde 3× the view's width), crowd (about 5,000 entities), "+
			"20k (the demo's way, 20,000 enemies), 200k (190,000 still sprites, 10,000 moving) or lights",
	)
	lightOpts := newLightFlags()
	profile := flag.String("cpuprofile", "", "write a CPU profile of the measured frames to this file")
	flag.Parse()
	w, _, err := build(*scene, *width, *height, lightOpts)
	if err != nil {
		fmt.Fprintln(os.Stdout, "bench2d:", err)
		return
	}
	measure(w, *scene, *width, *height, *frames, *profile, nil)
	w.MustRun()
}

// build is the scene named, in a window of width × height points; the
// lights scene takes lightOpts.
func build(scene string, width, height int, lightOpts *lightFlags) (*world.World, *platform.Window, error) {
	count, reach := 1, float32(1)
	switch scene {
	case "lights":
		win := platform.NewWindow(
			platform.Size(width, height),
			platform.Background(color.Hex("#1d2129")),
			platform.VSync(false),
		)
		return lights(win, width, height, lightOpts), win, nil
	case "20k", "200k":
		count = 0
	case "wide":
		reach = 3 * float32(width) / 2 / 540
	case "crowd":
		count = 5
	case "wave23":
	default:
		return nil, nil, fmt.Errorf("no scene %s", scene)
	}

	r := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // a game's randomness, not a secret
	win := platform.NewWindow(
		platform.Size(width, height),
		platform.Background(color.Hex("#1d2129")),
		platform.VSync(false),
	)

	floor := twod.NewLayer(0)
	shards := twod.NewLayer(1)
	crowd := twod.NewLayer(2, twod.SortByY())
	shots := twod.NewLayer(3)
	sparks := twod.NewLayer(4)
	addFloor(floor, crowd)

	heroSheet, heroClips := crypt.Character("elf_m")
	hero := twod.NewSprite(heroSheet, twod.Size(32, 56), twod.Animate(heroClips), twod.Playing("run"))
	crowd.Add(hero)

	b := &bench{hero: hero, reach: reach}
	b.spawn(r, count, crowd, shots, shards, sparks)

	eye := twod.NewCamera(twod.Smoothing(8), twod.Follow(hero), twod.Layers(floor, shards, crowd, shots, sparks))
	win.Add(floor, shards, crowd, shots, sparks, eye)
	marks := addMinimap(win, width, hero)
	w := ember.New().AddEntities(win)
	switch scene {
	case "20k":
		radius := geom.Sqrt(float32(width*height) * 10 / geom.Pi)
		horde(w, crowd, hero, 20000, radius, r)
	case "200k":
		radius := geom.Sqrt(float32(width*height) * 10 / geom.Pi)
		horde(w, crowd, hero, 10000, radius, r)
		still(floor, 190000, r)
	}

	b.marks = marks
	w.On(world.Update, "move", b.move)
	return w, win, nil
}

// addFloor puts the endless stone floor on floor and a few crates a chunk,
// standing in crowd.
func addFloor(floor, crowd *twod.Layer) {
	floor.Add(twod.NewInfiniteMap(crypt.Floor(), twod.CellSize(32, 32), twod.Generator(func(cx, cy int, cells [][]int) {
		g := rand.New(rand.NewPCG(uint64(cx), uint64(cy))) //nolint:gosec // a game's randomness, not a secret
		for y := range cells {
			for x := range cells[y] {
				cells[y][x] = crypt.Stone + g.IntN(8)
			}
		}
	})))
	floor.Add(
		twod.NewInfiniteMap(
			crypt.Things(),
			twod.CellSize(32, 32),
			twod.TallIn(crowd),
			twod.Generator(func(cx, cy int, cells [][]int) {
				g := rand.New(rand.NewPCG(uint64(cx)+7, uint64(cy))) //nolint:gosec // a game's randomness, not a secret
				for range 3 {
					cells[g.IntN(16)][g.IntN(16)] = crypt.Crate + g.IntN(2)
				}
			}),
		),
	)
}

// addMinimap adds a round minimap in win's top-right corner following
// hero, and returns the dots it draws.
func addMinimap(win *platform.Window, width int, hero *twod.Sprite) *dots {
	texture := twod.NewRenderTexture(160, 160)
	marks := &dots{Item: twod.Base()}
	marksLayer := twod.NewLayer(10).Add(twod.Build(marks, "dots"))
	minimap := twod.NewCamera(twod.Into(texture), twod.Zoom(0.2), twod.Follow(hero), twod.Layers(marksLayer))
	circle := make([]geom.Vec2, 48)
	for i := range circle {
		circle[i] = geom.FromAngle(float32(i) * 2 * geom.Pi / 48).Scale(80)
	}
	mapAt := geom.V2(float32(width)-100, 100)
	disc := twod.NewPolygon(circle, twod.Color(color.White), twod.At(mapAt.X, mapAt.Y))
	shape := twod.NewLayer(8, twod.Fixed(), twod.AsMask()).Add(disc)
	picture := twod.NewLayer(7, twod.Fixed(), twod.MaskedBy(shape)).Add(
		twod.NewPolygon(circle, twod.Color(color.RGBA(0.06, 0.05, 0.07, 0.85)), twod.At(mapAt.X, mapAt.Y)),
		twod.NewSprite(texture, twod.At(mapAt.X, mapAt.Y)))
	ring := twod.NewLayer(9, twod.Fixed()).
		Add(twod.NewLine(circle, twod.Closed(), twod.Width(3), twod.Color(color.Hex("#c9a36b")), twod.At(mapAt.X, mapAt.Y)))

	win.Add(marksLayer, minimap, shape, picture, ring)
	return marks
}

// bench is what the move system moves each frame.
type bench struct {
	hero    *twod.Sprite
	walkers []walker
	fliers  []flier
	numbers []*twod.Text
	marks   *dots
	reach   float32
	frame   int
}

// spawn fills the layers around the hero: count×600 walkers in crowd,
// count×200 arrows in shots, count×150 flasks in shards and count×50
// numbers in sparks.
func (b *bench) spawn(r *rand.Rand, count int, crowd, shots, shards, sparks *twod.Layer) {
	near := func(spread float32) geom.Vec2 {
		return geom.FromAngle(r.Float32() * 2 * geom.Pi).Scale(40 + r.Float32()*spread*b.reach)
	}
	for i, breed := range []string{"goblin", "skelet", "imp", "orc_warrior"} {
		sheet, clips := crypt.Character(breed)
		for range 150 * count {
			at := near(500)
			s := twod.NewSprite(
				sheet,
				twod.Size(32+float32(i/3)*8, 32+float32(i/3)*14),
				twod.Animate(clips),
				twod.Playing("run"),
				twod.At(at.X, at.Y),
			)
			crowd.Add(s)
			b.walkers = append(b.walkers, walker{s, 30 + r.Float32()*40})
		}
	}
	arrow := crypt.Sprite("weapon_arrow")
	for range 200 * count {
		at, dir := near(400), geom.FromAngle(r.Float32()*2*geom.Pi)
		s := twod.NewSprite(arrow, twod.At(at.X, at.Y), twod.Rotation(dir.Angle()+geom.Pi/2))
		shots.Add(s)
		b.fliers = append(b.fliers, flier{s, dir.Scale(300)})
	}
	flasks := []*twod.Texture{crypt.Sprite("flask_green"), crypt.Sprite("flask_yellow"), crypt.Sprite("flask_red")}
	for i := range 150 * count {
		at := near(600)
		shards.Add(twod.NewSprite(flasks[i%3], twod.At(at.X, at.Y)))
	}
	for range 50 * count {
		at := near(300)
		t := twod.NewText("12", twod.FontSize(14), twod.Color(color.Hex("#ffd27a")), twod.At(at.X, at.Y))
		sparks.Add(t)
		b.numbers = append(b.numbers, t)
	}
}

// move walks the crowd to the hero, flies the arrows, changes the numbers
// and puts a dot on the minimap for each walker.
func (b *bench) move(f world.Frame) {
	b.hero.Translate(geom.V2(60*f.DT, 20*f.DT))
	from := b.hero.Position()
	for _, e := range b.walkers {
		to := from.Sub(e.sprite.Position())
		if to.LenSq() > 1 {
			e.sprite.Translate(to.Normalize().Scale(e.speed * f.DT))
		}
	}
	for i := range b.fliers {
		fl := &b.fliers[i]
		fl.sprite.Translate(fl.vel.Scale(f.DT))
		if fl.sprite.Position().Sub(from).LenSq() > 500*500*b.reach*b.reach {
			fl.sprite.SetPosition(from)
		}
	}
	for i, t := range b.numbers {
		if (b.frame+i)%20 == 0 {
			t.SetText(fmt.Sprint(10 + (b.frame+i)%90))
		}
	}
	b.marks.at = b.marks.at[:0]
	for _, e := range b.walkers {
		b.marks.at = append(b.marks.at, e.sprite.Position())
	}
	b.frame++
}

func ms(d time.Duration) float64 { return float64(d) / 1e6 }

// measure runs frames after a 60-frame warm-up and prints the averages:
// the frame, the CPU tick, the render, the recorded costs and the
// slowest systems; with profile, a CPU profile of those frames.
func measure(w *world.World, scene string, width, height, frames int, profile string, _ func()) {
	frame := 0
	var render, total, tick time.Duration
	recorded := map[string]time.Duration{}
	var names []string
	w.On(world.PostUpdate, "measure", func(f world.Frame) {
		frame++
		s := f.World.Stats()
		if frame == 61 && profile != "" {
			if out, err := os.Create(profile); err == nil {
				_ = pprof.StartCPUProfile(out)
			}
		}
		if frame > 60 {
			render += s.RenderTime
			total += s.FrameTime
			tick += s.TickTime
			for _, r := range s.Recorded() {
				if _, ok := recorded[r.Name]; !ok {
					names = append(names, r.Name)
				}
				recorded[r.Name] += r.Time
			}
		}
		if frame < 60+frames {
			return
		}
		n := time.Duration(frames)
		fmt.Fprintf(
			os.Stdout,
			"%s, %dx%d, %d entities: frame %.2f ms, tick %.2f ms, render %.2f ms; "+
				"2D draws %d, binds %d, vertices %d, passes %d\n",
			scene,
			width,
			height,
			f.World.Len(),
			ms(total/n),
			ms(tick/n),
			ms(render/n),
			s.Draws2D,
			s.Binds2D,
			s.Vertices2D,
			s.Passes2D,
		)
		for _, name := range names {
			fmt.Fprintf(os.Stdout, "  %-24s %6.2f ms\n", name, ms(recorded[name]/n))
		}
		fmt.Fprintln(os.Stdout, "slowest systems:")
		for _, sys := range s.Slowest(8) {
			fmt.Fprintf(os.Stdout, "  %-24s %-11s %6.2f ms\n", sys.Name, sys.Phase, ms(sys.Time))
		}
		if profile != "" {
			pprof.StopCPUProfile()
		}
		f.Quit()
	})
}

// still adds n sprites that never move to layer, spread over a square
// round the origin as wide as n of them a 48-point cell apart.
func still(layer *twod.Layer, n int, r *rand.Rand) {
	side := geom.Sqrt(float32(n)) * 48
	flasks := []*twod.Texture{
		crypt.Sprite("flask_green"),
		crypt.Sprite("flask_yellow"),
		crypt.Sprite("flask_red"),
		crypt.Sprite("weapon_arrow"),
	}
	for i := range n {
		layer.Add(twod.NewSprite(flasks[i%len(flasks)], twod.At((r.Float32()-0.5)*side, (r.Float32()-0.5)*side)))
	}
}
