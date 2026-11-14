package automation //nolint:testpackage // white-box: the truncation limit (maxItems) and reflectValue

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/scene"
	"github.com/lantervale/ember/world"
)

type Stats struct {
	HP    int
	Tags  []string
	Items map[string]int
}

type hero struct {
	*world.BasicEntity
	scene.Transform
	Stats
	Name   string
	Tint   color.Color
	secret int
	OnHit  func()
}

type Settings struct {
	Bloom   float32
	Enabled bool
}

func heroWorld(t *testing.T) (*world.World, *hero) {
	t.Helper()
	w := world.NewWorld()
	h := &hero{
		BasicEntity: world.NewBasic(),
		Transform:   scene.At(1, 2, 3),
		Stats:       Stats{HP: 10, Tags: []string{"brave"}, Items: map[string]int{"wood": 3}},
		Name:        "Ayla",
		secret:      7,
	}
	w.AddEntity(h)
	world.SetResource(w, &Settings{Bloom: 0.5})
	return w, h
}

func path(h *hero, rest string) string { return "entity/" + strconv.FormatUint(h.ID(), 10) + rest }

func TestGetComponentsAndPromotedFields(t *testing.T) {
	w, h := heroWorld(t)
	v, typ, err := Get(w, path(h, "/Transform/Position"), 0)
	if err != nil || typ != "geom.Vec3" {
		t.Fatalf("%v %v", typ, err)
	}
	if m, ok := v.(map[string]any); !ok || m["X"] != 1.0 || m["Z"] != 3.0 {
		t.Fatalf("position %v", v)
	}
	if v, _, _ = Get(w, path(h, "/Position/Y"), 0); v != 2.0 {
		t.Fatalf("promoted field %v", v)
	}
	if v, _, _ = Get(w, path(h, "/Tags/0"), 0); v != "brave" {
		t.Fatalf("slice index %v", v)
	}
	if v, _, _ = Get(w, path(h, "/Items/wood"), 0); v != 3.0 {
		t.Fatalf("map key %v", v)
	}
}

func TestWholeEntityEncodingSkipsPrivateAndFuncs(t *testing.T) {
	w, h := heroWorld(t)
	v, _, err := Get(w, path(h, ""), 0)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("entity %v", v)
	}
	if _, ok = m["secret"]; ok {
		t.Fatal("unexported fields must not leak")
	}
	if _, ok = m["OnHit"]; ok {
		t.Fatal("funcs must be dropped")
	}
	if basic, isMap := m["BasicEntity"].(map[string]any); m["Name"] != "Ayla" || !isMap || basic["ID"] != float64(h.ID()) {
		t.Fatalf("entity %v", m)
	}
}

func TestSetFieldsStructsAndResources(t *testing.T) {
	w, h := heroWorld(t)
	if _, err := Set(w, path(h, "/HP"), 25.0); err != nil || h.HP != 25 {
		t.Fatalf("hp %d %v", h.HP, err)
	}
	if _, err := Set(w, path(h, "/Transform/Position"), map[string]any{"Y": 9.0}); err != nil {
		t.Fatal(err)
	}
	if h.Position != geom.V3(1, 9, 3) {
		t.Fatalf("partial struct update %v", h.Position)
	}
	if _, err := Set(w, path(h, "/Tags"), []any{"wise", "old"}); err != nil || len(h.Tags) != 2 {
		t.Fatalf("slice %v %v", h.Tags, err)
	}
	if _, err := Set(w, "resource/automation.Settings/Bloom", 1.5); err != nil {
		t.Fatal(err)
	}
	if _, err := Set(w, "resource/Settings/Enabled", true); err != nil {
		t.Fatal(err)
	}
	if s := world.Resource[Settings](w); s.Bloom != 1.5 || !s.Enabled {
		t.Fatalf("resource %+v", s)
	}
}

func TestPathErrors(t *testing.T) {
	w, h := heroWorld(t)
	for _, p := range []string{
		"entity/999999",
		path(h, "/secret"),
		path(h, "/Nope"),
		path(h, "/Tags/5"),
		"resource/Missing",
		"bogus/1",
		"entity",
	} {
		if _, _, err := Get(w, p, 0); err == nil {
			t.Fatalf("%q should fail", p)
		}
	}
	if _, err := Set(w, path(h, "/HP"), "lots"); err == nil {
		t.Fatal("type mismatch must fail")
	}
}

func TestLongSlicesAreTruncated(t *testing.T) {
	long := make([]int, 500)
	v, ok := Encode(reflectValue(long), 3).(map[string]any)
	if !ok {
		t.Fatal("a long slice encodes as a map")
	}
	if head, isList := v["Head"].([]any); v["Len"] != 500.0 || !isList || len(head) != maxItems {
		t.Fatalf("got %v", v["Len"])
	}
}

func TestEntitiesFilterAndDescribe(t *testing.T) {
	w, h := heroWorld(t)
	w.AddEntity(world.NewBasic())
	infos, total := Entities(w, EntityFilter{Component: "Transform"})
	if total != 1 || infos[0].ID != h.ID() || infos[0].Name != "Ayla" {
		t.Fatalf("%d %+v", total, infos)
	}
	if _, total = Entities(w, EntityFilter{Type: "hero", Name: "ayla"}); total != 1 {
		t.Fatal("type and name filters")
	}
	if _, total = Entities(w, EntityFilter{}); total != 2 {
		t.Fatalf("all entities: %d", total)
	}
}

func TestTheWorldsClockAndStatsHaveTheirOwnPaths(t *testing.T) {
	w, _ := heroWorld(t)
	if _, err := Set(w, "time/scale", 0.5); err != nil {
		t.Fatal(err)
	}
	if w.Time().Scale != 0.5 {
		t.Fatalf("scale %v", w.Time().Scale)
	}
	w.Frame(1.0 / 60)
	v, _, err := Get(w, "time/frame", 0)
	if err != nil || fmt.Sprint(v) != "1" {
		t.Fatalf("frame %s %v", v, err)
	}
	if _, _, err = Get(w, "stats/fps", 0); err != nil {
		t.Fatal(err)
	}
	if _, _, err = Get(w, "resource/world.Time/Scale", 0); err == nil {
		t.Fatal("the clock is no resource")
	}
}
