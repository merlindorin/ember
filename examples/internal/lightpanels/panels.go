package main

import (
	"math"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/examples/internal/crypt"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/twod/light"
)

// panels are every picture of the Light pages, by name.
func panels() map[string]panel {
	all := map[string]panel{}
	sets := []map[string]panel{
		overview(), litLayers(), points(), shadows(), cones(), directionals(), shapes(), debugViews(), budgets(),
	}
	for _, set := range sets {
		for name, p := range set {
			if _, taken := all[name]; taken {
				panic("lightpanels: two panels named " + name)
			}
			all[name] = p
		}
	}
	return all
}

// overview builds the lit picture up, one part at a time.
func overview() map[string]panel {
	dim := rgb(0.18, 0.18, 0.22)
	return map[string]panel{
		"parts-ambient": {call: "light.Ambient(c)", at: behind(), ambient: dim, sprites: crate()},
		"parts-light": {
			call:    "+ light.NewPoint(r, c)",
			at:      behind(),
			ambient: dim,
			lights:  point(180, warm(), light.Shadows(false)),
			sprites: crate(),
		},
		"parts-shadows": {
			call:    "+ tiles.Solid(crate, column)",
			at:      behind(),
			ambient: dim,
			lights:  point(180, warm()),
			sprites: crate(),
		},
		"parts-unlit": {
			call:    "+ light.Unlit() on a crate",
			at:      behind(),
			ambient: dim,
			lights:  point(180, warm()),
			sprites: crate(light.Unlit()),
		},
	}
}

func litLayers() map[string]panel {
	return map[string]panel{
		"lit-off":   {call: "twod.NewLayer(-1)", lights: point(140, warm()), floorUnlit: true},
		"lit-on":    {call: "twod.NewLayer(-1, light.Lit())", lights: point(140, warm())},
		"unlit-off": {call: "crate lit", at: behind(), lights: point(180, warm()), sprites: crate()},
		"unlit-on": {
			call:    "crate: light.Unlit()",
			at:      behind(),
			lights:  point(180, warm()),
			sprites: crate(light.Unlit()),
		},
		"ambient-0": {call: "light.Ambient(black)", ambient: rgb(0, 0, 0), lights: point(140, warm())},
		"ambient-0.15": {
			call:    "light.Ambient(color.RGB(0.15, 0.15, 0.2))",
			ambient: rgb(0.15, 0.15, 0.2),
			lights:  point(140, warm()),
		},
		"ambient-0.4": {
			call:    "light.Ambient(color.RGB(0.4, 0.4, 0.45))",
			ambient: rgb(0.4, 0.4, 0.45),
			lights:  point(140, warm()),
		},
		"ambient-map":  {call: "light.AmbientMap(moonlit)", ambientMap: true, lights: point(100, warm())},
		"perpixel-off": {call: "tall tiles lit at their foot", at: geom.V2(208, 150), lights: point(130, warm())},
		"perpixel-on": {
			call:   "things: light.PerPixel()",
			at:     geom.V2(208, 150),
			lights: point(130, warm()),
			things: []twod.TileMapOption{light.PerPixel()},
		},
	}
}

func points() map[string]panel {
	return map[string]panel{
		"radius-80":  {call: "light.NewPoint(80, warm)", lights: point(80, warm())},
		"radius-140": {call: "light.NewPoint(140, warm)", lights: point(140, warm())},
		"radius-220": {call: "light.NewPoint(220, warm)", lights: point(220, warm())},
		"falloff-0": {
			call:    "light.Falloff(0)",
			lights:  point(140, warm(), light.Falloff(0)),
			plot:    true,
			falloff: 0,
		},
		"falloff-0.5": {
			call:    "light.Falloff(0.5)",
			lights:  point(140, warm(), light.Falloff(0.5)),
			plot:    true,
			falloff: 0.5,
		},
		"falloff-1": {
			call:    "light.Falloff(1)",
			lights:  point(140, warm(), light.Falloff(1)),
			plot:    true,
			falloff: 1,
		},
		"intensity-0.25":   {call: "light.Intensity(0.25)", lights: point(140, warm(), light.Intensity(0.25))},
		"intensity-1":      {call: "light.Intensity(1)", lights: point(140, warm())},
		"intensity-2":      {call: "light.Intensity(2)", lights: point(140, warm(), light.Intensity(2))},
		"color-warm":       {call: "color.RGB(1, 0.8, 0.55)", lights: point(140, warm())},
		"color-cold":       {call: "color.RGB(0.55, 0.75, 1)", lights: point(140, color.RGB(0.55, 0.75, 1))},
		"color-green":      {call: "color.RGB(0.5, 1, 0.55)", lights: point(140, color.RGB(0.5, 1, 0.55))},
		"point-shadows-on": {call: "light.NewPoint(180, warm)", at: behind(), lights: point(180, warm())},
		"point-shadows-off": {
			call:   "light.Shadows(false)",
			at:     behind(),
			lights: point(180, warm(), light.Shadows(false)),
		},
		"static-off": {
			call:    "light.NewPoint(180, warm); crate: light.Occluder()",
			lights:  point(180, warm()),
			sprites: crate(light.Occluder()),
		},
		"static-on": {
			call:    "light.Static()",
			lights:  point(180, warm(), light.Static()),
			sprites: crate(light.Occluder()),
		},
	}
}

