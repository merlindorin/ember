package main

import "testing"

func run(g *Game, seconds float32) {
	for range int(seconds / Tick) {
		g.Step()
	}
}

func TestTheBirdFallsAndAFlapLiftsIt(t *testing.T) {
	g := New(1)
	y := g.Y
	run(g, 0.5)
	if g.Y <= y {
		t.Fatalf("gravity pulls it down: %v then %v", y, g.Y)
	}
	g.Flap()
	if g.VY >= 0 {
		t.Fatalf("a flap sends it up: vy %v", g.VY)
	}
}

func TestPipesComeAtTheirSpacingWithAGapInside(t *testing.T) {
	g := New(1)
	for range 720 { // 6 s
		g.Y, g.VY = 150, 0 // held in the air, whatever it touches
		g.Step()
		g.Over = false
	}
	if len(g.Pipes) < 2 {
		t.Fatalf("pipes keep coming: %d", len(g.Pipes))
	}
	for i := 1; i < len(g.Pipes); i++ {
		if d := g.Pipes[i].X - g.Pipes[i-1].X; d != Spacing {
			t.Fatalf("pipes %v apart, want %v", d, Spacing)
		}
	}
	for _, p := range g.Pipes {
		if p.Gap < Margin || p.Gap+GapHeight > Height-Margin {
			t.Fatalf("the gap stays off the edges: %+v", p)
		}
	}
}

func TestPassingAPipeScores(t *testing.T) {
	g := New(1)
	g.Pipes = []Pipe{{X: BirdX + 1, Gap: g.Y - GapHeight/2}}
	g.next = 1e9
	for range 120 {
		g.VY = 0
		g.Step()
	}
	if g.Score != 1 || g.Over {
		t.Fatalf("through the gap is a point: score %d, over %v", g.Score, g.Over)
	}
}

func TestAPipeTheGroundOrTheSkyEndsIt(t *testing.T) {
	g := New(1)
	g.Pipes = []Pipe{{X: BirdX + 10, Gap: Height - Margin - GapHeight}}
	g.next = 1e9
	for range 120 {
		g.VY = 0
		g.Step()
	}
	if !g.Over {
		t.Fatal("the pipe's top half stops it")
	}
	g = New(1)
	run(g, 5)
	if !g.Over || g.Y < Height-Ground-Radius-1 {
		t.Fatalf("without flapping it lands on the ground: over %v, y %v", g.Over, g.Y)
	}
	g = New(1)
	for range 200 {
		g.Flap()
		g.Step()
	}
	if !g.Over {
		t.Fatal("flapping without end hits the sky")
	}
}
