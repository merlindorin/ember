package anim

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/mesh"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/world"
)

// Animator is the component playing clips on a skeleton, with crossfades.
// After each update Globals holds the model-space joint matrices.
type Animator struct {
	Skeleton *Skeleton
	Clips    []*Clip
	// Speed scales playback (0 means 1); Paused freezes it.
	Speed  float32
	Paused bool
	// Constraints adjust the pose after sampling, in order (IK, aiming).
	Constraints []Constraint
	// RootMotion plays clips in place and reports their travel through
	// Motion, so gameplay moves the entity by what the feet do: no sliding
	// and no snap back at each loop. The travel of a loop is measured at
	// RootJoint in model space (the hips, whichever bones carry it) and
	// removed evenly over the loop, keeping the body's natural sway.
	RootMotion bool
	RootJoint  int
	motion     geom.Vec3
	drifts     map[*Clip]geom.Vec3
	scratch    Pose
	scratchG   []geom.Mat4

	current, previous playback
	fade, fadeTime    float32
	pose, from        Pose
	globals           []geom.Mat4
}

type playback struct {
	clip *Clip
	time float32
	once bool
}

func (a *Animator) GetAnimator() *Animator { return a }

// Find returns a clip by name.
func (a *Animator) Find(name string) *Clip {
	for _, c := range a.Clips {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// Play loops a clip, crossfading from the current one over fade seconds.
// It reports whether the clip exists. Playing the current clip is a no-op.
func (a *Animator) Play(name string, fade float32) bool { return a.start(name, fade, false) }

// PlaySynced crossfades to a looping clip at the same phase as the
// current one: walk, jog and run cycles keep their feet in step, with no
// stumble when the speed changes.
func (a *Animator) PlaySynced(name string, fade float32) bool {
	cur := a.current
	if !a.start(name, fade, false) {
		return false
	}
	if c := a.current.clip; c != cur.clip && cur.clip != nil && cur.clip.Length() > 0 && !cur.once {
		a.current.time = c.Start + cur.clip.Phase(cur.time)*c.Length()
	}
	return true
}

// PlayOnce plays a clip once and holds its last frame.
func (a *Animator) PlayOnce(name string, fade float32) bool { return a.start(name, fade, true) }

func (a *Animator) start(name string, fade float32, once bool) bool {
	c := a.Find(name)
	if c == nil {
		return false
	}
	if a.current.clip == c && !once {
		return true
	}
	a.previous, a.current = a.current, playback{clip: c, once: once, time: c.Start}
	a.fade, a.fadeTime = 0, fade
	if a.previous.clip == nil || fade <= 0 {
		a.fadeTime = 0
	}
	return true
}

// Stop returns to the rest pose.
func (a *Animator) Stop() { a.current, a.previous, a.fadeTime = playback{}, playback{}, 0 }

// Clip returns the playing clip (nil at rest) and its time.
func (a *Animator) Clip() (*Clip, float32) { return a.current.clip, a.current.time }

// Seek moves the playing clip to t seconds.
func (a *Animator) Seek(t float32) { a.current.time = t }

// Globals returns the model-space joint matrices of the last update.
func (a *Animator) Globals() []geom.Mat4 { return a.globals }

// Pose returns the local pose of the last update.
func (a *Animator) Pose() Pose { return a.pose }

// Motion returns the horizontal root displacement of the last update in
// model space (before the entity's rotation), with RootMotion on.
func (a *Animator) Motion() geom.Vec3 { return a.motion }

// drift returns how far a clip moves RootJoint horizontally over one
// loop, in model space.
func (a *Animator) drift(c *Clip) geom.Vec3 {
	if d, ok := a.drifts[c]; ok {
		return d
	}
	if a.drifts == nil {
		a.drifts = map[*Clip]geom.Vec3{}
	}
	at := func(t float32) geom.Vec3 {
		a.scratch = resize(a.scratch, len(a.Skeleton.Joints))
		a.Skeleton.Reset(a.scratch)
		c.Sample(t, a.scratch)
		a.scratchG = a.Skeleton.Globals(a.scratch, a.scratchG)
		return a.scratchG[a.RootJoint].Translation()
	}
	d := at(c.Duration).Sub(at(c.Start))
	d.Y = 0
	a.drifts[c] = d
	return d
}

// travel returns the root motion of a playback advancing from t0 to t1.
func (a *Animator) travel(p playback, t0, t1 float32) geom.Vec3 {
	if p.clip == nil || p.clip.Length() <= 0 {
		return geom.Vec3{}
	}
	dt := t1 - t0
	if dt < 0 && !p.once {
		dt += p.clip.Length()
	}
	return a.drift(p.clip).Scale(dt / p.clip.Length())
}

// inPlace removes the share of the loop's drift reached at t from a pose,
// at the top of the root joint's hierarchy.
func (a *Animator) inPlace(p playback, pose Pose) {
	if p.clip == nil || p.clip.Length() <= 0 {
		return
	}
	top := a.RootJoint
	for a.Skeleton.Joints[top].Parent >= 0 {
		top = a.Skeleton.Joints[top].Parent
	}
	pose[top].T = pose[top].T.Sub(a.drift(p.clip).Scale(p.clip.Phase(p.time)))
}

func (p *playback) advance(dt float32) {
	if p.clip == nil {
		return
	}
	p.time += dt
	if l := p.clip.Length(); l > 0 {
		if p.once {
			p.time = min(p.time, p.clip.Duration)
		} else {
			for p.time >= p.clip.Duration {
				p.time -= l
			}
		}
	}
}

// Update advances playback by dt and resolves the pose.
func (a *Animator) Update(dt float32) {
	if a.Skeleton == nil {
		return
	}
	if len(a.pose) != len(a.Skeleton.Joints) {
		a.pose, a.from = a.Skeleton.RestPose(), a.Skeleton.RestPose()
	}
	a.motion = geom.Vec3{}
	if !a.Paused {
		speed := a.Speed
		if speed == 0 {
			speed = 1
		}
		cur, prev := a.current, a.previous
		a.current.advance(dt * speed)
		a.previous.advance(dt * speed)
		a.fade += dt
		if a.RootMotion && valid(a.Skeleton, a.RootJoint) {
			a.motion = a.travel(a.current, cur.time, a.current.time)
			if a.fadeTime > 0 && a.fade < a.fadeTime {
				a.motion = a.travel(a.previous, prev.time, a.previous.time).Lerp(a.motion, a.fade/a.fadeTime)
			}
		}
	}
	rooted := a.RootMotion && valid(a.Skeleton, a.RootJoint)
	a.Skeleton.Reset(a.pose)
	if a.current.clip != nil {
		a.current.clip.Sample(a.current.time, a.pose)
		if rooted {
			a.inPlace(a.current, a.pose)
		}
	}
	if a.fadeTime > 0 && a.fade < a.fadeTime {
		a.Skeleton.Reset(a.from)
		if a.previous.clip != nil {
			a.previous.clip.Sample(a.previous.time, a.from)
			if rooted {
				a.inPlace(a.previous, a.from)
			}
		}
		Blend(a.from, a.pose, a.fade/a.fadeTime, a.pose)
	}
	a.globals = a.Skeleton.Globals(a.pose, a.globals)
	for _, c := range a.Constraints {
		c.Apply(a.Skeleton, a.pose, a.globals)
	}
}

// Skinned is the component of a mesh deformed by an Animator. Model.Mesh
// must be its own copy of Bind: the skinning system rewrites it every frame.
type Skinned struct {
	Animator   *Animator
	Skin       *Skin
	Bind       *mesh.Data
	Influences []Influence
	// Static shows the bind pose, skipping deformation.
	Static bool

	palette []geom.Mat4
}

func (s *Skinned) GetSkinned() *Skinned { return s }

// Apply deforms out from the animator's current joints. It reports false
// when there is nothing to apply yet.
func (s *Skinned) Apply(out *mesh.Data) bool {
	if s.Animator == nil || s.Bind == nil || out == nil || len(out.Vertices) != len(s.Bind.Vertices) {
		return false
	}
	globals := s.Animator.Globals()
	if len(globals) == 0 {
		return false
	}
	if s.Static {
		copy(out.Vertices, s.Bind.Vertices)
	} else {
		s.palette = s.Skin.Palette(globals, s.palette)
		Deform(s.Bind, s.Influences, s.palette, out)
	}
	out.Touch()
	return true
}

// NewSkinned returns a skinned component and the mesh to draw it with.
func NewSkinned(a *Animator, skin *Skin, bind *mesh.Data, influences []Influence) (Skinned, *mesh.Data) {
	return Skinned{Animator: a, Skin: skin, Bind: bind, Influences: influences}, bind.Clone()
}

type Animated interface{ GetAnimator() *Animator }

type Socketed interface {
	scene.Transformable
	GetSocket() *Socket
}

type SkinnedDrawable interface {
	render.Drawable
	GetSkinned() *Skinned
}

// AnimatorSystem advances every Animator late in Update, after gameplay
// picked the clips.
type AnimatorSystem struct{ world.Group[Animated] }

func (s *AnimatorSystem) Priority() int { return -500 }

func (s *AnimatorSystem) Update(frame world.Frame) {
	dt := frame.DT
	for e := range s.All() {
		e.GetAnimator().Update(dt)
	}
}

// SkinSystem deforms skinned meshes from their animator's joints.
type SkinSystem struct{ world.Group[SkinnedDrawable] }

func (s *SkinSystem) Priority() int { return -600 }

func (s *SkinSystem) Update(world.Frame) {
	for e := range s.All() {
		e.GetSkinned().Apply(e.GetModel().Mesh)
	}
}

// SocketSystem moves socketed entities onto their joints.
type SocketSystem struct{ world.Group[Socketed] }

func (s *SocketSystem) Priority() int { return -550 }

func (s *SocketSystem) Update(world.Frame) {
	for e := range s.All() {
		m, ok := e.GetSocket().Matrix()
		if !ok {
			continue
		}
		t := e.GetTransform()
		t.Position = m.Translation()
		t.Rotation = RotationOf(m)
		t.Scale = geom.V3(geom.V3(m[0], m[1], m[2]).Len(), geom.V3(m[4], m[5], m[6]).Len(), geom.V3(m[8], m[9], m[10]).Len())
	}
}

// Plugin installs the animation, socket and skinning systems.
type Plugin struct{}

func (Plugin) Build(a *world.World) { a.AddSystems(&AnimatorSystem{}, &SocketSystem{}, &SkinSystem{}) }
