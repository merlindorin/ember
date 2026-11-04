package net //nolint:testpackage // white-box: shares the package's test rig and unit

import (
	"slices"
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/world"
)

type Login struct{ Session string }

type Count struct{ N int }

// sessions is a server keeping each session's unit through disconnects,
// like a game's reconnect grace.
type sessions struct {
	srv     *Server
	units   map[string]*unit
	acks    map[ClientID]uint32
	byID    map[ClientID]string
	applied []int
	forget  bool
}

func newSessions(r *rig) *sessions {
	s := &sessions{srv: r.srv(), units: map[string]*unit{}, acks: map[ClientID]uint32{}, byID: map[ClientID]string{}}
	var left world.Reader[ClientDisconnected]
	r.server.On(world.FixedUpdate, "sessions.left", func(world.Frame) {
		for ev := range left.Read(r.server) {
			s.acks[ev.ID] = ev.Ack
		}
	})
	Handle(s.srv, func(c ClientID, m Login) {
		s.byID[c] = m.Session
		if u, ok := s.units[m.Session]; ok && !s.forget {
			var last uint32
			for old, sess := range s.byID {
				if sess == m.Session && old != c {
					last = max(last, s.acks[old])
				}
			}
			u.Owner = c
			s.srv.Send(c, Resumed{Client: c, LastSeq: last})
			return
		}
		u := newUnit(geom.V3(0, 0, 0), c)
		s.units[m.Session] = u
		r.server.AddEntities(u)
	})
	Handle(s.srv, func(_ ClientID, m Count) { s.applied = append(s.applied, m.N) })
	return s
}

func reconnectRig(t *testing.T) (*rig, *sessions) {
	srvConn, cliConn := Pipe()
	r := &rig{t: t, server: world.New(), client: world.New()}
	r.server.AddPlugins(NewServer(RequiredFocus()))
	s := newSessions(r)
	r.srv().Connect(srvConn)
	r.client.AddPlugins(NewClient(cliConn, Hello(Login{Session: "s"})))
	t.Cleanup(func() { r.cli().Close() })
	r.until("the session's unit", func() bool { _, ok := r.cli().Owned(); return ok })
	return r, s
}

func (r *rig) drop() {
	r.cli().Close()
	r.until("both sides to notice", func() bool { return !r.cli().Connected() && len(r.srv().Clients()) == 0 })
}

func (r *rig) redial(hello any) {
	srvConn, cliConn := Pipe()
	r.srv().Connect(srvConn)
	r.cli().Reconnect(cliConn, hello)
}

func TestReconnectResendsExactlyWhatTheServerMissed(t *testing.T) {
	r, s := reconnectRig(t)
	r.cli().Send(Count{1})
	acked := r.cli().Send(Count{2})
	r.until("acks", func() bool { return r.cli().Acked() == acked })

	r.cli().Send(Count{5})
	for !slices.Contains(s.applied, 5) {
		r.server.RunFrames(1)
	}
	r.drop()
	r.cli().Send(Count{3})
	r.cli().Send(Count{4})

	mine, _ := r.cli().Owned()
	r.redial(Login{Session: "s"})
	if seq := r.cli().Send(Count{6}); seq != 0 {
		t.Fatal("an intent sent while resuming must be held back")
	}
	r.until("every intent", func() bool { return len(s.applied) >= 6 })
	r.server.RunFrames(10)
	r.client.RunFrames(2)
	if want := []int{1, 2, 5, 3, 4, 6}; !slices.Equal(s.applied, want) {
		t.Fatalf("applied %v, want %v: no duplicate, no loss, in order", s.applied, want)
	}
	r.until("the unit owned again", func() bool { _, ok := r.cli().Owned(); return ok })
	if now, _ := r.cli().Owned(); now != mine {
		t.Fatal("the resumed session must keep the same mirror")
	}
}

func TestReconnectToAFreshSessionDropsStaleIntents(t *testing.T) {
	r, s := reconnectRig(t)
	r.drop()
	r.cli().Send(Count{3})
	s.forget = true
	r.redial(Login{Session: "s"})
	r.cli().Send(Count{6})
	r.until("the new session", func() bool { return slices.Contains(s.applied, 6) })
	r.server.RunFrames(5)
	if !slices.Equal(s.applied, []int{6}) {
		t.Fatalf("applied %v: a new session must not replay the old one's intents", s.applied)
	}
}

func TestRequireFocusReplicatesNothingBeforeJoining(t *testing.T) {
	srvConn, cliConn := Pipe()
	r := &rig{t: t, server: world.New(), client: world.New()}
	r.server.AddPlugins(NewServer(RequiredFocus()))
	r.srv().Connect(srvConn)
	r.client.AddPlugins(NewClient(cliConn))
	t.Cleanup(r.cli().Close)
	r.server.AddEntities(newUnit(geom.V3(0, 0, 0), 0))
	r.until("welcome", func() bool { return r.cli().ID() != 0 })
	r.server.RunFrames(10)
	r.client.RunFrames(2)
	if n := r.client.Len(); n != 0 {
		t.Fatalf("a client that has not joined received %d entities", n)
	}
}
