package anim //nolint:testpackage // white-box: samples Channel.at and reads the animator's current time

import (
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/mesh"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/world"
)

func rest(t geom.Vec3) TRS { return TRS{T: t, R: geom.QuatIdent(), S: geom.V3(1, 1, 1)} }

// arm is a root with a child one unit up the Y axis.
func arm() *Skeleton {
	return &Skeleton{Joints: []Joint{
		{Name: "shoulder", Parent: -1, Rest: rest(geom.Vec3{})},
		{Name: "elbow", Parent: 0, Rest: rest(geom.V3(0, 1, 0))},
	}}
}

func quarterTurn() geom.Vec4 {
	q := geom.QuatAxisAngle(geom.V3(0, 0, 1), geom.Pi/2)
	return geom.V4(q.X, q.Y, q.Z, q.W)
}

// bend rotates the shoulder a quarter turn around Z over one second.
func bend() *Clip {
	return &Clip{Name: "bend", Duration: 1, Channels: []Channel{{
		Joint: 0, Path: Rotation, Times: []float32{0, 1},
		Values: []geom.Vec4{geom.V4(0, 0, 0, 1), quarterTurn()},
	}}}
}

func TestGlobalsFollowTheHierarchy(t *testing.T) {
	s := arm()
	if s.Find("elbow") != 1 || s.Find("knee") != -1 {
		t.Fatal("find by name")
	}
	p := s.RestPose()
	p[0].R = geom.QuatAxisAngle(geom.V3(0, 0, 1), geom.Pi/2)
	g := s.Globals(p, nil)
	if elbow := g[1].Translation(); !elbow.ApproxEq(geom.V3(-1, 0, 0), 1e-5) {
		t.Fatalf("rotating the shoulder swings the elbow: %v", elbow)
	}
}

func TestSamplingInterpolates(t *testing.T) {
	ch := Channel{Path: Translation, Times: []float32{0, 1, 2},
		Values: []geom.Vec4{geom.V4(0, 0, 0, 0), geom.V4(10, 0, 0, 0), geom.V4(10, 20, 0, 0)}}
	for _, c := range []struct {
		t    float32
		want geom.Vec4
	}{{-1, geom.V4(0, 0, 0, 0)}, {0.5, geom.V4(5, 0, 0, 0)}, {1.5, geom.V4(10, 10, 0, 0)}, {9, geom.V4(10, 20, 0, 0)}} {
		if got := ch.at(c.t); got != c.want {
			t.Fatalf("linear at %v: %v", c.t, got)
		}
	}
	ch.Interpolation = Step
	if got := ch.at(0.9); got != geom.V4(0, 0, 0, 0) {
		t.Fatalf("step holds the key: %v", got)
	}
	cubic := Channel{Path: Translation, Interpolation: CubicSpline, Times: []float32{0, 1},
		Values: []geom.Vec4{{}, geom.V4(0, 0, 0, 0), {}, {}, geom.V4(4, 0, 0, 0), {}}}
	if got := cubic.at(0.5); geom.Abs(got.X-2) > 1e-5 {
		t.Fatalf("cubic with flat tangents is smoothstep: %v", got)
	}
	rot := bend().Channels[0]
	q := rot.at(0.5)
	half := geom.QuatAxisAngle(geom.V3(0, 0, 1), geom.Pi/4)
	if !geom.Quat(q).ApproxEq(half, 1e-4) {
		t.Fatalf("rotation slerps: %v", q)
	}
}

func TestAnimatorLoopsPlaysOnceAndCrossfades(t *testing.T) {
	a := &Animator{Skeleton: arm(), Clips: []*Clip{bend(), {Name: "still", Duration: 1}}}
	a.Update(0)
	if len(a.Globals()) != 2 || a.Globals()[1].Translation() != geom.V3(0, 1, 0) {
		t.Fatal("rest pose before any clip")
	}
	if a.Play("nope", 0) {
		t.Fatal("unknown clip")
	}
	a.Play("bend", 0)
	a.Update(1.25)
	if _, tm := a.Clip(); geom.Abs(tm-0.25) > 1e-5 {
		t.Fatalf("looping wraps: %v", tm)
	}
	a.PlayOnce("bend", 0)
	a.Update(3)
	if _, tm := a.Clip(); tm != 1 {
		t.Fatalf("once holds the end: %v", tm)
	}
	if e := a.Globals()[1].Translation(); !e.ApproxEq(geom.V3(-1, 0, 0), 1e-4) {
		t.Fatalf("end of bend: %v", e)
	}
	a.Play("still", 1)
	a.Update(0.5)
	if e := a.Globals()[1].Translation(); !e.ApproxEq(geom.V3(-geom.Sin(geom.Pi/4), geom.Cos(geom.Pi/4), 0), 1e-3) {
		t.Fatalf("halfway through the crossfade: %v", e)
	}
	a.Update(1)
	if e := a.Globals()[1].Translation(); !e.ApproxEq(geom.V3(0, 1, 0), 1e-4) {
		t.Fatalf("after the fade: %v", e)
	}
	a.Paused = true
	before := a.current.time
	a.Update(5)
	if a.current.time != before {
		t.Fatal("paused")
	}
}

