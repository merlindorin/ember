package twod

import (
	"math"
	"math/rand/v2"
	"slices"
	"sync/atomic"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/node"
	"github.com/lantervale/ember/world"
)

//nolint:gochecknoinits // registers the package's node types on import
func init() {
	node.Register(node.Type{New: func() node.Noded { return NewParticles(nil) }})
}

// Particles emits small pictures that fly, fade and die: fire, smoke,
// sparks, dust. Each particle shows the texture fitted to its size, or a
// round dot without one. They are simulated on the CPU before drawing, in
// the same order for the same seed and frame times.
type Particles struct {
	Item
	texture   *Texture    `node:"prop"`
	rate      float32     `node:"prop"`
	burst     int         `node:"prop"`
	oneShot   bool        `node:"prop"`
	lifetime  geom.Vec2   `node:"prop"`
	velocity  geom.Vec2   `node:"prop"`
	direction float32     `node:"prop"`
	spread    float32     `node:"prop"`
	gravity   geom.Vec2   `node:"prop"`
	drag      float32     `node:"prop"`
	start     color.Color `node:"prop"`
	end       color.Color `node:"prop"`
	sizes     geom.Vec2   `node:"prop"`
	spin      geom.Vec2   `node:"prop"`
	local     bool        `node:"prop"`
	max       int         `node:"prop"`
	seed      uint64      `node:"prop"`
	emitting  bool        `node:"prop"`
	random    *rand.Rand
	live      []particle
	carry     float32
	started   bool
	done      bool
}

type particle struct {
	at, speed       geom.Vec2
	age, life       float32
	rotation, twist float32
}

// ParticlesFinished is emitted when a OneShot's last particle dies.
type ParticlesFinished struct{ Particles *Particles }

//nolint:gochecknoglobals // the process-wide counter Seed's doc promises
var seeds atomic.Uint64

// NewParticles returns an emitter of tex, nil for round dots: by default
// 10 a second, living 1 second, launched up at 50 points a second in a
// narrow cone, white and 4 points wide.
func NewParticles(tex *Texture, opts ...ParticlesOption) *Particles {
	p := Build(&Particles{
		Item: Base(), texture: tex, rate: 10,
		lifetime: geom.V2(1, 1), velocity: geom.V2(50, 50), direction: -math.Pi / 2, spread: 0.2,
		start: color.White, end: color.White, sizes: geom.V2(4, 4),
		max: 1000, seed: seeds.Add(1), emitting: true,
	}, "particles")
	for _, o := range opts {
		o.particles(p)
	}
	return p
}

// ParticlesOption sets up Particles.
type ParticlesOption interface{ particles(*Particles) }

type particlesOption func(*Particles)

func (o particlesOption) particles(p *Particles) { o(p) }
func (o Option) particles(p *Particles)          { o(p) }

// Rate emits n particles a second while emitting.
func Rate(n float32) ParticlesOption { return particlesOption(func(p *Particles) { p.SetRate(n) }) }

// Burst emits n particles at once when the emitter starts.
func Burst(n int) ParticlesOption { return particlesOption(func(p *Particles) { p.burst = max(n, 0) }) }

// OneShot emits only the Burst, then is done once its particles die.
func OneShot() ParticlesOption { return particlesOption(func(p *Particles) { p.oneShot = true }) }

// Lifetime is how long each particle lives, in seconds, picked between lo
// and hi.
func Lifetime(lo, hi float32) ParticlesOption {
	return particlesOption(func(p *Particles) { p.SetLifetime(lo, hi) })
}

// Velocity is each particle's launch speed in points per second, picked
// between lo and hi.
func Velocity(lo, hi float32) ParticlesOption {
	return particlesOption(func(p *Particles) { p.SetVelocity(lo, hi) })
}

// Direction is the cone particles launch in: angle, in radians clockwise
// from the right, give or take spread.
func Direction(angle, spread float32) ParticlesOption {
	return particlesOption(func(p *Particles) { p.SetDirection(angle, spread) })
}

// Gravity accelerates every particle by g, in points per second squared.
func Gravity(g geom.Vec2) ParticlesOption {
	return particlesOption(func(p *Particles) { p.SetGravity(g) })
}

// Drag takes d of a particle's speed away each second, from 0 to 1.
func Drag(d float32) ParticlesOption { return particlesOption(func(p *Particles) { p.SetDrag(d) }) }

// Colors fades each particle from start, when it is born, to end, when it
// dies.
func ColorOverLife(start, end color.Color) ParticlesOption {
	return particlesOption(func(p *Particles) { p.SetColorOverLife(start, end) })
}

// Sizes grows or shrinks each particle from start to end over its life:
// the side of its box in points.
func SizeOverLife(start, end float32) ParticlesOption {
	return particlesOption(func(p *Particles) { p.SetSizeOverLife(start, end) })
}

// Spin turns each particle at a speed picked between lo and hi, in
// radians per second.
func Spin(lo, hi float32) ParticlesOption {
	return particlesOption(func(p *Particles) { p.SetSpin(lo, hi) })
}

// LocalSpace makes the particles move with the emitter. By default they
// stay where they were emitted, in its layer, as smoke behind a runner.
func LocalSpace() ParticlesOption { return particlesOption(func(p *Particles) { p.local = true }) }

// MaxParticles caps the live particles at n (default 1000); at the cap,
// new ones are skipped.
func MaxParticles(n int) ParticlesOption {
	return particlesOption(func(p *Particles) { p.max = max(n, 0) })
}

// Seed picks the random sequence: an emitter with an explicit seed gives
// the same particles for the same frame times, run after run. Without
// one, each emitter takes the next of a process-wide counter, which
// repeats only if the process makes its emitters in the same order.
func Seed(n uint64) ParticlesOption { return particlesOption(func(p *Particles) { p.seed = n }) }

