package main

import (
	"fmt"
	"slices"
	"strconv"
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

// another is a value of d other than now, which the tile shows.
func another(d twod.Property, now string) string {
	switch d.Kind {
	case twod.KindBool:
		if now == "true" {
			return "false"
		}
		return "true"
	case twod.KindEnum:
		for _, v := range d.Values {
			if v != now {
				return v
			}
		}
	case twod.KindInt:
		n := int(d.Min) + 1
		if d.Max <= d.Min {
			n = 3
		}
		if strconv.Itoa(n) == now {
			n++
		}
		return strconv.Itoa(n)
	case twod.KindFloat:
		return "2.5"
	case twod.KindString, twod.KindColor, twod.KindClass, twod.KindFile, twod.KindObject:
	}
	return "x"
}

// set changes property d of the picked tile to v through its own control
// in the panel.
func set(t *testing.T, w *world.World, _ *Tool, auto *automation.Automation, d twod.Property, v string) {
	t.Helper()
	in := inspect(t, w, auto)
	e := in.prop(d.Name)
	switch {
	case d.Kind == twod.KindBool:
		in.act(e, automation.ActSet, v == "true")
	case e.Role == automation.RoleCombo:
		in.choose(e, v)
	default:
		in.typeIn(e, v)
	}
}

// shown is property d of Tiled tile 27, which has no shapes so that solid
// shows, or for shadow of tile 0, which has some to cast with, as the
// preview's tileset reads it, its default when it isn't set.
func shown(tool *Tool, d twod.Property) string {
	id := 28
	if d.Name == "shadow" {
		id = 1
	}
	if d.Name == "tall" {
		return fmt.Sprint(tool.tiles.Tall(id))
	}
	if v, ok := tool.tiles.Value(id, d.Name); ok {
		return v.String()
	}
	return fmt.Sprint(d.Default)
}

func TestEveryPropertyChangedInThePanelShowsInTheNextFramesPreview(t *testing.T) {
	for _, static := range []bool{false, true} {
		for _, d := range engineProperties() {
			t.Run(fmt.Sprintf("%s static %v", d.Name, static), func(t *testing.T) {
				w, tool, auto, press := driven(t, copyOfWalls(t))
				if d.Name == "shadow" {
					press("tile-0")
				} else {
					press("tile-27")
				}
				if static {
					press("view")
					press("view/static")
				}
				w.RunFrames(2)
				versions := bakedVersions(w, tool)
				occluders := len(draw.LightingFor(w, tool.win).Occluders.Vertices)
				v := another(d, shown(tool, d))
				set(t, w, tool, auto, d, v)
				w.RunFrames(2)
				if got := shown(tool, d); got != v {
					t.Fatalf("the preview's tile reads %s %q, set to %q", d.Name, got, v)
				}
				if d.Name == "shadow" && len(draw.LightingFor(w, tool.win).Occluders.Vertices) == occluders {
					t.Fatalf("tile 0's shadow none takes its occluders out of the preview: %d", occluders)
				}
				if static && slices.Equal(versions, bakedVersions(w, tool)) {
					t.Fatalf("the static lamp's bake is drawn again after %s changes: %v", d.Name, versions)
				}
			})
		}
	}
}

// bakedVersions are the versions of the static bake's tiles in view.
func bakedVersions(w *world.World, tool *Tool) []uint64 {
	l := draw.LightingFor(w, tool.win)
	if l == nil || l.Static == nil {
		return nil
	}
	var out []uint64
	for _, tile := range l.Static.Tiles {
		out = append(out, tile.Version)
	}
	return out
}
