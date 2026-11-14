package main

import (
	"flag"
	"math/rand/v2"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/collide"
	"github.com/lantervale/ember/collide/tiles"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/examples/internal/crypt"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/twod/light"
	"github.com/lantervale/ember/world"
)

// lightFlags are the lights scene's command-line flags.
type lightFlags struct {
	count       int
	noShadows   bool
	radius      float64
	occluders   int
	occKind     string
	zoom        float64
	probes      int
	static      bool
	shaded      bool
	silhouettes bool
	affect      bool
}

// newLightFlags registers the lights scene's flags; they hold their
// values once flag.Parse has run.
func newLightFlags() *lightFlags {
	o := &lightFlags{}
	flag.IntVar(&o.count, "lights", 16, "lights in the lights scene; 0 turns lighting off")
	flag.BoolVar(&o.noShadows, "noshadows", false, "make every light of the lights scene cast no shadows")
	flag.BoolVar(&o.noShadows, "xray", false, "old name of -noshadows, removed in the next release")
	flag.Float64Var(&o.radius, "radius", 250, "the lights' radius in points")
	flag.IntVar(
		&o.occluders,
		"occluders",
		0,
		"solid objects in the lights scene, instead of a column every fourth cell",
	)
	flag.StringVar(
		&o.occKind,
		"occkind",
		"cells",
		"cells (solid tiles of a collide space) or sprites (Occluder sprites, half of them moving)",
	)
	flag.Float64Var(&o.zoom, "zoom", 1, "the camera's zoom in the lights scene; below 1 shows more of the world")
	flag.IntVar(&o.probes, "probes", 0, "light probes over the lights scene's view, asking the light where they stand")
	flag.BoolVar(&o.static, "static", false, "make every light of the lights scene static: baked once, standing still")
	flag.BoolVar(
		&o.shaded,
		"shaded",
		false,
		"make the lights scene's crowd ShadedBy(0, 1): a second light picture a frame",
	)
	flag.BoolVar(
		&o.silhouettes,
		"silhouette",
		false,
		"make the lights scene's Occluder sprites (-occkind sprites) Silhouettes",
	)
	flag.BoolVar(
		&o.affect,
		"affects",
		false,
		"make every fourth light Affect the floor alone, and the next the crowd alone: two more light pictures a frame",
	)
	return o
}

// lights is a stone floor with a solid column every fourth cell, 1,000
// flasks on the lit layer, and -lights point lights wandering over the
// view, casting shadows from the columns unless -noshadows.
func lights(win *platform.Window, width, height int, o *lightFlags) *world.World {
	r := rand.New(rand.NewPCG(3, 4)) //nolint:gosec // a game's randomness, not a secret
	zoom := float32(o.zoom)
	areaW, areaH := float32(width)/zoom, float32(height)/zoom
	cols, rows := int(areaW)/32+2, int(areaH)/32+2
	if o.occluders > 0 && o.occKind == "cells" {
		for cols*rows/2 < o.occluders {
			cols, rows = cols*5/4+1, rows*5/4+1
		}
	}
	floorCells, columnCells := grid(o, r, cols, rows)
	on := o.count > 0
	lit := func() []twod.LayerOption {
		if on {
			return []twod.LayerOption{light.Lit()}
		}
		return nil
	}
	floor := twod.NewLayer(0, lit()...)
	crowd := twod.NewLayer(1, append(lit(), twod.SortByY())...)
	if o.shaded {
		light.ShadedBy(0, 1)(crowd)
	}
	floor.Add(twod.NewTileMap(crypt.Floor(), twod.Cells(floorCells), twod.CellSize(32, 32)))
	columns := twod.NewTileMap(crypt.Things(), twod.Cells(columnCells), twod.CellSize(32, 32), twod.TallIn(crowd))
	floor.Add(columns)

	flasks := []*twod.Texture{crypt.Sprite("flask_green"), crypt.Sprite("flask_yellow"), crypt.Sprite("flask_red")}
	for i := range 1000 {
		crowd.Add(twod.NewSprite(flasks[i%3], twod.At(r.Float32()*areaW, r.Float32()*areaH)))
	}
	movers := occluderSprites(o, r, crowd, areaW, areaH)

	type wanderer struct {
		l   *light.Light
		vel geom.Vec2
	}
	var wanderers []wanderer
	for i := range o.count {
		opts := lightOptions(o, i, floor, crowd, light.At(r.Float32()*areaW, r.Float32()*areaH))
		c := color.HSV(float32(i)*47, 0.4, 0.6)
		l := light.NewPoint(float32(o.radius), c, opts...)
		crowd.Add(l)
		vel := geom.FromAngle(r.Float32() * 2 * geom.Pi).Scale(40 + r.Float32()*60)
		if !o.static {
			wanderers = append(wanderers, wanderer{l, vel})
		}
	}

	for range o.probes {
		floor.Add(light.NewProbe(twod.At(r.Float32()*areaW, r.Float32()*areaH)))
	}

	space := collide.NewSpace()
	space.AddSolid(tiles.Of(columns))
	camOpts := []twod.CameraOption{twod.At(areaW/2, areaH/2), twod.Zoom(zoom)}
	if on {
		camOpts = append(camOpts, light.Ambient(color.RGB(0.06, 0.06, 0.08)), light.OccludersFrom(space))
	}
	win.Add(floor, crowd, twod.NewCamera(camOpts...))

	w := ember.New().AddEntities(win)
	w.On(world.Update, "wander", func(f world.Frame) {
		for i := range wanderers {
			wd := &wanderers[i]
			p := wd.l.Position().Add(wd.vel.Scale(f.DT))
			if p.X < 0 || p.X > areaW {
				wd.vel.X = -wd.vel.X
			}
			if p.Y < 0 || p.Y > areaH {
				wd.vel.Y = -wd.vel.Y
			}
			wd.l.SetPosition(p)
		}
		for _, m := range movers {
			m.s.Translate(m.vel.Scale(f.DT))
		}
	})
	return w
}

