// Package render describes what to draw, never how. It holds the
// components entities embed (Model, PointLight, Camera), the Environment
// resource (sun, sky, fog, post-processing) and immediate-mode buffers
// (Gizmos, Billboards). It has no GPU dependency: dedicated servers link it,
// and the gpu package turns it into pixels on clients.
//
//	type Torch struct {
//		*world.BasicEntity
//		scene.Transform
//		render.Model
//		render.PointLight
//	}
package render

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/mesh"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/scene"
)

// Material controls surface shading. Materials are shared by pointer;
// entities with the same mesh and material are drawn in a single instanced
// draw call.
type Material struct {
	// Albedo multiplies the mesh vertex colors.
	Albedo color.Color
	// Emissive is added after lighting; values above 1 bloom.
	Emissive color.Color
	// Specular is the strength of highlights (0 = matte).
	Specular float32
	// Unlit skips lighting entirely (sky domes, markers).
	Unlit bool
	// Wind sways vertices with height, for foliage.
	Wind float32
	// Transparent blends with alpha and is drawn after opaque geometry.
	Transparent bool
	// Texture multiplies Albedo and the vertex colors through the mesh UVs.
	Texture *Texture
	// DoubleSided draws back faces too, lit from their side.
	DoubleSided bool
}

// Mat is a shorthand for a lit material of the given color.
func Mat(albedo color.Color) *Material { return &Material{Albedo: albedo} }

// Glow is a shorthand for an emissive, unshadowed material.
func Glow(c color.Color, intensity float32) *Material {
	return &Material{Albedo: c, Emissive: c.Scale(intensity)}
}

type Model struct {
	Mesh     *mesh.Data
	Material *Material
	// Tint multiplies the material per instance (hit flashes, team colors).
	// The zero value means white.
	Tint     color.Color
	Hidden   bool
	NoShadow bool
}

func (m *Model) GetModel() *Model { return m }

// Drawable is any entity the renderer draws.
type Drawable interface {
	scene.Transformable
	GetModel() *Model
}

// PointLight emits light in all directions from the entity's position.
type PointLight struct {
	Color     color.Color `node:"prop"`
	Intensity float32     `node:"prop"`
	Range     float32     `node:"prop"`
	// Flicker animates intensity like a flame (0 = steady).
	Flicker float32
}

func (l *PointLight) GetPointLight() *PointLight { return l }

type Light interface {
	scene.Transformable
	GetPointLight() *PointLight
}

// Camera turns an entity into a viewpoint. The entity's Transform positions
// it; -Z is the viewing direction.
type Camera struct {
	// FOV is the vertical field of view in radians (default 45°).
	FOV       float32
	Near, Far float32
	// OrthoHeight > 0 switches to an orthographic projection of that height.
	OrthoHeight float32
	// Priority picks the active camera when several target the same window
	// (highest wins).
	Priority int
	// Window is the window this camera renders to; nil means the window
	// the camera entity is under, else the primary.
	Window *platform.Window
	// Post turns what the camera renders into the final image; nil
	// presents it as rendered.
	Post *Post

	view       View
	presenting bool
	target     *platform.Window
}

func (c *Camera) GetCamera() *Camera { return c }

// View is what the camera saw at the end of the last frame: for picking and
// for placing UI over the world.
func (c *Camera) View() *View { return &c.view }

// RendersIn is the window the camera renders to, from the last frame.
func (c *Camera) RendersIn() *platform.Window { return c.target }

// Presenting reports whether the camera is the one its window shows: the
// highest Priority among those targeting it.
func (c *Camera) Presenting() bool { return c.presenting }

type Viewer interface {
	scene.Transformable
	GetCamera() *Camera
}
