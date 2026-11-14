package net //nolint:testpackage // white-box: the option fields each option sets

import (
	stdnet "net"
	"testing"
	"time"

	"github.com/lantervale/ember/world"
)

func TestServerOptionsAndDefaults(t *testing.T) {
	s := NewServer()
	if s.interval != 1.0/20 || s.radius != 60 || s.margin != 8 || s.maxIntents != 256 || s.budget != 256 ||
		s.needFocus {
		t.Fatalf("defaults %+v", s)
	}
	ln := &chanListener{}
	s = NewServer(WithListener(ln), TickRate(30), Interest(72, 4), MaxIntents(9), Region(3),
		RequiredFocus(), SnapshotBudget(7), WithCodec(gobFrames()))
	if s.listener != ln || s.interval != 1.0/30 || s.radius != 72 || s.margin != 4 || s.maxIntents != 9 ||
		s.region != 3 || !s.needFocus || s.budget != 7 || s.codec == nil {
		t.Fatalf("options %+v", s)
	}
	s = NewServer(TickRate(0), Interest(0, 0), MaxIntents(0), SnapshotBudget(0))
	if s.interval != 1.0/20 || s.radius != 60 || s.margin != 8 || s.maxIntents != 256 || s.budget != 256 {
		t.Fatal("zero values keep the defaults")
	}
}

func TestClientOptionsAndDefaults(t *testing.T) {
	_, e := Lag(time.Now, 0)
	c := NewClient(nil, WithEndpoint(e))
	if c.step != 1.0/30 || c.blend != 20 || c.now == nil || c.peer != e.l {
		t.Fatalf("defaults %+v", c)
	}
	clock := func() time.Time { return time.Unix(1, 0) }
	c = NewClient(
		nil,
		WithEndpoint(e),
		TickRate(60),
		Correction(5),
		Clock(clock),
		WithCodec(gobFrames()),
		Hello(Ping{N: 1}),
	)
	if c.step != 1.0/60 || c.blend != 5 || c.now() != time.Unix(1, 0) || c.codec == nil || c.hello != (Ping{N: 1}) {
		t.Fatalf("options %+v", c)
	}
}

func TestTheServerAndClientAreThePluginsTheGameHolds(t *testing.T) {
	r := newRig(t, NewServer())
	if serverOf(r.server) == nil || clientOf(r.client) == nil || serverOf(r.client) != nil {
		t.Fatal("each World knows its own side")
	}
	r.until("welcome", func() bool { return r.cli().ID() != 0 })
	if _, ok := world.TryResource[Server](r.server); ok {
		t.Fatal("the server is not a resource")
	}
}

type chanListener struct{}

func (*chanListener) Conns() <-chan stdnet.Conn { return nil }
func (*chanListener) Addr() string              { return "" }
func (*chanListener) Close() error              { return nil }
