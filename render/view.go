package render

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

// View is the resolved active camera for the current frame. Gameplay uses
// it for picking and for anchoring UI to world positions.
type View struct {
	Active     bool
	Position   geom.Vec3
	Forward    geom.Vec3
	ViewMat    geom.Mat4
	Proj       geom.Mat4
	ViewProj   geom.Mat4
	InvVP      geom.Mat4
	Viewport   geom.Vec2
	Focus      geom.Vec3
	Near, Far  float32
	Projection *Camera
}

// ScreenRay returns the world ray under a window point (logical pixels).
func (v *View) ScreenRay(p geom.Vec2) geom.Ray {
	ndc := geom.V2(2*p.X/v.Viewport.X-1, 1-2*p.Y/v.Viewport.Y)
	near := v.InvVP.MulPoint(geom.V3(ndc.X, ndc.Y, -1))
	far := v.InvVP.MulPoint(geom.V3(ndc.X, ndc.Y, 1))
	return geom.Ray{Origin: near, Dir: far.Sub(near).Normalize()}
}

// WorldToScreen projects a world point to window coordinates. ok is false
// when the point is behind the camera.
func (v *View) WorldToScreen(p geom.Vec3) (geom.Vec2, bool) {
	c := v.ViewProj.MulVec4(p.Vec4(1))
	if c.W <= 0 {
		return geom.Vec2{}, false
	}
	x, y := c.X/c.W, c.Y/c.W
	return geom.V2((x+1)/2*v.Viewport.X, (1-y)/2*v.Viewport.Y), true
}

// Frustum returns the culling planes of the view.
func (v *View) Frustum() geom.Frustum { return geom.FrustumFromMatrix(v.ViewProj) }

// ViewSystem resolves every camera's View at the end of PostUpdate, after
// transforms and camera controllers, and marks the camera each window
// presents: its highest Priority one.
type ViewSystem struct {
	world.Group[Viewer]
	world *world.World
	best  map[*platform.Window]Viewer
}

func (s *ViewSystem) New(w *world.World) {
	s.world = w
	s.best = map[*platform.Window]Viewer{}
}

func (s *ViewSystem) Phase() world.Phase { return world.PostUpdate }
func (s *ViewSystem) Priority() int      { return -1000 }

func (s *ViewSystem) Update(world.Frame) {
	clear(s.best)
	for c := range s.All() {
		cam := c.GetCamera()
		win := wm.Home(s.world, c, cam.Window)
		cam.target = win
		size, aspect := geom.V2(1280, 720), float32(16.0/9)
		if win != nil && win.Size().Y > 0 {
			size, aspect = win.Size(), win.Aspect()
		}
		m := c.GetTransform().World()
		if c.GetBasicEntity().Parent() == nil {
			m = c.GetTransform().Local()
		}
		cam.view, cam.presenting = Resolve(cam, m, size, aspect), false
		if b, ok := s.best[win]; !ok || cam.Priority > b.GetCamera().Priority {
			s.best[win] = c
		}
	}
	for _, c := range s.best {
		c.GetCamera().presenting = true
	}
}

// Resolve computes a View from a camera and its world matrix.
func Resolve(c *Camera, world geom.Mat4, viewport geom.Vec2, aspect float32) View {
	fov, near, far := c.FOV, c.Near, c.Far
	if fov == 0 {
		fov = 45 * geom.Deg2Rad
	}
	if near == 0 {
		near = 0.1
	}
	if far == 0 {
		far = 500
	}
	var proj geom.Mat4
	if c.OrthoHeight > 0 {
		h := c.OrthoHeight / 2
		proj = geom.Ortho(-h*aspect, h*aspect, -h, h, near, far)
	} else {
		proj = geom.Perspective(fov, aspect, near, far)
	}
	viewMat := world.Inverse()
	vp := proj.Mul(viewMat)
	pos := world.Translation()
	fwd := world.MulDir(geom.Forward).Normalize()
	focus := pos.Add(fwd.Scale(20))
	if hit, ok := (geom.Ray{Origin: pos, Dir: fwd}).IntersectGround(0); ok {
		focus = hit
	}
	return View{
		Active: true, Position: pos, Forward: fwd,
		ViewMat: viewMat, Proj: proj, ViewProj: vp, InvVP: vp.Inverse(),
		Viewport: viewport, Focus: focus, Near: near, Far: far, Projection: c,
	}
}
