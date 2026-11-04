package net //nolint:testpackage // white-box: shares the package's test rigs (serverOf, clientOf, Ping)

import (
	"testing"
	"time"

	"github.com/merlindorin/ember/world"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func lagRig(delay time.Duration) (*world.World, *world.World, *Server, *Client, *clock, Endpoint) {
	clk := &clock{t: time.Unix(1_800_000_000, 0)}
	se, ce := Lag(clk.now, delay)
	server, client := world.New().AddPlugins(NewServer()), world.New()
	srv := serverOf(server)
	srv.ConnectEndpoint(se)
	client.AddPlugins(NewClient(nil, WithEndpoint(ce), Clock(clk.now)))
	return server, client, srv, clientOf(client), clk, ce
}

func TestLagDeliversAfterItsDelayOnItsClock(t *testing.T) {
	server, client, srv, cli, clk, _ := lagRig(75 * time.Millisecond)
	var got []int
	Handle(srv, func(_ ClientID, p Ping) { got = append(got, p.N) })
	step := func(d time.Duration) {
		clk.t = clk.t.Add(d)
		server.RunFrames(1)
		client.RunFrames(1)
	}
	step(0)
	step(74 * time.Millisecond)
	if cli.ID() != 0 {
		t.Fatal("the welcome arrived before its 75 ms")
	}
	step(time.Millisecond)
	if cli.ID() == 0 {
		t.Fatal("the welcome is due after 75 ms")
	}
	sent := cli.Send(Ping{N: 1})
	cli.Send(Ping{N: 2})
	for i := 0; i < 20 && cli.Acked() != sent+1; i++ {
		step(10 * time.Millisecond)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("applied %v", got)
	}
	if rtt := cli.RTT(); rtt < 150*time.Millisecond || rtt > 200*time.Millisecond {
		t.Fatalf("round trip %v measured on the lag clock, want 150–200 ms", rtt)
	}
}

func TestLagEndpointCloseLosesWhatIsInFlight(t *testing.T) {
	server, client, srv, cli, clk, ce := lagRig(50 * time.Millisecond)
	var got []int
	Handle(srv, func(_ ClientID, p Ping) { got = append(got, p.N) })
	for i := 0; i < 10; i++ {
		clk.t = clk.t.Add(10 * time.Millisecond)
		server.RunFrames(1)
		client.RunFrames(1)
	}
	cli.Send(Ping{N: 1})
	ce.Close()
	for i := 0; i < 10; i++ {
		clk.t = clk.t.Add(10 * time.Millisecond)
		server.RunFrames(1)
		client.RunFrames(1)
	}
	if len(got) != 0 || cli.Connected() || len(srv.Clients()) != 0 {
		t.Fatalf("applied %v, client connected %v, server clients %v", got, cli.Connected(), srv.Clients())
	}
}
