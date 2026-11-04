//go:build gpu

package gpu //nolint:testpackage // white-box: drives the renderer's unexported passes

import (
	"image"
	stdcolor "image/color"
	"math"
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/twod/light"
)

// rgbAt is img's pixel at x, y from the middle of shine's window.
func rgbAt(img *image.RGBA, x, y int) stdcolor.RGBA { return img.RGBAAt(100+x, 50+y) }

// srgb is linear light l as the window shows it, 0 to 255.
func srgb(l float32) uint8 {
	v := float64(l)
	if v <= 0.0031308 {
		v *= 12.92
	} else {
		v = 1.055*math.Pow(v, 1/2.4) - 0.055
	}
	return uint8(math.Round(v * 255))
}

// near reports whether a is within d of b.
//
//nolint:unparam // the tolerance is named at each call, beside what it bounds
func near(a, b uint8, d int) bool { return math.Abs(float64(int(a)-int(b))) <= float64(d) }

// shadowScene is a lamp left of a sprite casting its box, or its
// picture when silhouette, the lamp made with opts.
func shadowScene(t *testing.T, silhouette bool, opts ...light.Option) *image.RGBA {
	t.Helper()
	cast := light.Occluder()
	if silhouette {
		cast = light.Silhouette()
	}
	sprite := twod.NewSprite(filled(10, 20, stdcolor.RGBA{255, 255, 255, 255}), twod.At(0, 0), cast)
	return shine(
		t,
		light.NewPoint(120, color.White, append([]light.Option{light.Falloff(1), light.At(-60, 10)}, opts...)...),
		sprite,
	)
}

func TestBlackShadowsOfFullOpacityAreTheDefault(t *testing.T) {
	for _, silhouette := range []bool{false, true} {
		plain := shadowScene(t, silhouette)
		set := shadowScene(t, silhouette, light.ShadowColor(color.Black), light.ShadowOpacity(1))
		for i := range plain.Pix {
			if plain.Pix[i] != set.Pix[i] {
				t.Fatalf(
					"silhouette %v: pixel-identical to setting nothing, byte %d: %d and %d",
					silhouette,
					i,
					plain.Pix[i],
					set.Pix[i],
				)
			}
		}
	}
}

func TestShadowColourAndOpacity(t *testing.T) {
	blue := color.RGB(0.1, 0.2, 0.6)
	for _, silhouette := range []bool{false, true} {
		open := rgbAt(shadowScene(t, silhouette, light.Shadows(false)), 12, 7)
		if open.R < 250 {
			t.Fatalf("without shadows the spot is lit: %v", open)
		}
		if c := rgbAt(shadowScene(t, silhouette), 12, 7); c.R > 5 || c.B > 5 {
			t.Fatalf("silhouette %v: by default the shadow is black: %v", silhouette, c)
		}
		half := rgbAt(shadowScene(t, silhouette, light.ShadowOpacity(0.5)), 12, 7)
		if !near(half.R, srgb(0.5), 6) || half.R != half.B {
			t.Fatalf("silhouette %v: opacity 0.5 lets half the light through, grey: %v", silhouette, half)
		}
		tinted := rgbAt(shadowScene(t, silhouette, light.ShadowColor(blue)), 12, 7)
		if !near(tinted.R, srgb(0.1), 6) || !near(tinted.G, srgb(0.2), 6) || !near(tinted.B, srgb(0.6), 6) {
			t.Fatalf("silhouette %v: a coloured shadow lets the light through times its colour: %v", silhouette, tinted)
		}
		both := rgbAt(shadowScene(t, silhouette, light.ShadowColor(blue), light.ShadowOpacity(0.5)), 12, 7)
		if !near(both.R, srgb(0.55), 6) || !near(both.G, srgb(0.6), 6) || !near(both.B, srgb(0.8), 6) {
			t.Fatalf("silhouette %v: both, half the light and half of it tinted: %v", silhouette, both)
		}
		if c := rgbAt(shadowScene(t, silhouette, light.ShadowColor(blue)), -20, 7); c.R < 250 || c.B < 250 {
			t.Fatalf("silhouette %v: out of the shadow the light is whole: %v", silhouette, c)
		}
	}
}

