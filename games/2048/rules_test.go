package main

import "testing"

func board(rows [4][4]int) *Board {
	b := New(1)
	b.Cells = rows
	b.Score = 0
	return b
}

func TestALineSlidesAndEachPairMergesOnce(t *testing.T) {
	for _, c := range []struct{ in, out [4]int }{
		{[4]int{2, 2, 0, 0}, [4]int{4, 0, 0, 0}},
		{[4]int{2, 2, 2, 2}, [4]int{4, 4, 0, 0}},
		{[4]int{4, 4, 8, 0}, [4]int{8, 8, 0, 0}},
		{[4]int{0, 2, 0, 2}, [4]int{4, 0, 0, 0}},
		{[4]int{2, 4, 2, 4}, [4]int{2, 4, 2, 4}},
	} {
		if got, _ := slide(c.in); got != c.out {
			t.Errorf("%v slides to %v, want %v", c.in, got, c.out)
		}
	}
}

func TestAMoveScoresItsMergesAndAddsATile(t *testing.T) {
	b := board([4][4]int{{2, 2, 0, 0}, {4, 0, 4, 0}})
	if !b.Move(Left) {
		t.Fatal("a move that merges is a move")
	}
	if b.Cells[0][0] != 4 || b.Cells[1][0] != 8 || b.Score != 12 {
		t.Fatalf("cells %v, score %d", b.Cells, b.Score)
	}
	if n := tiles(b); n != 3 {
		t.Fatalf("one new tile after the move: %d tiles", n)
	}
}

func TestAMoveThatChangesNothingAddsNothing(t *testing.T) {
	b := board([4][4]int{{2, 4, 0, 0}})
	if b.Move(Left) || tiles(b) != 2 {
		t.Fatalf("nothing moved, nothing added: %v", b.Cells)
	}
}

func TestEveryDirectionMovesTowardsIt(t *testing.T) {
	for d, want := range map[Dir][2]int{Left: {0, 0}, Right: {0, 3}, Up: {0, 1}, Down: {3, 1}} {
		b := board([4][4]int{{0, 2}})
		b.Move(d)
		if b.Cells[want[0]][want[1]] != 2 {
			t.Errorf("%v: the 2 is not at %v: %v", d, want, b.Cells)
		}
	}
}

func TestTheGameIsOverWhenNothingCanMove(t *testing.T) {
	full := board([4][4]int{{2, 4, 2, 4}, {4, 2, 4, 2}, {2, 4, 2, 4}, {4, 2, 4, 2}})
	if !full.Over() {
		t.Fatal("a full board without pairs is over")
	}
	pair := board([4][4]int{{2, 2, 2, 4}, {4, 2, 4, 2}, {2, 4, 2, 4}, {4, 2, 4, 2}})
	if pair.Over() {
		t.Fatal("a full board with a pair can still move")
	}
}

func TestANewGameStartsWithTwoTiles(t *testing.T) {
	if n := tiles(New(7)); n != 2 {
		t.Fatalf("%d tiles", n)
	}
}

func tiles(b *Board) int {
	n := 0
	for _, row := range b.Cells {
		for _, v := range row {
			if v != 0 {
				n++
			}
		}
	}
	return n
}
