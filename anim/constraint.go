package anim

import "github.com/merlindorin/ember/geom"

// Constraint adjusts a sampled pose before it is resolved, in model space:
// inverse kinematics, aiming, look-at. Animators apply theirs in order.
type Constraint interface {
	Apply(s *Skeleton, p Pose, globals []geom.Mat4)
}

// TwoBoneIK bends a limb (upper, lower, end: shoulder, elbow, hand) so the
// end joint reaches Target, bending toward Pole. Positions are in model
// space; Weight 0 disables it. With Follow set, the target is that socket
// as posed by the constraints before this one: a second hand on a handle.
type TwoBoneIK struct {
	Upper, Lower, End int
	Target, Pole      geom.Vec3
	Follow            *Socket
	Weight            float32
}

// Aim rotates a joint so its local Axis points along Direction, rolling
// its local Up toward UpHint, or toward the UpToward socket as posed at
// solve time (an elbow, to keep a wrist from twisting). Directions are in
// model space.
type Aim struct {
	Joint     int
	Axis, Up  geom.Vec3
	Direction geom.Vec3
	UpHint    geom.Vec3
	UpToward  *Socket
	// Along, when set, replaces Direction with AlongAxis as posed on that
	// socket at solve time: a second hand following a handle.
	Along     *Socket
	AlongAxis geom.Vec3
	Weight    float32
}

// Turn rotates a joint by a model-space rotation about its own position:
// twisting or bending a spine on top of a clip.
type Turn struct {
	Joint    int
	Rotation geom.Quat
	Weight   float32
}

// Adjust offsets a joint in its parent's space on top of whatever the
// clips do: moving a hand socket, fixing a shoulder for a whole rig.
// Offset.T is added, Offset.R applied after the joint's own rotation and
// Offset.S multiplied (zero scale means 1).
type Adjust struct {
	Joint  int
	Offset TRS
	Weight float32
}

func (a *Adjust) Apply(s *Skeleton, p Pose, globals []geom.Mat4) {
	if a.Weight <= 0 || !valid(s, a.Joint) {
		return
	}
	w := geom.Clamp01(a.Weight)
	j := &p[a.Joint]
	j.T = j.T.Add(a.Offset.T.Scale(w))
	if r := a.Offset.R; r != (geom.Quat{}) {
		j.R = j.R.Slerp(j.R.Mul(r.Normalize()), w)
	}
	if sc := a.Offset.S; sc != (geom.Vec3{}) {
		j.S = j.S.Mul(geom.V3(1, 1, 1).Lerp(sc, w))
	}
	s.Globals(p, globals)
}

// Overlay applies a clip's pose at Time to a subset of joints: closed
// fists from a punch over any clip, an upper body over legs.
type Overlay struct {
	Clip   *Clip
	Time   float32
	Joints []int
	Weight float32

	scratch Pose
}

func (o *Overlay) Apply(s *Skeleton, p Pose, globals []geom.Mat4) {
	if o.Weight <= 0 || o.Clip == nil || len(o.Joints) == 0 {
		return
	}
	o.scratch = resize(o.scratch, len(s.Joints))
	copy(o.scratch, p)
	o.Clip.Sample(o.Time, o.scratch)
	w := geom.Clamp01(o.Weight)
	for _, j := range o.Joints {
		if valid(s, j) {
			p[j] = p[j].Lerp(o.scratch[j], w)
		}
	}
	s.Globals(p, globals)
}

// Untwist hands part of a joint's twist about its bone over to the
// parent bone, like a forearm rolling to turn the hand over, so the wrist
// never wrings. Child keeps its model-space orientation and Joint rolls
// about its own axis, so positions do not move. Twist is measured from
// the child's rest orientation relative to Joint.
type Untwist struct {
	Joint, Child int
	// Share is how much of the twist Joint takes (0..1).
	Share  float32
	Weight float32
}

// Twist returns how far Child is twisted about the bone from Joint, in
// radians, and the bone axis in model space.
func (u *Untwist) Twist(s *Skeleton, globals []geom.Mat4) (float32, geom.Vec3) {
	axis := globals[u.Child].Translation().Sub(globals[u.Joint].Translation()).Normalize()
	neutral := RotationOf(globals[u.Joint]).Mul(s.Joints[u.Child].Rest.R)
	delta := RotationOf(globals[u.Child]).Mul(neutral.Conjugate())
	tw := twistAbout(delta, axis)
	angle := 2 * geom.Acos(geom.Clamp(tw.W, -1, 1))
	if geom.V3(tw.X, tw.Y, tw.Z).Dot(axis) < 0 {
		angle = -angle
	}
	return geom.WrapAngle(angle), axis
}

