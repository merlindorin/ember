package automation //nolint:testpackage // white-box: the act timeout (actTimeout)

import (
	"errors"
	"testing"
)

func enabled() *Registry {
	r := NewRegistry()
	r.SetRecording(true)
	return r
}

func TestActIsDeliveredToTheWidgetAndReportsItsState(t *testing.T) {
	r := enabled()
	done := r.Queue("hud/craft", ActPress, nil)
	if a := r.Take("hud/other"); a != nil {
		t.Fatal("acts are delivered by id")
	}
	a := r.Take("hud/craft")
	if a == nil || a.Kind != ActPress {
		t.Fatalf("got %+v", a)
	}
	r.Record(Element{ID: "hud/craft", Role: RoleButton, Label: "Craft"})
	select {
	case res := <-done:
		if res.Err != nil || res.Element.Label != "Craft" {
			t.Fatalf("result %+v", res)
		}
	default:
		t.Fatal("recording the element must complete the act")
	}
}

func TestActsTimeOutWhenTheElementIsNeverDrawn(t *testing.T) {
	r := enabled()
	done := r.Queue("ghost", ActPress, nil)
	for i := 0; i <= actTimeout; i++ {
		r.EndFrame(uint64(i))
	}
	if res := <-done; !errors.Is(res.Err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", res.Err)
	}
}

func TestSnapshotPublishesTheLastFrameAndFilters(t *testing.T) {
	r := enabled()
	r.Record(Element{ID: "hud/hp", Window: "game"})
	r.Record(Element{ID: "editor/console/command", Window: "tools"})
	if _, els := r.Snapshot("", ""); len(els) != 0 {
		t.Fatal("elements are published at frame end")
	}
	r.EndFrame(7)
	frame, els := r.Snapshot("", "hud")
	if frame != 7 || len(els) != 1 || els[0].ID != "hud/hp" {
		t.Fatalf("frame %d elements %+v", frame, els)
	}
	if _, els = r.Snapshot("tools", ""); len(els) != 1 {
		t.Fatal("window filter")
	}
	r.EndFrame(8)
	if _, els = r.Snapshot("", ""); len(els) != 0 {
		t.Fatal("each frame publishes only what it recorded")
	}
}

func TestDuplicateIDsStayAddressable(t *testing.T) {
	r := enabled()
	r.Record(Element{ID: "list/item"})
	r.Record(Element{ID: "list/item"})
	r.EndFrame(1)
	_, els := r.Snapshot("", "")
	if els[0].ID != "list/item" || els[1].ID != "list/item#2" {
		t.Fatalf("ids %q %q", els[0].ID, els[1].ID)
	}
}

func TestDisabledRegistryIsFree(t *testing.T) {
	r := NewRegistry()
	r.Record(Element{ID: "x"})
	r.EndFrame(1)
	if _, els := r.Snapshot("", ""); len(els) != 0 {
		t.Fatal("disabled registry must not record")
	}
	var nilRegistry *Registry
	if nilRegistry.Take("x") != nil {
		t.Fatal("nil registry is a no-op")
	}
}

func TestActions(t *testing.T) {
	r := enabled()
	allowed := false
	r.Define(Action{
		ID:      "craft.axe",
		Enabled: func() bool { return allowed },
		Run:     func(args map[string]any) (any, error) { return args["n"], nil },
	})
	r.Define(Action{ID: "cast.fireball", Run: func(map[string]any) (any, error) { return "boom", nil }})
	if got := r.Actions("craft"); len(got) != 1 || got[0].ID != "craft.axe" {
		t.Fatalf("prefix filter: %+v", got)
	}
	if _, err := r.Invoke("craft.axe", nil); err == nil {
		t.Fatal("disabled action must refuse")
	}
	allowed = true
	if out, err := r.Invoke("craft.axe", map[string]any{"n": 2.0}); err != nil || out != 2.0 {
		t.Fatalf("%v %v", out, err)
	}
	if _, err := r.Invoke("nope", nil); err == nil {
		t.Fatal("unknown action must fail")
	}
}

func TestSlugAndJoin(t *testing.T) {
	for in, want := range map[string]string{
		"Craft Axe!":        "craft-axe",
		"Ember Nova ":       "ember-nova",
		"Remove##entity-42": "entity-42",
		"1/2 Wood":          "1-2-wood",
	} {
		if got := Slug(in); got != want {
			t.Fatalf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
	if got := Join("hud", "", "/craft/", "axe"); got != "hud/craft/axe" {
		t.Fatalf("Join = %q", got)
	}
}

func TestZeroRegistryIsUsable(t *testing.T) {
	var r Registry
	r.Define(Action{ID: "a", Run: func(map[string]any) (any, error) { return 1, nil }})
	if out, err := r.Invoke("a", nil); err != nil || out != 1 {
		t.Fatalf("%v %v", out, err)
	}
	r.SetRecording(true)
	done := r.Queue("x", ActPress, nil)
	r.Take("x")
	r.Record(Element{ID: "x"})
	if res := <-done; res.Err != nil {
		t.Fatal(res.Err)
	}
}

func TestRecordingIsOffUntilTurnedOn(t *testing.T) {
	var none *Registry
	if none.Recording() || New().Recording() {
		t.Fatal("no server, no recording")
	}
	if !New(Recording()).Recording() {
		t.Fatal("the Recording option records")
	}
	r := NewRegistry()
	r.Record(Element{ID: "a"})
	r.SetRecording(true)
	r.Record(Element{ID: "b"})
	r.EndFrame(1)
	if _, all := r.Snapshot("", ""); len(all) != 1 || all[0].ID != "b" {
		t.Fatalf("only what was drawn while recording: %v", all)
	}
	r.SetRecording(false)
	if r.Recording() {
		t.Fatal("SetRecording(false) stops it")
	}
}
