package node3d_test

import (
	"testing"

	"github.com/merlindorin/ember/anim"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/gltf"
	"github.com/merlindorin/ember/mesh"
	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/node/node3d"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	armature = iota
	hips
	hand
	body
	sword
	blade
)

// hero is a skinned, animated character as Blender exports it, plus a
// sword that Ember typed: a Socket on the hand holding a
// Mesh blade.
func hero(t *testing.T) *node.Scene {
	t.Helper()
	sk := &anim.Skeleton{Joints: []anim.Joint{
		armature: {Name: "Armature", Parent: -1, Rest: rest(0, 0, 0)},
		hips:     {Name: "Hips", Parent: armature, Rest: rest(0, 1, 0)},
		hand:     {Name: "Hand", Parent: hips, Rest: rest(0.5, 0, 0)},
		body:     {Name: "Body", Parent: armature, Rest: rest(0, 0, 0)},
		sword:    {Name: "Sword", Parent: hand, Rest: rest(0, 0, 0)},
		blade:    {Name: "Blade", Parent: sword, Rest: rest(0, 0.2, 0)},
	}}
	g := sk.Globals(sk.RestPose(), nil)
	tri := &mesh.Data{
		Vertices: []mesh.Vertex{{Pos: geom.V3(0, 1, 0)}, {Pos: geom.V3(1, 1, 0)}, {Pos: geom.V3(0, 2, 0)}},
		Indices:  []uint32{0, 1, 2},
	}
	tri.RecomputeNormals()
	influences := make([]anim.Influence, 3)
	for i := range influences {
		influences[i].Weights[0] = 1
	}
	bodyPart := &gltf.Part{
		Name: "Body", Joint: body, Mesh: tri, Material: render.Mat(color.White), Influences: influences,
		Skin: &anim.Skin{Joints: []int{hips, hand}, InverseBind: []geom.Mat4{g[hips].Inverse(), g[hand].Inverse()}},
	}
	bladeMesh := &mesh.Data{
		Vertices: []mesh.Vertex{{Pos: geom.V3(0, 0, 0)}, {Pos: geom.V3(0.1, 0, 0)}, {Pos: geom.V3(0, 1, 0)}},
		Indices:  []uint32{0, 1, 2},
	}
	bladeMesh.RecomputeNormals()
	bladePart := &gltf.Part{Name: "Blade", Joint: blade, Mesh: bladeMesh, Material: render.Mat(color.White),
		Skin: &anim.Skin{Joints: []int{blade}, InverseBind: []geom.Mat4{geom.Ident()}}}
	key := func(j int, from, to geom.Vec3) anim.Channel {
		return anim.Channel{
			Joint:  j,
			Path:   anim.Translation,
			Times:  []float32{0, 1},
			Values: []geom.Vec4{from.Vec4(0), to.Vec4(0)},
		}
	}
	m := &gltf.Model{Skeleton: sk, Parts: []*gltf.Part{bodyPart, bladePart}, Clips: []*anim.Clip{
		{Name: "Jump", Duration: 1, Channels: []anim.Channel{key(hips, geom.V3(0, 1, 0), geom.V3(0, 3, 0))}},
		{Name: "Wave", Duration: 1, Channels: []anim.Channel{key(hand, geom.V3(0.5, 0, 0), geom.V3(0.5, 1, 0))}},
	}}
	data, err := gltf.Encode(m)
	require.NoError(t, err)
	s, err := node.Decode(data)
	require.NoError(t, err)
	s.Nodes[at(s, "Sword")].Type = "node3d.Socket"
	s.Nodes[at(s, "Blade")].Type = "node3d.Mesh"
	return s
}

func at(s *node.Scene, name string) int { return s.Model.Skeleton.Find(name) }

func animatedWorld() (*world.World, *node.Tree) {
	w := world.NewWorld()
	w.AddSystems(&anim.AnimatorSystem{}, &anim.SocketSystem{}, &anim.SkinSystem{}, &scene.TransformSystem{})
	return w, node.New(w)
}

func skinnedY(t *testing.T, w *world.World) float32 {
	t.Helper()
	for _, e := range w.Entities() {
		if s, ok := e.(anim.SkinnedDrawable); ok && len(s.GetSkinned().Skin.Joints) > 1 {
			return s.GetModel().Mesh.Vertices[0].Pos.Y
		}
	}
	t.Fatal("no skinned surface")
	return 0
}

