package net //nolint:testpackage // white-box: shares the package's test rig and walker

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/scene"
	"github.com/lantervale/ember/world"
)

type Walk struct{ V float32 }

// walker moves along X at a velocity only its owner and the server know.
type walker struct {
	*world.BasicEntity
	scene.Transform
	Network
	v float32
}

func (w *walker) PredictIntent(m any) {
	if walk, ok := m.(Walk); ok {
		w.v = walk.V
	}
}

func (w *walker) PredictStep(dt float32) { w.Position.X += w.v * dt }

func registerWalker() {
	RegisterKind("walker", func() Replicated {
		return &walker{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0)}
	})
}

type walkers struct{ world.Group[*walker] }

func (s *walkers) Phase() world.Phase { return world.FixedUpdate }
func (s *walkers) Update(frame world.Frame) {
	dt := frame.DT
	for w := range s.All() {
		w.Position.X += w.v * dt
	}
}

func TestOwnedEntitiesArePredictedAndReconciled(t *testing.T) {
	srvConn, cliConn := Pipe()
	r := &rig{t: t, server: world.New(), client: world.New()}
	r.server.Time().FixedDelta = 1.0 / 30
	r.server.AddPlugins(NewServer(TickRate(30)))
	r.server.AddSystems(&walkers{})
	r.srv().Connect(srvConn)
	r.client.AddPlugins(NewClient(cliConn, TickRate(30)))
	t.Cleanup(r.cli().Close)
	r.until("welcome", func() bool { return r.cli().ID() != 0 })

	w := &walker{
		BasicEntity: world.NewBasic(),
		Transform:   scene.At(0, 0, 0),
		Network:     Network{Kind: "walker", Owner: r.cli().ID()},
	}
	r.server.AddEntities(w)
	Handle(r.srv(), func(_ ClientID, m Walk) { w.v = m.V })
	var mine *walker
	r.until("the owned mirror", func() bool {
		if m, ok := r.cli().Owned(); ok {
			mine, _ = m.(*walker)
		}
		return mine != nil
	})

	r.cli().Send(Walk{V: 3})
	r.client.RunFrames(4)
	if mine.Position.X <= 0 || w.Position.X != 0 {
		t.Fatalf("the client must move before the server hears of it: client %v, server %v",
			mine.Position.X, w.Position.X)
	}

	r.until("walking on the server", func() bool { return w.Position.X > 4 })
	stop := r.cli().Send(Walk{V: 0})
	r.until("agreement once stopped", func() bool {
		return r.cli().Acked() == stop && geom.Abs(mine.Position.X-w.Position.X) < 1e-3
	})
	if r.cli().RTT() <= 0 {
		t.Fatal("acks must give a round trip estimate")
	}

	w.Position.X = 50
	before := mine.Position.X
	prev := before
	r.until("the correction to arrive", func() bool { return mine.Position.X != before })
	if mine.Position.X >= 50 {
		t.Fatalf("a correction must be blended, not snapped: at %v", mine.Position.X)
	}
	for i := 0; i < 30; i++ {
		r.server.RunFrames(1)
		r.client.RunFrames(1)
		if mine.Position.X < prev || mine.Position.X > 50+1e-3 {
			t.Fatalf("the correction must converge monotonically: %v after %v", mine.Position.X, prev)
		}
		prev = mine.Position.X
	}
	if geom.Abs(mine.Position.X-50) > 1e-2 {
		t.Fatalf("after half a second the client sits at %v, the server at 50", mine.Position.X)
	}
}
