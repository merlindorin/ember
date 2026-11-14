package node3d_test

import (
	"os"
	"testing"
	"time"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/net"
	"github.com/lantervale/ember/node"
	"github.com/lantervale/ember/node/node3d"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Chest struct {
	node3d.Node
	Open bool
}

func (c *Chest) NetState() any           { return c.Open }
func (c *Chest) ApplyNetState(state any) { c.Open, _ = state.(bool) }

func TestMain(m *testing.M) {
	node.Register(node.Type{
		Name: "test.Chest", Base: "node3d.Node",
		New: func() node.Noded { return &Chest{Node: *node3d.NewNode()} },
	})
	os.Exit(m.Run())
}

func chest() (*Chest, *node3d.Replicator) {
	c := &Chest{Node: *node3d.NewNode()}
	c.SetName("Chest")
	sync := node3d.NewReplicator()
	c.AddNode(sync)
	_ = sync.SetOwner(c.GetSceneNode())
	return c, sync
}

func TestTheSynchronizerReplicatesItsRoot(t *testing.T) {
	w := world.NewWorld()
	w.AddPlugins(net.NewServer())
	tree := node.New(w)
	c, sync := chest()
	tree.Root().AddNode(c)
	var _ net.Replicated = sync
	var stateful net.Stateful = sync
	c.Position, c.Open = geom.V3(1, 2, 3), true
	w.Update(0.016)
	assert.Equal(t, geom.V3(1, 2, 3), sync.GetTransform().Position, "the server sends the root's pose")
	assert.Equal(t, true, stateful.NetState())

	w = world.NewWorld()
	w.AddPlugins(net.NewClient(nil, net.WithEndpoint(cliEnd(t))))
	tree = node.New(w)
	c, sync = chest()
	tree.Root().AddNode(c)
	sync.GetTransform().Position = geom.V3(4, 5, 6)
	sync.ApplyNetState(true)
	w.Update(0.016)
	assert.Equal(t, geom.V3(4, 5, 6), c.Position, "a client moves the root to the received pose")
	assert.True(t, c.Open)
}

func TestReplicatedScenesSpawnAndLeaveOnClients(t *testing.T) {
	lib := library{}
	c, _ := chest()
	s := lib.add(t, c)
	w := world.NewWorld()
	w.AddPlugins(net.NewClient(nil, net.WithEndpoint(cliEnd(t))))
	tree := node.New(w)
	spawn := node3d.SceneSpawner(tree, lib.resolve)
	e := spawn(s.ID)
	require.NotNil(t, e)
	sync, ok := e.(*node3d.Replicator)
	require.True(t, ok)
	assert.Equal(t, s.ID, sync.Network.Kind)
	got, ok := node.Find[*Chest](tree.Root(), "Chest")
	require.True(t, ok, "the instance is in the tree")
	assert.True(t, w.Has(sync))

	w.RemoveEntity(sync)
	w.Update(0.016)
	assert.False(t, got.IsInsideTree(), "a despawned mirror takes its scene with it")
	assert.Nil(t, spawn("00000000000000ff"))
}

func TestRegisterScenesChecksEveryID(t *testing.T) {
	lib := library{}
	c, _ := chest()
	s := lib.add(t, c)
	tree := node.New(world.NewWorld())
	assert.NoError(t, node3d.RegisterScenes(tree, lib.resolve, s.ID))
	assert.ErrorContains(t, node3d.RegisterScenes(tree, lib.resolve, "00000000000000fe"), "00000000000000fe")
	w := lib.add(t, wolf())
	assert.ErrorContains(t, node3d.RegisterScenes(tree, lib.resolve, w.ID), "Replicator")

	placed, err := node3d.Instantiate(lib[s.ID], lib.resolve)
	require.NoError(t, err)
	tree.Root().AddNode(placed)
	sync, ok := node.Find[*node3d.Replicator](placed.GetSceneNode(), "Replicator")
	require.True(t, ok)
	assert.Equal(t, s.ID, sync.Network.Kind, "a scene's synchronizer replicates as that scene")
}

func cliEnd(t *testing.T) net.Endpoint {
	_, c := net.Lag(time.Now, 0)
	t.Cleanup(c.Close)
	return c
}