// grid is the floor's cells, stone in eight variants, and the cells of
// the columns: one every fourth cell, or with -occluders -occkind cells,
// that many in a checkerboard from the top.
func grid(o *lightFlags, r *rand.Rand, cols, rows int) ([][]int, [][]int) {
	floor := make([][]int, rows)
	columns := make([][]int, rows)
	for y := range rows {
		floor[y] = make([]int, cols)
		columns[y] = make([]int, cols)
		for x := range cols {
			floor[y][x] = crypt.Stone + r.IntN(8)
			if o.occluders == 0 && x%4 == 2 && y%4 == 2 {
				columns[y][x] = crypt.Column
			}
		}
	}
	if o.occluders > 0 && o.occKind == "cells" {
		placed := 0
		for y := 0; y < rows && placed < o.occluders; y++ {
			for x := (y % 2); x < cols && placed < o.occluders; x += 2 {
				columns[y][x] = crypt.Column
				placed++
			}
		}
	}
	return floor, columns
}

// lightOptions are the options of the scene's light i, standing at at.
func lightOptions(o *lightFlags, i int, floor, crowd *twod.Layer, at light.Option) []light.Option {
	opts := []light.Option{at, light.Falloff(0.3)}
	if o.noShadows {
		opts = append(opts, light.Shadows(false))
	}
	if o.static {
		opts = append(opts, light.Static())
	}
	if o.affect {
		switch i % 4 {
		case 0:
			opts = append(opts, light.Affects(floor))
		case 1:
			opts = append(opts, light.Affects(crowd))
		}
	}
	return opts
}

// mover is an occluder sprite and its velocity.
type mover struct {
	s   *twod.Sprite
	vel geom.Vec2
}

// occluderSprites adds -occluders Occluder sprites to crowd with -occkind
// sprites, and returns the half of them that move.
func occluderSprites(o *lightFlags, r *rand.Rand, crowd *twod.Layer, areaW, areaH float32) []mover {
	if o.occluders == 0 || o.occKind != "sprites" {
		return nil
	}
	var movers []mover
	box := crypt.Sprite("flask_red")
	for i := range o.occluders {
		s := twod.NewSprite(box, twod.Size(12, 12), twod.At(r.Float32()*areaW, r.Float32()*areaH), light.Occluder())
		if o.silhouettes {
			light.Silhouette()(s)
		}
		crowd.Add(s)
		if i%2 == 0 {
			movers = append(movers, mover{s, geom.FromAngle(r.Float32() * 2 * geom.Pi).Scale(30)})
		}
	}
	return movers
}