func shadows() map[string]panel {
	far := point(220, warm(), light.Elevation(100))
	fading := func(opts ...light.Option) func(geom.Vec2) []*light.Light {
		return point(260, warm(), append([]light.Option{light.Elevation(112), light.Falloff(0.6)}, opts...)...)
	}
	return map[string]panel{
		"occluders-off": {call: "crates and columns not solid", lights: point(220, warm()), noOccluders: true},
		"occluders-on":  {call: "tiles.Solid(crate, column)", lights: point(220, warm())},
		"height-24":     {call: "light.TileHeight(24, crate)", lights: far},
		"height-8": {
			call:   "light.TileHeight(8, crate)",
			lights: far,
			tiles:  []twod.TilesetOption{light.TileHeight(8, crypt.Crate)},
		},
		"elevation-0":   {call: "light.Elevation(0)", lights: point(220, warm())},
		"elevation-100": {call: "light.Elevation(100)", lights: far},
		"elevation-200": {call: "light.Elevation(200)", lights: point(220, warm(), light.Elevation(200))},
		"size-0":        {call: "light.Size(0)", lights: far},
		"size-32":       {call: "light.Size(32)", lights: point(220, warm(), light.Elevation(100), light.Size(32))},
		"fade-off":      {at: behind(), call: "no ShadowFade", lights: fading()},
		"fade-linear":   {at: behind(), call: "light.ShadowFade()", lights: fading(light.ShadowFade())},
		"fade-smooth": {
			at:     behind(),
			call:   "light.ShadowFade(light.FadeSmooth)",
			lights: fading(light.ShadowFade(light.FadeSmooth)),
		},
		"fade-soft": {
			at:     behind(),
			call:   "light.ShadowFade(light.FadeSoft)",
			lights: fading(light.ShadowFade(light.FadeSoft)),
		},
		"fade-from": {
			at:     behind(),
			call:   "light.FadeFrom(0.5)",
			lights: fading(light.ShadowFade(), light.FadeFrom(0.5)),
		},
		"groups-all": {
			call:   "light.TileOccluderGroup(1, crate)",
			lights: point(220, warm()),
			tiles:  []twod.TilesetOption{light.TileOccluderGroup(1, crypt.Crate)},
		},
		"groups-blockedby": {
			call:   "light.BlockedBy(0)",
			lights: point(220, warm(), light.BlockedBy(0)),
			tiles:  []twod.TilesetOption{light.TileOccluderGroup(1, crypt.Crate)},
		},
		"shadow-black": {call: "black shadows, the default", lights: point(220, warm())},
		"shadow-color": {
			call:   "light.ShadowColor(color.RGB(0.15, 0.3, 1.0))",
			lights: point(220, warm(), light.ShadowColor(color.RGB(0.15, 0.3, 1.0))),
		},
		"shadow-opacity": {call: "light.ShadowOpacity(0.5)", lights: point(220, warm(), light.ShadowOpacity(0.5))},
		"shadow-both": {
			call:   "ShadowColor(blue), ShadowOpacity(0.5)",
			lights: point(220, warm(), light.ShadowColor(color.RGB(0.15, 0.3, 1.0)), light.ShadowOpacity(0.5)),
		},
		"silhouette-box": {
			call:    "orc: light.Occluder()",
			at:      geom.V2(150, 190),
			lights:  point(220, warm()),
			sprites: orc(light.Occluder()),
		},
		"silhouette-on": {
			call:    "orc: light.Silhouette()",
			at:      geom.V2(150, 190),
			lights:  point(220, warm()),
			sprites: orc(light.Silhouette()),
		},
		"shadedby-all": {
			call:   "crowd: feet boxes light.Occluder(2)",
			lights: point(220, warm()),
			crowd:  []twod.Option{},
		},
		"shadedby-0-1": {
			call:   "crowd layer: light.ShadedBy(0, 1)",
			lights: point(220, warm()),
			crowd:  []twod.Option{light.ShadedBy(0, 1)},
		},
	}
}

