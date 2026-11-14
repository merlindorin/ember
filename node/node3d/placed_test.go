package node3d_test

import (
	"testing"
	"time"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/net"
	"github.com/lantervale/ember/node"
	"github.com/lantervale/ember/node/node3d"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/require"
)

// level is a scene placing two chests, each an instance of the chest scene.
func level(t *testing.T, lib library) *node.Scene {
	t.Helper()
	c, _ := chest()
	chestScene := lib.add(t, c)
	root := node3d.NewNode()
	root.SetName("Level")
	for i, at := range []geom.Vec3{geom.V3(1, 0, 0), geom.V3(-1, 0, 0)} {
		e := spawn(t, lib, chestScene.ID)
		e.GetSceneNode().SetName([]string{"Left", "Right"}[i])
		placed, ok := e.(*Chest)
		require.True(t, ok)
		placed.Position = at
		root.AddNode(e)
		require.NoError(t, e.GetSceneNode().SetOwner(&root.Node))
	}
	return lib.add(t, root)
}

func load(t *testing.T, a *world.World, lib library, s *node.Scene) node.Noded {
	t.Helper()
	tree := node.New(a)
	e, err := node3d.Instantiate(s, lib.resolve)
	require.NoError(t, err)
	tree.Root().AddNode(e)
	return e
}

func TestPlacedNodesReplicateByTheirSceneIDs(t *testing.T) {
	lib := library{}
	s := level(t, lib)
	srvConn, cliConn := net.Pipe()
	server, client := world.New(), world.New()
	srv := net.NewServer()
	server.AddPlugins(srv)
	srv.Connect(srvConn)
	client.AddPlugins(net.NewClient(cliConn))
	t.Cleanup(func() { _ = srvConn.Close(); _ = cliConn.Close() })

	onServer, onClient := load(t, server, lib, s), load(t, client, lib, s)
	left, _ := node.Find[*Chest](onServer.GetSceneNode(), "Left")
	mirror, _ := node.Find[*Chest](onClient.GetSceneNode(), "Left")
	right, _ := node.Find[*Chest](onClient.GetSceneNode(), "Right")
	sl, _ := node.Find[*node3d.Replicator](left.GetSceneNode(), "Replicator")
	cl, _ := node.Find[*node3d.Replicator](mirror.GetSceneNode(), "Replicator")
	cr, _ := node.Find[*node3d.Replicator](right.GetSceneNode(), "Replicator")
	require.Equal(t, net.PlacedRegion, sl.Network.NetID.Region())
	require.Equal(t, sl.Network.NetID, cl.Network.NetID, "both sides derive the same ID")
	require.NotEqual(t, cl.Network.NetID, cr.Network.NetID, "two instances of one scene differ")
	entities := client.Len()
	client.Frame(1.0 / 60)
	require.Equal(t, geom.V3(-1, 0, 0), right.Position, "a placed chest stays put until the server speaks")

	left.Position, left.Open = geom.V3(5, 0, 0), true
	deadline := time.Now().Add(3 * time.Second)
	for !(mirror.Position.ApproxEq(geom.V3(5, 0, 0), 1e-2) && mirror.Open) {
		require.True(t, time.Now().Before(deadline), "the client's chest follows the server's: %v", mirror.Position)
		server.Frame(1.0 / 60)
		client.Frame(1.0 / 60)
		time.Sleep(time.Millisecond)
	}
	require.Equal(t, entities, client.Len(), "nothing was spawned")
	require.Equal(t, geom.V3(-1, 0, 0), right.Position, "the other chest stays where the level put it")

	spawned, err := node3d.Instance(lib[s.Instances[0].Scene], lib.resolve)
	require.NoError(t, err)
	onServer.GetSceneNode().AddNode(spawned)
	sync, _ := node.Find[*node3d.Replicator](spawned.GetSceneNode(), "Replicator")
	require.NotEqual(t, net.PlacedRegion, sync.Network.NetID.Region(), "a chest spawned at runtime is not placed")
}