func TestDeformBlendsWeightedJoints(t *testing.T) {
	s := arm()
	skin := &Skin{Joints: []int{0, 1}, InverseBind: []geom.Mat4{geom.Ident(), geom.Translate(geom.V3(0, -1, 0))}}
	bind := &mesh.Data{Vertices: []mesh.Vertex{
		{Pos: geom.V3(0, 2, 0), Normal: geom.V3(1, 0, 0)},
		{Pos: geom.V3(0, 2, 0), Normal: geom.V3(1, 0, 0)},
		{Pos: geom.V3(5, 5, 5)},
	}}
	inf := []Influence{
		{Joints: [4]uint16{1}, Weights: [4]float32{1}},
		{Joints: [4]uint16{0, 1}, Weights: [4]float32{1, 1}},
		{},
	}
	p := s.RestPose()
	p[0].R = geom.QuatAxisAngle(geom.V3(0, 0, 1), geom.Pi/2)
	out := bind.Clone()
	Deform(bind, inf, skin.Palette(s.Globals(p, nil), nil), out)
	if !out.Vertices[0].Pos.ApproxEq(geom.V3(-2, 0, 0), 1e-5) || !out.Vertices[0].Normal.ApproxEq(geom.V3(0, 1, 0), 1e-5) {
		t.Fatalf("rigid vertex %+v", out.Vertices[0])
	}
	if !out.Vertices[1].Pos.ApproxEq(geom.V3(-2, 0, 0), 1e-5) {
		t.Fatalf("weights are normalized: %+v", out.Vertices[1])
	}
	if out.Vertices[2].Pos != geom.V3(5, 5, 5) {
		t.Fatal("unweighted vertices stay put")
	}
}

type rig struct {
	*world.BasicEntity
	scene.Transform
	Animator
}

type part struct {
	*world.BasicEntity
	scene.Transform
	render.Model
	Skinned
}

func TestSystemsAnimateAndSkinEntities(t *testing.T) {
	a := world.New().AddPlugins(render.Plugin{}, Plugin{})
	r := &rig{
		BasicEntity: world.NewBasic(),
		Transform:   scene.At(0, 0, 0),
		Animator:    Animator{Skeleton: arm(), Clips: []*Clip{bend()}},
	}
	bind := &mesh.Data{Vertices: []mesh.Vertex{{Pos: geom.V3(0, 2, 0), Color: color.White}}, Indices: []uint32{0, 0, 0}}
	skin := &Skin{Joints: []int{1}, InverseBind: []geom.Mat4{geom.Translate(geom.V3(0, -1, 0))}}
	sk, m := NewSkinned(&r.Animator, skin, bind, []Influence{{Weights: [4]float32{1}}})
	p := &part{world.NewBasic(), scene.At(0, 0, 0), render.Model{Mesh: m}, sk}
	a.AddEntities(r, p)
	r.PlayOnce("bend", 0)
	for range 10 {
		a.Tick(0.2)
	}
	version := m.Version()
	if !m.Vertices[0].Pos.ApproxEq(geom.V3(-2, 0, 0), 1e-4) || version == 0 {
		t.Fatalf("skinned vertex %v", m.Vertices[0].Pos)
	}
	if bind.Vertices[0].Pos != geom.V3(0, 2, 0) {
		t.Fatal("the bind mesh is never modified")
	}
	p.Static = true
	a.Tick(0.1)
	if m.Vertices[0].Pos != geom.V3(0, 2, 0) || m.Version() == version {
		t.Fatal("static shows the bind pose")
	}
}

