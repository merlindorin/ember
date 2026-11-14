package node3d_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/node"
	"github.com/lantervale/ember/node/node3d"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type library map[string]*node.Scene

func (l library) resolve(id string) (*node.Scene, error) {
	s, ok := l[id]
	if !ok {
		return nil, fmt.Errorf("no scene %s", id)
	}
	data, err := node.Encode(s)
	if err != nil {
		return nil, err
	}
	return node.Decode(data)
}

func (l library) add(t *testing.T, root node.Noded) *node.Scene {
	t.Helper()
	s, err := node3d.Pack(root, l.resolve)
	require.NoError(t, err)
	data, err := node.Encode(s)
	require.NoError(t, err)
	back, err := node.Decode(data)
	require.NoError(t, err)
	l[back.ID] = back
	return back
}

func wolf() node.Noded {
	w := node3d.NewNode()
	w.SetName("Wolf")
	howl := node.NewTimer()
	howl.SetName("Howl")
	howl.WaitTime = 5
	eyes := node3d.NewPointLight()
	eyes.SetName("Eyes")
	eyes.Range = 4
	w.AddNode(howl)
	w.AddNode(eyes)
	_ = howl.SetOwner(&w.Node)
	_ = eyes.SetOwner(&w.Node)
	return w
}

func spawn(t *testing.T, lib library, id string) node.Noded {
	t.Helper()
	s, err := lib.resolve(id)
	require.NoError(t, err)
	e, err := node3d.Instance(s, lib.resolve)
	require.NoError(t, err)
	return e
}

func TestInstancesSaveOnlyTheirOverrides(t *testing.T) {
	lib := library{}
	wolfScene := lib.add(t, wolf())

	camp := node3d.NewNode()
	camp.SetName("Camp")
	alpha, beta := spawn(t, lib, wolfScene.ID), spawn(t, lib, wolfScene.ID)
	alpha.GetSceneNode().SetName("Alpha")
	beta.GetSceneNode().SetName("Beta")
	camp.AddNode(alpha)
	camp.AddNode(beta)
	require.NoError(t, alpha.GetSceneNode().SetOwner(&camp.Node))
	require.NoError(t, beta.GetSceneNode().SetOwner(&camp.Node))
	alphaNode, ok := alpha.(*node3d.Node)
	require.True(t, ok)
	alphaNode.Position = geom.V3(3, 0, 0)
	howl, ok := node.Find[*node.Timer](alpha.GetSceneNode(), "Howl")
	require.True(t, ok)
	howl.WaitTime = 2
	eyes, _ := node.Find[*node3d.PointLight](beta.GetSceneNode(), "Eyes")
	eyes.SetName("Glow")
	eyes.Range = 7

	s, err := node3d.Pack(camp, lib.resolve)
	require.NoError(t, err)
	assert.Equal(t, []string{"Camp", "Alpha", "Beta"}, jointNames(s.Model.Skeleton), "the camp file holds no wolf")
	require.Len(t, s.Instances, 2)
	assert.Equal(t, wolfScene.ID, s.Instances[0].Scene)
	require.Len(t, s.Instances[0].Overrides, 1)
	o := s.Instances[0].Overrides[0]
	assert.Equal(t, "Howl", o.Path)
	assert.Equal(t, howl.SceneID, o.Node)
	assert.JSONEq(t, `{"wait_time":2}`, mustJSON(t, o.Properties))
	require.Len(t, s.Instances[1].Overrides, 1, "found by ID though renamed")
	assert.JSONEq(t, `{"range":7}`, mustJSON(t, s.Instances[1].Overrides[0].Properties))

	data, err := node.Encode(s)
	require.NoError(t, err)
	back, err := node.Decode(data)
	require.NoError(t, err)
	root, err := node3d.Instantiate(back, lib.resolve)
	require.NoError(t, err)
	a, ok := node.Find[*node3d.Node](root.GetSceneNode(), "Alpha")
	require.True(t, ok)
	assert.Equal(t, geom.V3(3, 0, 0), a.Position)
	ah, _ := node.Find[*node.Timer](root.GetSceneNode(), "Alpha/Howl")
	bh, _ := node.Find[*node.Timer](root.GetSceneNode(), "Beta/Howl")
	assert.Equal(t, float32(2), ah.WaitTime)
	assert.Equal(t, float32(5), bh.WaitTime)
	be, _ := node.Find[*node3d.PointLight](root.GetSceneNode(), "Beta/Eyes")
	assert.Equal(t, float32(7), be.Range)
	assert.Same(t, a.GetSceneNode(), ah.Owner(), "nodes of an instance are owned by its root")
	assert.Same(t, root.GetSceneNode(), a.Owner())

	again, err := node3d.Pack(root, lib.resolve)
	require.NoError(t, err)
	assert.Equal(t, s.Instances, again.Instances)
}

