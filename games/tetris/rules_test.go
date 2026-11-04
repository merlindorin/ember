package main

import "testing"

func game(kind int) *Game {
	g := New(1)
	g.Piece = spawn(kind)
	return g
}

func TestAPieceStopsAtTheWalls(t *testing.T) {
	g := game(O)
	for range W {
		g.Move(-1)
	}
	if g.Piece.X != 0 {
		t.Fatalf("O stops at the left wall: x %d", g.Piece.X)
	}
	for range W {
		g.Move(1)
	}
	if g.Piece.X != W-2 {
		t.Fatalf("O stops at the right wall: x %d", g.Piece.X)
	}
}

func TestRotatingTurnsTheCellsAQuarter(t *testing.T) {
	g := game(I)
	g.Rotate()
	xs := map[int]bool{}
	for _, c := range g.Piece.Cells() {
		xs[c[0]] = true
	}
	if len(xs) != 1 {
		t.Fatalf("an I turned stands upright: %v", g.Piece.Cells())
	}
}

func TestRotatingAgainstAWallKicksThePieceOff(t *testing.T) {
	g := game(I)
	g.Rotate()
	for range W {
		g.Move(-1)
	}
	if !g.Rotate() {
		t.Fatal("an upright I at the wall still turns")
	}
	for _, c := range g.Piece.Cells() {
		if c[0] < 0 {
			t.Fatalf("no cell past the wall: %v", g.Piece.Cells())
		}
	}
}

func TestAHardDropLandsLocksAndScores(t *testing.T) {
	g := game(O)
	g.Drop()
	if g.Board[H-1][4] == 0 || g.Board[H-2][5] == 0 {
		t.Fatalf("the O lies on the floor: %v", g.Board[H-2:])
	}
	if g.Score != 2*(H-2) {
		t.Fatalf("a hard drop scores 2 a row: %d", g.Score)
	}
	if g.Piece.Kind == 0 || g.Piece.Y != 0 {
		t.Fatalf("the next piece comes in at the top: %+v", g.Piece)
	}
}

func TestAFullRowClearsAndTheRestFalls(t *testing.T) {
	g := game(I)
	for x := range W - 4 {
		g.Board[H-1][x+4] = T
	}
	g.Board[H-2][9] = S
	for range W {
		g.Move(-1)
	}
	g.Drop()
	if g.Lines != 1 {
		t.Fatalf("one line: %d", g.Lines)
	}
	if g.Board[H-1][9] != S || g.Board[H-2][9] != 0 {
		t.Fatalf("the row above fell into the cleared one: %v", g.Board[H-2:])
	}
	if g.Score != 100+2*(H-2) {
		t.Fatalf("a line is 100 at level 1, plus the drop: %d", g.Score)
	}
}

func TestGravityStepsAndASoftDropIsFaster(t *testing.T) {
	g := game(T)
	g.Step(fall(1)+0.001, false)
	if g.Piece.Y != 1 {
		t.Fatalf("a fall interval moves it a row: y %d", g.Piece.Y)
	}
	g.Step(softFall+0.001, true)
	if g.Piece.Y != 2 || g.Score != 1 {
		t.Fatalf("a soft drop moves at its own pace and scores 1: y %d, score %d", g.Piece.Y, g.Score)
	}
}

func TestTheGameEndsWhenAPieceCannotComeIn(t *testing.T) {
	g := game(O)
	for y := 2; y < H; y++ {
		for x := range W - 1 {
			g.Board[y][x] = Z
		}
	}
	g.Drop()
	if !g.Over {
		t.Fatal("the O locks at the top, and the next piece has no room")
	}
}

func TestTheLevelRisesEveryTenLines(t *testing.T) {
	g := New(1)
	g.Lines = 19
	if g.Level() != 2 || fall(2) >= fall(1) {
		t.Fatalf("level %d, fall %v then %v", g.Level(), fall(1), fall(2))
	}
}

func TestEverySevenPiecesComeOnce(t *testing.T) {
	g := New(3)
	seen := map[int]bool{g.Piece.Kind: true, g.Next: true}
	for range 5 {
		seen[g.take()] = true
	}
	if len(seen) != 7 {
		t.Fatalf("a bag of seven: %v", seen)
	}
}
