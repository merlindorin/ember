package netside_test

import (
	"testing"

	"github.com/lantervale/ember/internal/netside"
	"github.com/lantervale/ember/world"
)

func TestEachWorldKeepsItsOwnSides(t *testing.T) {
	a, b := world.New(), world.New()
	if netside.Server(a) != nil || netside.Client(a) != nil {
		t.Fatal("no side yet")
	}
	netside.SetServer(a, "server")
	netside.SetClient(b, "client")
	if netside.Server(a) != "server" || netside.Client(a) != nil {
		t.Fatal("a is a server")
	}
	if netside.Client(b) != "client" || netside.Server(b) != nil {
		t.Fatal("b is a client")
	}
}
