package anim //nolint:testpackage // white-box: shares the rest, arm and bend fixtures of anim_test.go

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/world"
)

// limb is a straight arm along +X with a tilted root, so local and model
// frames differ.
func limb() *Skeleton {
	tilt := TRS{T: geom.V3(0, 1, 0), R: geom.QuatAxisAngle(geom.V3(0, 1, 0), 0.4), S: geom.V3(1, 1, 1)}
	return &Skeleton{Joints: []Joint{
		{Name: "body", Parent: -1, Rest: tilt},
		{Name: "shoulder", Parent: 0, Rest: rest(geom.V3(0.2, 0, 0))},
		{Name: "elbow", Parent: 1, Rest: rest(geom.V3(1, 0, 0))},
		{Name: "hand", Parent: 2, Rest: rest(geom.V3(1, 0, 0))},
	}}
}

func TestTwoBoneIKReachesTheTargetAndKeepsBoneLengths(t *testing.T) {
	for _, target := range []geom.Vec3{geom.V3(0.5, 2.2, 0.3), geom.V3(-0.6, 1.4, 1), geom.V3(0.9, 0.2, -0.5)} {
		s := limb()
		ik := &TwoBoneIK{Upper: 1, Lower: 2, End: 3, Target: target, Pole: geom.V3(0, 0, -3), Weight: 1}
		a := &Animator{Skeleton: s, Constraints: []Constraint{ik}}
		a.Update(0)
		g := a.Globals()
		if hand := g[3].Translation(); !hand.ApproxEq(target, 2e-3) {
			t.Fatalf("hand at %v, want %v", hand, target)
		}
		upper, lower := g[2].Translation().Dist(g[1].Translation()), g[3].Translation().Dist(g[2].Translation())
		if geom.Abs(upper-1) > 1e-3 || geom.Abs(lower-1) > 1e-3 {
			t.Fatalf("bones stretched: %v %v", upper, lower)
		}
	}
	s := limb()
	far := geom.V3(10, 1, 0)
	ik := &TwoBoneIK{Upper: 1, Lower: 2, End: 3, Target: far, Pole: geom.V3(0, 0, -3), Weight: 1}
	a := &Animator{Skeleton: s, Constraints: []Constraint{ik}}
	a.Update(0)
	shoulder, hand := a.Globals()[1].Translation(), a.Globals()[3].Translation()
	if d := far.Sub(shoulder).Normalize().Dot(hand.Sub(shoulder).Normalize()); d < 0.999 {
		t.Fatalf("out of reach, the arm points at the target: %v", d)
	}
	off := &TwoBoneIK{Upper: 1, Lower: 2, End: 3, Target: far}
	a.Constraints = []Constraint{off}
	a.Update(0)
	if a.Globals()[3].Translation().Dist(s.Globals(s.RestPose(), nil)[3].Translation()) > 1e-5 {
		t.Fatal("weight 0 leaves the pose alone")
	}
}

func TestAimPointsAnAxisAndRollsTowardTheHint(t *testing.T) {
	s := limb()
	aim := &Aim{
		Joint: 3, Axis: geom.V3(0, 1, 0), Up: geom.V3(1, 0, 0),
		Direction: geom.V3(0, 0, -1), UpHint: geom.V3(0, 1, 0), Weight: 1,
	}
	a := &Animator{Skeleton: s, Constraints: []Constraint{aim}}
	a.Update(0)
	r := RotationOf(a.Globals()[3])
	if got := r.Rotate(geom.V3(0, 1, 0)); !got.ApproxEq(geom.V3(0, 0, -1), 1e-4) {
		t.Fatalf("axis %v", got)
	}
	if got := r.Rotate(geom.V3(1, 0, 0)); !got.ApproxEq(geom.V3(0, 1, 0), 1e-4) {
		t.Fatalf("up %v", got)
	}
	if q := FromTo(geom.V3(1, 0, 0), geom.V3(-1, 0, 0)); !q.Rotate(geom.V3(1, 0, 0)).ApproxEq(geom.V3(-1, 0, 0), 1e-5) {
		t.Fatal("opposite directions")
	}
}

type holder struct {
	*world.BasicEntity
	scene.Transform
	Animator
}

type item struct {
	*world.BasicEntity
	scene.Transform
	Socket
}

