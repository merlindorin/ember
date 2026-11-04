package node //nolint:testpackage // white-box: encode, which writes scenes Encode refuses

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/merlindorin/ember/gltf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const blenderCamp = `{
  "asset": {"generator": "Khronos glTF Blender I/O v4.2.70", "version": "2.0"},
  "extensionsUsed": ["KHR_lights_punctual"],
  "extensions": {"KHR_lights_punctual": {"lights": [{"type": "point", "intensity": 40}]}},
  "scene": 0,
  "scenes": [{"name": "Scene", "nodes": [0]}],
  "nodes": [
    {"name": "Camp", "children": [1, 2], "extras": {"spawn_weight": 3}},
    {"name": "Fire", "translation": [0, 0.5, 0], "extensions": {"KHR_lights_punctual": {"light": 0}}},
    {"name": "Spawner"}
  ]
}`

var sceneID = regexp.MustCompile(`^[0-9a-f]{16}$`)

func TestABlenderFileLoadsAndGainsIDsOnSave(t *testing.T) {
	s, err := Decode([]byte(blenderCamp))
	require.NoError(t, err)
	require.Len(t, s.Nodes, 3)
	assert.Empty(t, s.ID)
	assert.Empty(t, s.Nodes[0].ID)
	assert.Equal(t, "Camp", s.Model.Skeleton.Joints[0].Name)

	out, err := Encode(s)
	require.NoError(t, err)
	assert.Regexp(t, sceneID, s.ID)
	for _, n := range s.Nodes {
		assert.Regexp(t, sceneID, n.ID)
	}

	plain, err := gltf.Decode(out)
	require.NoError(t, err, "a tool without Ember's extensions still reads the file")
	assert.NotContains(t, plain.ExtensionsRequired, "EMBER_node")
	assert.JSONEq(t, `{"light":0}`, string(plain.Nodes[1].Extensions["KHR_lights_punctual"]))

	back, err := Decode(out)
	require.NoError(t, err)
	assert.Equal(t, s.ID, back.ID)
	assert.Equal(t, s.Nodes, back.Nodes)
	assert.NotContains(
		t,
		back.Model.Nodes[0].Extensions,
		"EMBER_node",
		"Ember's data lives in the scene, not the raw map",
	)
	assert.NotContains(t, back.Model.Extensions, "EMBER_scene")
	assert.Contains(t, back.Model.Extensions, "KHR_lights_punctual")
	assert.JSONEq(t, `{"spawn_weight":3}`, string(back.Model.Nodes[0].Extras))
}

func TestSceneDataRoundTrips(t *testing.T) {
	s, err := Decode([]byte(blenderCamp))
	require.NoError(t, err)
	s.ID = "00000000000000c0"
	s.Inherits = "00000000000000b0"
	s.Root = &SceneNode{ID: "00000000000000f0", Name: "Hero", Type: "node3d.Skeleton", Groups: []string{"heroes"}}
	s.Nodes[0] = SceneNode{ID: "0000000000000001", Type: "node3d.Node", Groups: []string{"camps"}}
	s.Nodes[1] = SceneNode{ID: "0000000000000002", Type: "node3d.PointLight", ProcessMode: Always}
	s.Nodes[2] = SceneNode{
		ID:          "0000000000000003",
		Type:        "node.Timer",
		Behaviour:   "game.WolfSpawner",
		ProcessMode: Pausable,
		Properties: map[string]json.RawMessage{
			"autostart": json.RawMessage(`true`),
			"wait_time": json.RawMessage(`30`),
		},
		InstanceOf: "00000000000000d0",
	}
	s.Connections = []Connection{
		{
			Signal: "timeout",
			From:   "0000000000000003",
			To:     "0000000000000001",
			Method: "OnSpawn",
			Flags:  Persist | Deferred,
		},
	}
	s.Instances = []Instance{
		{
			Node:             "0000000000000003",
			Scene:            "00000000000000d0",
			EditableChildren: true,
			Overrides: []Override{
				{
					Node:       "00000000000000e1",
					Path:       "Wolf",
					Properties: map[string]json.RawMessage{"name": json.RawMessage(`"Alpha"`)},
				},
			},
		},
	}

	out, err := Encode(s)
	require.NoError(t, err)
	back, err := Decode(out)
	require.NoError(t, err)
	assert.Equal(t, s.ID, back.ID)
	assert.Equal(t, s.Inherits, back.Inherits)
	assert.Equal(t, s.Root, back.Root)
	assert.Equal(t, s.Connections, back.Connections)
	assert.Equal(t, s.Instances, back.Instances)
	require.Len(t, back.Nodes, 3)
	for i := range s.Nodes {
		assert.Equal(t, s.Nodes[i].Type, back.Nodes[i].Type)
		assert.Equal(t, s.Nodes[i].ProcessMode, back.Nodes[i].ProcessMode)
		assert.Equal(t, s.Nodes[i].Behaviour, back.Nodes[i].Behaviour)
		assert.Equal(t, s.Nodes[i].InstanceOf, back.Nodes[i].InstanceOf)
		assert.Equal(t, s.Nodes[i].Groups, back.Nodes[i].Groups)
		for k, v := range s.Nodes[i].Properties {
			assert.JSONEq(t, string(v), string(back.Nodes[i].Properties[k]), k)
		}
	}

	var raw struct {
		Nodes []struct {
			Extensions map[string]map[string]any `json:"extensions"`
		} `json:"nodes"`
	}
	require.NoError(t, json.Unmarshal(glbJSON(t, out), &raw))
	assert.Equal(
		t,
		"pausable",
		raw.Nodes[2].Extensions["EMBER_node"]["process_mode"],
		"process modes are written by name",
	)
}

