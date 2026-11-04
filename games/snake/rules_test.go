package main

import "testing"

func TestTheSnakeMovesAheadAndTurns(t *testing.T) {
	g := New(1)
	head := g.Head()
	g.Step()
	if g.Head() != (Cell{head.X + 1, head.Y}) {
		t.Fatalf("it starts heading right: %v then %v", head, g.Head())
	}
	g.Turn(Down)
	g.Step()
	if g.Head() != (Cell{head.X + 1, head.Y + 1}) {
		t.Fatalf("Down turns it: %v", g.Head())
	}
}

func TestItNeverTurnsBackOnItself(t *testing.T) {
	g := New(1)
	g.Turn(Left)
	g.Step()
	if g.Heading() != Right {
		t.Fatalf("heading right, Left is ignored: %v", g.Heading())
	}
}

func TestTwoQuickTurnsBothHappen(t *testing.T) {
	g := New(1)
	g.Turn(Up)
	g.Turn(Left)
	head := g.Head()
	g.Step()
	g.Step()
	if g.Head() != (Cell{head.X - 1, head.Y - 1}) {
		t.Fatalf("up then left, one a step: %v from %v", g.Head(), head)
	}
}

func TestEatingGrowsScoresAndMovesTheApple(t *testing.T) {
	g := New(1)
	head := g.Head()
	g.Apple = Cell{head.X + 1, head.Y}
	n := len(g.Body)
	g.Step()
	if len(g.Body) != n+1 || g.Score != 10 || g.Eaten != 1 {
		t.Fatalf("one longer, 10 points: len %d, score %d", len(g.Body), g.Score)
	}
	if g.Apple == g.Head() || g.on(g.Apple) {
		t.Fatalf("the new apple is off the snake: %v", g.Apple)
	}
}

func TestAWallOrItsOwnBodyEndsTheGame(t *testing.T) {
	g := New(1)
	for range W {
		g.Step()
	}
	if !g.Over {
		t.Fatal("the right wall ends it")
	}
	g = New(1)
	g.Body = []Cell{{5, 5}, {4, 5}, {4, 6}, {5, 6}, {6, 6}}
	g.Turn(Down)
	g.Step()
	if !g.Over {
		t.Fatal("running into itself ends it")
	}
}

func TestItSpeedsUpAsItEats(t *testing.T) {
	g := New(1)
	slow := g.Interval()
	g.Eaten = 10
	if g.Interval() >= slow {
		t.Fatalf("faster after ten apples: %v then %v", slow, g.Interval())
	}
}
