package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

func TestTheArcadeFontMeasuresItsCellsAndTheScoreCounts(t *testing.T) {
	arcade, err := twod.LoadBitmapFont(files, "arcade.png", twod.Cell(6, 8), twod.Charset(" 0123456789SCOREHILV"))
	if err != nil {
		t.Fatal(err)
	}
	score := twod.NewBitmapText("SCORE 000000", twod.Glyphs(arcade), twod.Scale(5, 5))
	if size := score.Measure(); size.Y == 0 || size.X < 12*5 {
		t.Fatalf("twelve characters of a 6-pixel-wide font: %v", size)
	}

	w := ember.New().AddEntities(score).AddSystems(&Score{text: score})
	w.RunFrames(60)
	if score.Text() != "SCORE 000250" {
		t.Fatalf("a second is 250 points: %q", score.Text())
	}
	if arcade.Measure("HI") == (geom.Vec2{}) {
		t.Fatal("the font measures the characters it has")
	}
}