func TestRootMotionKeepsTheRootInPlaceAndReportsTravel(t *testing.T) {
	walk := &Clip{Name: "walk", Duration: 1, Channels: []Channel{{
		Joint: 0, Path: Translation, Times: []float32{0, 1},
		Values: []geom.Vec4{geom.V4(0, 0, 0, 0), geom.V4(0, 0.1, 2, 0)},
	}}}
	a := &Animator{Skeleton: arm(), Clips: []*Clip{walk, bend()}, RootMotion: true}
	a.Play("walk", 0)
	a.Update(0.25)
	if m := a.Motion(); !m.ApproxEq(geom.V3(0, 0, 0.5), 1e-5) {
		t.Fatalf("motion %v", m)
	}
	if p := a.Globals()[0].Translation(); geom.Abs(p.Z) > 1e-5 || geom.Abs(p.Y-0.025) > 1e-5 {
		t.Fatalf("the root stays put horizontally but keeps its bob: %v", p)
	}
	a.Seek(0.9)
	a.Update(0.2)
	if m := a.Motion(); !m.ApproxEq(geom.V3(0, 0, 0.4), 1e-4) {
		t.Fatalf("travel across the loop wrap: %v", m)
	}
	a.Play("bend", 1)
	a.Update(0.5)
	if m := a.Motion(); !m.ApproxEq(geom.V3(0, 0, 0.5), 1e-4) {
		t.Fatalf("crossfading out of a walk keeps half its travel: %v", m)
	}
	a.Paused = true
	a.Update(1)
	if a.Motion() != (geom.Vec3{}) {
		t.Fatal("no motion while paused")
	}
}

func TestRootMotionMeasuresTheHipsAndLoopsSeamlessly(t *testing.T) {
	// The root slides 1 per loop and the hips, below it, another 0.2 that
	// they snap back from: the case of the pack's run clip.
	s := &Skeleton{Joints: []Joint{
		{Name: "Root", Parent: -1, Rest: rest(geom.Vec3{})},
		{Name: "Hips", Parent: 0, Rest: rest(geom.V3(0, 1, 0))},
	}}
	run := &Clip{Name: "run", Duration: 1, Channels: []Channel{
		{Joint: 0, Path: Translation, Times: []float32{0, 1}, Values: []geom.Vec4{geom.V4(0, 0, 0, 0), geom.V4(0, 0, 1, 0)}},
		{Joint: 1, Path: Translation, Times: []float32{0, 0.5, 1},
			Values: []geom.Vec4{geom.V4(0, 1, 0, 0), geom.V4(0.1, 1, 0.1, 0), geom.V4(0, 1, 0.2, 0)}},
	}}
	a := &Animator{Skeleton: s, Clips: []*Clip{run}, RootMotion: true, RootJoint: 1}
	a.Play("run", 0)
	var last geom.Vec3
	for i := range 20 {
		a.Update(0.1)
		if m := a.Motion(); !m.ApproxEq(geom.V3(0, 0, 0.12), 1e-4) {
			t.Fatalf("frame %d: motion %v, want the loop travel spread evenly", i, m)
		}
		hips := a.Globals()[1].Translation()
		if i > 0 && hips.Dist(last) > 0.06 {
			t.Fatalf("frame %d: the hips jump %v at the loop seam", i, hips.Dist(last))
		}
		last = hips
	}
	a.Seek(0.5)
	a.Update(0)
	if hips := a.Globals()[1].Translation(); !hips.ApproxEq(geom.V3(0.1, 1, 0), 1e-4) {
		t.Fatalf("the sideways sway stays in the pose: %v", hips)
	}
}

func TestPlaySyncedKeepsThePhase(t *testing.T) {
	a := &Animator{Skeleton: arm(), Clips: []*Clip{{Name: "walk", Duration: 1}, {Name: "run", Duration: 0.5}}}
	a.Play("walk", 0)
	a.Update(0.6)
	a.PlaySynced("run", 0.2)
	if c, tm := a.Clip(); c.Name != "run" || geom.Abs(tm-0.3) > 1e-5 {
		t.Fatalf("run starts 60%% in like the walk was: %v", tm)
	}
	a.PlaySynced("run", 0.2)
	if _, tm := a.Clip(); geom.Abs(tm-0.3) > 1e-5 {
		t.Fatal("replaying the current clip changes nothing")
	}
	if a.PlaySynced("swim", 0.2) {
		t.Fatal("unknown clip")
	}
}

