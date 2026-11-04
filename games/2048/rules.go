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

// Move pushes every line towards d and reports whether anything moved.
func (b *Board) Move(d Dir) bool {
	moved := false
	for i := range 4 {
		var line [4]int
		for j := range 4 {
			line[j] = *b.at(d, i, j)
		}
		slid, gained := slide(line)
		if slid != line {
			moved = true
		}
		b.Score += gained
		for j := range 4 {
			*b.at(d, i, j) = slid[j]
		}
	}
	if moved {
		b.add()
	}
	return moved
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

// at is the j-th cell of line i, counted from the side d pushes towards.
func (b *Board) at(d Dir, i, j int) *int {
	switch d {
	case Right:
		return &b.Cells[i][3-j]
	case Up:
		return &b.Cells[j][i]
	case Down:
		return &b.Cells[3-j][i]
	case Left:
	}
	return &b.Cells[i][j]
}

func (b *Board) add() {
	var empty [][2]int
	for r := range 4 {
		for c := range 4 {
			if b.Cells[r][c] == 0 {
				empty = append(empty, [2]int{r, c})
			}
		}
	}
	if len(empty) == 0 {
		return
	}
	at := empty[b.rng.IntN(len(empty))]
	b.Cells[at[0]][at[1]] = 2
	if b.rng.IntN(10) == 0 {
		b.Cells[at[0]][at[1]] = 4
	}
}

// slide pushes a line towards its start, merging each pair once, and
// returns it with the points the merges made.
func slide(line [4]int) ([4]int, int) {
	tiles := make([]int, 0, 4)
	gained, open := 0, false // open: the last tile placed may still merge
	for _, v := range line {
		if v == 0 {
			continue
		}
		if last := len(tiles) - 1; open && tiles[last] == v {
			tiles[last] *= 2
			gained += tiles[last]
			open = false
			continue
		}
		tiles = append(tiles, v)
		open = true
	}
	var out [4]int
	copy(out[:], tiles)
	return out, gained
}