func cone(spread float32, angle float32, opts ...light.Option) func(geom.Vec2) []*light.Light {
	return func(at geom.Vec2) []*light.Light {
		return []*light.Light{
			light.NewCone(220, warm(), angle, spread, append([]light.Option{light.At(at.X, at.Y)}, opts...)...),
		}
	}
}

func cones() map[string]panel {
	return map[string]panel{
		"angle-0":    {call: "light.NewCone(220, warm, 0, 0.8)", lights: cone(0.8, 0, light.Intensity(1.5))},
		"angle-up":   {call: "angle -math.Pi/2", lights: cone(0.8, -math.Pi/2, light.Intensity(1.5))},
		"angle-down": {call: "angle math.Pi/4", lights: cone(0.8, math.Pi/4, light.Intensity(1.5))},
		"spread-0.2": {call: "spread 0.2", lights: cone(0.2, -0.3)},
		"spread-0.5": {call: "spread 0.5", lights: cone(0.5, -0.3)},
		"spread-1.2": {call: "spread 1.2", lights: cone(1.2, -0.3)},
		"cone-size-0": {
			call:   "light.Size(0)",
			at:     geom.V2(304, 104),
			lights: cone(0.6, math.Pi, light.Intensity(1.5)),
		},
		"cone-size-32": {
			call:   "light.Size(32)",
			at:     geom.V2(304, 104),
			lights: cone(0.6, math.Pi, light.Intensity(1.5), light.Size(32)),
		},
	}
}

func sun(angle float32, opts ...light.Option) func(geom.Vec2) []*light.Light {
	return func(geom.Vec2) []*light.Light {
		return []*light.Light{light.NewDirectional(color.RGB(1, 0.9, 0.75), angle, opts...)}
	}
}

func directionals() map[string]panel {
	day := rgb(0.3, 0.3, 0.34)
	return map[string]panel{
		"sun-angle-0.2": {
			call:    "light.NewDirectional(c, 0.2*math.Pi)",
			ambient: day,
			lights:  sun(0.2 * math.Pi),
			noDot:   true,
		},
		"sun-angle-0.5": {call: "angle 0.5*math.Pi", ambient: day, lights: sun(0.5 * math.Pi), noDot: true},
		"sun-angle-0.8": {call: "angle 0.8*math.Pi", ambient: day, lights: sun(0.8 * math.Pi), noDot: true},
		"sun-length-40": {
			call:    "light.ShadowLength(40)",
			ambient: day,
			lights:  sun(0.25*math.Pi, light.ShadowLength(40)),
			noDot:   true,
		},
		"sun-length-120": {
			call:    "light.ShadowLength(120)",
			ambient: day,
			lights:  sun(0.25*math.Pi, light.ShadowLength(120)),
			noDot:   true,
		},
		"sun-length-256": {call: "light.ShadowLength(256)", ambient: day, lights: sun(0.25 * math.Pi), noDot: true},
		"day-dawn": {
			call:    "dawn 0.15π · Intensity 0.4 · ShadowLength 260",
			ambient: rgb(0.12, 0.1, 0.16),
			lights:  sun(0.15*math.Pi, light.Intensity(0.4), light.ShadowLength(260)),
			noDot:   true,
		},
		"day-noon": {
			call:    "noon 0.5π · Intensity 1 · ShadowLength 16",
			ambient: rgb(0.35, 0.35, 0.4),
			lights:  sun(0.5*math.Pi, light.ShadowLength(16)),
			noDot:   true,
		},
		"day-dusk": {
			call:    "dusk 0.85π · Intensity 0.4 · ShadowLength 260",
			ambient: rgb(0.14, 0.09, 0.1),
			lights:  sun(0.85*math.Pi, light.Intensity(0.4), light.ShadowLength(260)),
			noDot:   true,
		},
	}
}

func shaped(s light.Shape, reach float32, opts ...light.Option) func(geom.Vec2) []*light.Light {
	return func(at geom.Vec2) []*light.Light {
		return []*light.Light{
			light.NewShape(s, color.RGB(0.45, 0.9, 1), reach, append([]light.Option{light.At(at.X, at.Y)}, opts...)...),
		}
	}
}

