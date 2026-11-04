// Look3d is a 3D scene that uses everything the 3D renderer draws, for
// the golden captures (gpu's TestGolden): a textured,
// mipmapped ground, opaque, transparent, unlit, double-sided and swaying
// meshes, point lights, the sun's shadows, fog, bloom, a colour grade and
// a vignette, blended and additive billboards, a 2D layer behind the world
// and a label in front. LOOK3D=post adds the camera's post chain: half
// resolution, a nearest upscale and an effect reading the scene's depth;
// LOOK3D=lines adds a grid of gizmo lines, which the golden captures leave
// out: OpenGL drivers, which drew the references, rasterize lines a little
// differently.
//
//	go run ./examples/internal/look3d
package main

import (
	"image"
	"image/color"
	"os"
	"strings"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/camera"
	ecolor "github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/mesh"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/merlindorin/ember/world"
)

type prop struct {
	*world.BasicEntity
	scene.Transform
	render.Model
}

func newProp(at scene.Transform, m *mesh.Data, mat *render.Material) *prop {
	return &prop{world.NewBasic(), at, render.Model{Mesh: m, Material: mat}}
}

type lamp struct {
	*world.BasicEntity
	scene.Transform
	render.PointLight
}

func newLamp(x, y, z float32, c ecolor.Color) *lamp {
	return &lamp{world.NewBasic(), scene.At(x, y, z), render.PointLight{Color: c, Intensity: 3, Range: 6}}
}

// checker is a 64×64 checkerboard that mipmaps.
func checker() *render.Texture {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			c := color.RGBA{200, 180, 140, 255}
			if (x/8+y/8)%2 == 1 {
				c = color.RGBA{90, 110, 80, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return &render.Texture{Image: img, Mipmaps: true, Repeat: true}
}

const depthTint = `
fn effect(vUV: vec2f) -> vec4f {
	let c = color(vUV).rgb;
	let d = clamp(linearDepth(vUV) / 30.0, 0.0, 1.0);
	return vec4f(mix(c, c * vec3f(0.8, 0.9, 1.2), d) + vec3f((1.0 - vUV.y) * 0.08, 0.0, 0.0), 1.0);
}`

func main() {
	win := platform.NewWindow(
		platform.Title("ember · look3d"),
		platform.Size(800, 500),
		platform.FixedSize(),
		platform.Background(ecolor.Hex("#1a1d24")),
	)
	env := render.NewEnvironment(
		render.Sun(geom.V3(-0.4, -1, -0.3), ecolor.Hex("#fff4e0"), 2.2),
		render.Ambient(ecolor.Hex("#8aa4c8"), ecolor.Hex("#5a4a3a"), 0.6),
		render.Fog(ecolor.Hex("#303846"), 0.025),
		render.Bloom(0.6, 0.9),
		render.Vignette(0.35),
		render.ColorGrade(1.1, 1.05, ecolor.Hex("#e8ecff"), ecolor.Hex("#fff0dc")),
	)
	ground := render.Mat(ecolor.White)
	ground.Texture, ground.Specular = checker(), 0.2
	glass := render.Mat(ecolor.RGBA(0.4, 0.7, 1, 0.45))
	glass.Transparent = true
	glow := render.Mat(ecolor.Hex("#ffb060"))
	glow.Unlit, glow.Emissive = true, ecolor.RGB(2.5, 1.2, 0.4)
	leaf := render.Mat(ecolor.Hex("#5c8a3c"))
	leaf.DoubleSided = true
	tree := render.Mat(ecolor.Hex("#4f7a36"))
	tree.Wind = 1.5
	shiny := render.Mat(ecolor.Hex("#c04040"))
	shiny.Specular = 1

	gizmos, bills := render.NewGizmos(), render.NewBillboards()
	eye := camera.NewOrbit(camera.Distance(12), camera.Angles(0.7, 0.42))
	if strings.Contains(os.Getenv("LOOK3D"), "post") {
		eye.Post = render.NewPost()
		eye.Post.Resolution = func(fb geom.Vec2) (int, int) { return int(fb.X / 2), int(fb.Y / 2) }
		eye.Post.Upscale = render.FilterNearest
		eye.Post.Add(render.NewEffect(depthTint, render.Name("depth tint")))
	}
	sky := twod.NewLayer(-1, twod.Fixed()).
		Add(twod.NewPolygon([]geom.Vec2{{}, {X: 800}, {X: 800, Y: 160}, {Y: 220}}, twod.Color(ecolor.Hex("#2e3a52"))))
	win.Add(
		sky,
		label("look3d"),
		newProp(scene.At(0, -0.1, 0), mesh.Box(geom.V3(10, 0.2, 10)), ground),
		newProp(scene.At(-1.5, 0.6, 0), mesh.Cube(1.2), shiny),
		newProp(scene.At(1.2, 0.8, 1.2), mesh.Sphere(0.8, 24, 16), glass),
		newProp(scene.At(2.4, 0.5, -1.8), mesh.Cube(1), glow),
		newProp(scene.At(-2.8, 1, -2), mesh.Quad(), leaf),
		newProp(scene.At(0.5, 1, -3), mesh.Cone(0.8, 2.4, 12), tree),
		newLamp(-2, 1.5, 2, ecolor.Hex("#ff8040")),
		newLamp(2, 1.2, 2.5, ecolor.Hex("#40a0ff")),
		newLamp(0, 2, -1, ecolor.Hex("#80ff80")),
		gizmos, bills, eye,
	)
	ember.New().AddEntities(win, env).AddSystems(&marks{gizmos: gizmos, bills: bills}).MustRun()
}

// label is a Document holding text at the window's top-left, over the
// scene.
func label(text string) *ui.Document {
	doc := ui.NewDocument(
		kit.Theme(),
		ui.CSS(`body { background-color: transparent; } .tag { position: absolute; left: 16px; top: 16px; }`),
	)
	doc.Body().Append(kit.Label(text, ui.Class("tag")))
	return doc
}

// marks draws, each frame, a grid of gizmo lines over the ground and two
// billboards, one blended and one additive.
type marks struct {
	gizmos *render.Gizmos
	bills  *render.Billboards
}

func (m *marks) Update(world.Frame) {
	for i := -5; i <= 5 && strings.Contains(os.Getenv("LOOK3D"), "lines"); i++ {
		f := float32(i)
		m.gizmos.Line(geom.V3(f, 0.02, -5), geom.V3(f, 0.02, 5), ecolor.RGBA(1, 1, 1, 0.3))
		m.gizmos.Line(geom.V3(-5, 0.02, f), geom.V3(5, 0.02, f), ecolor.RGBA(1, 1, 1, 0.3))
	}
	m.bills.Items = append(
		m.bills.Items,
		render.Billboard{Position: geom.V3(-2, 2.2, 1), Size: 0.8, Color: ecolor.RGBA(1, 0.8, 0.3, 0.9)},
		render.Billboard{
			Position: geom.V3(2.5, 1.6, -1),
			Size:     1.2,
			Color:    ecolor.RGBA(0.3, 0.6, 1, 0.8),
			Additive: true,
			Rotation: 0.6,
		},
	)
}
