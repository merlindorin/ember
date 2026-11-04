package main

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/world"
)

// inspector is a test's hold on the inspector through automation alone,
// so the same tests run on every inspector view: properties are found
// under the node named properties, object rows under objects, each by
// the leaf of its ID (see inspector_view.go).
type inspector struct {
	t    *testing.T
	w    *world.World
	auto *automation.Automation
}

func inspect(t *testing.T, w *world.World, auto *automation.Automation) inspector {
	return inspector{t, w, auto}
}

// find is the element whose ID ends in leaf, one part or several split
// by "/", under a node named scope, "" for anywhere; ok false when none
// shows. Parts compare as slugs.
func (in inspector) find(scope, leaf string) (automation.Element, bool) {
	in.t.Helper()
	_, els := in.auto.Snapshot("", "")
	want := strings.Split(leaf, "/")
	for _, e := range els {
		parts := strings.Split(e.ID, "/")
		if len(parts) < len(want) || !sameIDs(parts[len(parts)-len(want):], want) {
			continue
		}
		if scope == "" ||
			slices.ContainsFunc(
				parts[:len(parts)-len(want)],
				func(p string) bool { return automation.Slug(p) == automation.Slug(scope) },
			) {
			return e, true
		}
	}
	return automation.Element{}, false
}

// sameIDs reports whether the ID parts a and b match, as slugs: a view
// may name occluder_group so or occluder-group.
func sameIDs(a, b []string) bool {
	return slices.EqualFunc(a, b, func(x, y string) bool { return automation.Slug(x) == automation.Slug(y) })
}

// root is the inspector's own element and its bounds: the one a view
// records as inspector, else, for the old ember/ui view, whose scroll records
// nothing, the Tool's inspector scroll.
func (in inspector) root(tool *Tool) (string, geom.Rect) {
	if e, ok := in.find("", "inspector"); ok {
		return e.ID, e.Bounds
	}
	return tool.inspector.Path(), tool.inspector.Rect()
}

func (in inspector) must(scope, leaf string) automation.Element {
	in.t.Helper()
	e, ok := in.find(scope, leaf)
	if !ok {
		in.t.Fatalf("no %s %s shows in the inspector", scope, leaf)
	}
	return e
}

// prop is property name's control, and obj the open object's field key.
func (in inspector) prop(name string) automation.Element {
	in.t.Helper()
	return in.must("properties", name)
}
func (in inspector) obj(key string) automation.Element { in.t.Helper(); return in.must("objects", key) }

// line is folded object n's summary line, from 1, and whether it shows.
func (in inspector) line(n int) (automation.Element, bool) {
	return in.find("objects", fmt.Sprintf("object-%d-summary", n))
}

// title is the open object's title.
func (in inspector) title() string { in.t.Helper(); return in.obj("title").Label }

// value is e's value as text: a toggle's "true"/"false", a combo's shown
// item or placeholder, a field's or label's text.
func value(e automation.Element) string {
	switch e.Role {
	case automation.RoleToggle:
		return fmt.Sprint(e.Value)
	case automation.RoleCombo:
		return e.Label
	case automation.RoleLabel:
		return e.Label
	}
	return fmt.Sprint(e.Value)
}

// unset reports whether a property shows its default, not a set value:
// a toggle followed by its <name>-default label, a combo with nothing
// selected, a field empty.
func (in inspector) unset(name string) bool {
	in.t.Helper()
	e := in.prop(name)
	switch e.Role {
	case automation.RoleToggle:
		_, shows := in.find("properties", name+"-default")
		return shows
	case automation.RoleCombo:
		n, _ := e.Value.(int)
		return n < 0
	}
	return value(e) == ""
}

// act queues act with v on e and runs the frames it takes.
func (in inspector) act(e automation.Element, act string, v any) {
	in.t.Helper()
	in.auto.Queue(e.ID, act, v)
	in.w.RunFrames(3)
}

// press presses e: a toggle flips, a line selects its object, a button
// fires.
func (in inspector) press(e automation.Element) { in.t.Helper(); in.act(e, automation.ActPress, nil) }

// choose picks item, by its text or index, in combo e.
func (in inspector) choose(e automation.Element, item any) {
	in.t.Helper()
	in.act(e, automation.ActSelect, item)
}

// typeIn types text into field e and enters it.
func (in inspector) typeIn(e automation.Element, text string) {
	in.t.Helper()
	in.auto.Queue(e.ID, automation.ActSet, text)
	in.w.RunFrames(2)
	in.auto.Queue(e.ID, automation.ActSubmit, nil)
	in.w.RunFrames(3)
}