func TestInheritedScenesStoreOnlyTheDifferences(t *testing.T) {
	lib := library{}
	wolfScene := lib.add(t, wolf())

	alpha := spawn(t, lib, wolfScene.ID)
	alpha.GetSceneNode().SetName("AlphaWolf")
	crown := node3d.NewMarker()
	crown.SetName("Crown")
	alpha.GetSceneNode().AddNode(crown)
	require.NoError(t, crown.SetOwner(alpha.GetSceneNode()))
	eyes, _ := node.Find[*node3d.PointLight](alpha.GetSceneNode(), "Eyes")
	eyes.Range = 9

	s, err := node3d.Pack(alpha, lib.resolve)
	require.NoError(t, err)
	assert.Equal(t, wolfScene.ID, s.Inherits)
	assert.Equal(t, []string{"AlphaWolf", "Crown"}, jointNames(s.Model.Skeleton))
	require.Len(t, s.Instances, 1)
	assert.Equal(t, s.Nodes[0].ID, s.Instances[0].Node, "the root stands for the base scene")
	assert.NotEqual(t, wolfScene.ID, s.ID)

	data, err := node.Encode(s)
	require.NoError(t, err)
	back, err := node.Decode(data)
	require.NoError(t, err)
	root, err := node3d.Instantiate(back, lib.resolve)
	require.NoError(t, err)
	assert.Equal(t, "AlphaWolf", root.GetSceneNode().Name())
	e, ok := node.Find[*node3d.PointLight](root.GetSceneNode(), "Eyes")
	require.True(t, ok)
	assert.Equal(t, float32(9), e.Range)
	_, ok = node.Find[*node3d.Marker](root.GetSceneNode(), "Crown")
	assert.True(t, ok)
	_, ok = node.Find[*node.Timer](root.GetSceneNode(), "Howl")
	assert.True(t, ok)
	assert.Equal(t, s.ID, root.GetSceneNode().File.ID)
}

func TestEmbeddedInstancesKeepTheirContent(t *testing.T) {
	lib := library{}
	wolfScene := lib.add(t, wolf())
	camp := node3d.NewNode()
	camp.SetName("Camp")
	w := spawn(t, lib, wolfScene.ID)
	camp.AddNode(w)
	for _, e := range append([]node.Noded{w}, subtree(w.GetSceneNode())...) {
		e.GetSceneNode().Instance = nil
		require.NoError(t, e.GetSceneNode().SetOwner(&camp.Node))
	}
	w.GetSceneNode().Instance = &node.Instanced{Scene: wolfScene.ID, Root: w.GetSceneNode(), Embedded: true}

	s, err := node3d.Pack(camp, nil)
	require.NoError(t, err)
	assert.Empty(t, s.Instances)
	assert.Equal(t, []string{"Camp", "Wolf", "Howl", "Eyes"}, jointNames(s.Model.Skeleton))
	assert.Equal(t, wolfScene.ID, s.Nodes[1].InstanceOf)
	root, err := node3d.Instantiate(s, nil)
	require.NoError(t, err)
	got, _ := node.Find[*node3d.Node](root.GetSceneNode(), "Wolf")
	require.NotNil(t, got.Instance)
	assert.True(t, got.Instance.Embedded)
	assert.Equal(t, wolfScene.ID, got.Instance.Scene)
}

func TestInstancingFailsLoudly(t *testing.T) {
	lib := library{}
	wolfScene := lib.add(t, wolf())
	camp := node3d.NewNode()
	camp.SetName("Camp")
	w := spawn(t, lib, wolfScene.ID)
	w.GetSceneNode().SetName("Alpha")
	camp.AddNode(w)
	require.NoError(t, w.GetSceneNode().SetOwner(&camp.Node))
	campScene := lib.add(t, camp)

	_, err := node3d.Instantiate(campScene, nil)
	assert.ErrorContains(t, err, wolfScene.ID, "no resolver")
	assert.ErrorContains(t, err, "Camp/Alpha")

	delete(lib, wolfScene.ID)
	_, err = node3d.Instantiate(campScene, lib.resolve)
	assert.ErrorContains(t, err, wolfScene.ID)

	lib.add(t, wolf())
	loop := lib[campScene.ID]
	loop.Instances[0].Scene = campScene.ID
	_, err = node3d.Instantiate(loop, lib.resolve)
	assert.ErrorContains(t, err, "cycle")

	lib = library{}
	wolfScene = lib.add(t, wolf())
	camp = node3d.NewNode()
	w = spawn(t, lib, wolfScene.ID)
	camp.AddNode(w)
	require.NoError(t, w.GetSceneNode().SetOwner(&camp.Node))
	howl, _ := node.Find[*node.Timer](w.GetSceneNode(), "Howl")
	extra := node3d.NewMarker()
	extra.SetName("Extra")
	howl.AddNode(extra)
	require.NoError(t, extra.SetOwner(&camp.Node))
	_, err = node3d.Pack(camp, lib.resolve)
	assert.ErrorContains(t, err, "Extra")
}

func subtree(n *node.Node) []node.Noded {
	var out []node.Noded
	for _, c := range n.ChildNodes() {
		out = append(out, c.Entity())
		out = append(out, subtree(c)...)
	}
	return out
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}
