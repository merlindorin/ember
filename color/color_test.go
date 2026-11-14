package color_test

import (
	"testing"

	"github.com/lantervale/ember/color"
)

func near(a, b float32) bool { return a-b < 1e-3 && b-a < 1e-3 }

func TestHex(t *testing.T) {
	c := color.Hex("#ff000080")
	if !near(c.R, 1) || c.G != 0 || c.B != 0 || !near(c.A, 128.0/255) {
		t.Fatalf("got %v", c)
	}
	if color.Hex("#fff") != color.White {
		t.Fatal("short form")
	}
	if _, err := color.ParseHex("#12"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSRGBMidGrayIsDarkerInLinear(t *testing.T) {
	if c := color.RGB8(128, 128, 128); !near(c.R, 0.2158) {
		t.Fatalf("got %v", c.R)
	}
}

func TestHSVPrimaries(t *testing.T) {
	for h, want := range map[float32]color.Color{0: color.Red, 120: color.Green, 240: color.Blue} {
		if got := color.HSV(h, 1, 1); !near(got.R, want.R) || !near(got.G, want.G) || !near(got.B, want.B) {
			t.Fatalf("hue %v: %v", h, got)
		}
	}
}

func TestScaleKeepsAlpha(t *testing.T) {
	if c := color.RGBA(1, 0.5, 0, 0.5).Scale(4); c != (color.Color{4, 2, 0, 0.5}) {
		t.Fatalf("got %v", c)
	}
}

func TestHexRoundTripsThroughSRGB(t *testing.T) {
	for _, h := range []string{"#fff2d6", "#000000", "#ff8800", "#123456"} {
		if got := color.Hex(h).Hex(); got != h {
			t.Fatalf("%s came back %s", h, got)
		}
	}
	c := color.Hex("#ff8800")
	if s := c.SRGB(); !near(s[0], 1) || !near(s[1], 0x88/255.0) || s[2] != 0 {
		t.Fatalf("sRGB %v", s)
	}
	if back := color.FromSRGB(c.SRGB()); !near(back.R, c.R) || !near(back.G, c.G) || !near(back.B, c.B) {
		t.Fatal("FromSRGB inverts SRGB")
	}
}