func TestADirectionalLightsShadowsTakeItsShadowColour(t *testing.T) {
	blue := color.RGB(0.1, 0.2, 0.6)
	for _, silhouette := range []bool{false, true} {
		cast := light.Occluder()
		if silhouette {
			cast = light.Silhouette()
		}
		sprite := twod.NewSprite(filled(10, 20, stdcolor.RGBA{255, 255, 255, 255}), twod.At(0, -20), cast)
		img := shine(
			t,
			light.NewDirectional(color.White, math.Pi/2, light.ShadowLength(30), light.ShadowColor(blue)),
			sprite,
		)
		if c := rgbAt(img, 0, 0); !near(c.R, srgb(0.1), 6) || !near(c.B, srgb(0.6), 6) {
			t.Fatalf("silhouette %v: the sun's shadow is its colour: %v", silhouette, c)
		}
		if c := rgbAt(img, 20, 0); c.R < 250 {
			t.Fatalf("silhouette %v: beside it the sun shines whole: %v", silhouette, c)
		}
	}
}

func TestAShadedPictureTakesASilhouetteOutOnce(t *testing.T) {
	for _, reached := range []bool{false, true} {
		sprite := twod.NewSprite(filled(10, 20, stdcolor.RGBA{255, 255, 255, 255}), twod.At(0, 0), light.Silhouette())
		patch := twod.NewLayer(0, light.Lit(), light.ShadedBy(0, 1)).
			Add(twod.NewSprite(filled(20, 6, stdcolor.RGBA{255, 255, 255, 255}), twod.At(15, 4)))
		ents := []twod.Entity{sprite, patch}
		if reached {
			ents = append(
				ents,
				twod.NewSprite(filled(4, 4, stdcolor.RGBA{255, 255, 255, 255}), twod.At(-60, -30), light.Occluder(2)),
			)
		}
		img := shine(
			t,
			append(
				[]twod.Entity{
					light.NewPoint(120, color.White, light.Falloff(1), light.At(-60, 10), light.ShadowOpacity(0.5)),
				},
				ents...)...)
		if c := rgbAt(img, 12, 11); !near(c.R, srgb(0.5), 6) {
			t.Fatalf("reached %v: on the floor, the silhouette lets half through: %v", reached, c)
		}
		if c := rgbAt(img, 12, 6); !near(c.R, srgb(0.5), 6) {
			t.Fatalf(
				"reached %v: on a layer ShadedBy(0, 1), the silhouette of group 0 lets half through, as on the floor: %v",
				reached,
				c,
			)
		}
	}
}

func TestAShadedPictureAddsBackASilhouetteOfAGroupLeftOut(t *testing.T) {
	sprite := twod.NewSprite(filled(10, 20, stdcolor.RGBA{255, 255, 255, 255}), twod.At(0, 0), light.Silhouette(2))
	patch := twod.NewLayer(0, light.Lit(), light.ShadedBy(0, 1)).
		Add(twod.NewSprite(filled(20, 6, stdcolor.RGBA{255, 255, 255, 255}), twod.At(15, 4)))
	img := shine(
		t,
		light.NewPoint(120, color.White, light.Falloff(1), light.At(-60, 10), light.ShadowOpacity(0.5)),
		sprite,
		patch,
	)
	if c := rgbAt(img, 12, 11); !near(c.R, srgb(0.5), 6) {
		t.Fatalf("on the floor, the silhouette of group 2 lets half through: %v", c)
	}
	if c := rgbAt(img, 12, 6); c.R < 250 {
		t.Fatalf("a layer ShadedBy(0, 1) is never shaded by it: %v", c)
	}
}
