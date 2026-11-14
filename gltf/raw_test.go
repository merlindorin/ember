package gltf //nolint:testpackage // white-box: reads the written JSON chunk with splitGLB

import (
	"encoding/json"
	"testing"

	"github.com/lantervale/ember/anim"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/mesh"
	"github.com/lantervale/ember/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const blenderStyle = `{
  "asset": {"generator": "Khronos glTF Blender I/O v4.2.70", "version": "2.0"},
  "extensionsUsed": ["KHR_lights_punctual", "EXT_vendor_thing"],
  "extensionsRequired": ["EXT_vendor_thing"],
  "extensions": {"KHR_lights_punctual": {"lights": [{"type": "point", "color": [1, 0.5, 0], "intensity": 40}]}},
  "extras": {"blender_scene": "Main"},
  "scene": 0,
  "scenes": [{"name": "Scene", "nodes": [0]}],
  "nodes": [
    {"name": "Camp", "children": [1, 2], "extras": {"spawn_weight": 3}},
    {"name": "Fire", "translation": [0, 0.5, 0], "extensions": {"KHR_lights_punctual": {"light": 0}}},
    {"name": "Tent", "extensions": {"EXT_vendor_thing": {"a": [1, 2]}}}
  ]
}`

func TestUnknownExtensionsAndExtrasSurviveARoundTrip(t *testing.T) {
	m, err := Decode([]byte(blenderStyle))
	require.NoError(t, err)
	require.Len(t, m.Nodes, 3, "one entry per skeleton joint")
	assert.JSONEq(t, `{"spawn_weight": 3}`, string(m.Nodes[m.Skeleton.Find("Camp")].Extras))
	assert.JSONEq(t, `{"light": 0}`, string(m.Nodes[m.Skeleton.Find("Fire")].Extensions["KHR_lights_punctual"]))

	m.Nodes[m.Skeleton.Find("Tent")].Extensions["EMBER_node"] = json.RawMessage(`{"id":"00000000000000aa"}`)
	m.Extensions["EMBER_scene"] = json.RawMessage(`{"id":"00000000000000bb"}`)
	m.ExtensionsRequired = append(m.ExtensionsRequired, "EMBER_node")
	out, err := Encode(m)
	require.NoError(t, err)

	var doc struct {
		Used       []string                   `json:"extensionsUsed"`
		Required   []string                   `json:"extensionsRequired"`
		Extensions map[string]json.RawMessage `json:"extensions"`
		Extras     json.RawMessage            `json:"extras"`
	}
	js, _, err := splitGLB(out)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(js, &doc))
	assert.ElementsMatch(t, []string{"KHR_lights_punctual", "EXT_vendor_thing", "EMBER_node", "EMBER_scene"}, doc.Used)
	assert.Equal(t, []string{"EXT_vendor_thing"}, doc.Required, "Ember's extensions are never required")
	assert.JSONEq(t, `{"blender_scene": "Main"}`, string(doc.Extras))

	again, err := Decode(out)
	require.NoError(t, err)
	sameJSON(t, m.Extensions, again.Extensions)
	sameJSON(t, m.Extras, again.Extras)
	for i := range m.Nodes {
		sameJSON(t, m.Nodes[i], again.Nodes[i])
	}
}

func TestEncodeListsTheExtensionsItWrites(t *testing.T) {
	m, err := Decode([]byte(blenderStyle))
	require.NoError(t, err)
	m.Parts = append(m.Parts, unlitPart())
	out, err := Encode(m)
	require.NoError(t, err)
	js, _, err := splitGLB(out)
	require.NoError(t, err)
	var doc struct {
		Used []string `json:"extensionsUsed"`
	}
	require.NoError(t, json.Unmarshal(js, &doc))
	assert.Contains(t, doc.Used, "KHR_materials_unlit")
}

func unlitPart() *Part {
	tri := &mesh.Data{Vertices: []mesh.Vertex{
		{Pos: geom.V3(0, 0, 0)}, {Pos: geom.V3(1, 0, 0)}, {Pos: geom.V3(0, 1, 0)},
	}, Indices: []uint32{0, 1, 2}}
	tri.RecomputeNormals()
	return &Part{Name: "Tent", Mesh: tri, Material: &render.Material{Albedo: color.White, Unlit: true}}
}

func sameJSON(t *testing.T, want, got any) {
	t.Helper()
	w, err := json.Marshal(want)
	require.NoError(t, err)
	g, err := json.Marshal(got)
	require.NoError(t, err)
	assert.JSONEq(t, string(w), string(g))
}

func TestPartsAndCamerasStayOnTheirNodes(t *testing.T) {
	sk := &anim.Skeleton{Joints: []anim.Joint{
		{Name: "Camp", Parent: -1, Rest: anim.TRS{R: geom.QuatIdent(), S: geom.V3(1, 1, 1)}},
		{Name: "Mesh", Parent: 0, Rest: anim.TRS{T: geom.V3(2, 0, 0), R: geom.QuatIdent(), S: geom.V3(1, 1, 1)}},
		{Name: "Mesh", Parent: 0, Rest: anim.TRS{T: geom.V3(-2, 0, 0), R: geom.QuatIdent(), S: geom.V3(1, 1, 1)}},
	}}
	left, right := unlitPart(), unlitPart()
	left.Name, left.Joint = "Mesh", 1
	right.Name, right.Joint = "Mesh", 2
	right.Mesh = right.Mesh.Transform(geom.Translate(geom.V3(-2, 0, 0)))
	left.Mesh = left.Mesh.Transform(geom.Translate(geom.V3(2, 0, 0)))
	m := &Model{Skeleton: sk, Parts: []*Part{left, right}, Cameras: map[int]render.Camera{
		0: {FOV: 0.8, Near: 0.1, Far: 300},
		2: {OrthoHeight: 20, Near: 1, Far: 50},
	}}
	back := roundTrip(t, m)
	require.Len(t, back.Parts, 2)
	for _, p := range back.Parts {
		want := sk.Joints[p.Joint].Rest.T.X
		assert.InDelta(t, want, p.Mesh.Vertices[0].Pos.X, 1e-5, "part on joint %d", p.Joint)
	}
	assert.ElementsMatch(t, []int{1, 2}, []int{back.Parts[0].Joint, back.Parts[1].Joint})
	assert.Equal(t, m.Cameras, back.Cameras)
}
