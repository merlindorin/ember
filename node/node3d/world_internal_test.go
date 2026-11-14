package node3d

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/node"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTheEnvironmentNodeLightsTheSceneWithTheSunsDirection(t *testing.T) {
	w := world.NewWorld()
	tree := node.New(w)
	root := NewNode()
	root.SetName("Level")
	we := NewEnvironment()
	we.Environment.SetFog(color.Gray(0.5), 0.2)
	sky, ground, _ := we.Environment.Ambient()
	we.Environment.SetAmbient(sky, ground, 0.5)
	sun := NewSun()
	sun.Energy = 3
	root.AddNode(sun)
	root.AddNode(we)
	_ = we.SetOwner(&root.Node)
	_ = sun.SetOwner(&root.Node)
	tree.Root().AddNode(root)
	w.Update(0.016)
	require.Same(t, we.Environment, environment(w), "the node is the World's environment")
	_, density := we.Environment.Fog()
	_, _, intensity := we.Environment.Sun()
	assert.Equal(t, float32(0.2), density)
	assert.Equal(t, float32(3), intensity, "the scene's Sun lights it")

	s, err := Pack(root, nil)
	require.NoError(t, err)
	data, err := node.Encode(s)
	require.NoError(t, err)
	back, err := node.Decode(data)
	require.NoError(t, err)
	again, err := Instantiate(back, nil)
	require.NoError(t, err)
	got, ok := node.Find[*Environment](again.GetSceneNode(), "Environment")
	require.True(t, ok)
	_, saved := got.Environment.Fog()
	assert.Equal(t, float32(0.2), saved)
}
