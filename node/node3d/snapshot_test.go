package node3d_test

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/node/node3d"
	"github.com/merlindorin/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotTakesTheLiveTree(t *testing.T) {
	lib := library{}
	wolfScene := lib.add(t, wolf())
	w := world.NewWorld()
	tree := node.New(w)
	level := node3d.NewNode()
	level.SetName("Level")
	tree.Root().AddNode(level)
	near := spawn(t, lib, wolfScene.ID)
	near.GetSceneNode().SetName("Near")
	far := spawn(t, lib, wolfScene.ID)
	far.GetSceneNode().SetName("Far")
	farNode, ok := far.(*node3d.Node)
	require.True(t, ok)
	farNode.Position = geom.V3(100, 0, 0)
	level.AddNode(near)
	level.AddNode(far)
	dropped := node3d.NewMarker()
	dropped.SetName("Loot")
	dropped.Position = geom.V3(1, 0, 0)
	level.AddNode(dropped)

	s, err := node3d.Snapshot(tree.Root().Entity(), lib.resolve, nil)
	require.NoError(t, err)
	assert.Equal(
		t,
		[]string{"root", "Level", "Near", "Far", "Loot"},
		jointNames(s.Model.Skeleton),
		"runtime nodes without an owner are in",
	)
	assert.Equal(t, "node.Empty", s.Nodes[0].Type, "the tree root is a plain Node")
	assert.Len(t, s.Instances, 2)
	assert.Nil(t, tree.Root().File, "a snapshot leaves the tree as it was")
	assert.Nil(t, dropped.Owner())

	w.Update(0)
	s, err = node3d.Snapshot(level, lib.resolve, node3d.Within(geom.V3(0, 0, 0), 10))
	require.NoError(t, err)
	assert.Equal(t, []string{"Level", "Near", "Loot"}, jointNames(s.Model.Skeleton))

	data, err := node.Encode(s)
	require.NoError(t, err)
	back, err := node.Decode(data)
	require.NoError(t, err)
	again, err := node3d.Instantiate(back, lib.resolve)
	require.NoError(t, err)
	_, ok = node.Find[*node.Timer](again.GetSceneNode(), "Near/Howl")
	assert.True(t, ok, "a snapshot is just a scene")
}

func TestASkeletonInsideAScenePacksOnlyAsAnInstance(t *testing.T) {
	hero, err := node3d.Instantiate(hero(t), nil)
	require.NoError(t, err)
	level := node3d.NewNode()
	level.SetName("Level")
	level.AddNode(hero)
	require.NoError(t, hero.GetSceneNode().SetOwner(&level.Node))
	_, err = node3d.Pack(level, nil)
	assert.ErrorContains(t, err, "Armature")
	assert.ErrorContains(t, err, "instance")
}