func checkHero(t *testing.T, root node.Noded) {
	t.Helper()
	skel, ok := root.(*node3d.Skeleton)
	require.True(t, ok, "an animated file's root is a Skeleton, %T", root)
	assert.Equal(t, "Armature", skel.Name())
	assert.ElementsMatch(
		t,
		[]string{"Armature", "Hips", "Hand", "Body"},
		jointNames(skel.Skeleton),
		"bones are not nodes",
	)

	w, tree := animatedWorld()
	tree.Root().AddNode(root)
	player, ok := node.Find[*node3d.Animator](skel.GetSceneNode(), "Animator")
	require.True(t, ok)
	attachment, ok := node.Find[*node3d.Socket](skel.GetSceneNode(), "Sword")
	require.True(t, ok)
	assert.Equal(t, "Hand", attachment.BoneName)
	bladeNode, ok := node.Find[*node3d.Mesh](skel.GetSceneNode(), "Sword/Blade")
	require.True(t, ok)
	require.Len(t, bladeNode.Surfaces, 1, "the blade keeps its mesh")
	assert.Len(t, skel.Model.Parts, 1, "and the skeleton draws only the body")

	w.Update(0.016)
	assert.InDelta(t, 1, skinnedY(t, w), 1e-4, "rest pose")
	assert.InDelta(t, 1.2, bladeNode.World().Translation().Y, 1e-4, "the blade sits in the hand")

	require.True(t, player.Play("Jump"))
	w.Update(0.5)
	assert.InDelta(t, 2, skinnedY(t, w), 1e-3, "half a jump lifts the skin by 1")
	assert.InDelta(t, 2.2, bladeNode.World().Translation().Y, 1e-3, "and the sword with it")

	var finished []string
	player.AnimationFinished.Connect(func(name string) { finished = append(finished, name) }, 0)
	require.True(t, player.PlayOnce("Wave"))
	for range 4 {
		w.Update(0.4)
	}
	assert.Equal(t, []string{"Wave"}, finished)
	assert.False(t, player.Play("Nope"))
}

func jointNames(sk *anim.Skeleton) []string {
	var out []string
	for _, j := range sk.Joints {
		out = append(out, j.Name)
	}
	return out
}

func TestAnAnimatedFileBecomesASkeleton(t *testing.T) {
	root, err := node3d.Instantiate(hero(t), nil)
	require.NoError(t, err)
	checkHero(t, root)
}

func TestAnAnimatedSceneSurvivesPackAndReload(t *testing.T) {
	root, err := node3d.Instantiate(hero(t), nil)
	require.NoError(t, err)
	root.GetSceneNode().AddToGroup("heroes")
	s, err := node3d.Pack(root, nil)
	require.NoError(t, err)
	require.NotNil(t, s.Root)
	assert.Equal(t, "node3d.Skeleton", s.Root.Type)
	assert.Equal(
		t,
		at(s, "Hand"),
		s.Model.Skeleton.Joints[at(s, "Sword")].Parent,
		"the attachment is written under its bone",
	)
	data, err := node.Encode(s)
	require.NoError(t, err)
	back, err := node.Decode(data)
	require.NoError(t, err)
	again, err := node3d.Instantiate(back, nil)
	require.NoError(t, err)
	assert.Equal(t, root.GetSceneNode().SceneID, again.GetSceneNode().SceneID)
	assert.True(t, again.GetSceneNode().IsInGroup("heroes"))
	assert.Len(t, again.GetSceneNode().ChildNodes(), 2, "the player and the attachment, no duplicate player")
	checkHero(t, again)
}

func TestAnimatedFilesKeepTypedNodesOffBones(t *testing.T) {
	s := hero(t)
	s.Nodes[at(s, "Sword")].Type = "node3d.Marker"
	_, err := node3d.Instantiate(s, nil)
	assert.ErrorContains(t, err, "Armature/Hips/Hand/Sword")
	assert.ErrorContains(t, err, "Socket")

	s = hero(t)
	s.Nodes[at(s, "Sword")].Type = ""
	_, err = node3d.Instantiate(s, nil)
	assert.ErrorContains(t, err, "Blade", "a typed node under a bone")
}
