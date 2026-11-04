package main

import "math/rand/v2"

// The world, in points: the bird stays at BirdX while the pipes come.
const (
	Width, Height = 340, 400
	Ground        = 40 // the ground's height at the bottom
	BirdX         = 90
	Radius        = 12 // the bird's, for collisions
	PipeWidth     = 52
	GapHeight     = 120
	Spacing       = 190 // between two pipes
	Margin        = 50  // a gap's least distance from the sky and the ground
	Speed         = 120 // how fast the pipes come, points a second
	Gravity       = 900
	Lift          = -300 // a flap's speed
	Tick          = float32(1.0 / 120)
)

// A Pipe is a pair: from the sky down to Gap, and from Gap+GapHeight to the
// ground.
type Pipe struct {
	X, Gap float32
	passed bool
}

// Game is the rules: the bird falls, a flap lifts it, pipes come at a steady
// speed, each passed is a point, and touching one, the ground or the sky
// ends it. Step advances one Tick, so a run plays the same at any frame rate.
type Game struct {
	Y, VY  float32
	Pipes  []Pipe
	Score  int
	Over   bool
	Travel float32 // how far the world has come, for drawing it
	next   float32 // the distance to the next pipe
	rng    *rand.Rand
}

func New(seed uint64) *Game {
	return &Game{Y: (Height - Ground) / 2, next: Width - BirdX + 60,
		rng: rand.New(rand.NewPCG(seed, seed))} //nolint:gosec // a game's randomness, not a secret
}

// Flap lifts the bird.
func (g *Game) Flap() {
	if !g.Over {
		g.VY = Lift
	}
}

// Step lets one Tick pass.
func (g *Game) Step() {
	if g.Over {
		return
	}
	g.VY += Gravity * Tick
	g.Y += g.VY * Tick
	move := Speed * Tick
	g.Travel += move
	g.next -= move
	if g.next <= 0 {
		g.Pipes = append(g.Pipes, Pipe{X: Width, Gap: Margin + g.rng.Float32()*(Height-Ground-2*Margin-GapHeight)})
		g.next += Spacing
	}
	kept := g.Pipes[:0]
	for _, p := range g.Pipes {
		p.X -= move
		if !p.passed && p.X+PipeWidth < BirdX-Radius {
			p.passed = true
			g.Score++
		}
		if p.X+PipeWidth > 0 {
			kept = append(kept, p)
		}
	}
	g.Pipes = kept
	g.Over = g.hit()
}

func (g *Game) hit() bool {
	if g.Y-Radius < 0 || g.Y+Radius > Height-Ground {
		if g.Y+Radius > Height-Ground {
			g.Y = Height - Ground - Radius
		}
		return true
	}
	for _, p := range g.Pipes {
		if BirdX+Radius > p.X && BirdX-Radius < p.X+PipeWidth && (g.Y-Radius < p.Gap || g.Y+Radius > p.Gap+GapHeight) {
			return true
		}
	}
	return false
}