func glbJSON(t *testing.T, glb []byte) []byte {
	t.Helper()
	require.Greater(t, len(glb), 20)
	n := int(glb[12]) | int(glb[13])<<8 | int(glb[14])<<16 | int(glb[15])<<24
	return glb[20 : 20+n]
}

func TestBrokenSceneDataFailsLoudly(t *testing.T) {
	with := func(mutate func(s *Scene)) []byte {
		s, err := Decode([]byte(blenderCamp))
		require.NoError(t, err)
		mutate(s)
		_, err = Encode(s)
		require.Error(t, err, "Encode refuses what Decode would")
		out, err := encode(s)
		require.NoError(t, err)
		return out
	}
	for _, c := range []struct {
		name   string
		mutate func(*Scene)
		err    []string
	}{
		{"duplicate node ID", func(s *Scene) { s.Nodes[1].ID, s.Nodes[2].ID = "00000000000000aa", "00000000000000aa" },
			[]string{"00000000000000aa", "Camp/Fire", "Camp/Spawner"}},
		{"malformed node ID", func(s *Scene) { s.Nodes[2].ID = "spawner" }, []string{"spawner", "Camp/Spawner"}},
		{"malformed scene ID", func(s *Scene) { s.ID = "camp" }, []string{"camp"}},
		{"root ID on a node too", func(s *Scene) {
			s.Root = &SceneNode{ID: "00000000000000aa", Name: "Hero"}
			s.Nodes[1].ID = "00000000000000aa"
		}, []string{"00000000000000aa", "Hero", "Camp/Fire"}},
		{"connection to a missing node", func(s *Scene) {
			s.Nodes[0].ID = "0000000000000001"
			s.Connections = []Connection{
				{Signal: "timeout", From: "0000000000000001", To: "00000000000000ff", Method: "OnSpawn"},
			}
		}, []string{"00000000000000ff", "timeout"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := Decode(with(c.mutate))
			require.Error(t, err)
			for _, want := range c.err {
				assert.Contains(t, err.Error(), want)
			}
		})
	}

	var mode ProcessMode
	assert.ErrorContains(t, mode.UnmarshalText([]byte("Sometimes")), "Sometimes")
}

func TestReadIDReadsOnlyTheHeader(t *testing.T) {
	s, err := Decode([]byte(blenderCamp))
	require.NoError(t, err)
	_, err = Encode(s)
	require.NoError(t, err)
	out, err := Encode(s)
	require.NoError(t, err)
	id, err := ReadID(bytes.NewReader(out[:20+int(binary.LittleEndian.Uint32(out[12:]))]))
	require.NoError(t, err)
	assert.Equal(t, s.ID, id)

	plain, err := gltf.Encode(s.Model)
	require.NoError(t, err)
	id, err = ReadID(bytes.NewReader(plain))
	require.NoError(t, err)
	assert.Empty(t, id, "a file without EMBER_scene has no ID")

	id, err = ReadID(strings.NewReader(blenderCamp))
	require.NoError(t, err)
	assert.Empty(t, id)
	_, err = ReadID(strings.NewReader("glTF"))
	assert.Error(t, err)
}
