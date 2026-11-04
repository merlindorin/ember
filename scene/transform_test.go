package scene_test

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/world"
)

type node struct {
	*world.BasicEntity
	scene.Transform
}

func newNode(t scene.Transform) *node { return &node{world.NewBasic(), t} }

func TestTransformSystemResolvesHierarchy(t *testing.T) {
	w := world.NewWorld()
	w.AddSystem(&scene.TransformSystem{})
	root := newNode(scene.At(10, 0, 0).WithYaw(geom.Pi / 2))
	child := newNode(scene.At(0, 0, -1))
	grandchild := newNode(scene.At(0, 2, 0))
	w.AddEntity(root)
	scene.Attach(w, root, child)
	scene.Attach(w, child, grandchild)
	w.Update(0)

	if got := child.WorldPosition(); !got.ApproxEq(geom.V3(9, 0, 0), 1e-4) {
		t.Fatalf("child at %v", got)
	}
	if got := grandchild.WorldPosition(); !got.ApproxEq(geom.V3(9, 2, 0), 1e-4) {
		t.Fatalf("grandchild at %v", got)
	}
}

func TestWorldFallsBackToLocalBeforeFirstUpdate(t *testing.T) {
	n := newNode(scene.At(1, 2, 3))
	if n.WorldPosition() != geom.V3(1, 2, 3) {
		t.Fatalf("got %v", n.WorldPosition())
	}
}

func TestYaw(t *testing.T) {
	n := newNode(scene.At(0, 0, 0).WithYaw(0.8))
	if geom.Abs(n.Yaw()-0.8) > 1e-4 {
		t.Fatalf("yaw = %v", n.Yaw())
	}
}

func TestRemovingParentRemovesChildFromSystem(t *testing.T) {
	w := world.NewWorld()
	sys := &scene.TransformSystem{}
	w.AddSystem(sys)
	root := newNode(scene.At(0, 0, 0))
	w.AddEntity(root)
	scene.Attach(w, root, newNode(scene.At(1, 0, 0)))
	w.RemoveEntity(root)
	if sys.Len() != 0 {
		t.Fatalf("%d entities left", sys.Len())
	}
}

func TestAnEntityUnderOneWithoutATransformIsATopLevelRoot(t *testing.T) {
	w := world.NewWorld()
	w.AddSystem(&scene.TransformSystem{})
	group := world.NewBasic()
	parent := newNode(scene.At(0, 1, 0))
	child := newNode(scene.At(0, 2, 0))
	group.AppendChild(parent.BasicEntity)
	w.AddEntity(parent)
	scene.Attach(w, parent, child)
	w.Update(0)
	if got := child.WorldPosition(); !got.ApproxEq(geom.V3(0, 3, 0), 1e-4) {
		t.Fatalf("child at %v", got)
	}
}
