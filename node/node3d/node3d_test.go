package node3d_test

import (
	"encoding/json"
	"testing"

	"github.com/merlindorin/ember/anim"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/gltf"
	"github.com/merlindorin/ember/mesh"
	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/node/node3d"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rest(x, y, z float32) anim.TRS {
	return anim.TRS{T: geom.V3(x, y, z), R: geom.QuatIdent(), S: geom.V3(1, 1, 1)}
}

// camp is what Blender exports: a root with a point light, a mesh, a
// camera and a sun, and no Ember data.
func camp(t *testing.T) *node.Scene {
	t.Helper()
	sk := &anim.Skeleton{Joints: []anim.Joint{
		{Name: "Camp", Parent: -1, Rest: rest(0, 0, 0)},
		{Name: "Fire", Parent: 0, Rest: rest(0, 0.5, 0)},
		{Name: "Tent", Parent: 0, Rest: rest(3, 0, 0)},
		{Name: "Eye", Parent: 0, Rest: rest(0, 10, 10)},
		{Name: "Sun", Parent: 0, Rest: anim.TRS{R: geom.QuatYaw(1), S: geom.V3(1, 1, 1)}},
	}}
	tri := &mesh.Data{
		Vertices: []mesh.Vertex{{Pos: geom.V3(0, 0, 0)}, {Pos: geom.V3(1, 0, 0)}, {Pos: geom.V3(0, 1, 0)}},
		Indices:  []uint32{0, 1, 2},
	}
	tri.RecomputeNormals()
	tent := &gltf.Part{
		Name:     "Tent",
		Joint:    2,
		Mesh:     tri.Transform(geom.Translate(geom.V3(3, 0, 0))),
		Material: render.Mat(color.White),
	}
	m := &gltf.Model{
		Skeleton: sk, Parts: []*gltf.Part{tent},
		Cameras: map[int]render.Camera{3: {FOV: 0.8, Near: 0.1, Far: 300}},
		Nodes: []gltf.Raw{
			{Extras: json.RawMessage(`{"spawn_weight":3}`)},
			{Extensions: map[string]json.RawMessage{"KHR_lights_punctual": json.RawMessage(`{"light":0}`)}},
			{}, {},
			{Extensions: map[string]json.RawMessage{
				"KHR_lights_punctual": json.RawMessage(`{"light":1}`), "EXT_vendor": json.RawMessage(`{"a":1}`),
			}},
		},
		Extensions: map[string]json.RawMessage{"KHR_lights_punctual": json.RawMessage(`{"lights":[` +
			`{"type":"point","color":[1,0.5,0],"intensity":4,"range":12},` +
			`{"type":"directional","color":[1,1,0.9],"intensity":2}]}`)},
	}
	data, err := gltf.Encode(m)
	require.NoError(t, err)
	s, err := node.Decode(data)
	require.NoError(t, err)
	return s
}

func TestABlenderSceneInstantiatesAsNodes(t *testing.T) {
	root, err := node3d.Instantiate(camp(t), nil)
	require.NoError(t, err)
	w := world.NewWorld()
	env := render.NewEnvironment()
	w.AddEntities(env)
	tree := node.New(w)
	tree.Root().AddNode(root)

	r := root.GetSceneNode()
	assert.Equal(t, "/root/Camp", r.Path())
	fire, ok := node.Find[*node3d.PointLight](r, "Fire")
	require.True(t, ok)
	assert.Equal(t, float32(12), fire.Range)
	assert.Equal(t, float32(4), fire.Intensity)
	assert.Equal(t, color.RGB(1, 0.5, 0), fire.Color)
	assert.Equal(t, geom.V3(0, 0.5, 0), fire.Position)

	eye, ok := node.Find[*node3d.Camera](r, "Eye")
	require.True(t, ok)
	assert.Equal(t, float32(0.8), eye.FOV)

	tent, ok := node.Find[*node3d.Mesh](r, "Tent")
	require.True(t, ok)
	require.Len(t, tent.Surfaces, 1)
	assert.InDelta(t, 0, tent.Surfaces[0].Mesh.Vertices[0].Pos.X, 1e-5, "meshes are in their node's space")
	drawn := 0
	for _, e := range w.Entities() {
		if _, isDrawn := e.(render.Drawable); isDrawn {
			drawn++
		}
	}
	assert.Equal(t, 1, drawn, "each surface is drawn")

	_, ok = node.Find[*node3d.Sun](r, "Sun")
	require.True(t, ok)
	w.Update(0.016)
	dir, _, intensity := env.Sun()
	defaultDir, _, _ := render.NewEnvironment().Sun()
	assert.Equal(t, float32(2), intensity)
	assert.InDelta(t, 1, dir.Len(), 1e-5)
	assert.NotEqual(t, defaultDir, dir, "the sun follows the node")

	for _, n := range r.ChildNodes() {
		assert.Same(t, r, n.Owner(), n.Name())
	}
}