func twistAbout(q geom.Quat, axis geom.Vec3) geom.Quat {
	d := geom.V3(q.X, q.Y, q.Z).Dot(axis)
	t := geom.Quat{X: axis.X * d, Y: axis.Y * d, Z: axis.Z * d, W: q.W}
	if t.Dot(t) < 1e-12 {
		return geom.QuatIdent()
	}
	t = t.Normalize()
	if t.W < 0 {
		t = geom.Quat{X: -t.X, Y: -t.Y, Z: -t.Z, W: -t.W}
	}
	return t
}

func (u *Untwist) Apply(s *Skeleton, p Pose, globals []geom.Mat4) {
	if u.Weight <= 0 || u.Share <= 0 || !valid(s, u.Joint, u.Child) || s.Joints[u.Child].Parent != u.Joint {
		return
	}
	angle, axis := u.Twist(s, globals)
	child := RotationOf(globals[u.Child])
	roll := geom.QuatAxisAngle(axis, angle*geom.Clamp01(u.Share)*geom.Clamp01(u.Weight))
	s.setGlobalRotation(p, globals, u.Joint, roll.Mul(RotationOf(globals[u.Joint])), 1)
	s.Globals(p, globals)
	s.setGlobalRotation(p, globals, u.Child, child, 1)
	s.Globals(p, globals)
}

// Shift moves a joint by a model-space offset, carrying its children:
// dropping the hips into a stance.
type Shift struct {
	Joint  int
	Offset geom.Vec3
	Weight float32
}

func (sh *Shift) Apply(s *Skeleton, p Pose, globals []geom.Mat4) {
	if sh.Weight <= 0 || !valid(s, sh.Joint) || sh.Offset == (geom.Vec3{}) {
		return
	}
	d := sh.Offset.Scale(geom.Clamp01(sh.Weight))
	if parent := s.Joints[sh.Joint].Parent; parent >= 0 {
		d = globals[parent].Inverse().MulDir(d)
	}
	p[sh.Joint].T = p[sh.Joint].T.Add(d)
	s.Globals(p, globals)
}

func (t *Turn) Apply(s *Skeleton, p Pose, globals []geom.Mat4) {
	if t.Weight <= 0 || !valid(s, t.Joint) || t.Rotation == (geom.Quat{}) {
		return
	}
	s.setGlobalRotation(p, globals, t.Joint, t.Rotation.Normalize().Mul(RotationOf(globals[t.Joint])), t.Weight)
	s.Globals(p, globals)
}

// RotationOf returns the rotation part of a matrix, ignoring scale.
func RotationOf(m geom.Mat4) geom.Quat {
	for c := range 3 {
		l := geom.V3(m[c*4], m[c*4+1], m[c*4+2]).Len()
		if l > 0 {
			m[c*4], m[c*4+1], m[c*4+2] = m[c*4]/l, m[c*4+1]/l, m[c*4+2]/l
		}
	}
	return m.Quat().Normalize()
}

// FromTo returns the shortest rotation taking direction a onto b.
func FromTo(a, b geom.Vec3) geom.Quat {
	a, b = a.Normalize(), b.Normalize()
	d := a.Dot(b)
	if d > 0.99999 {
		return geom.QuatIdent()
	}
	if d < -0.99999 {
		axis := geom.Right.Cross(a)
		if axis.LenSq() < 1e-6 {
			axis = geom.Up.Cross(a)
		}
		return geom.QuatAxisAngle(axis.Normalize(), geom.Pi)
	}
	c := a.Cross(b)
	return geom.Quat{X: c.X, Y: c.Y, Z: c.Z, W: 1 + d}.Normalize()
}

func (s *Skeleton) parentRotation(globals []geom.Mat4, j int) geom.Quat {
	if p := s.Joints[j].Parent; p >= 0 {
		return RotationOf(globals[p])
	}
	return geom.QuatIdent()
}

// setGlobalRotation gives joint j the model-space rotation q.
func (s *Skeleton) setGlobalRotation(p Pose, globals []geom.Mat4, j int, q geom.Quat, weight float32) {
	local := s.parentRotation(globals, j).Conjugate().Mul(q).Normalize()
	p[j].R = p[j].R.Slerp(local, geom.Clamp01(weight))
}

