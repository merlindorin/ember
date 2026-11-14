package net_test

import (
	"testing"
	"time"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/net"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

type token struct {
	twod.Item
	net.Network
}

func newToken(opts ...twod.Option) *token {
	return twod.Build(&token{Item: twod.Base(), Network: net.Network{Kind: "token"}}, "token", opts...)
}

func init() { //nolint:gochecknoinits // the kind is registered before any client mirrors it
	net.RegisterKind("token", func() net.Replicated { return newToken() })
}

func TestATwodItemReplicatesItsPlacement(t *testing.T) {
	srvConn, cliConn := net.Pipe()
	srv := net.NewServer()
	server := world.New().AddPlugins(srv)
	srv.Connect(srvConn)
	cli := net.NewClient(cliConn)
	client := world.New().AddPlugins(cli)
	t.Cleanup(cli.Close)

	tok := newToken(twod.At(64, 32), twod.Rotation(0.75))
	server.AddEntities(tok)

	var mirror *token
	for deadline := time.Now().Add(3 * time.Second); mirror == nil || mirror.Position().Dist(geom.V2(80, 32)) > 1e-2; {
		if time.Now().After(deadline) {
			t.Fatalf("no mirror at (80, 32): %+v", mirror)
		}
		if mirror != nil {
			tok.SetPosition(geom.V2(80, 32))
		}
		server.RunFrames(1)
		time.Sleep(time.Millisecond)
		client.RunFrames(1)
		if e, ok := cli.Lookup(tok.NetID); ok {
			mirror, _ = e.(*token)
		}
	}
	if r := mirror.Rotation(); r < 0.749 || r > 0.751 {
		t.Fatalf("mirror turned %v, want 0.75", r)
	}
}
