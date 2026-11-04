package world_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/world"
)

func TestBasicEntity_HasAncestor(t *testing.T) {
	t.Run("should be true for child", func(t *testing.T) {
		ancestor := world.NewBasic()
		n1 := &world.BasicEntity{}

		ancestor.AppendChild(n1)

		assert.Truef(t, n1.HasAncestor(ancestor), "Expected HasAncestor to return true for ancestor")
	})

	t.Run("should be true for deep child", func(t *testing.T) {
		ancestor := world.NewBasic()

		n1 := &world.BasicEntity{}
		n2 := &world.BasicEntity{}

		n1.AppendChild(n2)
		ancestor.AppendChild(n1)

		assert.Truef(t, n2.HasAncestor(ancestor), "Expected HasAncestor to return true for ancestor")
	})

	t.Run("should be true for complex tree", func(t *testing.T) {
		root := world.NewBasic()
		ancestor := world.NewBasic()

		n1 := world.NewBasic()
		n2 := world.NewBasic()
		n3 := world.NewBasic()
		n4 := world.NewBasic()
		n5 := world.NewBasic()
		n6 := world.NewBasic()

		n1.AppendChild(n2)
		n1.AppendChild(n3)
		n3.AppendChild(n4)
		n4.AppendChild(n5)
		n4.AppendChild(n6)

		root.AppendChild(ancestor)
		ancestor.AppendChild(n1)

		assert.Truef(t, ancestor.HasAncestor(root), "Expected HasAncestor to return true for ancestor")
		assert.Truef(t, n1.HasAncestor(root), "Expected HasAncestor to return true for ancestor")
		assert.Truef(t, n1.HasAncestor(ancestor), "Expected HasAncestor to return true for ancestor")
		assert.Truef(t, n2.HasAncestor(ancestor), "Expected HasAncestor to return true for ancestor")

		assert.Truef(t, n3.HasAncestor(ancestor), "Expected HasAncestor to return true for ancestor")
		assert.Falsef(t, n3.HasAncestor(n2), "Expected HasAncestor to return false for ancestor")

		assert.Truef(t, n4.HasAncestor(ancestor), "Expected HasAncestor to return true for ancestor")
		assert.Truef(t, n5.HasAncestor(ancestor), "Expected HasAncestor to return true for ancestor")
		assert.Truef(t, n6.HasAncestor(ancestor), "Expected HasAncestor to return true for ancestor")
	})
}

func TestHierarchyLinksStayConsistent(t *testing.T) {
	a, b, c := world.NewBasic(), world.NewBasic(), world.NewBasic()
	a.AppendChild(c)
	b.AppendChild(c)
	if c.Parent() != b || len(a.Children()) != 0 {
		t.Fatalf("appending to b must detach c from a")
	}
	b.RemoveChild(c)
	if c.HasParent() {
		t.Fatalf("a removed child keeps no parent")
	}
}

type node struct{ *world.BasicEntity }

func newNode() *node { return &node{world.NewBasic()} }

func TestAChildAppendedToAnEntityInAWorldJoinsIt(t *testing.T) {
	w := world.New()
	parent, child, grandchild := newNode(), newNode(), newNode()
	child.AppendChild(grandchild)
	parent.AppendChild(child)
	if w.Has(child) {
		t.Fatal("a parent outside any World takes nobody in")
	}
	w.AddEntity(parent)
	late, lateKid := newNode(), newNode()
	late.AppendChild(lateKid)
	parent.AppendChild(late)
	if !w.Has(child) || !w.Has(grandchild) || !w.Has(late) || !w.Has(lateKid) {
		t.Fatal("appending to an entity in a World adds the child and its subtree at once")
	}
	parent.RemoveChild(late.BasicEntity)
	if !w.Has(late) {
		t.Fatal("leaving its parent does not take an entity out of the World")
	}
	w.RemoveEntity(parent)
	if w.Has(child) || w.Has(grandchild) {
		t.Fatal("removing the parent removes its children")
	}
	parent.AppendChild(newNode())
	if w.Len() != 2 {
		t.Fatalf("a removed parent adds nothing: %d entities", w.Len())
	}
}

type effect struct {
	*world.BasicEntity
	left float32
}

func TestChildrenAreTheWholeEntitiesAGameAppended(t *testing.T) {
	hero := world.NewBasic()
	burn := &effect{BasicEntity: world.NewBasic(), left: 3}
	spark := &effect{BasicEntity: world.NewBasic(), left: 1}
	bare := world.NewBasic()
	hero.AppendChild(burn)
	hero.AppendChild(bare)
	burn.AppendChild(spark)
	kids := hero.Children()
	if len(kids) != 2 {
		t.Fatalf("two children: %d", len(kids))
	}
	fx, ok := kids[0].(*effect)
	if !ok || fx != burn {
		t.Fatalf("the child is the game's own entity, the same one: %T", kids[0])
	}
	fx.left = 0
	if burn.left != 0 {
		t.Fatal("not a copy")
	}
	if kids[1] != world.BasicFace(bare) {
		t.Fatal("one appended bare is its BasicEntity")
	}
	all := hero.Descendents()
	if len(all) != 3 || all[0] != world.BasicFace(spark) || !hero.HasDescendents() {
		t.Fatalf("descendants, depth first, whole too: %v", all)
	}
}
