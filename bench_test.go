package ember_test

import (
	"image"
	"math/rand/v2"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/collide"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// bulletHell is the busiest frame the engine aims for: 500 enemies walking
// through a collision space, 1,000 trigger bullets, four particle fountains
// and 300 health bars following enemies, drawn as the demo draws its marks
// over the horde: one twod item, a back and a fill over each enemy.
func bulletHell(tb testing.TB) *world.World {
	tb.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	tex := twod.NewTexture(img)
	win := platform.NewWindow(platform.Size(1280, 720))
	space := collide.NewSpace(collide.Grid(32))
	random := rand.New(rand.NewPCG(1, 2))
	at := func() (float32, float32) { return random.Float32() * 1280, random.Float32() * 720 }

	field := twod.NewLayer(0)
	var enemies []*twod.Sprite
	for range 500 {
		x, y := at()
		e := twod.NewSprite(tex, twod.At(x, y), twod.Scale(2, 2))
		e.Add(collide.NewBody(collide.Box(14, 14), collide.Group(1), collide.CollidesWith(1)))
		field.Add(e)
		enemies = append(enemies, e)
	}
	type bullet struct {
		*twod.Sprite
		v geom.Vec2
	}
	var bullets []bullet
	for range 1000 {
		x, y := at()
		b := twod.NewSprite(tex, twod.At(x, y))
		b.Add(collide.NewBody(collide.Circle(3), collide.Trigger(), collide.Group(2), collide.CollidesWith(1)))
		field.Add(b)
		a := random.Float32() * 2 * geom.Pi
		bullets = append(bullets, bullet{b, geom.V2(geom.Cos(a), geom.Sin(a)).Scale(240)})
	}
	for i := range 4 {
		field.Add(twod.NewParticles(tex, twod.At(200+float32(i)*300, 600),
			twod.Rate(200), twod.Lifetime(1, 2), twod.Velocity(40, 120)))
	}
	bars := twod.Build(&healthBars{Item: twod.Base(), of: enemies[:300], fill: 0.7}, "bars")
	win.Add(field, twod.NewLayer(1).Add(bars))

	w := ember.New(ember.Headless()).AddSystems(collide.NewTracker(space)).AddEntities(win)
	centre := geom.V2(640, 360)
	w.On(world.FixedUpdate, "enemies", func(f world.Frame) {
		for _, e := range enemies {
			space.Move(e, centre.Sub(e.Position()).Normalize().Scale(60*f.DT))
		}
	})
	w.On(world.FixedUpdate, "bullets", func(f world.Frame) {
		for _, b := range bullets {
			p := b.Position().Add(b.v.Scale(f.DT))
			p.X, p.Y = wrap(p.X, 1280), wrap(p.Y, 720)
			b.SetPosition(p)
		}
	})
	w.RunFrames(10)
	return w
}

// barBack and barFill are a bar's colours.
//
//nolint:gochecknoglobals // parsed once, not each frame
var barBack, barFill = color.Hex("#1a1210"), color.Hex("#d04030")

// healthBars draws a bar over each enemy, filled to fill.
type healthBars struct {
	twod.Item
	of   []*twod.Sprite
	fill float32
}

func (b *healthBars) Draw(c *twod.Canvas) {
	for _, e := range b.of {
		at := e.Position().Add(geom.V2(-12, -16))
		c.Rect(geom.Rect{X: at.X, Y: at.Y, W: 24, H: 4}, barBack)
		c.Rect(geom.Rect{X: at.X, Y: at.Y, W: 24 * b.fill, H: 4}, barFill)
	}
}

func wrap(v, size float32) float32 {
	switch {
	case v < 0:
		return v + size
	case v > size:
		return v - size
	}
	return v
}

func BenchmarkBulletHellFrame(b *testing.B) {
	w := bulletHell(b)
	b.ReportAllocs()
	for b.Loop() {
		w.Frame(1.0 / 60)
	}
}

// TestTheBulletHellSceneStaysFast guards against a frame gone quadratic,
// not a timing: about a millisecond alone, it has a generous ceiling so a
// loaded machine running every package doesn't fail it.
// BenchmarkBulletHellFrame measures it.
func TestTheBulletHellSceneStaysFast(t *testing.T) {
	if testing.Short() {
		t.Skip("a timing check")
	}
	r := testing.Benchmark(BenchmarkBulletHellFrame)
	if per := r.NsPerOp(); per > 50_000_000 {
		t.Fatalf("a frame takes %.1f ms of CPU", float64(per)/1e6)
	}
}
