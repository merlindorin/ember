package main

import "math/rand/v2"

// Dir is where a move pushes the tiles.
type Dir int

const (
	Left Dir = iota
	Right
	Up
	Down
)

// Board is 2048's rules: tiles slide towards a side, two equal tiles meeting
// merge into their sum, and each move that changes the board adds a 2 (or,
// one time in ten, a 4).
type Board struct {
	Cells [4][4]int // [row][column], 0 is empty
	Score int
	rng   *rand.Rand
}

func New(seed uint64) *Board {
	b := &Board{rng: rand.New(rand.NewPCG(seed, seed))} //nolint:gosec // a game's randomness, not a secret
	b.add()
	b.add()
	return b
}

// A Slide is one tile's way in a move: From and To are {row, column}.
type Slide struct {
	From, To [2]int
	Value    int
}

// Moved is what a move did, for drawing it: every tile's slide, the cells
// whose tile is a merge, and where the new tile came.
type Moved struct {
	Slides []Slide
	Merged [][2]int
	Added  [2]int
}

// Move pushes every line towards d and reports what moved, if anything did.
func (b *Board) Move(d Dir) (Moved, bool) {
	var moved Moved
	changed := false
	for i := range 4 {
		var in [4]int
		for j := range 4 {
			in[j] = b.cell(at(d, i, j))
		}
		l := slide(in)
		changed = changed || l.out != in
		b.Score += l.gained
		for j := range 4 {
			if in[j] != 0 {
				moved.Slides = append(moved.Slides, Slide{From: at(d, i, j), To: at(d, i, l.dest[j]), Value: in[j]})
			}
			if l.merged[j] {
				moved.Merged = append(moved.Merged, at(d, i, j))
			}
			to := at(d, i, j)
			b.Cells[to[0]][to[1]] = l.out[j]
		}
	}
	if !changed {
		return Moved{}, false
	}
	moved.Added = b.add()
	return moved, true
}

// Over reports whether no move is left: the board is full and no two
// neighbours are equal.
func (b *Board) Over() bool {
	for r := range 4 {
		for c := range 4 {
			v := b.Cells[r][c]
			if v == 0 || c < 3 && v == b.Cells[r][c+1] || r < 3 && v == b.Cells[r+1][c] {
				return false
			}
		}
	}
	return true
}

// Best is the largest tile.
func (b *Board) Best() int {
	best := 0
	for _, row := range b.Cells {
		for _, v := range row {
			best = max(best, v)
		}
	}
	return best
}

// at is the {row, column} of the j-th cell of line i, counted from the side
// d pushes towards.
func at(d Dir, i, j int) [2]int {
	switch d {
	case Right:
		return [2]int{i, 3 - j}
	case Up:
		return [2]int{j, i}
	case Down:
		return [2]int{3 - j, i}
	case Left:
	}
	return [2]int{i, j}
}

func (b *Board) cell(at [2]int) int { return b.Cells[at[0]][at[1]] }

// add puts a 2 (or, one time in ten, a 4) on an empty cell and returns it.
func (b *Board) add() [2]int {
	var empty [][2]int
	for r := range 4 {
		for c := range 4 {
			if b.Cells[r][c] == 0 {
				empty = append(empty, [2]int{r, c})
			}
		}
	}
	if len(empty) == 0 {
		return [2]int{-1, -1}
	}
	at := empty[b.rng.IntN(len(empty))]
	b.Cells[at[0]][at[1]] = 2
	if b.rng.IntN(10) == 0 {
		b.Cells[at[0]][at[1]] = 4
	}
	return at
}

// line is a line after a slide: its cells, the points its merges made, the
// index each tile went to, and which cells hold a merge.
type line struct {
	out, dest [4]int
	gained    int
	merged    [4]bool
}

// slide pushes a line towards its start, merging each pair once.
func slide(in [4]int) line {
	var l line
	n, open := 0, false // open: the last tile placed may still merge
	for j, v := range in {
		switch {
		case v == 0:
		case open && l.out[n-1] == v:
			l.out[n-1] *= 2
			l.gained += l.out[n-1]
			l.merged[n-1] = true
			l.dest[j] = n - 1
			open = false
		default:
			l.out[n] = v
			l.dest[j] = n
			n++
			open = true
		}
	}
	return l
}
