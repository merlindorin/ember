package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParticlesEmitAtTheirRateAndDie(t *testing.T) {
	p := NewParticles(nil, Rate(10), Lifetime(1, 1), Seed(1))
	for range 50 {
		p.step(0.01)
	}
	assert.Equal(t, 5, p.Live(), "half a second at 10 a second")
	for range 100 {
		p.step(0.01)
	}
	assert.Equal(t, 10, p.Live(), "a second's worth alive once the first die")
	p.SetEmitting(false)
	for range 101 {
		p.step(0.01)
	}
	assert.Zero(t, p.Live(), "paused, the live ones die")
}

func TestTheSameSeedGivesTheSameParticles(t *testing.T) {
	run := func(seed uint64) []particle {
		p := NewParticles(nil, Rate(30), Velocity(20, 80), Direction(0, 1), Spin(-2, 2), Seed(seed))
		for range 20 {
			p.step(1.0 / 60)
		}
		return append([]particle(nil), p.live...)
	}
	assert.Equal(t, run(7), run(7))
	assert.NotEqual(t, run(7), run(8))
	assert.NotEqual(t, NewParticles(nil).seed, NewParticles(nil).seed, "each emitter its own sequence by default")
}

func TestAOneShotBurstsThenIsDone(t *testing.T) {
	w := world.New().AddPlugins(Plugin{})
	var done world.Reader[ParticlesFinished]
	p := NewParticles(nil, Burst(12), OneShot(), Lifetime(0.5, 0.5), Rate(100))
	w.AddEntities(p)
	w.Tick(0.1)
	assert.Equal(t, 12, p.Live(), "the burst, and no rate")
	var events []ParticlesFinished
	for range 10 {
		w.Tick(0.1)
		for ev := range done.Read(w) {
			events = append(events, ev)
		}
	}
	assert.True(t, p.Done())
	require.Len(t, events, 1, "said once")
	assert.Equal(t, p, events[0].Particles)
}

func TestGravityDragAndTheCap(t *testing.T) {
	p := NewParticles(
		nil,
		Rate(0),
		Velocity(100, 100),
		Direction(0, 0),
		Gravity(geom.V2(0, 100)),
		Drag(0.5),
		MaxParticles(3),
		Lifetime(10, 10),
	)
	p.Emit(5)
	require.Equal(t, 3, p.Live(), "capped")
	p.step(0.1)
	q := p.live[0]
	assert.InDelta(t, 95, q.speed.X, 1e-3, "half its speed lost a second, a tenth of a second")
	assert.InDelta(t, 9.5, q.speed.Y, 1e-3, "gravity pulls it down")
	assert.InDelta(t, 9.5, q.at.X, 1e-3)
}

func TestParticlesStayWhereTheyWereEmittedUnlessLocal(t *testing.T) {
	world := NewParticles(nil, Rate(0), Velocity(0, 0), At(100, 50))
	local := NewParticles(nil, Rate(0), Velocity(0, 0), At(100, 50), LocalSpace())
	for _, p := range []*Particles{world, local} {
		p.Emit(1)
		p.SetPosition(geom.V2(300, 50))
	}
	assert.Equal(t, geom.V2(100, 50), world.live[0].at, "in the layer, where the emitter was")
	assert.Equal(t, geom.Vec2{}, local.live[0].at, "on the emitter, wherever it goes")
}

func TestParticlesFadeGrowAndDrawBatched(t *testing.T) {
	w, win, c := app()
	p := NewParticles(atlas(4, 4), Rate(0), Burst(3), Velocity(0, 0), Lifetime(1, 1), At(200, 100),
		ColorOverLife(color.White, color.RGBA(1, 1, 1, 0)), SizeOverLife(10, 2), Blend(Add))
	win.Add(p)
	w.AddEntities(p)
	for range 15 {
		w.Tick(1.0 / 30)
	}
	require.Len(t, c.Unindexed(), 18, "three textured quads")
	require.Len(t, c.Batches, 1, "in one batch")
	assert.Equal(t, draw.Add, c.Batches[0].Blend)
	lo, hi := corners(c.Unindexed()[:6])
	assert.Equal(t, geom.V2(197, 97), lo, "halfway: 6 points wide, centred where it was emitted")
	assert.Equal(t, geom.V2(203, 103), hi)
	assert.InDelta(t, 0.5, c.Unindexed()[0].Color.A, 1e-3, "halfway faded")
}