// Each option reads back under its own name; what a game changes while
// particles fly has a setter taking the option's arguments.
func (p *Particles) Rate() float32                             { return p.rate }
func (p *Particles) Burst() int                                { return p.burst }
func (p *Particles) OneShot() bool                             { return p.oneShot }
func (p *Particles) Lifetime() (float32, float32)              { return p.lifetime.X, p.lifetime.Y }
func (p *Particles) Velocity() (float32, float32)              { return p.velocity.X, p.velocity.Y }
func (p *Particles) Direction() (float32, float32)             { return p.direction, p.spread }
func (p *Particles) Gravity() geom.Vec2                        { return p.gravity }
func (p *Particles) Drag() float32                             { return p.drag }
func (p *Particles) ColorOverLife() (color.Color, color.Color) { return p.start, p.end }
func (p *Particles) SizeOverLife() (float32, float32)          { return p.sizes.X, p.sizes.Y }
func (p *Particles) Spin() (float32, float32)                  { return p.spin.X, p.spin.Y }
func (p *Particles) LocalSpace() bool                          { return p.local }
func (p *Particles) MaxParticles() int                         { return p.max }
func (p *Particles) Seed() uint64                              { return p.seed }

func (p *Particles) SetRate(n float32)                       { p.rate = max(n, 0) }
func (p *Particles) SetLifetime(lo, hi float32)              { p.lifetime = geom.V2(lo, hi) }
func (p *Particles) SetVelocity(lo, hi float32)              { p.velocity = geom.V2(lo, hi) }
func (p *Particles) SetDirection(angle, spread float32)      { p.direction, p.spread = angle, spread }
func (p *Particles) SetGravity(g geom.Vec2)                  { p.gravity = g }
func (p *Particles) SetDrag(d float32)                       { p.drag = geom.Clamp01(d) }
func (p *Particles) SetColorOverLife(start, end color.Color) { p.start, p.end = start, end }
func (p *Particles) SetSizeOverLife(start, end float32)      { p.sizes = geom.V2(start, end) }
func (p *Particles) SetSpin(lo, hi float32)                  { p.spin = geom.V2(lo, hi) }

// Emitting reports whether the rate emits; SetEmitting pauses or resumes
// it, leaving the live particles be.
func (p *Particles) Emitting() bool      { return p.emitting }
func (p *Particles) SetEmitting(on bool) { p.emitting = on }

// Live is how many particles are alive.
func (p *Particles) Live() int { return len(p.live) }

// Done reports whether a OneShot has emitted and its particles are gone.
func (p *Particles) Done() bool { return p.done }

// Emit adds n particles now, such as a burst on a click.
func (p *Particles) Emit(n int) {
	origin, turn := geom.Vec2{}, float32(0)
	if !p.local {
		g := p.Global()
		origin, turn = g.Origin, float32(math.Atan2(float64(g.X.Y), float64(g.X.X)))
	}
	if p.random == nil {
		p.random = rand.New(rand.NewPCG(p.seed, 0x9e3779b97f4a7c15)) //nolint:gosec // seeded to repeat, not for secrets
	}
	for range n {
		if len(p.live) >= p.max {
			return
		}
		angle := turn + p.direction + p.spread*(2*p.random.Float32()-1)
		s, c := math.Sincos(float64(angle))
		speed := p.pick(p.velocity)
		p.live = append(p.live, particle{
			at:    origin,
			speed: geom.V2(float32(c), float32(s)).Scale(speed),
			life:  max(p.pick(p.lifetime), 1e-3),
			twist: p.pick(p.spin),
		})
	}
}

func (p *Particles) pick(r geom.Vec2) float32 { return r.X + (r.Y-r.X)*p.random.Float32() }

// step moves the particles on by dt seconds, emits what the rate asks and
// reports whether a OneShot just finished.
func (p *Particles) step(dt float32) bool {
	if !p.started {
		p.started = true
		p.Emit(p.burst)
	}
	if p.emitting && !p.oneShot && p.rate > 0 {
		p.carry += p.rate * dt
		n := int(p.carry)
		p.carry -= float32(n)
		p.Emit(n)
	}
	keep := 1 - p.drag*dt
	p.live = slices.DeleteFunc(p.live, func(q particle) bool { return q.age+dt >= q.life })
	for i := range p.live {
		q := &p.live[i]
		q.age += dt
		q.speed = q.speed.Add(p.gravity.Scale(dt)).Scale(max(keep, 0))
		q.at = q.at.Add(q.speed.Scale(dt))
		q.rotation += q.twist * dt
	}
	if p.oneShot && !p.done && len(p.live) == 0 {
		p.done = true
		return true
	}
	return false
}

func (p *Particles) Draw(c *Canvas) {
	toLocal := geom.Identity2D()
	if !p.local {
		toLocal = p.Global().Inverse()
	}
	base := c.Transform()
	for _, q := range p.live {
		t := q.age / q.life
		col := p.start.Lerp(p.end, t)
		size := p.sizes.X + (p.sizes.Y-p.sizes.X)*t
		at := toLocal.Xform(q.at)
		if p.texture == nil || p.texture.Size() == (geom.Vec2{}) {
			c.Circle(at, size/2, col)
			continue
		}
		c.SetTransform(base.Mul(geom.Transform2DFrom(at, q.rotation, 0, geom.V2(1, 1))))
		c.Image(geom.Rect{X: -size / 2, Y: -size / 2, W: size, H: size}, p.texture, geom.Rect{W: 1, H: 1}, col)
	}
	c.SetTransform(base)
}

// emitDone tells the World a OneShot is done.
func emitDone(w *world.World, p *Particles) { world.Emit(w, ParticlesFinished{Particles: p}) }