func TestPackWritesBackWhatInstantiateRead(t *testing.T) {
	root, err := node3d.Instantiate(camp(t), nil)
	require.NoError(t, err)
	tent, _ := node.Find[*node3d.Mesh](root.GetSceneNode(), "Tent")
	tent.Position = geom.V3(5, 0, 0)

	s, err := node3d.Pack(root, nil)
	require.NoError(t, err)
	data, err := node.Encode(s)
	require.NoError(t, err)
	back, err := node.Decode(data)
	require.NoError(t, err)
	again, err := node3d.Instantiate(back, nil)
	require.NoError(t, err)

	r := again.GetSceneNode()
	assert.Equal(t, root.GetSceneNode().SceneID, r.SceneID)
	assert.NotEmpty(t, r.SceneID)
	moved, ok := node.Find[*node3d.Mesh](r, "Tent")
	require.True(t, ok)
	assert.Equal(t, geom.V3(5, 0, 0), moved.Position)
	assert.InDelta(t, 0, moved.Surfaces[0].Mesh.Vertices[0].Pos.X, 1e-5)
	fire, _ := node.Find[*node3d.PointLight](r, "Fire")
	assert.Equal(t, float32(12), fire.Range)
	_, ok = node.Find[*node3d.Camera](r, "Eye")
	assert.True(t, ok)
	sun, _ := node.Find[*node3d.Sun](r, "Sun")
	assert.Equal(t, float32(2), sun.Energy)
	assert.JSONEq(t, `{"a":1}`, string(sun.Raw.Extensions["EXT_vendor"]), "unknown extensions survive")
	assert.JSONEq(t, `{"spawn_weight":3}`, string(r.Raw.Extras))
	assert.Equal(t, s.ID, r.File.ID, "the scene keeps its ID")
}

type Spawner struct {
	node.Timer
	log *[]string
}

func (s *Spawner) OnTimeout() { *s.log = append(*s.log, "spawn") }

func TestTypedNodesBehavioursAndConnections(t *testing.T) {
	var spawned []string
	node.Register(node.Type{Name: "test.Spawner", Base: "node.Timer", New: func() node.Noded {
		s := &Spawner{log: &spawned}
		s.Timer = *node.NewTimer()
		return s
	}})
	s := camp(t)
	s.Nodes[2].Type, s.Nodes[2].Behaviour = "node3d.Mesh", ""
	s.Model.Skeleton.Joints = append(s.Model.Skeleton.Joints,
		anim.Joint{Name: "Wolves", Parent: 0, Rest: rest(0, 0, 0)})
	s.Nodes = append(s.Nodes, node.SceneNode{
		ID:        "00000000000000a1",
		Type:      "node.Timer",
		Behaviour: "test.Spawner",
		Groups:    []string{"spawners"},
		Properties: map[string]json.RawMessage{
			"wait_time": json.RawMessage(`0.5`),
			"autostart": json.RawMessage(`true`),
		},
	})
	s.Nodes[0].ID = "00000000000000a0"
	s.Connections = []node.Connection{
		{Signal: "timeout", From: "00000000000000a1", To: "00000000000000a1", Method: "OnTimeout"},
	}

	root, err := node3d.Instantiate(s, nil)
	require.NoError(t, err)
	w := world.NewWorld()
	tree := node.New(w)
	tree.Root().AddNode(root)
	sp, ok := node.Find[*Spawner](root.GetSceneNode(), "Wolves")
	require.True(t, ok)
	assert.True(t, sp.IsInGroup("spawners"))
	for range 4 {
		w.Update(0.3)
	}
	assert.Equal(t, []string{"spawn", "spawn"}, spawned, "1.2 s of a 0.5 s autostarted timer")

	packed, err := node3d.Pack(root, nil)
	require.NoError(t, err)
	var wolves node.SceneNode
	for _, n := range packed.Nodes {
		if n.ID == "00000000000000a1" {
			wolves = n
		}
	}
	assert.Equal(t, "node.Timer", wolves.Type)
	assert.Equal(t, "test.Spawner", wolves.Behaviour)
	assert.JSONEq(t, `0.5`, string(wolves.Properties["wait_time"]))
	assert.NotContains(t, wolves.Properties, "one_shot", "defaults are not written")
	want := s.Connections[0]
	want.Flags = node.Persist
	assert.Equal(t, []node.Connection{want}, packed.Connections, "scene connections persist")
}

func TestScenesThatCannotBeBuiltFailLoudly(t *testing.T) {
	for _, c := range []struct {
		name   string
		mutate func(*node.Scene)
		err    []string
	}{
		{"unknown property", func(s *node.Scene) {
			s.Nodes[1].Properties = map[string]json.RawMessage{"flicker_speed": json.RawMessage(`2`)}
		}, []string{"Camp/Fire", "flicker_speed"}},
		{"mesh on a type that draws none", func(s *node.Scene) {
			s.Nodes[2].Type = "node3d.Marker"
		}, []string{"Camp/Tent", "Marker"}},
		{"unknown class", func(s *node.Scene) { s.Nodes[3].Type = "Camera4D" }, []string{"Camp/Eye", "Camera4D"}},
		{"connection to a missing method", func(s *node.Scene) {
			s.Nodes[0].ID, s.Nodes[1].ID = "00000000000000a0", "00000000000000a1"
			s.Connections = []node.Connection{
				{Signal: "ready", From: "00000000000000a1", To: "00000000000000a0", Method: "Nope"},
			}
		}, []string{"ready"}},
		{"instance of a missing node", func(s *node.Scene) {
			s.Instances = []node.Instance{{Node: "00000000000000ee", Scene: "00000000000000ef"}}
		}, []string{"00000000000000ee"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := camp(t)
			c.mutate(s)
			_, err := node3d.Instantiate(s, nil)
			require.Error(t, err)
			for _, want := range c.err {
				assert.Contains(t, err.Error(), want)
			}
		})
	}
}
