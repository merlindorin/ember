package world //nolint:testpackage // white-box: queueOf, the event queue behind Emit

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Velocity struct{ X float32 }

func (v *Velocity) GetVelocity() *Velocity { return v }

type Health struct{ HP int }

func (h *Health) GetHealth() *Health { return h }

type Frozen struct{}

func (f *Frozen) GetFrozen() *Frozen { return f }

type Movable interface {
	BasicFace
	GetVelocity() *Velocity
}

type Mortal interface {
	BasicFace
	GetHealth() *Health
}

type Freezable interface{ GetFrozen() *Frozen }

type mover struct {
	*BasicEntity
	Velocity
	X float32
}

type monster struct {
	*BasicEntity
	Velocity
	Health
}

type statue struct {
	*BasicEntity
	Velocity
	Frozen
}

type moveSystem struct{ Group[Movable] }

func (s *moveSystem) Update(frame Frame) {
	dt := frame.DT
	for m := range s.All() {
		if mv, ok := m.(*mover); ok {
			mv.X += m.GetVelocity().X * dt
		}
	}
}

type reaper struct {
	Group[Mortal]
	world *World
}

func (r *reaper) New(w *World) { r.world = w }

func (r *reaper) Update(Frame) {
	for m := range r.All() {
		m.GetHealth().HP--
		if m.GetHealth().HP <= 0 {
			r.world.RemoveEntity(m)
		}
	}
}

func TestGroupSystemReceivesMatchingEntities(t *testing.T) {
	w := NewWorld()
	sys := &moveSystem{}
	w.AddSystem(sys)
	a := &mover{BasicEntity: NewBasic(), Velocity: Velocity{2}}
	w.AddEntities(a, &monster{BasicEntity: NewBasic()}, NewBasic())
	assert.Equal(t, 2, sys.Len())
	w.Update(0.5)
	assert.Equal(t, float32(1), a.X)
}

func TestSystemAddedLateSeesExistingEntities(t *testing.T) {
	w := NewWorld()
	w.AddEntity(&mover{BasicEntity: NewBasic()})
	sys := &moveSystem{}
	w.AddSystem(sys)
	assert.Equal(t, 1, sys.Len())
}

func TestAddSystemInterfaceExclusion(t *testing.T) {
	w := NewWorld()
	sys := &moveSystem{}
	w.AddSystemInterface(sys, (*Movable)(nil), (*Freezable)(nil))
	w.AddEntities(&mover{BasicEntity: NewBasic()}, &statue{BasicEntity: NewBasic()})
	assert.Equal(t, 1, sys.Len(), "frozen entity must be excluded")
}

func TestTwoSystemsOfSameTypeKeepTheirOwnFilters(t *testing.T) {
	w := NewWorld()
	all, notFrozen := &moveSystem{}, &moveSystem{}
	w.AddSystemInterface(all, (*Movable)(nil), nil)
	w.AddSystemInterface(notFrozen, (*Movable)(nil), (*Freezable)(nil))
	w.AddEntity(&statue{BasicEntity: NewBasic()})
	assert.Equal(t, 1, all.Len())
	assert.Equal(t, 0, notFrozen.Len())
}

func TestRemovalDuringUpdateIsDeferredAndSafe(t *testing.T) {
	w := NewWorld()
	r := &reaper{}
	mv := &moveSystem{}
	w.AddSystems(r, mv)
	for i := 1; i <= 5; i++ {
		w.AddEntity(&monster{BasicEntity: NewBasic(), Health: Health{i}})
	}
	w.Update(0)
	assert.Equal(t, 4, r.Len())
	assert.Equal(t, 4, mv.Len(), "removal must propagate to every system")
	assert.Equal(t, 4, w.Len())
}

func TestRemoveEntityRemovesDescendants(t *testing.T) {
	w := NewWorld()
	sys := &moveSystem{}
	w.AddSystem(sys)
	parent := &mover{BasicEntity: NewBasic()}
	child := &mover{BasicEntity: NewBasic()}
	parent.AppendChild(child.BasicEntity)
	w.AddEntities(parent, child)
	w.RemoveEntity(parent)
	assert.Equal(t, 0, sys.Len())
	assert.False(t, w.Has(child))
	assert.Same(t, parent.BasicEntity, child.Parent(), "the removed subtree stays whole")
}

