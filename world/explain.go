package world

import (
	"fmt"
	"reflect"
	"strings"
)

// wanter is a group that can say what it collects.
type wanter interface{ wants() reflect.Type }

func (g *Group[T]) wants() reflect.Type { return reflect.TypeFor[T]() }

// holder is a group that tells whether it holds an entity.
type holder interface{ holds(id uint64) bool }

func (g *Group[T]) holds(id uint64) bool { _, ok := g.index[id]; return ok }

// Explain says, for every system and view of w, whether e is in its group
// and, when it is not, what e lacks: the methods of the group's interface
// it misses, and the component fields whose names hide one of them. Print
// it when an entity is silently left out of a system. A group that matches
// e without holding it, or holds it without matching it, was changed
// after e was added: world.Regroup(e) offers it again.
func (w *World) Explain(e Identifier) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%T %d:\n", e, e.ID())
	for _, c := range w.collectors {
		name := fmt.Sprintf("%T", c.target)
		if c.name != "" {
			name = c.name
		}
		h, holds := c.target.(holder)
		switch {
		case c.matcher != nil && c.matcher.Matches(e) && holds && !h.holds(e.ID()):
			fmt.Fprintf(&b, "  out %s: matches since it was added; world.Regroup(e) offers it again\n", name)
		case c.matcher != nil && !c.matcher.Matches(e) && holds && h.holds(e.ID()):
			fmt.Fprintf(&b, "  in  %s: no longer matches; world.Regroup(e) lets it go\n", name)
		case c.matcher != nil && c.matcher.Matches(e):
			fmt.Fprintf(&b, "  in  %s\n", name)
		case c.matcher != nil:
			want, ok := c.matcher.(wanter)
			if !ok {
				fmt.Fprintf(&b, "  out %s\n", name)
				continue
			}
			fmt.Fprintf(&b, "  out %s: wants %s%s\n", name, want.wants(), lacks(e, want.wants()))
		default:
			c.explainInterfaces(&b, name, e)
		}
	}
	return b.String()
}

func (c *collector) explainInterfaces(b *strings.Builder, name string, e Identifier) {
	t := reflect.TypeOf(e)
	for _, x := range c.ex {
		if t.Implements(x) {
			fmt.Fprintf(b, "  out %s: excludes %s\n", name, x)
			return
		}
	}
	for _, i := range c.in {
		if t.Implements(i) {
			fmt.Fprintf(b, "  in  %s\n", name)
			return
		}
	}
	for _, i := range c.in {
		fmt.Fprintf(b, "  out %s: wants %s%s\n", name, i, lacks(e, i))
	}
}

// lacks lists what e misses of want, an interface: each missing method,
// and why when a field of the same name hides it or its signature differs.
func lacks(e Identifier, want reflect.Type) string {
	if want.Kind() != reflect.Interface {
		return ""
	}
	t := reflect.TypeOf(e)
	var why []string
	for i := range want.NumMethod() {
		m := want.Method(i)
		got, ok := t.MethodByName(m.Name)
		switch {
		case ok && sameSignature(got.Type, m.Type):
		case ok:
			why = append(why, fmt.Sprintf("%s is %s, not %s", m.Name, got.Type, m.Type))
		case hasField(t, m.Name):
			why = append(why, fmt.Sprintf("%s is missing: a field named %s hides the method", m.Name, m.Name))
		default:
			why = append(why, m.Name+" is missing")
		}
	}
	if len(why) == 0 {
		return ""
	}
	return "; " + strings.Join(why, "; ")
}

// sameSignature compares a method, its receiver first, with an interface's.
func sameSignature(method, want reflect.Type) bool {
	if method.NumIn()-1 != want.NumIn() || method.NumOut() != want.NumOut() || method.IsVariadic() != want.IsVariadic() {
		return false
	}
	for i := range want.NumIn() {
		if method.In(i+1) != want.In(i) {
			return false
		}
	}
	for i := range want.NumOut() {
		if method.Out(i) != want.Out(i) {
			return false
		}
	}
	return true
}

func hasField(t reflect.Type, name string) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	_, ok := t.FieldByName(name)
	return ok
}