func TestSocketsFollowTheirJoint(t *testing.T) {
	a := world.New().AddPlugins(render.Plugin{}, Plugin{})
	h := &holder{
		BasicEntity: world.NewBasic(),
		Transform:   scene.At(5, 0, 0),
		Animator:    Animator{Skeleton: arm(), Clips: []*Clip{bend()}},
	}
	it := &item{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0)}
	it.Socket = Socket{Animator: &h.Animator, Joint: 1, Offset: TRS{T: geom.V3(0, 0.5, 0)}}
	a.AddEntities(h, it)
	scene.Attach(a, h, it)
	h.PlayOnce("bend", 0)
	for range 10 {
		a.Tick(0.2)
	}
	if p := it.WorldPosition(); !p.ApproxEq(geom.V3(5-1.5, 0, 0), 1e-3) {
		t.Fatalf("socket at %v", p)
	}
	if up := it.Rotation.Rotate(geom.V3(0, 1, 0)); !up.ApproxEq(geom.V3(-1, 0, 0), 1e-3) {
		t.Fatalf("socket rotates with the joint: %v", up)
	}
	var empty Socket
	if _, ok := empty.Matrix(); ok {
		t.Fatal("a socket without animator has no matrix")
	}
}

func TestTurnFollowAndUpTowardCompose(t *testing.T) {
	s := limb()
	turn := &Turn{Joint: 0, Rotation: geom.QuatAxisAngle(geom.Up, geom.Pi/2), Weight: 1}
	a := &Animator{Skeleton: s, Constraints: []Constraint{turn}}
	a.Update(0)
	before := s.Globals(s.RestPose(), nil)
	if got := RotationOf(a.Globals()[0]); !got.ApproxEq(turn.Rotation.Mul(RotationOf(before[0])), 1e-4) {
		t.Fatalf("turn adds a model-space rotation: %v", got)
	}
	if a.Globals()[0].Translation() != before[0].Translation() {
		t.Fatal("turning keeps the joint in place")
	}

	// A second limb reaching for the first hand, offset along it.
	s.Joints = append(s.Joints,
		Joint{Name: "shoulder2", Parent: 0, Rest: rest(geom.V3(-0.2, 0, 0))},
		Joint{Name: "elbow2", Parent: 4, Rest: rest(geom.V3(-1, 0, 0))},
		Joint{Name: "hand2", Parent: 5, Rest: rest(geom.V3(-1, 0, 0))},
	)
	grip := &Socket{Joint: 3, Offset: TRS{T: geom.V3(-0.1, 0, 0)}}
	main := &TwoBoneIK{Upper: 1, Lower: 2, End: 3, Target: geom.V3(0.3, 1.5, 0.9), Pole: geom.V3(0, 0, -3), Weight: 1}
	off := &TwoBoneIK{Upper: 4, Lower: 5, End: 6, Follow: grip, Pole: geom.V3(0, 0, -3), Weight: 1}
	wrist := &Aim{
		Joint: 6, Axis: geom.V3(0, 1, 0), Up: geom.V3(-1, 0, 0),
		Direction: geom.V3(0, 1, 0), UpToward: &Socket{Joint: 5}, Weight: 1,
	}
	a = &Animator{Skeleton: s, Constraints: []Constraint{main, off, wrist}}
	a.Update(0)
	g := a.Globals()
	want, _ := grip.In(g)
	if got := g[6].Translation(); !got.ApproxEq(want.Translation(), 3e-3) {
		t.Fatalf("second hand %v, want the grip %v", got, want.Translation())
	}
	toElbow := g[5].Translation().Sub(g[6].Translation())
	up := RotationOf(g[6]).Rotate(geom.V3(-1, 0, 0))
	flat := toElbow.Sub(geom.V3(0, toElbow.Y, 0)).Normalize()
	if up.Dot(flat) < 0.99 {
		t.Fatalf("the wrist rolls toward the elbow: up %v elbow %v", up, flat)
	}
}

func TestShiftMovesAJointAndItsChildrenInModelSpace(t *testing.T) {
	s := limb()
	s.Joints[0].Rest.S = geom.V3(2, 2, 2)
	rest := s.Globals(s.RestPose(), nil)
	shift := &Shift{Joint: 1, Offset: geom.V3(0, -0.5, 0.25), Weight: 1}
	a := &Animator{Skeleton: s, Constraints: []Constraint{shift}}
	a.Update(0)
	for _, j := range []int{1, 3} {
		if d := a.Globals()[j].Translation().Sub(rest[j].Translation()); !d.ApproxEq(geom.V3(0, -0.5, 0.25), 1e-4) {
			t.Fatalf("joint %d moved by %v under a scaled, turned parent", j, d)
		}
	}
	shift.Weight = 0.5
	a.Update(0)
	if d := a.Globals()[1].Translation().Sub(rest[1].Translation()); !d.ApproxEq(geom.V3(0, -0.25, 0.125), 1e-4) {
		t.Fatalf("half weight moves half way: %v", d)
	}
}