func TestStartupRunsOnce(t *testing.T) {
	w := NewWorld()
	n := 0
	w.AddSystem(Func(Startup, "boot", func(Frame) { n++ }))
	w.Update(0)
	w.Update(0)
	assert.Equal(t, 1, n)
}

func TestViewIsSyncedButNotUpdated(t *testing.T) {
	w := NewWorld()
	view := &Group[Mortal]{}
	w.AddView(view)
	m := &monster{BasicEntity: NewBasic()}
	w.AddEntity(m)
	require.Equal(t, 1, view.Len())
	w.RemoveEntity(m)
	assert.Equal(t, 0, view.Len())
	assert.Empty(t, w.Systems())
}

func TestStatsNameSystems(t *testing.T) {
	w := NewWorld()
	w.AddSystems(&moveSystem{}, Func(Update, "custom", func(Frame) {}))
	w.Update(0)
	var names []string
	for _, s := range w.Timings() {
		names = append(names, s.Name)
	}
	slices.Sort(names)
	assert.Equal(t, []string{"custom", "moveSystem"}, names)
}

func TestResources(t *testing.T) {
	type score struct{ V int }
	w := NewWorld()
	_, ok := TryResource[score](w)
	assert.False(t, ok)
	SetResource(w, &score{3})
	Resource[score](w).V++
	assert.Equal(t, 4, Resource[score](w).V)
	assert.Same(t, ResourceOrInit[Health](w), ResourceOrInit[Health](w))
	assert.Panics(t, func() { Resource[Velocity](w) })
}

func TestEventsDeliveredOnceAndExpire(t *testing.T) {
	type hit struct{ N int }
	w := NewWorld()
	var early, late Reader[hit]
	read := func(r *Reader[hit]) []int {
		var out []int
		for ev := range r.Read(w) {
			out = append(out, ev.N)
		}
		return out
	}
	assert.Empty(t, read(&early))
	Emit(w, hit{1})
	Emit(w, hit{2})
	assert.Equal(t, []int{1, 2}, read(&late))
	w.EndFrame()
	assert.Equal(t, []int{1, 2}, read(&early), "a reader that ran before the emit sees it next frame")
	assert.Empty(t, read(&late), "events are never delivered twice")
	w.EndFrame()
	var fresh Reader[hit]
	assert.Empty(t, read(&fresh), "events expire after two frames")
}

func TestAnyAndCountReadWhatIsNew(t *testing.T) {
	type hit struct{}
	w := NewWorld()
	var anyReader, count Reader[hit]
	assert.False(t, anyReader.Any(w))
	Emit(w, hit{})
	Emit(w, hit{})
	assert.True(t, anyReader.Any(w))
	assert.False(t, anyReader.Any(w), "Any reads them: asked again, nothing new")
	assert.Equal(t, 2, count.Count(w))
	w.EndFrame()
	assert.False(t, anyReader.Any(w), "still alive next frame, but already read")
	Emit(w, hit{})
	assert.Equal(t, 1, count.Count(w))
	w.EndFrame()
	w.EndFrame()
	var late Reader[hit]
	assert.Zero(t, late.Count(w), "expired events count for nothing")
}

func TestAFixedStepReaderGetsEveryEventAtAnyFrameRate(t *testing.T) {
	type hit struct{ N int }
	w := NewWorld()
	var fixedReader, updateReader Reader[hit]
	var fixedGot, updateGot []int
	w.On(FixedUpdate, "fixed", func(f Frame) {
		for e := range fixedReader.Read(f.World) {
			fixedGot = append(fixedGot, e.N)
		}
	})
	frame := 0
	w.On(Update, "emit", func(f Frame) {
		frame++
		if frame%5 == 0 {
			Emit(f.World, hit{frame})
		}
		for e := range updateReader.Read(f.World) {
			updateGot = append(updateGot, e.N)
		}
	})
	step := w.Time().FixedDelta / 8
	for range 100 {
		w.Frame(step)
	}
	want := []int{5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100}
	assert.Equal(t, want, updateGot, "Update readers see each once")
	assert.Equal(t, want[:len(fixedGot)], fixedGot, "in order, never twice")
	assert.GreaterOrEqual(t, len(fixedGot), len(want)-1,
		"at 480 frames a second the fixed step sees them all, the last one at its next step")
	q := queueOf[hit](w)
	assert.Less(t, len(q.events), 3, "and the queue empties as steps pass")
}

