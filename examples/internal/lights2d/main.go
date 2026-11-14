// Lights2d is a night scene using every kind of 2D light, for the golden
// check (gpu's TestGolden): a hero walking between pillars with a cone of
// soft-shadowed light, an x-ray point light, a dim sun casting long
// shadows from the pillars and the walls, a rectangle and a line of light,
// and an emissive block glowing round it.
//
//	go run ./examples/internal/lights2d
package main

import (
	"image"
	stdcolor "image/color"
	"log"
	"math"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/collide"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/twod/light"
	"github.com/lantervale/ember/world"
)

type walls []geom.Rect

func (ws walls) SolidIn(geom.Rect) []geom.Rect { return ws }

func checker() *twod.Texture {
	img := image.NewRGBA(image.Rect(0, 0, 640, 384))
	for y := range 384 {
		for x := range 640 {
			c := stdcolor.RGBA{150, 140, 120, 255}
			if (x/32+y/32)%2 == 0 {
				c = stdcolor.RGBA{120, 112, 96, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return twod.NewTexture(img)
}

func block(w, h int, c stdcolor.RGBA) *twod.Texture {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return twod.NewTexture(img)
}

type walker struct {
	hero *twod.Sprite
	t    float32
}

func (s *walker) Update(f world.Frame) {
	s.t += f.DT
	s.hero.SetPosition(geom.V2(float32(math.Sin(float64(s.t)*0.7))*240, float32(math.Sin(float64(s.t)*1.3))*110))
}

func main() {
	win := platform.NewWindow(platform.Title("lights"), platform.Size(640, 384), platform.FixedSize())
	space := collide.NewSpace()
	space.AddSolid(walls{{X: -320, Y: -192, W: 640, H: 16}, {X: -320, Y: 176, W: 640, H: 16}})
	win.Add(twod.NewCamera(light.Ambient(color.RGB(0.06, 0.06, 0.1)), light.OccludersFrom(space)))
	world := twod.NewLayer(0, light.Lit()).Add(twod.NewSprite(checker()))
	for _, x := range []float32{-160, -60, 60, 160} {
		for _, y := range []float32{-70, 70} {
			world.Add(
				twod.NewSprite(
					block(24, 24, stdcolor.RGBA{90, 80, 70, 255}),
					twod.At(x, y),
					light.Occluder(),
					light.LitAt(0, 12),
				),
			)
		}
	}
	for _, y := range []float32{-184, 184} {
		world.Add(twod.NewSprite(block(640, 16, stdcolor.RGBA{70, 60, 55, 255}), twod.At(0, y)))
	}
	hero := twod.NewSprite(block(12, 16, stdcolor.RGBA{230, 200, 120, 255}), light.Unlit())
	hero.Add(light.NewCone(260, color.RGB(1, 0.85, 0.6), 0, 0.9, light.Falloff(0.2), light.Size(30)))
	world.Add(hero, light.NewPoint(120, color.RGB(0.4, 0.6, 1), light.At(250, -120), light.Shadows(false)))
	world.Add(light.NewDirectional(color.RGB(0.12, 0.1, 0.2), 1.0, light.ShadowLength(80)))
	world.Add(light.NewShape(light.Rect(80, 30), color.RGB(0.3, 1, 0.5), 50, light.At(-230, 120)))
	world.Add(light.NewShape(light.Line(geom.V2(-280, -150), geom.V2(-120, -150)), color.RGB(1, 0.2, 0.6), 40))
	world.Add(
		twod.NewSprite(
			block(24, 24, stdcolor.RGBA{255, 120, 40, 255}),
			twod.At(0, 0),
			light.Emissive(40),
			light.Unlit(),
		),
	)
	win.Add(world)
	w := ember.New()
	w.AddEntities(win).AddSystems(&walker{hero: hero})
	if err := w.Run(); err != nil {
		log.Fatal(err)
	}
}
