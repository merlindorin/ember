package main

import "math/rand/v2"

// The board's size, in cells.
const W, H = 18, 18

// A Cell is a place on the board.
type Cell struct{ X, Y int }

// A Dir is where the snake heads.
type Dir int

const (
	Right Dir = iota
	Down
	Left
	Up
)

func (d Dir) step() Cell { return [4]Cell{{1, 0}, {0, 1}, {-1, 0}, {0, -1}}[d] }

// Game is Snake's rules: the snake moves a cell a step, turns at most once
// a step, grows on an apple and dies on a wall or on itself.
type Game struct {
	Body  []Cell // the head first
	Apple Cell
	Score int
	Eaten int
	Over  bool
	dir   Dir
	turns []Dir // turns asked for, one taken a step
	rng   *rand.Rand
}

func New(seed uint64) *Game {
	g := &Game{rng: rand.New(rand.NewPCG(seed, seed))} //nolint:gosec // a game's randomness, not a secret
	for i := range 3 {
		g.Body = append(g.Body, Cell{W/2 - 3 - i, H / 2})
	}
	g.place()
	return g
}

func (g *Game) Head() Cell { return g.Body[0] }

// Heading is where the snake heads now.
func (g *Game) Heading() Dir { return g.dir }

// Interval is how long a step takes, in seconds: shorter as the snake eats.
func (g *Game) Interval() float32 { return max(0.14-0.005*float32(g.Eaten), 0.06) }

// Turn asks the snake to head d at a coming step. A turn back on itself, or
// along the way it already goes, is ignored; two can wait in line.
func (g *Game) Turn(d Dir) {
	last := g.dir
	if len(g.turns) > 0 {
		last = g.turns[len(g.turns)-1]
	}
	if d == last || d == (last+2)%4 || len(g.turns) == 2 {
		return
	}
	g.turns = append(g.turns, d)
}

// Step moves the snake a cell.
func (g *Game) Step() {
	if g.Over {
		return
	}
	if len(g.turns) > 0 {
		g.dir, g.turns = g.turns[0], g.turns[1:]
	}
	s := g.dir.step()
	head := Cell{g.Head().X + s.X, g.Head().Y + s.Y}
	eats := head == g.Apple
	tail := g.Body
	if !eats {
		tail = g.Body[:len(g.Body)-1]
	}
	if head.X < 0 || head.Y < 0 || head.X >= W || head.Y >= H || contains(tail, head) {
		g.Over = true
		return
	}
	g.Body = append([]Cell{head}, tail...)
	if eats {
		g.Eaten++
		g.Score += 10
		g.place()
	}
}

func (g *Game) on(c Cell) bool { return contains(g.Body, c) }

// place puts the apple on a free cell.
func (g *Game) place() {
	for {
		c := Cell{g.rng.IntN(W), g.rng.IntN(H)}
		if !g.on(c) {
			g.Apple = c
			return
		}
	}
}

func contains(cells []Cell, c Cell) bool {
	for _, b := range cells {
		if b == c {
			return true
		}
	}
	return false
}