func TestRealElapsedRunsWhileTimeIsScaledOrPaused(t *testing.T) {
	w := NewWorld()
	w.Time().Scale = 0
	w.Frame(0.2)
	w.Time().Scale = 2
	w.Frame(0.1)
	assert.InDelta(t, 0.3, w.Time().RealElapsed, 1e-5, "every frame's real time")
	assert.InDelta(t, 0.2, w.Time().Elapsed, 1e-5, "while Elapsed follows the scale")
	w.Frame(5)
	assert.InDelta(t, 0.3+w.Time().MaxDelta, w.Time().RealElapsed, 1e-5, "a long frame is clamped like the rest")
}

func BenchmarkGroupIteration(b *testing.B) {
	w := NewWorld()
	sys := &moveSystem{}
	w.AddSystem(sys)
	for i := 0; i < 10000; i++ {
		w.AddEntity(&mover{BasicEntity: NewBasic(), Velocity: Velocity{1}})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sys.Update(Frame{DT: 0.016, World: w})
	}
}

func TestEntitiesAndResourcesEnumeration(t *testing.T) {
	w := NewWorld()
	a, b := &mover{BasicEntity: NewBasic()}, &monster{BasicEntity: NewBasic()}
	w.AddEntities(a, b)
	w.RemoveEntity(a)
	w.AddEntity(a)
	got := w.Entities()
	require.Len(t, got, 2)
	assert.Same(t, b, got[0])
	assert.Same(t, a, got[1])

	SetResource(w, &Velocity{X: 1})
	SetResource(w, &Health{HP: 2})
	res := w.Resources()
	require.Len(t, res, 2, "only what was set")
	assert.Equal(t, "world.Health", res[0].Type.String())
	assert.Equal(t, &Velocity{X: 1}, res[1].Value)
}

func TestRemovalKeepsInsertionOrderAcrossCompaction(t *testing.T) {
	w := NewWorld()
	var all []*mover
	for i := 0; i < 1000; i++ {
		m := &mover{BasicEntity: NewBasic()}
		all = append(all, m)
		w.AddEntity(m)
	}
	var want []uint64
	for i, m := range all {
		if i%3 == 0 {
			want = append(want, m.ID())
		} else {
			w.RemoveEntity(m)
		}
	}
	extra := &mover{BasicEntity: NewBasic()}
	w.AddEntity(extra)
	want = append(want, extra.ID())
	var got []uint64
	for _, e := range w.Entities() {
		got = append(got, e.ID())
	}
	assert.Equal(t, want, got)
	assert.Equal(t, len(want), w.Len())

	view := &Group[*mover]{}
	w.AddView(view)
	assert.Equal(t, len(want), view.Len(), "a late view sees only live entities")
	w.RemoveEntity(extra)
	w.AddEntity(extra)
	assert.Equal(t, extra.ID(), w.Entities()[len(w.Entities())-1].ID(), "re-adding appends at the end")
}

func BenchmarkRemoveEntity(b *testing.B) {
	w := NewWorld()
	ents := make([]*mover, b.N)
	for i := range ents {
		ents[i] = &mover{BasicEntity: NewBasic()}
		w.AddEntity(ents[i])
	}
	b.ResetTimer()
	for _, e := range ents {
		w.RemoveEntity(e)
	}
}

func TestReAddingDuringUpdateCancelsTheDeferredRemoval(t *testing.T) {
	w := NewWorld()
	sys := &moveSystem{}
	m := &mover{BasicEntity: NewBasic()}
	w.AddSystem(sys).AddEntity(m)
	w.AddSystem(Func(Update, "move", func(Frame) {
		w.RemoveEntity(m)
		w.AddEntity(m)
	}))
	w.Update(0)
	assert.True(t, w.Has(m))
	assert.Equal(t, 1, sys.Len())
}

type member struct {
	*BasicEntity
	left int
}

func (m *member) LeaveTree() { m.left++ }

func TestARemovedTreeMemberLeavesItsTree(t *testing.T) {
	w := New()
	parent, child := &member{BasicEntity: NewBasic()}, &member{BasicEntity: NewBasic()}
	parent.AppendChild(child)
	w.AddEntities(parent)
	w.RemoveEntity(parent)
	if parent.left != 1 || child.left != 0 {
		t.Fatalf("only the removed entity leaves its parent's tree: %d %d", parent.left, child.left)
	}
	w.RemoveEntity(parent)
	if parent.left != 1 {
		t.Fatal("not again once out of the World")
	}
}