// Apply solves the limb analytically: the triangle formed by the two
// bones and the target fixes the shoulder and elbow angles, bending toward
// the pole without moving the shoulder-to-hand direction, then the whole
// chain swings onto the target.
func (ik *TwoBoneIK) Apply(s *Skeleton, p Pose, globals []geom.Mat4) {
	if ik.Weight <= 0 || !valid(s, ik.Upper, ik.Lower, ik.End) {
		return
	}
	a, b, c := globals[ik.Upper].Translation(), globals[ik.Lower].Translation(), globals[ik.End].Translation()
	t := ik.Target
	if ik.Follow != nil {
		m, ok := ik.Follow.In(globals)
		if !ok {
			return
		}
		t = m.Translation()
	}
	lab, lcb := b.Dist(a), b.Dist(c)
	if lab < 1e-6 || lcb < 1e-6 {
		return
	}
	lat := geom.Clamp(t.Dist(a), 1e-4, (lab+lcb)*0.9999)
	angle := func(x, y geom.Vec3) float32 { return geom.Acos(x.Normalize().Dot(y.Normalize())) }
	acab0 := angle(c.Sub(a), b.Sub(a))
	babc0 := angle(a.Sub(b), c.Sub(b))
	acat0 := angle(c.Sub(a), t.Sub(a))
	acab1 := geom.Acos((lcb*lcb - lab*lab - lat*lat) / (-2 * lab * lat))
	babc1 := geom.Acos((lat*lat - lab*lab - lcb*lcb) / (-2 * lab * lcb))

	bend := c.Sub(a).Cross(ik.Pole.Sub(a))
	if bend.LenSq() < 1e-10 {
		bend = c.Sub(a).Cross(b.Sub(a))
	}
	if bend.LenSq() < 1e-10 {
		return
	}
	bend = bend.Normalize()
	ga, gb := RotationOf(globals[ik.Upper]).Conjugate(), RotationOf(globals[ik.Lower]).Conjugate()
	r0 := geom.QuatAxisAngle(ga.Rotate(bend), acab1-acab0)
	r1 := geom.QuatAxisAngle(gb.Rotate(bend), babc1-babc0)
	r2 := geom.QuatIdent()
	if swing := c.Sub(a).Cross(t.Sub(a)); swing.LenSq() > 1e-10 {
		r2 = geom.QuatAxisAngle(ga.Rotate(swing.Normalize()), acat0)
	}
	w := geom.Clamp01(ik.Weight)
	p[ik.Upper].R = p[ik.Upper].R.Slerp(p[ik.Upper].R.Mul(r2.Mul(r0)).Normalize(), w)
	p[ik.Lower].R = p[ik.Lower].R.Slerp(p[ik.Lower].R.Mul(r1).Normalize(), w)
	s.Globals(p, globals)
}

func (aim *Aim) Apply(s *Skeleton, p Pose, globals []geom.Mat4) {
	dir := aim.Direction
	if aim.Along != nil {
		m, ok := aim.Along.In(globals)
		if !ok {
			return
		}
		dir = RotationOf(m).Rotate(aim.AlongAxis)
	}
	if aim.Weight <= 0 || !valid(s, aim.Joint) || dir.LenSq() == 0 || aim.Axis.LenSq() == 0 {
		return
	}
	g := RotationOf(globals[aim.Joint])
	dir = dir.Normalize()
	q := FromTo(g.Rotate(aim.Axis), dir).Mul(g)
	hint := aim.UpHint
	if aim.UpToward != nil {
		if m, ok := aim.UpToward.In(globals); ok {
			hint = m.Translation().Sub(globals[aim.Joint].Translation())
		}
	}
	if hint.LenSq() > 0 && aim.Up.LenSq() > 0 {
		up := q.Rotate(aim.Up)
		want := hint.Sub(dir.Scale(hint.Dot(dir)))
		have := up.Sub(dir.Scale(up.Dot(dir)))
		if want.LenSq() > 1e-8 && have.LenSq() > 1e-8 {
			have, want = have.Normalize(), want.Normalize()
			roll := geom.Atan2(have.Cross(want).Dot(dir), have.Dot(want))
			q = geom.QuatAxisAngle(dir, roll).Mul(q)
		}
	}
	s.setGlobalRotation(p, globals, aim.Joint, q, aim.Weight)
	s.Globals(p, globals)
}

func valid(s *Skeleton, joints ...int) bool {
	for _, j := range joints {
		if j < 0 || j >= len(s.Joints) {
			return false
		}
	}
	return true
}

// Socket makes an entity follow a joint of an Animator, offset in the
// joint's space. The entity must be a child of the animator's entity
// (scene.Attach) so the joint's model space is its parent space.
type Socket struct {
	Animator *Animator
	Joint    int
	Offset   TRS
}

func (s *Socket) GetSocket() *Socket { return s }

// Matrix returns the socket transform in the animator's model space.
func (s *Socket) Matrix() (geom.Mat4, bool) {
	if s.Animator == nil {
		return geom.Mat4{}, false
	}
	return s.In(s.Animator.Globals())
}

// In returns the socket transform for given model-space joint matrices.
func (s *Socket) In(g []geom.Mat4) (geom.Mat4, bool) {
	if s.Joint < 0 || s.Joint >= len(g) {
		return geom.Mat4{}, false
	}
	off := s.Offset
	if off.S == (geom.Vec3{}) {
		off.S = geom.V3(1, 1, 1)
	}
	if off.R == (geom.Quat{}) {
		off.R = geom.QuatIdent()
	}
	return g[s.Joint].Mul(off.Mat4()), true
}
