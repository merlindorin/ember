package main

import "math/rand/v2"

// The board's size, in cells.
const W, H = 10, 20

// The seven pieces; a cell of the board holds the kind that filled it, 0
// when empty.
const (
	I = iota + 1
	O
	T
	S
	Z
	J
	L
)

// shape is a piece's box size and its cells in the box, as it comes in.
func shape(kind int) (int, [4][2]int) {
	switch kind {
	case I:
		return 4, [4][2]int{{0, 1}, {1, 1}, {2, 1}, {3, 1}}
	case O:
		return 2, [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}}
	case T:
		return 3, [4][2]int{{1, 0}, {0, 1}, {1, 1}, {2, 1}}
	case S:
		return 3, [4][2]int{{1, 0}, {2, 0}, {0, 1}, {1, 1}}
	case Z:
		return 3, [4][2]int{{0, 0}, {1, 0}, {1, 1}, {2, 1}}
	case J:
		return 3, [4][2]int{{0, 0}, {0, 1}, {1, 1}, {2, 1}}
	default:
		return 3, [4][2]int{{2, 0}, {0, 1}, {1, 1}, {2, 1}}
	}
}

// softFall is how long a row takes while Down is held.
const softFall = 0.04

// Piece is the falling piece: its kind, its cells in its box, and where the
// box's top-left is on the board.
type Piece struct {
	Kind, X, Y int
	cells      [4][2]int
}

func spawn(kind int) Piece {
	box, cells := shape(kind)
	return Piece{Kind: kind, X: (W - box) / 2, cells: cells}
}

// Cells are the piece's cells on the board, as {x, y}.
func (p Piece) Cells() [4][2]int {
	var out [4][2]int
	for i, c := range p.cells {
		out[i] = [2]int{p.X + c[0], p.Y + c[1]}
	}
	return out
}

// Game is Tetris's rules: pieces from a bag of seven fall a row at a time,
// lock where they land, and full rows clear for points.
type Game struct {
	Board        [H][W]int // [row][column]
	Piece        Piece
	Next         int
	Score, Lines int
	Over         bool
	bag          []int
	rng          *rand.Rand
	falling      float32
}

func New(seed uint64) *Game {
	g := &Game{rng: rand.New(rand.NewPCG(seed, seed))} //nolint:gosec // a game's randomness, not a secret
	g.Piece = spawn(g.take())
	g.Next = g.take()
	return g
}

// Level rises every ten lines; the pieces fall faster with it.
func (g *Game) Level() int { return g.Lines/10 + 1 }

// fall is how long a row takes at a level, in seconds.
func fall(level int) float32 { return max(0.8-0.07*float32(level-1), 0.08) }

// Move shifts the piece by dx columns, if there is room.
func (g *Game) Move(dx int) { g.try(g.Piece.X+dx, g.Piece.Y, g.Piece.cells) }

// Rotate turns the piece a quarter clockwise, pushing it off a wall or the
// stack by up to two columns when it doesn't fit in place.
func (g *Game) Rotate() bool {
	box, _ := shape(g.Piece.Kind)
	var turned [4][2]int
	for i, c := range g.Piece.cells {
		turned[i] = [2]int{box - 1 - c[1], c[0]}
	}
	for _, kick := range []int{0, 1, -1, 2, -2} {
		if g.try(g.Piece.X+kick, g.Piece.Y, turned) {
			return true
		}
	}
	return false
}

// Drop sends the piece straight down and locks it: 2 points a row.
func (g *Game) Drop() {
	for g.try(g.Piece.X, g.Piece.Y+1, g.Piece.cells) {
		g.Score += 2
	}
	g.lock()
}

// Ghost is where the piece would land.
func (g *Game) Ghost() Piece {
	p := g.Piece
	for g.fits(p.X, p.Y+1, p.cells) {
		p.Y++
	}
	return p
}

// Step lets dt seconds pass: the piece falls a row each fall interval, or
// each softFall while soft (1 point a row), and locks when it can't.
func (g *Game) Step(dt float32, soft bool) {
	if g.Over {
		return
	}
	every := fall(g.Level())
	if soft {
		every = softFall
	}
	for g.falling += dt; g.falling >= every; g.falling -= every {
		if !g.try(g.Piece.X, g.Piece.Y+1, g.Piece.cells) {
			g.lock()
			return
		}
		if soft {
			g.Score++
		}
	}
}

func (g *Game) try(x, y int, cells [4][2]int) bool {
	if !g.fits(x, y, cells) {
		return false
	}
	g.Piece.X, g.Piece.Y, g.Piece.cells = x, y, cells
	return true
}

func (g *Game) fits(x, y int, cells [4][2]int) bool {
	for _, c := range cells {
		cx, cy := x+c[0], y+c[1]
		if cx < 0 || cx >= W || cy >= H || cy >= 0 && g.Board[cy][cx] != 0 {
			return false
		}
	}
	return true
}

func (g *Game) lock() {
	for _, c := range g.Piece.Cells() {
		if c[1] >= 0 {
			g.Board[c[1]][c[0]] = g.Piece.Kind
		}
	}
	g.clear()
	g.falling = 0
	g.Piece, g.Next = spawn(g.Next), g.take()
	if !g.fits(g.Piece.X, g.Piece.Y, g.Piece.cells) {
		g.Over = true
	}
}

// clear removes the full rows, the rows above falling in, and scores them:
// 100, 300, 500 or 800 times the level for one to four at once.
func (g *Game) clear() {
	cleared := 0
	for y := H - 1; y >= 0; y-- {
		full := true
		for _, v := range g.Board[y] {
			full = full && v != 0
		}
		if !full {
			continue
		}
		copy(g.Board[1:y+1], g.Board[:y])
		g.Board[0] = [W]int{}
		cleared++
		y++
	}
	g.Score += [5]int{0, 100, 300, 500, 800}[cleared] * g.Level()
	g.Lines += cleared
}

// take is the next kind from a shuffled bag of the seven.
func (g *Game) take() int {
	if len(g.bag) == 0 {
		g.bag = []int{I, O, T, S, Z, J, L}
		g.rng.Shuffle(len(g.bag), func(i, j int) { g.bag[i], g.bag[j] = g.bag[j], g.bag[i] })
	}
	k := g.bag[0]
	g.bag = g.bag[1:]
	return k
}