func TestOverlayPosesOnlyItsJoints(t *testing.T) {
	s := limb()
	wave := &Clip{Name: "wave", Duration: 1, Channels: []Channel{
		{Joint: 2, Path: Rotation, Times: []float32{0, 1}, Values: []geom.Vec4{geom.V4(0, 0, 0, 1), quarterTurn()}},
		{Joint: 1, Path: Rotation, Times: []float32{0, 1}, Values: []geom.Vec4{geom.V4(0, 0, 0, 1), quarterTurn()}},
	}}
	o := &Overlay{Clip: wave, Time: 1, Joints: []int{2}, Weight: 1}
	a := &Animator{Skeleton: s, Constraints: []Constraint{o}}
	a.Update(0)
	q := geom.QuatAxisAngle(geom.V3(0, 0, 1), geom.Pi/2)
	if !a.Pose()[2].R.ApproxEq(q, 1e-4) || !a.Pose()[1].R.ApproxEq(geom.QuatIdent(), 1e-6) {
		t.Fatalf("only the masked joint takes the clip: %v %v", a.Pose()[2].R, a.Pose()[1].R)
	}
	o.Weight = 0.5
	a.Update(0)
	if !a.Pose()[2].R.ApproxEq(geom.QuatAxisAngle(geom.V3(0, 0, 1), geom.Pi/4), 1e-4) {
		t.Fatal("weight blends")
	}
}

func TestUntwistRollsTheForearmAndKeepsTheHand(t *testing.T) {
	s := limb()
	wring := &Turn{Joint: 3, Rotation: geom.QuatAxisAngle(geom.V3(1, 0, 0), 2.6), Weight: 1}
	ik := &TwoBoneIK{Upper: 1, Lower: 2, End: 3, Target: geom.V3(0.6, 2, 0.5), Pole: geom.V3(0, 0, -3), Weight: 1}
	a := &Animator{Skeleton: s, Constraints: []Constraint{ik, wring}}
	a.Update(0)
	before := append([]geom.Mat4(nil), a.Globals()...)
	u := &Untwist{Joint: 2, Child: 3, Share: 0.75, Weight: 1}
	twisted, _ := u.Twist(s, before)
	if geom.Abs(twisted) < 1 {
		t.Fatalf("setup should wring the wrist, twist %v", twisted)
	}
	a.Constraints = append(a.Constraints, u)
	a.Update(0)
	g := a.Globals()
	if left, _ := u.Twist(s, g); geom.Abs(left-twisted*0.25) > 0.02 {
		t.Fatalf("the wrist keeps a quarter of %v, has %v", twisted, left)
	}
	if hand, was := RotationOf(g[3]), RotationOf(before[3]); !hand.ApproxEq(was, 1e-3) &&
		!hand.ApproxEq(was.Mul(geom.Quat{W: -1}), 1e-3) {
		t.Fatal("the hand keeps its orientation")
	}
	for _, j := range []int{2, 3} {
		if !g[j].Translation().ApproxEq(before[j].Translation(), 1e-4) {
			t.Fatalf("joint %d moved", j)
		}
	}
}

func TestAimAlongFollowsAnotherJoint(t *testing.T) {
	s := limb()
	turn := &Turn{Joint: 1, Rotation: geom.QuatAxisAngle(geom.V3(0, 0, 1), 0.7), Weight: 1}
	aim := &Aim{Joint: 3, Axis: geom.V3(1, 0, 0), Along: &Socket{Joint: 2}, AlongAxis: geom.V3(0, 1, 0), Weight: 1}
	a := &Animator{Skeleton: s, Constraints: []Constraint{turn, aim}}
	a.Update(0)
	g := a.Globals()
	want := RotationOf(g[2]).Rotate(geom.V3(0, 1, 0))
	if got := RotationOf(g[3]).Rotate(geom.V3(1, 0, 0)); !got.ApproxEq(want, 1e-4) {
		t.Fatalf("hand axis %v, want the elbow's Y %v", got, want)
	}
}

func TestAdjustOffsetsAJointOnTopOfClips(t *testing.T) {
	s := limb()
	a := &Animator{Skeleton: s, Clips: []*Clip{bend()}}
	a.PlayOnce("bend", 0)
	a.Update(1)
	before := a.Globals()[3].Translation()
	adj := &Adjust{
		Joint:  2,
		Offset: TRS{T: geom.V3(0.5, 0, 0), R: geom.QuatAxisAngle(geom.V3(0, 0, 1), geom.Pi/2)},
		Weight: 1,
	}
	a.Constraints = []Constraint{adj}
	a.Update(0)
	g := a.Globals()
	if g[3].Translation().ApproxEq(before, 1e-3) {
		t.Fatal("the adjusted elbow moves the hand")
	}
	elbowLocal := a.Pose()[2]
	if !elbowLocal.T.ApproxEq(geom.V3(1.5, 0, 0), 1e-4) ||
		!elbowLocal.R.ApproxEq(geom.QuatAxisAngle(geom.V3(0, 0, 1), geom.Pi/2), 1e-4) || elbowLocal.S != geom.V3(1, 1, 1) {
		t.Fatalf("offset in parent space: %+v", elbowLocal)
	}
	adj.Weight = 0
	a.Update(0)
	if !a.Globals()[3].Translation().ApproxEq(before, 1e-4) {
		t.Fatal("weight 0 leaves the clip alone")
	}
}
