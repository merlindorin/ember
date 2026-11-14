package main

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
)

func TestASlimeLurksInTheDarkAndShowsInTheLight(t *testing.T) {
	if !lurks(color.RGB(0.1, 0.2, 0.1), false) {
		t.Fatal("below a quarter, a slime lurks")
	}
	if lurks(color.RGB(0.6, 0.4, 0.2), true) {
		t.Fatal("in the torchlight, it shows")
	}
}

func TestASlimeDoesNotBlinkInAFlickeringLight(t *testing.T) {
	between := color.RGB(0.26, 0.2, 0.1)
	if !lurks(between, true) || lurks(between, false) {
		t.Fatal("between 25% and 27%, a slime stays as it was")
	}
}

func TestASlimeStartsLurking(t *testing.T) {
	s := newSlime(geom.V2(0, 0))
	if !s.lurking || !s.body.Hidden() || s.eyes.Hidden() {
		t.Fatal("until its probe answers, only a slime's eyes show")
	}
}