func TestClipsLoopOverTheirKeysNotFromZero(t *testing.T) {
	// Keys from 1/30 s like the pack's exports, closed loop (last pose = first).
	s := &Skeleton{Joints: []Joint{
		{Name: "Root", Parent: -1, Rest: rest(geom.Vec3{})},
		{Name: "Hips", Parent: 0, Rest: rest(geom.V3(0, 1, 0))},
	}}
	start := float32(1.0 / 30)
	run := &Clip{Name: "run", Start: start, Duration: start + 0.8, Channels: []Channel{
		{Joint: 0, Path: Translation, Times: []float32{start, start + 0.8},
			Values: []geom.Vec4{geom.V4(0, 0, 0, 0), geom.V4(0, 0, 2, 0)}},
	}}
	a := &Animator{Skeleton: s, Clips: []*Clip{run}, RootMotion: true, RootJoint: 1}
	a.Play("run", 0)
	if _, tm := a.Clip(); tm != start {
		t.Fatalf("playback starts on the first key: %v", tm)
	}
	world := float32(0)
	prev := float32(0)
	for i := range 200 {
		a.Update(1.0 / 60)
		world += a.Motion().Z
		z := world + a.Globals()[1].Translation().Z
		if i > 0 && geom.Abs((z-prev)-2.0/0.8/60) > 1e-4 {
			t.Fatalf("frame %d: the body moved %.4f, want a steady %.4f", i, z-prev, 2.0/0.8/60)
		}
		prev = z
	}
	if c := (&Clip{Start: 0.5, Duration: 1.5}); c.Length() != 1 || c.Phase(1) != 0.5 || c.Phase(9) != 1 {
		t.Fatal("length and phase")
	}
}

func TestRetargetAcrossRestPosesAndProportions(t *testing.T) {
	tall := &Skeleton{Joints: []Joint{
		{Name: "hips", Parent: -1, Rest: rest(geom.V3(0, 1, 0))},
		{Name: "spine", Parent: 0, Rest: rest(geom.V3(0, 0.5, 0))},
		{Name: "tail", Parent: 0, Rest: rest(geom.V3(0, 0, -0.5))},
	}}
	turned := geom.QuatAxisAngle(geom.V3(0, 1, 0), geom.Pi/2)
	short := &Skeleton{Joints: []Joint{
		{Name: "spine", Parent: 1, Rest: TRS{T: geom.V3(0, 0.5, 0), R: turned, S: geom.V3(1, 1, 1)}},
		{Name: "hips", Parent: -1, Rest: rest(geom.V3(0, 0.5, 0))},
	}}
	short.Joints[0].Parent, short.Joints[1].Parent = 1, -1
	bow := geom.QuatAxisAngle(geom.V3(1, 0, 0), 0.5)
	clip := &Clip{Name: "walk", Start: 0.1, Duration: 1, Channels: []Channel{
		{Joint: 0, Path: Translation, Times: []float32{0}, Values: []geom.Vec4{geom.V4(0, 1, 2, 0)}},
		{Joint: 1, Path: Rotation, Times: []float32{0}, Values: []geom.Vec4{geom.V4(bow.X, bow.Y, bow.Z, bow.W)}},
		{Joint: 1, Path: Translation, Times: []float32{0}, Values: []geom.Vec4{geom.V4(0, 0.6, 0, 0)}},
		{Joint: 2, Path: Rotation, Times: []float32{0}, Values: []geom.Vec4{geom.V4(0, 0, 0, 1)}},
	}}
	out := Retarget(clip, tall, short)
	if out.Name != "walk" || out.Start != 0.1 || len(out.Channels) != 3 {
		t.Fatalf("missing joints are dropped: %+v", out)
	}
	p := short.RestPose()
	out.Sample(0, p)
	if !p[1].T.ApproxEq(geom.V3(0, 0.5, 1), 1e-4) {
		t.Fatalf("the hips travel half as far on a half-size rig: %v", p[1].T)
	}
	if !p[0].R.ApproxEq(turned.Mul(bow), 1e-4) {
		t.Fatalf("rotation is applied from the target's rest: %v", p[0].R)
	}
	if !p[0].T.ApproxEq(geom.V3(0, 0.6, 0), 1e-4) {
		t.Fatalf("same-length bones keep their translation: %v", p[0].T)
	}
	same := Retarget(clip, tall, tall)
	a, b := tall.RestPose(), tall.RestPose()
	clip.Sample(0, a)
	same.Sample(0, b)
	for i := range a {
		if !a[i].T.ApproxEq(b[i].T, 1e-5) || !a[i].R.ApproxEq(b[i].R, 1e-5) {
			t.Fatal("retargeting onto the same skeleton changes nothing")
		}
	}
}