func shapes() map[string]panel {
	return map[string]panel{
		"shape-rect":   {call: "light.Rect(120, 40)", lights: shaped(light.Rect(120, 40), 24), noDot: true},
		"shape-circle": {call: "light.Circle(36)", lights: shaped(light.Circle(36), 24), noDot: true},
		"shape-capsule": {
			call:   "light.Capsule(a, b, 16)",
			lights: shaped(light.Capsule(geom.V2(-50, -20), geom.V2(50, 20), 16), 24),
			noDot:  true,
		},
		"shape-line": {
			call:   "light.Line(a, b)",
			lights: shaped(light.Line(geom.V2(-70, 30), geom.V2(70, -30)), 24),
			noDot:  true,
		},
		"shape-polygon": {
			call:   "light.Polygon(p…)",
			lights: shaped(light.Polygon(geom.V2(0, -50), geom.V2(50, 36), geom.V2(-50, 36)), 24),
			noDot:  true,
		},
		"reach-8":  {call: "reach 8", lights: shaped(light.Circle(36), 8), noDot: true},
		"reach-24": {call: "reach 24", lights: shaped(light.Circle(36), 24), noDot: true},
		"reach-80": {call: "reach 80", lights: shaped(light.Circle(36), 80), noDot: true},
		"shadows-off": {
			call:   "light.NewShape(…), no shadows",
			at:     behind(),
			lights: shaped(light.Circle(10), 160),
			noDot:  true,
		},
		"shadows-on": {
			call:   "light.Shadows(true)",
			at:     behind(),
			lights: shaped(light.Circle(10), 160, light.Shadows(true)),
			noDot:  true,
		},
		"emissive-off": {call: "flask: light.Unlit()", sprites: flask()},
		"emissive-on":  {call: "flask: light.Emissive(90), light.Unlit()", sprites: flask(light.Emissive(90))},
	}
}

// crate is a crate sprite standing in the column's shadow when the lamp
// is beside the column, with opts.
func crate(opts ...twod.Option) func() []*twod.Sprite {
	return func() []*twod.Sprite {
		o := []twod.SpriteOption{twod.At(72, 120), twod.Scale(2, 2)}
		for _, opt := range opts {
			o = append(o, opt)
		}
		return []*twod.Sprite{twod.NewSprite(crypt.Sprite("crate"), o...)}
	}
}

// flask is a red flask on the floor, unlit so it shows in the dark, with
// opts.
func flask(opts ...twod.Option) func() []*twod.Sprite {
	return func() []*twod.Sprite {
		o := []twod.SpriteOption{twod.At(lampAt().X, lampAt().Y), twod.Scale(2, 2), light.Unlit()}
		for _, opt := range opts {
			o = append(o, opt)
		}
		return []*twod.Sprite{twod.NewSprite(crypt.Sprite("flask_big_red"), o...)}
	}
}

func debugViews() map[string]panel {
	return map[string]panel{
		"debug-light": {
			call:   "light.SetDebug(win, light.DebugLight)",
			lights: point(220, warm()),
			debug:  light.DebugLight,
		},
		"debug-occluders": {
			call:     "light.DebugOccluders + SetOutlines",
			lights:   point(220, warm()),
			debug:    light.DebugOccluders,
			outlines: true,
		},
		"debug-shadows": {
			call:   "light.SetDebug(win, light.DebugShadows)",
			lights: point(220, warm()),
			debug:  light.DebugShadows,
		},
	}
}

// lamps are five small lamps along the floor, with opts.
func lamps(opts ...light.Option) func(geom.Vec2) []*light.Light {
	return func(geom.Vec2) []*light.Light {
		var out []*light.Light
		for _, x := range []float32{32, 96, 160, 224, 288} {
			out = append(out, light.NewPoint(70, warm(), append([]light.Option{light.At(x, 190)}, opts...)...))
		}
		return out
	}
}

func budgets() map[string]panel {
	return map[string]panel{
		"budget-none": {call: "five lamps, no Budget", lights: lamps(), noDot: true, dots: true},
		"exposure-0.6": {
			call:   "camera: light.Exposure(0.6)",
			lights: point(220, warm()),
			camera: []twod.CameraOption{light.Exposure(0.6)},
		},
		"exposure-1": {call: "camera: light.Exposure(1), the default", lights: point(220, warm())},
		"exposure-1.6": {
			call:   "camera: light.Exposure(1.6)",
			lights: point(220, warm()),
			camera: []twod.CameraOption{light.Exposure(1.6)},
		},
		"affects-all":   {call: "a lamp lights every lit layer", lights: point(220, warm())},
		"affects-floor": {call: "light.Affects(floor)", lights: point(220, warm()), floorOnly: true},
		"budget-3": {
			call:   "camera: light.Budget(3)",
			lights: lamps(),
			noDot:  true,
			dots:   true,
			camera: []twod.CameraOption{light.Budget(3)},
		},
	}
}

// orc is one orc, three times its size, left of the lamp, with opts.
func orc(opts ...twod.Option) func() []*twod.Sprite {
	return func() []*twod.Sprite {
		o := []twod.SpriteOption{twod.At(96, 140), twod.Scale(3, 3)}
		for _, opt := range opts {
			o = append(o, opt)
		}
		return []*twod.Sprite{twod.NewSprite(crypt.Sprite("orc_warrior_idle_anim_f0"), o...)}
	}
}
