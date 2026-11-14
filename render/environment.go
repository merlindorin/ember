package render

import (
	"encoding/json"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/world"
)

// Environment is the look of the world: sun, ambient light, fog and the
// image's post-processing. It is an entity: the first one added to the World
// lights the scene, and a neutral one does when there is none. Set it up with
// options, and change it at runtime through its setters (a day/night
// cycle). Bloom, vignette and fog are off unless asked for.
type Environment struct {
	*world.BasicEntity
	sunDirection   geom.Vec3
	sunColor       color.Color
	sunIntensity   float32
	shadows        bool
	shadowDistance float32
	skyColor       color.Color
	groundColor    color.Color
	ambient        float32
	fogColor       color.Color
	fogDensity     float32
	exposure       float32
	bloom          float32
	bloomThreshold float32
	vignette       float32
	saturation     float32
	contrast       float32
	shadowTint     color.Color
	highlightTint  color.Color
}

func (e *Environment) GetEnvironment() *Environment { return e }

// Lit is any entity holding an environment, such as a scene's environment
// node.
type Lit interface {
	world.BasicFace
	GetEnvironment() *Environment
}

// Option sets up an Environment.
type Option func(*Environment)

// NewEnvironment returns a neutral daylight environment set up by opts.
func NewEnvironment(opts ...Option) *Environment {
	e := &Environment{
		BasicEntity:  world.NewBasic(),
		sunDirection: geom.V3(-0.4, -1, -0.3).Normalize(), sunColor: color.RGB(1, 0.95, 0.85), sunIntensity: 2.5,
		shadows: true, shadowDistance: 30,
		skyColor: color.RGB(0.45, 0.55, 0.75), groundColor: color.RGB(0.2, 0.18, 0.15), ambient: 0.35,
		fogColor: color.RGB(0.55, 0.62, 0.75),
		exposure: 1, bloomThreshold: 1, saturation: 1, contrast: 1,
		shadowTint: color.White, highlightTint: color.White,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Sun sets the sun: direction points from the sun toward the ground.
func Sun(direction geom.Vec3, c color.Color, intensity float32) Option {
	return func(e *Environment) { e.SetSun(direction, c, intensity) }
}

// Shadows casts sun shadows over distance around the camera's focus; 0
// turns them off.
func Shadows(distance float32) Option {
	return func(e *Environment) { e.SetShadows(distance) }
}

// Ambient sets the ambient light, blended from sky above to ground below.
func Ambient(sky, ground color.Color, amount float32) Option {
	return func(e *Environment) { e.SetAmbient(sky, ground, amount) }
}

// Fog fades distant geometry toward c.
func Fog(c color.Color, density float32) Option {
	return func(e *Environment) { e.SetFog(c, density) }
}

// Exposure scales the image's brightness before tone mapping.
func Exposure(exposure float32) Option { return func(e *Environment) { e.SetExposure(exposure) } }

// Bloom makes what is brighter than threshold glow by strength.
func Bloom(strength, threshold float32) Option {
	return func(e *Environment) { e.SetBloom(strength, threshold) }
}

// Vignette darkens the image's corners by amount.
func Vignette(amount float32) Option { return func(e *Environment) { e.SetVignette(amount) } }

// ColorGrade grades the image: saturation and contrast (1 is neutral),
// and tints for its shadows and highlights.
func ColorGrade(saturation, contrast float32, shadowTint, highlightTint color.Color) Option {
	return func(e *Environment) { e.SetColorGrade(saturation, contrast, shadowTint, highlightTint) }
}

// Sun is the sun's direction, colour and intensity.
func (e *Environment) Sun() (geom.Vec3, color.Color, float32) {
	return e.sunDirection, e.sunColor, e.sunIntensity
}

func (e *Environment) SetSun(direction geom.Vec3, c color.Color, intensity float32) {
	e.sunDirection, e.sunColor, e.sunIntensity = direction, c, intensity
}

// Shadows is how far round the camera's focus sun shadows are cast, 0
// when they are off.
func (e *Environment) Shadows() float32 {
	if !e.shadows {
		return 0
	}
	return e.shadowDistance
}

func (e *Environment) SetShadows(distance float32) {
	e.shadows = distance > 0
	if distance > 0 {
		e.shadowDistance = distance
	}
}

// Ambient is the sky's and the ground's colours, and how much they light.
func (e *Environment) Ambient() (color.Color, color.Color, float32) {
	return e.skyColor, e.groundColor, e.ambient
}

func (e *Environment) SetAmbient(sky, ground color.Color, amount float32) {
	e.skyColor, e.groundColor, e.ambient = sky, ground, amount
}

func (e *Environment) Fog() (color.Color, float32) { return e.fogColor, e.fogDensity }

func (e *Environment) SetFog(c color.Color, density float32) { e.fogColor, e.fogDensity = c, density }

func (e *Environment) Exposure() float32 { return e.exposure }

func (e *Environment) SetExposure(exposure float32) { e.exposure = exposure }

// Bloom is the bloom's strength and threshold.
func (e *Environment) Bloom() (float32, float32) { return e.bloom, e.bloomThreshold }

func (e *Environment) SetBloom(strength, threshold float32) {
	e.bloom, e.bloomThreshold = strength, threshold
}

func (e *Environment) Vignette() float32 { return e.vignette }

func (e *Environment) SetVignette(amount float32) { e.vignette = amount }

// ColorGrade is the saturation, contrast, shadow tint and highlight tint.
func (e *Environment) ColorGrade() (float32, float32, color.Color, color.Color) {
	return e.saturation, e.contrast, e.shadowTint, e.highlightTint
}

func (e *Environment) SetColorGrade(saturation, contrast float32, shadowTint, highlightTint color.Color) {
	e.saturation, e.contrast, e.shadowTint, e.highlightTint = saturation, contrast, shadowTint, highlightTint
}

// CopyLook takes everything from o but its sun, which a DirectionalLight
// owns.
func (e *Environment) CopyLook(o *Environment) {
	dir, c, i := e.Sun()
	basic := e.BasicEntity
	*e = *o
	e.BasicEntity = basic
	e.SetSun(dir, c, i)
}

type environmentJSON struct {
	SunDirection   geom.Vec3   `json:"sun_direction"`
	SunColor       color.Color `json:"sun_color"`
	SunIntensity   float32     `json:"sun_intensity"`
	Shadows        bool        `json:"shadows"`
	ShadowDistance float32     `json:"shadow_distance"`
	SkyColor       color.Color `json:"sky_color"`
	GroundColor    color.Color `json:"ground_color"`
	Ambient        float32     `json:"ambient"`
	FogColor       color.Color `json:"fog_color"`
	FogDensity     float32     `json:"fog_density"`
	Exposure       float32     `json:"exposure"`
	Bloom          float32     `json:"bloom"`
	BloomThreshold float32     `json:"bloom_threshold"`
	Vignette       float32     `json:"vignette"`
	Saturation     float32     `json:"saturation"`
	Contrast       float32     `json:"contrast"`
	ShadowTint     color.Color `json:"shadow_tint"`
	HighlightTint  color.Color `json:"highlight_tint"`
}

func (e *Environment) MarshalJSON() ([]byte, error) { return json.Marshal(e.json()) }

func (e *Environment) UnmarshalJSON(b []byte) error {
	d := *NewEnvironment().json()
	if err := json.Unmarshal(b, &d); err != nil {
		return err
	}
	basic := e.BasicEntity
	if basic == nil {
		basic = world.NewBasic()
	}
	*e = Environment{
		basic, d.SunDirection, d.SunColor, d.SunIntensity, d.Shadows, d.ShadowDistance,
		d.SkyColor, d.GroundColor, d.Ambient, d.FogColor, d.FogDensity,
		d.Exposure, d.Bloom, d.BloomThreshold, d.Vignette, d.Saturation, d.Contrast, d.ShadowTint, d.HighlightTint,
	}
	return nil
}

func (e *Environment) json() *environmentJSON {
	return &environmentJSON{
		e.sunDirection, e.sunColor, e.sunIntensity, e.shadows, e.shadowDistance,
		e.skyColor, e.groundColor, e.ambient, e.fogColor, e.fogDensity,
		e.exposure, e.bloom, e.bloomThreshold, e.vignette, e.saturation, e.contrast, e.shadowTint, e.highlightTint,
	}
}
