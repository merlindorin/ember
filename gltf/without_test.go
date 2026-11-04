package gltf_test

import (
	"testing"

	"github.com/merlindorin/ember/anim"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/gltf"
	"github.com/merlindorin/ember/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithoutRemapsWhatRefersToJoints(t *testing.T) {
	id := anim.TRS{R: geom.QuatIdent(), S: geom.V3(1, 1, 1)}
	m := &gltf.Model{
		Skeleton: &anim.Skeleton{Joints: []anim.Joint{
			{Name: "Root", Parent: -1, Rest: id},
			{Name: "Player", Parent: 0, Rest: id},
			{Name: "Hips", Parent: 0, Rest: id},
			{Name: "Sword", Parent: 2, Rest: id},
			{Name: "Hand", Parent: 2, Rest: id},
		}},
		Nodes: []gltf.Raw{
			{Extras: []byte(`0`)}, {Extras: []byte(`1`)}, {Extras: []byte(`2`)}, {Extras: []byte(`3`)}, {Extras: []byte(`4`)},
		},
		Cameras: map[int]render.Camera{1: {FOV: 1}, 4: {FOV: 4}},
		Parts: []*gltf.Part{
			{Name: "Body", Joint: 2, Skin: &anim.Skin{
				Joints: []int{2, 4}, InverseBind: []geom.Mat4{geom.Ident(), geom.Ident()},
			}},
			{Name: "Sword", Joint: 3},
		},
		Clips: []*anim.Clip{{Name: "Swing", Channels: []anim.Channel{{Joint: 4}, {Joint: 3}, {Joint: 1}}}},
	}
	out, err := m.Without([]bool{false, true, false, true, false})
	require.NoError(t, err)
	var names []string
	var parents []int
	for _, j := range out.Skeleton.Joints {
		names, parents = append(names, j.Name), append(parents, j.Parent)
	}
	assert.Equal(t, []string{"Root", "Hips", "Hand"}, names)
	assert.Equal(t, []int{-1, 0, 1}, parents)
	assert.Equal(t, []gltf.Raw{{Extras: []byte(`0`)}, {Extras: []byte(`2`)}, {Extras: []byte(`4`)}}, out.Nodes)
	assert.Equal(t, map[int]render.Camera{2: {FOV: 4}}, out.Cameras)
	require.Len(t, out.Parts, 1)
	assert.Equal(t, 1, out.Parts[0].Joint)
	assert.Equal(t, []int{1, 2}, out.Parts[0].Skin.Joints)
	assert.Equal(t, []int{2}, []int{out.Clips[0].Channels[0].Joint})
	assert.Len(t, out.Clips[0].Channels, 1)
	assert.Equal(t, []int{2, 4}, m.Parts[0].Skin.Joints, "the original is untouched")
	assert.Len(t, m.Clips[0].Channels, 3)

	_, err = m.Without([]bool{false, false, true, false, false})
	assert.ErrorContains(t, err, "Sword", "a kept joint under a dropped one")
	_, err = m.Without([]bool{false, false, false, false, true})
	assert.ErrorContains(t, err, "Body", "a skin on a dropped joint")
}
