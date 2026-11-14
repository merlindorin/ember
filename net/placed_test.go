package net //nolint:testpackage // white-box: Client.unplaced

import (
	"testing"

	"github.com/lantervale/ember/geom"
)

func TestPlacedIDsAreStableAndApart(t *testing.T) {
	a, b := PlacedID("level/door"), PlacedID("level/door")
	if a != b || a.Region() != PlacedRegion || PlacedID("level/gate") == a {
		t.Fatalf("placed IDs %v %v", a, b)
	}
}

func TestAPlacedEntityIsAdoptedNotSpawned(t *testing.T) {
	r := newRig(t, NewServer())
	door := newUnit(geom.V3(1, 0, 0), 0)
	door.NetID = PlacedID("level/door")
	r.server.AddEntities(door)
	local := newUnit(geom.V3(1, 0, 0), 0)
	local.NetID = door.NetID
	r.client.AddEntities(local)
	r.cli().Place(local)

	door.Position, door.Vitals.HP = geom.V3(4, 0, 0), 2
	r.until("the placed door follows", func() bool {
		return local.Position.ApproxEq(geom.V3(4, 0, 0), 1e-2) && local.Vitals.HP == 2
	})
	if r.client.Len() != 1 {
		t.Fatalf("%d entities: the door was spawned again", r.client.Len())
	}
	if door.NetID != PlacedID("level/door") {
		t.Fatal("the server keeps a placed ID")
	}

	r.server.RemoveEntity(door)
	r.until("despawn", func() bool { _, ok := r.cli().Lookup(local.NetID); return !ok })
	if !r.client.Has(local) {
		t.Fatal("net never removes a placed entity")
	}
}

func TestAPlacedSpawnWaitsForItsEntity(t *testing.T) {
	r := newRig(t, NewServer())
	door := newUnit(geom.V3(2, 0, 0), 0)
	door.NetID = PlacedID("level/door")
	r.server.AddEntities(door)
	r.until("the spawn is held", func() bool { _, ok := r.cli().unplaced[door.NetID]; return ok })
	if r.client.Len() != 0 {
		t.Fatal("a placed ID is never spawned from its kind")
	}
	local := newUnit(geom.V3(0, 0, 0), 0)
	local.NetID = door.NetID
	r.client.AddEntities(local)
	r.cli().Place(local)
	r.until("adopted once placed", func() bool { return local.Position.ApproxEq(geom.V3(2, 0, 0), 1e-2) })
}
